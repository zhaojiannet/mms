package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
	"golang.org/x/crypto/bcrypt"

	pauth "github.com/zhaojiannet/mms/backend/internal/platform/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// --------------- package state ---------------

// 独立 pool 用于失败计数的 mini tx（不受业务 handler tx rollback 影响）
var dbPool *pgxpool.Pool

// dummyBcryptHash 对不存在的用户走一次 CompareHashAndPassword 保持等时响应
// 初始化时一次性生成，避免运行时增加 bcrypt 开销
var dummyBcryptHash []byte

// Init 由 main.go 启动时调用：注入 pool，预生成 dummy hash，启动 captcha GC
func Init(pool *pgxpool.Pool) error {
	dbPool = pool
	h, err := bcrypt.GenerateFromPassword([]byte("dummy-never-matches"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	dummyBcryptHash = h
	startCaptchaGC()
	return nil
}

// genericLoginErr 所有登录失败统一返回同一错误，防止账号枚举
const genericLoginErr = "邮箱或密码错误"

// --------------- types ---------------

type LoginRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	Remember      bool   `json:"remember"`                 // 信任此设备：7 天免登录
	CaptchaID     string `json:"captcha_id,omitempty"`     // 图形验证码 ID（可选）
	CaptchaAnswer string `json:"captcha_answer,omitempty"` // 用户输入的验证码
}

type LoginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        UserDTO   `json:"user"`
}

type UserDTO struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
	Role  string    `json:"role"`
}

// --------------- handlers ---------------

// LoginHandler POST /api/login
//
// 安全措施：
//   - 统一错误消息 "邮箱或密码错误" — 防账号枚举
//   - 用户不存在时也跑一次 dummy bcrypt — 防时序攻击
//   - 密码错误时用独立 pool 连接写失败计数 — 防业务 tx rollback 导致锁定失效
//   - 依赖 TenantResolver + TenantTx（SET LOCAL app.current_tenant）提供 RLS 隔离
func LoginHandler(c *echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "邮箱和密码不能为空")
	}

	tenant := mw.TenantFrom(c)
	queries := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()

	user, err := queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 等时：不存在用户也消耗一次 bcrypt 时间
			_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(req.Password))
			return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
		}
		return mw.InternalError(c, "lookup user: ", err)
	}

	// 账号被停用：统一 generic 错误（不透露状态）
	if user.Status != "active" {
		_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(req.Password))
		return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
	}

	// 验证码校验：如果请求带了 captcha，必须校验通过；否则统一返回 generic 错误
	// 不暴露"captcha 错误"和"密码错误"的差异，防账号枚举 + 防验证码穷举
	captchaProvided := req.CaptchaID != "" && req.CaptchaAnswer != ""
	captchaOK := false
	if captchaProvided {
		captchaOK = verifyCaptcha(req.CaptchaID, req.CaptchaAnswer)
		if !captchaOK {
			_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(req.Password))
			return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
		}
	}

	// 账号锁定中：需要正确的 captcha 才能继续到密码校验（允许被 DoS 锁号的用户自救）
	// 没 captcha / captcha 错 → 拒绝
	locked := user.LockedUntil.Valid && user.LockedUntil.Time.After(time.Now())
	if locked && !captchaOK {
		_ = bcrypt.CompareHashAndPassword(dummyBcryptHash, []byte(req.Password))
		return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
	}

	ok, err := pauth.VerifyPassword(req.Password, user.PasswordHash)
	if err != nil {
		return mw.InternalError(c, "verify password: ", err)
	}
	if !ok {
		// 独立 pool 连接写失败计数：绕过业务 tx 的 rollback
		recordFailureIndependently(tenant.ID, user.ID)
		return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
	}

	// 成功：UpdateUserLastLogin 同时清零 failed_login_attempts + locked_until
	// （包含"锁定期内 captcha + 密码都对"的自救场景）
	if err := queries.UpdateUserLastLogin(ctx, user.ID); err != nil {
		slog.Warn("update last_login_at failed", "user_id", user.ID, "err", err)
	}

	// token 版本：当前 DB 的 token_version（改密/降级后会 +1 立即失效旧 token）
	// 新登录无 originalIssuedAt，iat = now（从此刻起计算 30 天绝对寿命）
	token, expiresAt, err := pauth.SignAccessToken(user.ID, tenant.ID, user.Email, user.Role, req.Remember, user.TokenVersion, nil)
	if err != nil {
		return mw.InternalError(c, "sign token: ", err)
	}

	return c.JSON(http.StatusOK, LoginResponse{
		AccessToken: token,
		ExpiresAt:   expiresAt,
		User: UserDTO{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Role:  user.Role,
		},
	})
}

