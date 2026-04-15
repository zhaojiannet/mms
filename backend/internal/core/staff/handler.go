package staff

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
	ID                    uuid.UUID       `json:"id"`
	Name                  string          `json:"name"`
	Position              string          `json:"position"`
	Phone                 *string         `json:"phone"`
	HireDate              *string         `json:"hire_date"`
	Status                string          `json:"status"`
	CountsCommission      bool            `json:"counts_commission"`
	DefaultCommissionRate decimal.Decimal `json:"default_commission_rate"`
	SortOrder             int32           `json:"sort_order"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Name                  string           `json:"name"`
	Position              string           `json:"position"`
	Phone                 *string          `json:"phone"`
	HireDate              *string          `json:"hire_date"` // YYYY-MM-DD
	CountsCommission      *bool            `json:"counts_commission"`
	DefaultCommissionRate *decimal.Decimal `json:"default_commission_rate"`
	SortOrder             *int32           `json:"sort_order"`
}

type UpdateRequest struct {
	Name                  *string          `json:"name"`
	Position              *string          `json:"position"`
	Phone                 *string          `json:"phone"`
	HireDate              *string          `json:"hire_date"`
	Status                *string          `json:"status"`
	CountsCommission      *bool            `json:"counts_commission"`
	DefaultCommissionRate *decimal.Decimal `json:"default_commission_rate"`
	SortOrder             *int32           `json:"sort_order"`
}

// List GET /api/staff?status=active
func List(c *echo.Context) error {
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	var status *string
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}
	rows, err := q.ListStaff(c.Request().Context(), status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list staff: "+err.Error())
	}
	items := make([]DTO, 0, len(rows))
	for _, m := range rows {
		items = append(items, toDTO(m))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Get GET /api/staff/:id
func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.GetStaffByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "staff not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Create POST /api/staff
func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Name == "" || req.Position == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name and position are required")
	}

	hireDate, err := parseDate(req.HireDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hire_date: "+err.Error())
	}

	var commNull decimal.NullDecimal
	if req.DefaultCommissionRate != nil {
		commNull = decimal.NullDecimal{Decimal: *req.DefaultCommissionRate, Valid: true}
	}

	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	m, err := q.CreateStaff(c.Request().Context(), sqlc.CreateStaffParams{
		TenantID:              t.ID,
		Name:                  req.Name,
		Position:              req.Position,
		Phone:                 req.Phone,
		HireDate:              hireDate,
		CountsCommission:      req.CountsCommission,
		DefaultCommissionRate: commNull,
		SortOrder:             req.SortOrder,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create: "+err.Error())
	}
	return c.JSON(http.StatusCreated, toDTO(m))
}

// Update PUT /api/staff/:id
func Update(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	hireDate, err := parseDate(req.HireDate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid hire_date: "+err.Error())
	}
	var commNull decimal.NullDecimal
	if req.DefaultCommissionRate != nil {
		commNull = decimal.NullDecimal{Decimal: *req.DefaultCommissionRate, Valid: true}
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.UpdateStaff(c.Request().Context(), sqlc.UpdateStaffParams{
		ID:                    id,
		Name:                  req.Name,
		Position:              req.Position,
		Phone:                 req.Phone,
		HireDate:              hireDate,
		Status:                req.Status,
		CountsCommission:      req.CountsCommission,
		DefaultCommissionRate: commNull,
		SortOrder:             req.SortOrder,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "staff not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "update: "+err.Error())
	}
	return c.JSON(http.StatusOK, toDTO(m))
}

// Delete DELETE /api/staff/:id
//   - 前置：status=inactive
//   - 同时解除 transactions / appointments 的 staff_id 引用（与老系统对齐）
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	m, err := q.GetStaffByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "staff not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	if m.Status != "inactive" {
		return echo.NewHTTPError(http.StatusBadRequest, "必须先将员工状态设为离职（status=inactive）再删除")
	}

	ctx := c.Request().Context()
	if err := q.UnlinkStaffFromTransactions(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "unlink transactions: "+err.Error())
	}
	if err := q.UnlinkStaffFromAppointments(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "unlink appointments: "+err.Error())
	}
	if err := q.DeleteStaff(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func toDTO(m sqlc.Staff) DTO {
	dto := DTO{
		ID:                    m.ID,
		Name:                  m.Name,
		Position:              m.Position,
		Phone:                 m.Phone,
		Status:                m.Status,
		CountsCommission:      m.CountsCommission,
		DefaultCommissionRate: m.DefaultCommissionRate,
		SortOrder:             m.SortOrder,
		CreatedAt:             m.CreatedAt.Time,
		UpdatedAt:             m.UpdatedAt.Time,
	}
	if m.HireDate.Valid {
		s := m.HireDate.Time.Format("2006-01-02")
		dto.HireDate = &s
	}
	return dto
}

func parseDate(s *string) (pgtype.Date, error) {
	if s == nil || *s == "" {
		return pgtype.Date{Valid: false}, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}
