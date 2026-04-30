// Package tenantsettings 提供 KV 配置管理 + 特殊的 "开启撤单" / "生成预约码" 接口
package tenantsettings

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type SettingDTO struct {
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// List GET /api/tenant-settings
func List(c *echo.Context) error {
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	rows, err := q.ListTenantSettings(c.Request().Context(), t.ID)
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}
	items := make([]SettingDTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, SettingDTO{Key: r.Key, Value: r.Value, UpdatedAt: r.UpdatedAt.Time})
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Get GET /api/tenant-settings/:key
func Get(c *echo.Context) error {
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	row, err := q.GetTenantSetting(c.Request().Context(), sqlc.GetTenantSettingParams{
		TenantID: t.ID, Key: c.Param("key"),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "setting not found")
		}
		return mw.InternalError(c, "get: ", err)
	}
	return c.JSON(http.StatusOK, SettingDTO{Key: row.Key, Value: row.Value, UpdatedAt: row.UpdatedAt.Time})
}

// Upsert PUT /api/tenant-settings/:key   body: {"value": ...}
// 简单 KV：key 已知就覆盖，不存在就创建。不走预设开关逻辑（走 EnableVoid / RegenerateBookingCode 等特殊接口）
func Upsert(c *echo.Context) error {
	key := c.Param("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "key required")
	}
	var body struct {
		Value json.RawMessage `json:"value"`
	}
	if err := c.Bind(&body); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid body: "+err.Error())
	}
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	row, err := q.UpsertTenantSetting(c.Request().Context(), sqlc.UpsertTenantSettingParams{
		TenantID: t.ID, Key: key, Value: body.Value,
	})
	if err != nil {
		return mw.InternalError(c, "upsert: ", err)
	}
	return c.JSON(http.StatusOK, SettingDTO{Key: row.Key, Value: row.Value, UpdatedAt: row.UpdatedAt.Time})
}