// recordFailureIndependently 用独立 pool 连接执行 RecordLoginFailure
//
// 关键点：
//   - 不走业务 handler 的 tx（handler 返回 401 后 tx 会 rollback，失败计数会丢）
//   - 必须 SET LOCAL tenant 才能穿过 RLS 更新 users 表
//   - 失败不影响主流程（只 log）
func recordFailureIndependently(tenantID, userID uuid.UUID) {
	if dbPool == nil {
		slog.Error("auth.dbPool not initialized; call auth.Init(pool) in main")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		slog.Warn("record failure: begin tx", "err", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 用 set_config(..., true)（SET LOCAL 不支持预处理参数）
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		slog.Warn("record failure: set tenant", "err", err)
		return
	}

	if _, err := sqlc.New(tx).RecordLoginFailure(ctx, sqlc.RecordLoginFailureParams{
		ID:          userID,
		MaxAttempts: 5,
		LockMinutes: 15,
	}); err != nil {
		slog.Warn("record failure: update", "err", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Warn("record failure: commit", "err", err)
	}
}

// RefreshHandler POST /api/auth/refresh
//
// 安全措施：
//   - 从 DB 重取 role/status/token_version，被 disable/降级/改密后立即拒绝
//   - 保留原 token 的 iat：refresh 不重置"绝对寿命"，防永续会话攻击
//     攻击场景：被盗 token → 每日 refresh 永久续 → 有 MaxTokenLifetime 则最多 30 天
//   - 不保留 remember 标志：每次 refresh 按 access TTL 签（更安全）
func RefreshHandler(c *echo.Context) error {
	claims := mw.ClaimsFrom(c)
	tenant := mw.TenantFrom(c)

	// RequireAuth 已校验 status == active 且 claims.Ver == u.TokenVersion，下游直接复用
	u := mw.UserFrom(c)
	// 绝对寿命校验：原 iat + MaxTokenLifetime < now 则拒绝
	origIat := claims.IssuedAt.Time
	if time.Since(origIat) > pauth.MaxTokenLifetime {
		return echo.NewHTTPError(http.StatusUnauthorized, "会话已超过最长有效期，请重新登录")
	}

	// 保留原 iat，防攻击者通过不断 refresh 拉长寿命
	token, expiresAt, err := pauth.SignAccessToken(u.ID, tenant.ID, u.Email, u.Role, false, u.TokenVersion, &origIat)
	if err != nil {
		return mw.InternalError(c, "sign token: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"access_token": token,
		"expires_at":   expiresAt,
	})
}

// LogoutHandler POST /api/auth/logout
//
// 服务端吊销：token_version +1，该用户所有已签发 token（含被盗的 remember
// 7 天长 token）立即失效。语义是"登出所有设备"——无状态 JWT 下不引入
// refresh_token 存储的最小吊销手段
func LogoutHandler(c *echo.Context) error {
	u := mw.UserFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	if err := q.IncrementUserTokenVersion(c.Request().Context(), u.ID); err != nil {
		return mw.InternalError(c, "logout: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"message": "logged out"})
}

// ChangePasswordRequest 修改当前登录用户的密码
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// ChangePasswordHandler POST /api/auth/change-password
func ChangePasswordHandler(c *echo.Context) error {
	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if len(req.NewPassword) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "新密码至少 8 位")
	}

	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))

	user := mw.UserFrom(c)
	ok, err := pauth.VerifyPassword(req.OldPassword, user.PasswordHash)
	if err != nil || !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "当前密码不正确")
	}
	hash, err := pauth.HashPassword(req.NewPassword)
	if err != nil {
		return mw.InternalError(c, "hash: ", err)
	}
	if err := q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID: user.ID, PasswordHash: hash,
	}); err != nil {
		return mw.InternalError(c, "update: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}
