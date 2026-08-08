package platform

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

type editionDTO struct {
	Code         string           `json:"code"`
	Name         string           `json:"name"`
	PriceMonthly *decimal.Decimal `json:"price_monthly"`
	PriceYearly  *decimal.Decimal `json:"price_yearly"`
	Quotas       json.RawMessage  `json:"quotas"`
	Features     json.RawMessage  `json:"features"`
	SortOrder    int16            `json:"sort_order"`
	Active       bool             `json:"active"`
}

// ListEditions GET /api/platform/editions
func ListEditions(c *echo.Context) error {
	rows, err := mw.TxFrom(c).Query(c.Request().Context(), `
		SELECT code, name, price_monthly, price_yearly, quotas, features, sort_order, active
		FROM editions ORDER BY sort_order
	`)
	if err != nil {
		return mw.InternalError(c, "editions.list", err)
	}
	defer rows.Close()

	items := make([]editionDTO, 0, 5)
	for rows.Next() {
		var e editionDTO
		if err := rows.Scan(&e.Code, &e.Name, &e.PriceMonthly, &e.PriceYearly,
			&e.Quotas, &e.Features, &e.SortOrder, &e.Active); err != nil {
			return mw.InternalError(c, "editions.scan", err)
		}
		items = append(items, e)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

type updateEditionRequest struct {
	PriceMonthly *decimal.Decimal `json:"price_monthly"`
	PriceYearly  *decimal.Decimal `json:"price_yearly"`
	Quotas       map[string]int64 `json:"quotas"`
}

// UpdateEdition PUT /api/platform/editions/:code
// 改价格与限额（含 free 档）；限额变更即刻作用于该档全部商户的后续新增
func UpdateEdition(c *echo.Context) error {
	code := c.Param("code")
	var req updateEditionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if len(req.Quotas) == 0 {
		// 空对象若整体覆盖，缺键会被 quota 判为"不限"，等于静默清零该档全部商户的限额
		return echo.NewHTTPError(http.StatusBadRequest, "quotas 必填且不能为空")
	}
	for k, v := range req.Quotas {
		if v < 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "限额不能为负数："+k)
		}
	}
	quotasJSON, err := json.Marshal(req.Quotas)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "quotas 格式错误")
	}

	// `||` 增量合并而非整体替换：UI 只提交它认识的几个键，
	// 整体覆盖会把将来新增的限额键在每次保存时丢掉
	tag, err := mw.TxFrom(c).Exec(c.Request().Context(), `
		UPDATE editions
		SET price_monthly = $2, price_yearly = $3, quotas = quotas || $4::jsonb, updated_at = now()
		WHERE code = $1
	`, code, req.PriceMonthly, req.PriceYearly, quotasJSON)
	if err != nil {
		return mw.InternalError(c, "editions.update", err)
	}
	if tag.RowsAffected() == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "套餐不存在")
	}
	slog.Info("platform: edition updated", "operator", mw.OperatorFrom(c).Email,
		"code", code, "quotas", string(quotasJSON))
	return c.JSON(http.StatusOK, map[string]any{"code": code})
}
