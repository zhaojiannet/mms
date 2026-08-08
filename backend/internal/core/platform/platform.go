// Package platform 运营后台：操作员登录、商户申请审批、商户与套餐管理
//
// CUSTOM: 本包用裸 pgx 而非 sqlc——查询要在同一事务里交替切换 app.platform_op /
// app.current_tenant 两种 RLS 上下文，且多为一次性运营操作，不适合 sqlc 的固定租户模型
package platform

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	coreauth "github.com/zhaojiannet/mms/backend/internal/core/auth"
	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

var dbPool *pgxpool.Pool

// dummyHash 对不存在的操作员也跑一次密码校验，保持等时响应防账号枚举
var dummyHash string

// Init 由 main.go 注入 pool（登录与公开申请不经过事务中间件，直接用 pool）
func Init(pool *pgxpool.Pool) error {
	dbPool = pool
	h, err := auth.HashPassword("dummy-never-matches")
	if err != nil {
		return err
	}
	dummyHash = h
	return nil
}

const genericLoginErr = "邮箱或密码错误"

type loginRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	CaptchaID     string `json:"captcha_id,omitempty"`
	CaptchaAnswer string `json:"captcha_answer,omitempty"`
}

// Login POST /api/platform/login（路由层挂 IP 限流）
func Login(c *echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Email == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, genericLoginErr)
	}
	ctx := c.Request().Context()

	var (
		id          uuid.UUID
		hash, name  string
		status      string
		ver         int32
		failed      int
		lockedUntil *time.Time
	)
	err := dbPool.QueryRow(ctx, `
		SELECT id, password_hash, name, status, token_version, failed_login_attempts, locked_until
		FROM platform_operators WHERE email = $1
	`, req.Email).Scan(&id, &hash, &name, &status, &ver, &failed, &lockedUntil)
	if err != nil || status != "active" {
		// 不存在与已停用一律走 dummy 校验 + 统一错误：防账号枚举与时序探测（与商户登录同策略）
		_, _ = auth.VerifyPassword(req.Password, dummyHash)
		return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
	}

	// 验证码与锁定的每个失败分支都回同一句 generic 错误：
	// 任何差异化响应都会把「该邮箱是不是操作员」告诉攻击者（与商户登录同策略）
	// 带了 captcha 就必须对；锁定期内需正确 captcha 才能继续（允许被恶意锁号者自救）
	captchaProvided := req.CaptchaID != "" && req.CaptchaAnswer != ""
	captchaOK := false
	if captchaProvided {
		captchaOK = coreauth.VerifyCaptcha(req.CaptchaID, req.CaptchaAnswer)
		if !captchaOK {
			_, _ = auth.VerifyPassword(req.Password, dummyHash)
			return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
		}
	}
	if failed >= 2 && !captchaOK {
		_, _ = auth.VerifyPassword(req.Password, dummyHash)
		return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
	}
	if lockedUntil != nil && time.Now().Before(*lockedUntil) && !captchaOK {
		_, _ = auth.VerifyPassword(req.Password, dummyHash)
		return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
	}

	ok, err := auth.VerifyPassword(req.Password, hash)
	if err != nil || !ok {
		_, _ = dbPool.Exec(ctx, `
			UPDATE platform_operators
			SET failed_login_attempts = failed_login_attempts + 1,
			    locked_until = CASE WHEN failed_login_attempts + 1 >= 5
			                        THEN now() + interval '15 minutes' END,
			    updated_at = now()
			WHERE id = $1
		`, id)
		return echo.NewHTTPError(http.StatusUnauthorized, genericLoginErr)
	}

	_, _ = dbPool.Exec(ctx, `
		UPDATE platform_operators
		SET failed_login_attempts = 0, locked_until = NULL, last_login_at = now(), updated_at = now()
		WHERE id = $1
	`, id)

	token, expiresAt, err := auth.SignOperatorToken(id, req.Email, ver)
	if err != nil {
		return mw.InternalError(c, "platform.login.sign", err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"access_token": token,
		"expires_at":   expiresAt,
		"operator":     map[string]any{"email": req.Email, "name": name},
	})
}

// Me GET /api/platform/me（前端启动时校验 token 有效性）
func Me(c *echo.Context) error {
	op := mw.OperatorFrom(c)
	return c.JSON(http.StatusOK, map[string]any{"email": op.Email, "name": op.Name})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword POST /api/platform/password（路由层挂密码类限流）
//
// 运营账号原先只能由 `PLATFORM_ADMIN_*` 首次启动创建、之后无法更换，
// 疑似泄露时只剩连库改 hash 一条路。与商户改密同策略：验当前密码、
// 递增 token_version 吊销全部已签发 token（含本次会话，前端随后重新登录）。
func ChangePassword(c *echo.Context) error {
	var req changePasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if len(req.NewPassword) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "新密码至少 8 位")
	}
	if req.NewPassword == req.CurrentPassword {
		return echo.NewHTTPError(http.StatusBadRequest, "新密码不能与当前密码相同")
	}

	ctx := c.Request().Context()
	tx := mw.TxFrom(c)
	op := mw.OperatorFrom(c)

	var hash string
	if err := tx.QueryRow(ctx,
		"SELECT password_hash FROM platform_operators WHERE id = $1", op.ID,
	).Scan(&hash); err != nil {
		return mw.InternalError(c, "platform.password.load", err)
	}
	ok, err := auth.VerifyPassword(req.CurrentPassword, hash)
	if err != nil || !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "当前密码不正确")
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return mw.InternalError(c, "platform.password.hash", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE platform_operators
		SET password_hash = $2, token_version = token_version + 1,
		    password_changed_at = now(), failed_login_attempts = 0, locked_until = NULL,
		    updated_at = now()
		WHERE id = $1
	`, op.ID, newHash); err != nil {
		return mw.InternalError(c, "platform.password.update", err)
	}

	slog.Info("platform: operator password changed", "operator", op.Email)
	return c.NoContent(http.StatusNoContent)
}

// --------------- 共享工具 ---------------

// passwordAlphabet 去掉易混字符（0O1lI）的随机密码字符集
const passwordAlphabet = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// genPassword 生成 16 位一次性初始密码（crypto/rand）
func genPassword() (string, error) {
	b := make([]byte, 16)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = passwordAlphabet[n.Int64()]
	}
	return string(b), nil
}

// setTenantCtx 在平台事务内切入某租户的 RLS 上下文（写 users / subscriptions 用）
// 同时清掉 platform_op 哨兵：切入单租户后跨租户读策略即不再需要，
// 后续查询若漏写租户过滤仍有 RLS 保护（防御纵深）
func setTenantCtx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant', $1, true), set_config('app.platform_op', '', true)",
		tenantID.String())
	return err
}

// editionByCode 查套餐；不存在返回 (0, nil, pgx.ErrNoRows)
func editionByCode(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, code string) (int16, json.RawMessage, error) {
	var id int16
	var quotas json.RawMessage
	err := q.QueryRow(ctx, "SELECT id, quotas FROM editions WHERE code = $1 AND active", code).Scan(&id, &quotas)
	return id, quotas, err
}
