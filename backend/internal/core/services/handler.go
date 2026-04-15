package services

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type DTO struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Price       decimal.Decimal `json:"price"`
	DurationMin *int32          `json:"duration_min"`
	Category    *string         `json:"category"`
	Description *string         `json:"description"`
	NoDiscount  bool            `json:"no_discount"`
	SortOrder   int32           `json:"sort_order"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type ListResponse struct {
	Items []DTO `json:"items"`
	Total int64 `json:"total"`
}

type CreateRequest struct {
	Name        string           `json:"name"`
	Price       *decimal.Decimal `json:"price"`
	DurationMin *int32           `json:"duration_min"`
	Category    *string          `json:"category"`
	Description *string          `json:"description"`
	NoDiscount  *bool            `json:"no_discount"`
	SortOrder   *int32           `json:"sort_order"`
}

type UpdateRequest struct {
	Name        *string          `json:"name"`
	Price       *decimal.Decimal `json:"price"`
	DurationMin *int32           `json:"duration_min"`
	Category    *string          `json:"category"`
	Description *string          `json:"description"`
	NoDiscount  *bool            `json:"no_discount"`
	SortOrder   *int32           `json:"sort_order"`
	Status      *string          `json:"status"`
}

// List GET /api/services?status=active
func List(c *echo.Context) error {
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	var status *string
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}

	rows, err := q.ListServices(c.Request().Context(), status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list services: "+err.Error())
	}
	total, err := q.CountServices(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "count: "+err.Error())
	}

	items := make([]DTO, 0, len(rows))
	for _, m := range rows {
		items = append(items, toDTO(m))
	}
	return c.JSON(http.StatusOK, ListResponse{Items: items, Total: total})
}

// Get GET /api/services/:id
func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.GetServiceByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Create POST /api/services
func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.Price == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "price is required")
	}
	if req.Price.IsNegative() {
		return echo.NewHTTPError(http.StatusBadRequest, "price must be non-negative")
	}

	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	m, err := q.CreateService(c.Request().Context(), sqlc.CreateServiceParams{
		TenantID:    t.ID,
		Name:        req.Name,
		Price:       *req.Price,
		DurationMin: req.DurationMin,
		Category:    req.Category,
		Description: req.Description,
		NoDiscount:  req.NoDiscount,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create: "+err.Error())
	}
	return c.JSON(http.StatusCreated, toDTO(m))
}

// Update PUT /api/services/:id
func Update(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}

	var priceNull decimal.NullDecimal
	if req.Price != nil {
		priceNull = decimal.NullDecimal{Decimal: *req.Price, Valid: true}
	}

	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.UpdateService(c.Request().Context(), sqlc.UpdateServiceParams{
		ID:          id,
		Name:        req.Name,
		Price:       priceNull,
		DurationMin: req.DurationMin,
		Category:    req.Category,
		Description: req.Description,
		NoDiscount:  req.NoDiscount,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "update: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Delete DELETE /api/services/:id
//   - 前置：状态为 inactive + 无 transaction_items 或 appointment_services 引用
//   - 有引用 → 提示改为停用
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	// 状态校验
	m, err := q.GetServiceByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "service not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	if m.Status != "inactive" {
		return echo.NewHTTPError(http.StatusBadRequest, "必须先将服务项目下架（status=inactive）再删除")
	}

	usage, err := q.CountServiceUsage(c.Request().Context(), pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "check usage: "+err.Error())
	}
	if usage.TxItemCount > 0 || usage.ApptCount > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "该服务项目已被交易或预约引用，不能删除，建议只下架")
	}

	if err := q.DeleteService(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func toDTO(m sqlc.Service) DTO {
	return DTO{
		ID:          m.ID,
		Name:        m.Name,
		Price:       m.Price,
		DurationMin: m.DurationMin,
		Category:    m.Category,
		Description: m.Description,
		NoDiscount:  m.NoDiscount,
		SortOrder:   m.SortOrder,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt.Time,
		UpdatedAt:   m.UpdatedAt.Time,
	}
}
