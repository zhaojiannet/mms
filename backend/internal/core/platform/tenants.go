package platform

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

type tenantDTO struct {
	ID          uuid.UUID  `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	Edition     *string    `json:"edition"`      // 无订阅行 = null（hosted 下按 free 限额执行）
	EditionName *string    `json:"edition_name"`
	PeriodEnd   *time.Time `json:"current_period_end"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ListTenants GET /api/platform/tenants
// 每租户取最新一条订阅（created_at 最新者生效，与限额读取口径一致）
func ListTenants(c *echo.Context) error {
	rows, err := mw.TxFrom(c).Query(c.Request().Context(), `
		SELECT t.id, t.slug, t.name, t.status, e.code, e.name, s.current_period_end, t.created_at
		FROM tenants t
		LEFT JOIN LATERAL (
			SELECT edition_id, current_period_end FROM subscriptions
			WHERE tenant_id = t.id ORDER BY created_at DESC LIMIT 1
		) s ON TRUE
		LEFT JOIN editions e ON e.id = s.edition_id
		WHERE t.status != 'deleted'
		ORDER BY t.created_at DESC
	`)
	if err != nil {
		return mw.InternalError(c, "tenants.list", err)
	}
	defer rows.Close()

	items := make([]tenantDTO, 0)
	for rows.Next() {
		var t tenantDTO
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Status,
			&t.Edition, &t.EditionName, &t.PeriodEnd, &t.CreatedAt); err != nil {
			return mw.InternalError(c, "tenants.scan", err)
		}
		items = append(items, t)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

type createTenantRequest struct {
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Edition    string `json:"edition"`
	Months     int    `json:"months"`
	Source     string `json:"source"`
	AdminEmail string `json:"admin_email"`
	AdminName  string `json:"admin_name"`
}

// CreateTenant POST /api/platform/tenants（不走申请表单，线下已谈定的商户直接开通）
func CreateTenant(c *echo.Context) error {
	var req createTenantRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	res, err := provisionTenant(c.Request().Context(), mw.TxFrom(c), provisionInput{
		Slug:       req.Slug,
		Name:       req.Name,
		Edition:    req.Edition,
		Months:     req.Months,
		Source:     req.Source,
		AdminEmail: req.AdminEmail,
		AdminName:  req.AdminName,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// 平台高危操作统一 slog 留痕（audit_logs 是租户内体系，平台侧独立成表待 billing 阶段一并设计）
	slog.Info("platform: tenant created", "operator", mw.OperatorFrom(c).Email,
		"slug", req.Slug, "edition", req.Edition, "months", req.Months)
	return c.JSON(http.StatusOK, map[string]any{
		"tenant_id":      res.TenantID,
		"slug":           req.Slug,
		"admin_email":    req.AdminEmail,
		"admin_password": res.AdminPassword,
	})
}

// setTenantStatus 停用 / 恢复共用
func setTenantStatus(c *echo.Context, status string) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
	}
	tag, err := mw.TxFrom(c).Exec(c.Request().Context(),
		"UPDATE tenants SET status = $2, updated_at = now() WHERE id = $1", id, status)
	if err != nil {
		return mw.InternalError(c, "tenants.set_status", err)
	}
	if tag.RowsAffected() == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "商户不存在")
	}
	slog.Info("platform: tenant status changed", "operator", mw.OperatorFrom(c).Email,
		"tenant_id", id.String(), "status", status)
	return c.JSON(http.StatusOK, map[string]any{"status": status})
}

// SuspendTenant POST /api/platform/tenants/:id/suspend（中间件对非 active 租户全端 403，缓存 5s 内生效）
func SuspendTenant(c *echo.Context) error { return setTenantStatus(c, "suspended") }

// ResumeTenant POST /api/platform/tenants/:id/resume
func ResumeTenant(c *echo.Context) error { return setTenantStatus(c, "active") }

type setSubscriptionRequest struct {
	Edition string `json:"edition"`
	Months  int    `json:"months"`
	Source  string `json:"source"`
}

