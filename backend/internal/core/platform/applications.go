package platform

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

type applicationDTO struct {
	ID           uuid.UUID  `json:"id"`
	StoreName    string     `json:"store_name"`
	Industry     string     `json:"industry"`
	ContactName  string     `json:"contact_name"`
	Phone        string     `json:"phone"`
	DesiredSlug  string     `json:"desired_slug"`
	Status       string     `json:"status"`
	RejectReason *string    `json:"reject_reason"`
	TenantID     *uuid.UUID `json:"tenant_id"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ListApplications GET /api/platform/applications?status=pending
func ListApplications(c *echo.Context) error {
	status := c.QueryParam("status")
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "approved" && status != "rejected" && status != "all" {
		return echo.NewHTTPError(http.StatusBadRequest, "无效的状态筛选")
	}

	rows, err := mw.TxFrom(c).Query(c.Request().Context(), `
		SELECT id, store_name, industry, contact_name, phone, desired_slug,
		       status, reject_reason, tenant_id, reviewed_at, created_at
		FROM signup_applications
		WHERE ($1 = 'all' OR status = $1)
		ORDER BY created_at DESC
		LIMIT 200
	`, status)
	if err != nil {
		return mw.InternalError(c, "applications.list", err)
	}
	defer rows.Close()

	items := make([]applicationDTO, 0)
	for rows.Next() {
		var a applicationDTO
		if err := rows.Scan(&a.ID, &a.StoreName, &a.Industry, &a.ContactName, &a.Phone,
			&a.DesiredSlug, &a.Status, &a.RejectReason, &a.TenantID, &a.ReviewedAt, &a.CreatedAt); err != nil {
			return mw.InternalError(c, "applications.scan", err)
		}
		items = append(items, a)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

type approveRequest struct {
	Edition    string `json:"edition"`
	Months     int    `json:"months"`
	Source     string `json:"source"`
	AdminEmail string `json:"admin_email"`
	AdminName  string `json:"admin_name"`
}

// ApproveApplication POST /api/platform/applications/:id/approve
// 事务内：锁申请行 → 开通（租户+订阅+管理员）→ 更新申请状态
func ApproveApplication(c *echo.Context) error {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid application id")
	}
	var req approveRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	tx := mw.TxFrom(c)

	var storeName, slug, status string
	err = tx.QueryRow(ctx, `
		SELECT store_name, desired_slug, status FROM signup_applications
		WHERE id = $1 FOR UPDATE
	`, appID).Scan(&storeName, &slug, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "申请不存在")
		}
		return mw.InternalError(c, "applications.approve.load", err)
	}
	if status != "pending" {
		return echo.NewHTTPError(http.StatusConflict, "该申请已处理过")
	}

	res, err := provisionTenant(ctx, tx, provisionInput{
		Slug:       slug,
		Name:       storeName,
		Edition:    req.Edition,
		Months:     req.Months,
		Source:     req.Source,
		AdminEmail: req.AdminEmail,
		AdminName:  req.AdminName,
	})
	if err != nil {
		var ie InputError
		if errors.As(err, &ie) {
			return echo.NewHTTPError(http.StatusBadRequest, ie.Error())
		}
		return mw.InternalError(c, "platform.approve_application", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE signup_applications
		SET status = 'approved', tenant_id = $2, reviewed_at = now(), updated_at = now()
		WHERE id = $1
	`, appID, res.TenantID); err != nil {
		return mw.InternalError(c, "applications.approve.update", err)
	}

	slog.Info("platform: application approved", "operator", mw.OperatorFrom(c).Email,
		"application_id", appID.String(), "slug", slug, "edition", req.Edition)
	return c.JSON(http.StatusOK, map[string]any{
		"tenant_id":      res.TenantID,
		"slug":           slug,
		"admin_email":    req.AdminEmail,
		"admin_password": res.AdminPassword, // 一次性展示，后端不留明文
	})
}

type rejectRequest struct {
	Reason string `json:"reason"`
}

// RejectApplication POST /api/platform/applications/:id/reject
func RejectApplication(c *echo.Context) error {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid application id")
	}
	var req rejectRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	tag, err := mw.TxFrom(c).Exec(c.Request().Context(), `
		UPDATE signup_applications
		SET status = 'rejected', reject_reason = $2, reviewed_at = now(), updated_at = now()
		WHERE id = $1 AND status = 'pending'
	`, appID, req.Reason)
	if err != nil {
		return mw.InternalError(c, "applications.reject", err)
	}
	if tag.RowsAffected() == 0 {
		return echo.NewHTTPError(http.StatusConflict, "申请不存在或已处理过")
	}
	slog.Info("platform: application rejected", "operator", mw.OperatorFrom(c).Email,
		"application_id", appID.String())
	return c.JSON(http.StatusOK, map[string]any{"message": "已拒绝"})
}
