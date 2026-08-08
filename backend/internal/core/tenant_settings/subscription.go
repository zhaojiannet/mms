package tenantsettings

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

// StoreSubscription GET /api/store/subscription（admin 及以上）
// 商户端到期提示条数据源；无订阅行返回 edition=null，前端静默
func StoreSubscription(c *echo.Context) error {
	t := mw.TenantFrom(c)

	var (
		code, name string
		periodEnd  *time.Time
	)
	err := mw.TxFrom(c).QueryRow(c.Request().Context(), `
		SELECT e.code, e.name, s.current_period_end
		FROM subscriptions s
		JOIN editions e ON e.id = s.edition_id
		WHERE s.tenant_id = $1
		ORDER BY s.created_at DESC LIMIT 1
	`, t.ID).Scan(&code, &name, &periodEnd)
	if err != nil {
		// 无订阅行是预期路径（自建部署、手工建的租户）；真实 DB 错误不能折叠成 200，
		// 否则事务已 abort、提交注定失败，而客户端拿到的是正常响应且无任何日志
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusOK, map[string]any{"edition": nil})
		}
		return mw.InternalError(c, "store.subscription", err)
	}

	resp := map[string]any{
		"edition":            code,
		"edition_name":       name,
		"current_period_end": periodEnd,
	}
	if periodEnd != nil {
		resp["days_left"] = int(math.Ceil(time.Until(*periodEnd).Hours() / 24))
	}
	return c.JSON(http.StatusOK, resp)
}
