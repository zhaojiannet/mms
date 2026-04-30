package transactions

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/pgtypex"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/timex"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// SettleAllCredits POST /api/members/:memberId/pending/settle-all
// 批量清挂账：一次建立 credit_settlement transaction 总额 = Σ(未清挂账)，批量更新
type SettleAllRequest struct {
	PaymentMethodID uuid.UUID  `json:"payment_method_id"`
	CardID          *uuid.UUID `json:"card_id"`
	TransactionTime *string    `json:"transaction_time"`
}

func SettleAllCredits(c *echo.Context) error {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}
	var req SettleAllRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.PaymentMethodID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payment_method_id required")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	// FOR UPDATE 锁定该会员所有未清挂账：防并发双扣（同一会员两个请求同时"一键清账"）
	credits, err := q.LockPendingCreditsByMember(ctx, memberID)
	if err != nil {
		return mw.InternalError(c, "lock pending: ", err)
	}
	if len(credits) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "no unsettled credits")
	}

	total := decimal.Zero
	summaryParts := make([]string, 0, len(credits))
	for _, cr := range credits {
		total = total.Add(cr.Amount)
		if cr.Summary != nil && *cr.Summary != "" {
			summaryParts = append(summaryParts, fmt.Sprintf("%s(¥%s)", *cr.Summary, cr.Amount.String()))
		}
	}

	// 卡支付校验
	var primaryCardID pgtype.UUID
	var cardBefore decimal.Decimal
	var cardID uuid.UUID
	if req.CardID != nil {
		card, err := q.LockCardForUpdate(ctx, *req.CardID)
		if err != nil {
			return mw.InternalError(c, "get card: ", err)
		}
		if card.MemberID != memberID {
			return echo.NewHTTPError(http.StatusBadRequest, "card does not belong to member")
		}
		if card.Status != "active" {
			return echo.NewHTTPError(http.StatusBadRequest, "card not active")
		}
		if card.Balance.LessThan(total) {
			return echo.NewHTTPError(http.StatusBadRequest, "insufficient balance for batch settle")
		}
		cardBefore = card.Balance
		cardID = card.ID
		primaryCardID = pgtype.UUID{Bytes: cardID, Valid: true}
	}

	// summary（前 3 条详情 + "等 N 笔"）
	summary := "批量清账：" + joinBriefSummary(summaryParts, 3)

	txTime, err := timex.ParseRFC3339Tz(req.TransactionTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transaction_time: "+err.Error())
	}

	trx, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		TenantID:         t.ID,
		Kind:             "credit_settlement",
		PaymentMethodID:  req.PaymentMethodID,
		TotalAmount:      total,
		ActualPaidAmount: total,
		MemberID:         pgtype.UUID{Bytes: memberID, Valid: true},
		CardID:           primaryCardID,
		TransactionTime:  txTime,
		Summary:          pgtypex.StrPtr(summary),
	})
	if err != nil {
		return mw.InternalError(c, "create tx: ", err)
	}

	// 走卡扣款 + 流水
	if req.CardID != nil {
		updated, err := q.AdjustCardBalance(ctx, sqlc.AdjustCardBalanceParams{
			ID:      cardID,
			Balance: total.Neg(),
		})
		if err != nil {
			return mw.InternalError(c, "adjust balance: ", err)
		}
		if _, err := q.CreateCardBalanceLog(ctx, sqlc.CreateCardBalanceLogParams{
			TenantID:      t.ID,
			CardID:        cardID,
			ChangeType:    "consume",
			Delta:         total.Neg(),
			BalanceBefore: cardBefore,
			BalanceAfter:  updated.Balance,
			TransactionID: pgtype.UUID{Bytes: trx.ID, Valid: true},
			Note:          pgtypex.StrPtr("batch clear pending"),
		}); err != nil {
			return mw.InternalError(c, "log: ", err)
		}
	}

	// 批量标记已清
	if err := q.MarkCreditsSettledByMember(ctx, sqlc.MarkCreditsSettledByMemberParams{
		MemberID:       memberID,
		SettledAt:      pgtype.Timestamptz{Time: time.Now(), Valid: true},
		SettlementTxID: pgtype.UUID{Bytes: trx.ID, Valid: true},
	}); err != nil {
		return mw.InternalError(c, "mark settled: ", err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"transaction": toDTO(trx),
		"count":       len(credits),
		"total":       total.String(),
	})
}

// joinBriefSummary 拼"X、Y、Z 等 N 笔"
func joinBriefSummary(parts []string, maxN int) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) <= maxN {
		return strings.Join(parts, "、")
	}
	return strings.Join(parts[:maxN], "、") + fmt.Sprintf("等%d笔", len(parts))
}
