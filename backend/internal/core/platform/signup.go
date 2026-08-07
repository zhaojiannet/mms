package platform

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v5"

	coreauth "github.com/zhaojiannet/mms/backend/internal/core/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

type applyRequest struct {
	StoreName     string `json:"store_name"`
	Industry      string `json:"industry"`
	ContactName   string `json:"contact_name"`
	Phone         string `json:"phone"`
	DesiredSlug   string `json:"desired_slug"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

// SubmitApplication POST /api/signup/applications（公开，验证码必填 + 路由层 IP 限流）
func SubmitApplication(c *echo.Context) error {
	var req applyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	req.StoreName = strings.TrimSpace(req.StoreName)
	req.Industry = strings.TrimSpace(req.Industry)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.Phone = strings.TrimSpace(req.Phone)
	req.DesiredSlug = strings.ToLower(strings.TrimSpace(req.DesiredSlug))

	if !coreauth.VerifyCaptcha(req.CaptchaID, req.CaptchaAnswer) {
		return echo.NewHTTPError(http.StatusBadRequest, "验证码错误或已过期")
	}
	if req.StoreName == "" || req.ContactName == "" || req.Phone == "" || req.Industry == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "请完整填写店铺名、行业、联系人和手机号")
	}
	if len(req.StoreName) > 60 || len(req.ContactName) > 60 || len(req.Industry) > 60 || len(req.Phone) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "字段长度超限")
	}
	if err := ValidateSlug(req.DesiredSlug); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	ctx := c.Request().Context()

	var taken int
	if err := dbPool.QueryRow(ctx,
		"SELECT count(*) FROM tenants WHERE slug = $1", req.DesiredSlug,
	).Scan(&taken); err != nil {
		return mw.InternalError(c, "signup.check_slug", err)
	}
	if taken > 0 {
		return echo.NewHTTPError(http.StatusConflict, "该子域已被使用，换一个吧")
	}
	if err := dbPool.QueryRow(ctx,
		"SELECT count(*) FROM signup_applications WHERE desired_slug = $1 AND status = 'pending'",
		req.DesiredSlug,
	).Scan(&taken); err != nil {
		return mw.InternalError(c, "signup.check_pending", err)
	}
	if taken > 0 {
		return echo.NewHTTPError(http.StatusConflict, "该子域已有待审核的申请")
	}

	if _, err := dbPool.Exec(ctx, `
		INSERT INTO signup_applications (store_name, industry, contact_name, phone, desired_slug)
		VALUES ($1, $2, $3, $4, $5)
	`, req.StoreName, req.Industry, req.ContactName, req.Phone, req.DesiredSlug); err != nil {
		// 并发提交同 slug：唯一部分索引拦截（上面的计数检查有竞态窗口，索引才是权威）
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return echo.NewHTTPError(http.StatusConflict, "该子域已有待审核的申请")
		}
		return mw.InternalError(c, "signup.insert", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "申请已提交，我们会尽快联系你开通",
	})
}
