package cardtypes

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type DTO struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Price        decimal.Decimal `json:"price"`
	DiscountRate decimal.Decimal `json:"discount_rate"`
	SortOrder    int32           `json:"sort_order"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Name         string           `json:"name"`
	Price        *decimal.Decimal `json:"price"`
	DiscountRate *decimal.Decimal `json:"discount_rate"`
	SortOrder    *int32           `json:"sort_order"`
}

type UpdateRequest struct {
	Name         *string          `json:"name"`
	Price        *decimal.Decimal `json:"price"`
	DiscountRate *decimal.Decimal `json:"discount_rate"`
	SortOrder    *int32           `json:"sort_order"`
	Status       *string          `json:"status"`
}

func List(c *echo.Context) error {
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	var status *string
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}
	rows, err := q.ListCardTypes(c.Request().Context(), status)
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}
	items := make([]DTO, 0, len(rows))
	for _, m := range rows {
		items = append(items, toDTO(m))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.GetCardTypeByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card type not found")
		}
		return mw.InternalError(c, "get: ", err)
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.Price == nil || req.DiscountRate == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "price and discount_rate are required")
	}
	rate := *req.DiscountRate
	if rate.LessThanOrEqual(decimal.Zero) || rate.GreaterThan(decimal.NewFromInt(1)) {
		return echo.NewHTTPError(http.StatusBadRequest, "discount_rate must be in (0, 1]")
	}

	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.CreateCardType(c.Request().Context(), sqlc.CreateCardTypeParams{
		TenantID:     t.ID,
		Name:         req.Name,
		Price:        *req.Price,
		DiscountRate: rate,
		SortOrder:    req.SortOrder,
	})
	if err != nil {
		return mw.InternalError(c, "create: ", err)
	}
	return c.JSON(http.StatusCreated, toDTO(m))
}

func Update(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}

	var priceNull, rateNull decimal.NullDecimal
	if req.Price != nil {
		priceNull = decimal.NullDecimal{Decimal: *req.Price, Valid: true}
	}
	if req.DiscountRate != nil {
		if req.DiscountRate.LessThanOrEqual(decimal.Zero) || req.DiscountRate.GreaterThan(decimal.NewFromInt(1)) {
			return echo.NewHTTPError(http.StatusBadRequest, "discount_rate must be in (0, 1]")
		}
		rateNull = decimal.NullDecimal{Decimal: *req.DiscountRate, Valid: true}
	}

	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.UpdateCardType(c.Request().Context(), sqlc.UpdateCardTypeParams{
		ID:           id,
		Name:         req.Name,
		Price:        priceNull,
		DiscountRate: rateNull,
		SortOrder:    req.SortOrder,
		Status:       req.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card type not found")
		}
		return mw.InternalError(c, "update: ", err)
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Delete 前置：status=inactive + 无 cards 引用
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.GetCardTypeByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card type not found")
		}
		return mw.InternalError(c, "get: ", err)
	}
	if m.Status != "inactive" {
		return echo.NewHTTPError(http.StatusBadRequest, "必须先下架（status=inactive）再删除")
	}
	usage, err := q.CountCardTypeUsage(c.Request().Context(), id)
	if err != nil {
		return mw.InternalError(c, "check usage: ", err)
	}
	if usage > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "已有会员办理了该类型的卡，不能删除")
	}
	if err := q.DeleteCardType(c.Request().Context(), id); err != nil {
		return mw.InternalError(c, "delete: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

func toDTO(m sqlc.CardType) DTO {
	return DTO{
		ID:           m.ID,
		Name:         m.Name,
		Price:        m.Price,
		DiscountRate: m.DiscountRate,
		SortOrder:    m.SortOrder,
		Status:       m.Status,
		CreatedAt:    m.CreatedAt.Time,
		UpdatedAt:    m.UpdatedAt.Time,
	}
}
