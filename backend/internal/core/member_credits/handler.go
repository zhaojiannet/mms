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
	"github.com/zhaojiannet/mms/backend/internal/platform/util/decx"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/timex"
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
	// 粗筛必须先于与 0 的比较：比较本身就会触发 rescale
	if err := decx.Amount("amount", req.Amount); err != nil {
		return err
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return echo.NewHTTPError(http.StatusBadRequest, "amount must be positive")
	}

	chargedAt, err := timex.ParseRFC3339Tz(req.ChargedAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid charged_at: "+err.Error())
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	// 挂账同时登记一条 0 实收交易：收银台挂账走的就是这条路径，只写
	// member_credits 会让这笔欠款在流水与营业额里彻底看不到。金额口径与
	// 老系统迁入的挂账一致——标价即挂账额、实收 0、支付方式记「其他」
	active := true
	pms, err := q.ListPaymentMethods(ctx, &active)
	if err != nil {
		return mw.InternalError(c, "list payment methods: ", err)
	}
	otherIdx := -1
	for i := range pms {
		if pms[i].Name == "其他" {
			otherIdx = i
			break
		}
	}
	if otherIdx < 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "缺少「其他」支付方式，请先在设置中恢复后再登记挂账")
	}

	trx, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		TenantID:         t.ID,
		Kind:             "sale",
		PaymentMethodID:  pms[otherIdx].ID,
		TotalAmount:      req.Amount,
		ActualPaidAmount: decimal.Zero,
		MemberID:         pgtype.UUID{Bytes: memberID, Valid: true},
		TransactionTime:  chargedAt,
		Summary:          req.Summary,
	})
	if err != nil {
		return mw.InternalError(c, "create credit transaction: ", err)
	}

	m, err := q.CreateCredit(ctx, sqlc.CreateCreditParams{
		TenantID:    t.ID,
		MemberID:    memberID,
		Amount:      req.Amount,
		Summary:     req.Summary,
		ChargedAt:   chargedAt,
		ChargedTxID: pgtype.UUID{Bytes: trx.ID, Valid: true},
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
	// 挂账已计入流水与营业额，删掉就是改账，必须留痕：走撤单（记操作人与原因）。
	// 早期只写挂账表、没有关联交易的历史记录不受此限，仍可直接删
	if m.ChargedTxID.Valid {
		return echo.NewHTTPError(http.StatusBadRequest, "该挂账已计入营业流水，请在报表流水或收银记录中撤销对应交易")
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