// SetSubscription POST /api/platform/tenants/:id/subscription
// 改套餐 / 续期统一为追加一条新订阅行（保留历史，读取口径 = created_at 最新）
func SetSubscription(c *echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
	}
	var req setSubscriptionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Months < 1 || req.Months > 120 {
		return echo.NewHTTPError(http.StatusBadRequest, "期限须在 1-120 个月之间")
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}
	if source != "gift" && source != "manual" && source != "contract" {
		return echo.NewHTTPError(http.StatusBadRequest, "无效的订阅来源")
	}

	ctx := c.Request().Context()
	tx := mw.TxFrom(c)

	editionID, _, err := editionByCode(ctx, tx, req.Edition)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusBadRequest, "套餐不存在："+req.Edition)
		}
		return mw.InternalError(c, "tenants.subscription.edition", err)
	}

	var exists int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM tenants WHERE id = $1", tenantID).Scan(&exists); err != nil {
		return mw.InternalError(c, "tenants.subscription.check", err)
	}
	if exists == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "商户不存在")
	}

	// 续期从「当前到期日与现在的较晚者」起算：提前续费不吃亏，过期续费从今天起算
	var base time.Time
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(current_period_end, now()) FROM subscriptions
		WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT 1
	`, tenantID).Scan(&base)
	if err != nil || base.Before(time.Now()) {
		base = time.Now()
	}

	cycle := "monthly"
	switch {
	case source == "gift":
		cycle = "gift"
	case source == "contract":
		cycle = "contract"
	case req.Months >= 12:
		cycle = "yearly"
	}

	if err := setTenantCtx(ctx, tx, tenantID); err != nil {
		return mw.InternalError(c, "tenants.subscription.ctx", err)
	}
	var periodEnd time.Time = base.AddDate(0, req.Months, 0)
	if _, err := tx.Exec(ctx, `
		INSERT INTO subscriptions (tenant_id, edition_id, source, billing_cycle, current_period_end, auto_renew)
		VALUES ($1, $2, $3, $4, $5, FALSE)
	`, tenantID, editionID, source, cycle, periodEnd); err != nil {
		return mw.InternalError(c, "tenants.subscription.insert", err)
	}

	slog.Info("platform: subscription changed", "operator", mw.OperatorFrom(c).Email,
		"tenant_id", tenantID.String(), "edition", req.Edition, "months", req.Months, "source", source)
	return c.JSON(http.StatusOK, map[string]any{
		"edition":            req.Edition,
		"current_period_end": periodEnd,
	})
}

type resetAdminPasswordRequest struct {
	Email string `json:"email"`
}

// ResetAdminPassword POST /api/platform/tenants/:id/reset-admin-password
// 商户管理员把自己锁死时救急：生成新密码一次性返回，同时吊销其所有旧会话
func ResetAdminPassword(c *echo.Context) error {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
	}
	var req resetAdminPasswordRequest
	if err := c.Bind(&req); err != nil || req.Email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "email 必填")
	}

	ctx := c.Request().Context()
	tx := mw.TxFrom(c)

	password, err := genPassword()
	if err != nil {
		return mw.InternalError(c, "tenants.reset.gen", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return mw.InternalError(c, "tenants.reset.hash", err)
	}

	if err := setTenantCtx(ctx, tx, tenantID); err != nil {
		return mw.InternalError(c, "tenants.reset.ctx", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE users
		SET password_hash = $3, token_version = token_version + 1,
		    password_changed_at = now(), failed_login_attempts = 0, locked_until = NULL,
		    updated_at = now()
		WHERE tenant_id = $1 AND email = $2 AND role = 'super_admin'
	`, tenantID, req.Email, hash)
	if err != nil {
		return mw.InternalError(c, "tenants.reset.update", err)
	}
	if tag.RowsAffected() == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "该商户下不存在此邮箱的超级管理员")
	}

	slog.Info("platform: merchant admin password reset", "operator", mw.OperatorFrom(c).Email,
		"tenant_id", tenantID.String(), "target_email", req.Email)
	return c.JSON(http.StatusOK, map[string]any{
		"email":        req.Email,
		"new_password": password, // 一次性展示
	})
}
