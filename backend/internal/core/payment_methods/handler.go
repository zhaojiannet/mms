package paymentmethods

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// DTO 隐藏 tenant_id / legacy_id
type DTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	SortOrder int32     `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name      string `json:"name"`
	SortOrder *int32 `json:"sort_order"`
}

type UpdateRequest struct {
	Name      *string `json:"name"`
	SortOrder *int32  `json:"sort_order"`
	IsActive  *bool   `json:"is_active"`
}

// List GET /api/payment-methods?active=1
func List(c *echo.Context) error {
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	var isActive *bool
	switch c.QueryParam("active") {
	case "1", "true":
		v := true
		isActive = &v
	case "0", "false":
		v := false
		isActive = &v
	}

	rows, err := q.ListPaymentMethods(c.Request().Context(), isActive)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list payment_methods: "+err.Error())
	}

	items := make([]DTO, 0, len(rows))
	for _, m := range rows {
		items = append(items, toDTO(m))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Get GET /api/payment-methods/:id
func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.GetPaymentMethodByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "payment method not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Create POST /api/payment-methods
func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	m, err := q.CreatePaymentMethod(c.Request().Context(), sqlc.CreatePaymentMethodParams{
		TenantID:  t.ID,
		Name:      req.Name,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create: "+err.Error())
	}
	return c.JSON(http.StatusCreated, toDTO(m))
}

// Update PUT /api/payment-methods/:id
func Update(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}

	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.UpdatePaymentMethod(c.Request().Context(), sqlc.UpdatePaymentMethodParams{
		ID:        id,
		Name:      req.Name,
		SortOrder: req.SortOrder,
		IsActive:  req.IsActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "payment method not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "update: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Delete DELETE /api/payment-methods/:id
//   - 前置：该方式无交易引用
//   - 若有引用，提示改为停用（is_active=false）
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	usage, err := q.CountPaymentMethodUsage(c.Request().Context(), id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "check usage: "+err.Error())
	}
	if usage > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "该支付方式已被交易引用，不能删除，请改为停用")
	}

	if err := q.DeletePaymentMethod(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func toDTO(m sqlc.PaymentMethod) DTO {
	return DTO{
		ID:        m.ID,
		Name:      m.Name,
		SortOrder: m.SortOrder,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt.Time,
		UpdatedAt: m.UpdatedAt.Time,
	}
}
