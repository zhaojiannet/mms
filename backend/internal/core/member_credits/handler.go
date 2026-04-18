package membercredits

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
	ID             uuid.UUID       `json:"id"`
	MemberID       uuid.UUID       `json:"member_id"`
	Amount         decimal.Decimal `json:"amount"`
	Summary        *string         `json:"summary"`
	ChargedAt      time.Time       `json:"charged_at"`
	ChargedTxID    *uuid.UUID      `json:"charged_tx_id"`
	SettledAt      *time.Time      `json:"settled_at"`
	SettlementTxID *uuid.UUID      `json:"settlement_tx_id"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateRequest struct {
	Amount    decimal.Decimal `json:"amount"`
	Summary   *string         `json:"summary"`
	ChargedAt *string         `json:"charged_at"` // 可选，不传用 now()
}

// ListPending GET /api/members/:memberId/pending
func ListPending(c *echo.Context) error {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	rows, err := q.ListPendingCreditsByMember(c.Request().Context(), memberID)
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}
	total, err := q.SumUnsettledByMember(c.Request().Context(), memberID)
	if err != nil {
		return mw.InternalError(c, "sum: ", err)
	}

	items := make([]DTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, toDTO(r))
	}
	return c.JSON(http.StatusOK, map[string]any{
		"items":        items,
		"total_amount": total,
	})
}

// Create POST /api/members/:memberId/pending
func Create(c *echo.Context) error {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return echo.NewHTTPError(http.StatusBadRequest, "amount must be positive")
	}

	chargedAt, err := parseTimestamp(req.ChargedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid charged_at: "+err.Error())
	}

	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)
	m, err := q.CreateCredit(c.Request().Context(), sqlc.CreateCreditParams{
		TenantID:    t.ID,
		MemberID:    memberID,
		Amount:      req.Amount,
		Summary:     req.Summary,
		ChargedAt:   chargedAt,
		ChargedTxID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		return mw.InternalError(c, "create: ", err)
	}
	return c.JSON(http.StatusCreated, toDTO(m))
}

// Delete DELETE /api/members/:memberId/pending/:pendingId
//   - 仅可删未清的（输入错误时用）
//   - 已清的不能直接删（有 settlement_tx 关联，RESTRICT）
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("pendingId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pending id")
	}
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	m, err := q.GetCreditByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "credit not found")
		}
		return mw.InternalError(c, "get: ", err)
	}
	if m.SettledAt.Valid {
		return echo.NewHTTPError(http.StatusBadRequest, "已清挂账不能直接删除，请走撤单流程")
	}

	if err := q.DeleteCredit(c.Request().Context(), id); err != nil {
		return mw.InternalError(c, "delete: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// --- 工具 ---

func toDTO(m sqlc.MemberCredit) DTO {
	dto := DTO{
		ID:        m.ID,
		MemberID:  m.MemberID,
		Amount:    m.Amount,
		Summary:   m.Summary,
		ChargedAt: m.ChargedAt.Time,
		CreatedAt: m.CreatedAt.Time,
		UpdatedAt: m.UpdatedAt.Time,
	}
	if m.ChargedTxID.Valid {
		v := uuid.UUID(m.ChargedTxID.Bytes)
		dto.ChargedTxID = &v
	}
	if m.SettledAt.Valid {
		v := m.SettledAt.Time
		dto.SettledAt = &v
	}
	if m.SettlementTxID.Valid {
		v := uuid.UUID(m.SettlementTxID.Bytes)
		dto.SettlementTxID = &v
	}
	return dto
}

func parseTimestamp(s *string) (pgtype.Timestamptz, error) {
	if s == nil || *s == "" {
		return pgtype.Timestamptz{Valid: false}, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}
