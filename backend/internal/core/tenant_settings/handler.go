// Package tenantsettings 提供 KV 配置管理 + 特殊的 "开启撤单" / "生成预约码" 接口
package tenantsettings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
		return echo.NewHTTPError(http.StatusInternalServerError, "list: "+err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, "upsert: "+err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, "delete: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

// ================ 特殊接口 ================

// EnableVoidRequest：开启撤单功能需要输入"验证密码"（用户密码 + "!!!" 后缀）
type EnableVoidRequest struct {
	Password string `json:"password"`
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

	// 必须以 !!! 结尾
	if !strings.HasSuffix(req.Password, "!!!") {
		recordAttempt(key)
		return echo.NewHTTPError(http.StatusBadRequest, "验证密码错误")
	}
	actualPwd := strings.TrimSuffix(req.Password, "!!!")

	// 查用户密码 hash
	user, err := q.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "get user: "+err.Error())
	}
	ok, verr := auth.VerifyPassword(actualPwd, user.PasswordHash)
	if verr != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "verify password: "+verr.Error())
	}
	if !ok {
		recordAttempt(key)
		return echo.NewHTTPError(http.StatusBadRequest, "验证密码错误")
	}

	// 成功：清计数 + upsert 两条配置
	clearAttempts(key)

	if err := setBool(ctx, q, t.ID, "enable_transaction_void", true); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	if err := setString(ctx, q, t.ID, "void_enabled_at", &nowISO); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if err := setString(ctx, q, t.ID, "void_enabled_at", nil); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{"enabled": false})
}

// RegenerateBookingCode POST /api/tenant-settings/booking-code  body: {"code":"optional"}
// 若不传 code：自动生成 8 位 hex
// 传 code：必须 2-20 位 字母数字下划线
type BookingCodeRequest struct {
	Code *string `json:"code"`
}

func RegenerateBookingCode(c *echo.Context) error {
	var req BookingCodeRequest
	_ = c.Bind(&req)
	var newCode string
	if req.Code != nil && *req.Code != "" {
		if !isValidBookingCode(*req.Code) {
			return echo.NewHTTPError(http.StatusBadRequest, "code 只能 2-20 位字母/数字/下划线")
		}
		newCode = *req.Code
	} else {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "gen code: "+err.Error())
		}
		newCode = hex.EncodeToString(b)
	}

	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()
	if err := setString(ctx, q, t.ID, "booking_code", &newCode); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	nowISO := time.Now().UTC().Format(time.RFC3339)
	if err := setString(ctx, q, t.ID, "booking_code_updated_at", &nowISO); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]any{
		"booking_code": newCode,
		"updated_at":   nowISO,
	})
}

// --- helpers ---

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
