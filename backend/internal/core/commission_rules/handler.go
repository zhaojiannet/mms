// Package commissionrules 提成规则 CRUD（员工×服务，rate 或 fixed_amount 二选一）
package commissionrules

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type DTO struct {
	ID          uuid.UUID        `json:"id"`
	StaffID     uuid.UUID        `json:"staff_id"`
	StaffName   string           `json:"staff_name"`
	ServiceID   *uuid.UUID       `json:"service_id"`
	ServiceName *string          `json:"service_name"`
	Rate        *decimal.Decimal `json:"rate"`
	FixedAmount *decimal.Decimal `json:"fixed_amount"`
	Note        *string          `json:"note"`
}

type CreateRequest struct {
	StaffID     uuid.UUID        `json:"staff_id"`
	ServiceID   *uuid.UUID       `json:"service_id"`
	Rate        *decimal.Decimal `json:"rate"`
	FixedAmount *decimal.Decimal `json:"fixed_amount"`
	Note        *string          `json:"note"`
}

type UpdateRequest struct {
	Rate        *decimal.Decimal `json:"rate"`
	FixedAmount *decimal.Decimal `json:"fixed_amount"`
	Note        *string          `json:"note"`
}

func List(c *echo.Context) error {
	q := sqlc.New(mw.TxFrom(c))
	var staffID pgtype.UUID
	if s := c.QueryParam("staff_id"); s != "" {
		u, err := uuid.Parse(s)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid staff_id")
		}
		staffID = pgtype.UUID{Bytes: u, Valid: true}
	}
	rows, err := q.ListCommissionRules(c.Request().Context(), staffID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list: "+err.Error())
	}
	items := make([]DTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, fromRow(r))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.StaffID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "staff_id required")
	}
	hasRate := req.Rate != nil
	hasFixed := req.FixedAmount != nil
	if hasRate == hasFixed {
		return echo.NewHTTPError(http.StatusBadRequest, "rate 和 fixed_amount 二选一")
	}
	if hasRate && (req.Rate.LessThan(decimal.Zero) || req.Rate.GreaterThan(decimal.NewFromInt(1))) {
		return echo.NewHTTPError(http.StatusBadRequest, "rate 必须 [0, 1]")
	}

	var rateNull, fixedNull decimal.NullDecimal
	if hasRate {
		rateNull = decimal.NullDecimal{Decimal: *req.Rate, Valid: true}
	}
	if hasFixed {
		fixedNull = decimal.NullDecimal{Decimal: *req.FixedAmount, Valid: true}
	}

	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	row, err := q.CreateCommissionRule(c.Request().Context(), sqlc.CreateCommissionRuleParams{
		TenantID:    t.ID,
		StaffID:     req.StaffID,
		ServiceID:   optUUID(req.ServiceID),
		Rate:        rateNull,
		FixedAmount: fixedNull,
		Note:        req.Note,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "create: "+err.Error())
	}
	return c.JSON(http.StatusCreated, fromModel(row))
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
	var rateNull, fixedNull decimal.NullDecimal
	if req.Rate != nil {
		rateNull = decimal.NullDecimal{Decimal: *req.Rate, Valid: true}
	}
	if req.FixedAmount != nil {
		fixedNull = decimal.NullDecimal{Decimal: *req.FixedAmount, Valid: true}
	}
	q := sqlc.New(mw.TxFrom(c))
	row, err := q.UpdateCommissionRule(c.Request().Context(), sqlc.UpdateCommissionRuleParams{
		ID: id, Rate: rateNull, FixedAmount: fixedNull, Note: req.Note,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "update: "+err.Error())
	}
	return c.JSON(http.StatusOK, fromModel(row))
}

func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	q := sqlc.New(mw.TxFrom(c))
	if err := q.DeleteCommissionRule(c.Request().Context(), id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete: "+err.Error())
	}
	return c.NoContent(http.StatusNoContent)
}

func optUUID(p *uuid.UUID) pgtype.UUID {
	if p == nil || *p == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *p, Valid: true}
}

func fromRow(r sqlc.ListCommissionRulesRow) DTO {
	dto := DTO{ID: r.ID, StaffID: r.StaffID, StaffName: r.StaffName, Note: r.Note}
	if r.ServiceID.Valid {
		v := uuid.UUID(r.ServiceID.Bytes); dto.ServiceID = &v
	}
	if r.ServiceName != nil {
		dto.ServiceName = r.ServiceName
	}
	if r.Rate.Valid {
		v := r.Rate.Decimal; dto.Rate = &v
	}
	if r.FixedAmount.Valid {
		v := r.FixedAmount.Decimal; dto.FixedAmount = &v
	}
	return dto
}

func fromModel(r sqlc.CommissionRule) DTO {
	dto := DTO{ID: r.ID, StaffID: r.StaffID, Note: r.Note}
	if r.ServiceID.Valid {
		v := uuid.UUID(r.ServiceID.Bytes); dto.ServiceID = &v
	}
	if r.Rate.Valid {
		v := r.Rate.Decimal; dto.Rate = &v
	}
	if r.FixedAmount.Valid {
		v := r.FixedAmount.Decimal; dto.FixedAmount = &v
	}
	return dto
}