// Delete DELETE /api/tenant-settings/:key
func Delete(c *echo.Context) error {
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	if err := q.DeleteTenantSetting(c.Request().Context(), sqlc.DeleteTenantSettingParams{
		TenantID: t.ID, Key: c.Param("key"),
	}); err != nil {
		return mw.InternalError(c, "delete: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ================ 公开店铺展示信息 ================

// StoreInfoDTO 登录页等未登录场景使用的店铺展示信息
type StoreInfoDTO struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`     // 展示名：优先 store_name 覆盖，否则 tenants.name
	LoginBg string `json:"login_bg"` // 背景主题 key，前端映射到样式
	LogoURL string `json:"logo_url"` // 店铺 logo 相对路径（前端拼 apiBase）
}

// StoreInfo GET /api/store/info
// 公开接口：返回当前租户的展示名 + 登录背景主题 + logo URL
func StoreInfo(c *echo.Context) error {
	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	dto := StoreInfoDTO{Slug: t.Slug, Name: t.Name, LoginBg: "beauty"}
	if s, err := getStringSetting(ctx, q, t.ID, "store_name"); err == nil && s != "" {
		dto.Name = s
	}
	if s, err := getStringSetting(ctx, q, t.ID, "login_bg_theme"); err == nil && s != "" {
		dto.LoginBg = s
	}
	if s, err := getStringSetting(ctx, q, t.ID, "store_logo_url"); err == nil && s != "" {
		dto.LogoURL = s
	}
	return c.JSON(http.StatusOK, dto)
}

func getStringSetting(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, key string) (string, error) {
	row, err := q.GetTenantSetting(ctx, sqlc.GetTenantSettingParams{TenantID: tenantID, Key: key})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	var s string
	if len(row.Value) > 0 {
		_ = json.Unmarshal(row.Value, &s)
	}
	return s, nil
}

// ================ 特殊接口 ================

// EnableVoidRequest：开启撤单功能需要输入"超级密码"（管理员在"个人设置"中单独设置）
type EnableVoidRequest struct {
	Password string `json:"password"`
}

// SetSuperPasswordRequest：设置/修改超级密码（需要当前登录密码确认）
type SetSuperPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// 内存中的错误次数限制（每 user 3 次，锁 5 分钟）
var (
	voidAttemptsMu sync.Mutex
	voidAttempts   = map[string]*attemptRecord{}
)

type attemptRecord struct {
	count       int
	lockedUntil time.Time
}

const (
	maxVoidAttempts = 3
	voidLockTime    = 5 * time.Minute
)

// EnableVoid POST /api/tenant-settings/enable-void
// body: {"password": "userpwd!!!"}
// 开启后 10 分钟自动过期（由 transactions.Void 按需检查）
func EnableVoid(c *echo.Context) error {
	var req EnableVoidRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid: "+err.Error())
	}
	if req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "password required")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	claims := mw.ClaimsFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	// 校验锁定
	key := claims.UserID.String()
	voidAttemptsMu.Lock()
	rec, ok := voidAttempts[key]
	if ok && !rec.lockedUntil.IsZero() && time.Now().Before(rec.lockedUntil) {
		remaining := int(time.Until(rec.lockedUntil).Seconds())
		voidAttemptsMu.Unlock()
		return echo.NewHTTPError(http.StatusTooManyRequests,
			fmt.Sprintf("密码错误次数过多，请 %d 秒后重试", remaining))
	}
	voidAttemptsMu.Unlock()

	// 超级密码必须已设置
	superHash, err := getSuperPasswordHash(ctx, q, t.ID)
	if err != nil {
		return mw.InternalError(c, "get super password: ", err)
	}
	if superHash == "" {
		return echo.NewHTTPError(http.StatusPreconditionRequired, "尚未设置超级密码，请在个人设置中先设置")
	}

	ok, verr := auth.VerifyPassword(req.Password, superHash)
	if verr != nil {
		return mw.InternalError(c, "tenant_settings.verify_super", verr)
	}
	if !ok {
		recordAttempt(key)
		return echo.NewHTTPError(http.StatusBadRequest, "超级密码错误")
	}

	// 成功：清计数 + upsert 两条配置
	clearAttempts(key)

	if err := setBool(ctx, q, t.ID, "enable_transaction_void", true); err != nil {
		return mw.InternalError(c, "tenant_settings.enable_void", err)
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	if err := setString(ctx, q, t.ID, "void_enabled_at", &nowISO); err != nil {
		return mw.InternalError(c, "tenant_settings.set_void_at", err)
	}
	// 记录开启者 ID：多超管租户时，Void handler 校验 claims.UserID == enabler
	// 防止 "A 开了 B 趁窗口撤单" 的攻击
	enablerID := claims.UserID.String()
	if err := setString(ctx, q, t.ID, "void_enabled_by", &enablerID); err != nil {
		return mw.InternalError(c, "tenant_settings.set_void_by", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"enabled":          true,
		"enabled_at":       nowISO,
		"auto_disable_sec": 600,
	})
}

// DisableVoid POST /api/tenant-settings/disable-void
func DisableVoid(c *echo.Context) error {
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()
	if err := setBool(ctx, q, t.ID, "enable_transaction_void", false); err != nil {
		return mw.InternalError(c, "tenant_settings.disable_void", err)
	}
	if err := setString(ctx, q, t.ID, "void_enabled_at", nil); err != nil {
		return mw.InternalError(c, "tenant_settings.clear_void_at", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"enabled": false})
}

// RegenerateBookingCode POST /api/tenant-settings/booking-code  body: {"code":"optional"}
// 若不传 code：自动生成 4 位数字
// 传 code：必须 2-20 位 字母数字下划线
type BookingCodeRequest struct {
	Code *string `json:"code"`
}

func RegenerateBookingCode(c *echo.Context) error {
	var req BookingCodeRequest
	// Bind 失败（无 body / 非法 JSON）按"未传 code"走自动生成分支，是有意 UX
	_ = c.Bind(&req)
	var newCode string
	if req.Code != nil && *req.Code != "" {
		if !isValidBookingCode(*req.Code) {
			return echo.NewHTTPError(http.StatusBadRequest, "code 只能 2-20 位字母/数字/下划线")
		}
		newCode = *req.Code
	} else {
		// 8 位字母+数字（无易混淆字符），约 2.8 * 10^12 种 — 防暴力枚举
		newCode = randomBookingCode(8)
	}

	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()
	if err := setString(ctx, q, t.ID, "booking_code", &newCode); err != nil {
		return mw.InternalError(c, "tenant_settings.set_booking_code", err)
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	if err := setString(ctx, q, t.ID, "booking_code_updated_at", &nowISO); err != nil {
		return mw.InternalError(c, "tenant_settings.set_booking_updated_at", err)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"booking_code": newCode,
		"updated_at":   nowISO,
	})
}

// SetSuperPassword POST /api/tenant-settings/super-password
// body: { current_password, new_password }
// 要求当前登录用户确认本人登录密码，然后设置 tenant 级超级密码
func SetSuperPassword(c *echo.Context) error {
	var req SetSuperPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid: "+err.Error())
	}
	if len(req.NewPassword) < 8 {
		return echo.NewHTTPError(http.StatusBadRequest, "新超级密码至少 8 位")
	}
	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	user := mw.UserFrom(c)
	ok, verr := auth.VerifyPassword(req.CurrentPassword, user.PasswordHash)
	if verr != nil || !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "当前登录密码不正确")
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return mw.InternalError(c, "hash: ", err)
	}
	if err := setString(ctx, q, t.ID, "super_password_hash", &hash); err != nil {
		return mw.InternalError(c, "tenant_settings.set_super_password", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// SuperPasswordStatus GET /api/tenant-settings/super-password/status
func SuperPasswordStatus(c *echo.Context) error {
	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	hash, err := getSuperPasswordHash(ctx, q, t.ID)
	if err != nil {
		return mw.InternalError(c, "tenant_settings.get_super_password", err)
	}
	return c.JSON(http.StatusOK, map[string]bool{"is_set": hash != ""})
}

// --- helpers ---

func getSuperPasswordHash(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID) (string, error) {
	row, err := q.GetTenantSetting(ctx, sqlc.GetTenantSettingParams{
		TenantID: tenantID, Key: "super_password_hash",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	var s string
	if len(row.Value) > 0 {
		_ = json.Unmarshal(row.Value, &s)
	}
	return s, nil
}

func setBool(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, key string, v bool) error {
	raw, _ := json.Marshal(v)
	_, err := q.UpsertTenantSetting(ctx, sqlc.UpsertTenantSettingParams{
		TenantID: tenantID, Key: key, Value: raw,
	})
	return err
}

func setString(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, key string, v *string) error {
	raw, _ := json.Marshal(v)
	_, err := q.UpsertTenantSetting(ctx, sqlc.UpsertTenantSettingParams{
		TenantID: tenantID, Key: key, Value: raw,
	})
	return err
}

func recordAttempt(key string) {
	voidAttemptsMu.Lock()
	defer voidAttemptsMu.Unlock()
	rec, ok := voidAttempts[key]
	if !ok {
		rec = &attemptRecord{}
		voidAttempts[key] = rec
	}
	rec.count++
	if rec.count >= maxVoidAttempts {
		rec.lockedUntil = time.Now().Add(voidLockTime)
	}
}

func clearAttempts(key string) {
	voidAttemptsMu.Lock()
	defer voidAttemptsMu.Unlock()
	delete(voidAttempts, key)
}

// randomBookingCode 生成 n 位 booking_code（字母+数字，去掉 O/0/I/l 等易混淆字符）
// 约 52^8 ≈ 5.3 万亿种组合，防暴力枚举
func randomBookingCode(n int) string {
	const charset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789abcdefghjkmnpqrstuvwxyz"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// 极不可能：fallback 用时间戳避免空串
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	out := make([]byte, n)
	for i, v := range b {
		out[i] = charset[int(v)%len(charset)]
	}
	return string(out)
}

func isValidBookingCode(s string) bool {
	if len(s) < 2 || len(s) > 20 {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}
