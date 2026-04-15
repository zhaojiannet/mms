// Package transactions 提供三类交易的核心接口：消费(sale) / 办卡(recharge) / 清挂账(credit_settlement)
// 所有事务都在 mw.TxFrom(c) 事务里跑，handler 不 panic 即 COMMIT，panic 或 return err 即 ROLLBACK
package transactions

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// =============================== DTO ===============================

type DTO struct {
	ID               uuid.UUID       `json:"id"`
	Kind             string          `json:"kind"`
	Status           string          `json:"status"`
	MemberID         *uuid.UUID      `json:"member_id"`
	MemberName       *string         `json:"member_name,omitempty"`    // 列表场景由 LEFT JOIN members 填充
	CustomerName     *string         `json:"customer_name"`
	StaffID          *uuid.UUID      `json:"staff_id"`
	PaymentMethodID  uuid.UUID       `json:"payment_method_id"`
	CardID           *uuid.UUID      `json:"card_id"`
	CardTypeName     *string         `json:"card_type_name,omitempty"` // 列表场景由 LEFT JOIN card_types 填充
	TotalAmount      decimal.Decimal `json:"total_amount"`
	ActualPaidAmount decimal.Decimal `json:"actual_paid_amount"`
	DiscountAmount   decimal.Decimal `json:"discount_amount"`
	TransactionTime  time.Time       `json:"transaction_time"`
	Summary          *string         `json:"summary"`
	ItemQty          int32           `json:"item_qty,omitempty"`       // 列表场景由 SUM(transaction_items.quantity) 填充
	Notes            *string         `json:"notes"`
	VoidedAt         *time.Time      `json:"voided_at,omitempty"`
	VoidedByName     *string         `json:"voided_by_name,omitempty"`
	VoidReason       *string         `json:"void_reason,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	CardSnapshots    []CardSnapshot  `json:"card_snapshots,omitempty"` // 列表场景一次性附带卡余额变化（POS hover 显示用）
}

// CardSnapshot 单笔交易里某张卡的扣款前后余额（多卡支付每张一条）
type CardSnapshot struct {
	CardID        uuid.UUID       `json:"card_id"`
	CardTypeName  string          `json:"card_type_name"`
	BalanceBefore decimal.Decimal `json:"balance_before"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
	Delta         decimal.Decimal `json:"delta"`
	ChangeType    string          `json:"change_type"` // consume / issue / void_restore ...
}

type ItemDTO struct {
	ID         uuid.UUID       `json:"id"`
	ServiceID  *uuid.UUID      `json:"service_id"`
	Name       string          `json:"name"` // service_name_snapshot
	Price      decimal.Decimal `json:"price"`
	Quantity   int32           `json:"quantity"`
	Commission decimal.Decimal `json:"commission_amount"`
}

// ============ POST /api/transactions  消费交易 ============

type CardAllocation struct {
	CardID uuid.UUID       `json:"card_id"`
	Deduct decimal.Decimal `json:"deduct"` // 从这张卡扣多少
}

type ItemInput struct {
	ServiceID uuid.UUID `json:"service_id"`
	Quantity  int32     `json:"quantity"`
}

type CreateRequest struct {
	MemberID         *uuid.UUID       `json:"member_id"`
	CustomerName     *string          `json:"customer_name"`
	StaffID          *uuid.UUID       `json:"staff_id"`
	PaymentMethodID  uuid.UUID        `json:"payment_method_id"`
	Items            []ItemInput      `json:"items"`
	CardAllocations  []CardAllocation `json:"card_allocations"` // 走会员卡时才有
	ManualPrice      *decimal.Decimal `json:"manual_price"`     // 手动定价，覆盖标准折扣
	TransactionTime  *string          `json:"transaction_time"` // 可手动补录历史时间（ISO）
	Notes            *string          `json:"notes"`
}

func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if len(req.Items) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "items is required")
	}
	if req.PaymentMethodID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payment_method_id is required")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	// 1. 查 services + 算 total_amount（总应收，按标价）
	total := decimal.Zero
	itemRows := make([]struct {
		svc sqlc.Service
		qty int32
	}, 0, len(req.Items))
	summaryParts := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "quantity must be > 0")
		}
		svc, err := q.GetServiceByID(ctx, it.ServiceID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusNotFound, "service not found: "+it.ServiceID.String())
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "get service: "+err.Error())
		}
		total = total.Add(svc.Price.Mul(decimal.NewFromInt32(it.Quantity)))
		itemRows = append(itemRows, struct {
			svc sqlc.Service
			qty int32
		}{svc, it.Quantity})
		if it.Quantity == 1 {
			summaryParts = append(summaryParts, svc.Name)
		} else {
			summaryParts = append(summaryParts, fmt.Sprintf("%s×%d", svc.Name, it.Quantity))
		}
	}

	// 2. 计算 actual_paid
	actualPaid := total
	if req.ManualPrice != nil {
		actualPaid = *req.ManualPrice
	} else if len(req.CardAllocations) > 0 {
		actualPaid = decimal.Zero
		for _, a := range req.CardAllocations {
			if a.Deduct.IsNegative() {
				return echo.NewHTTPError(http.StatusBadRequest, "card_allocations.deduct must be non-negative")
			}
			actualPaid = actualPaid.Add(a.Deduct)
		}
	}
	if actualPaid.IsNegative() {
		return echo.NewHTTPError(http.StatusBadRequest, "actual_paid cannot be negative")
	}
	discount := total.Sub(actualPaid)

	// 3. 扣卡前置校验
	var primaryCardID *uuid.UUID
	type allocLine struct {
		card   sqlc.GetCardByIDRow
		deduct decimal.Decimal
	}
	allocs := make([]allocLine, 0, len(req.CardAllocations))
	memberIDForCardCheck := uuid.Nil
	if req.MemberID != nil {
		memberIDForCardCheck = *req.MemberID
	}
	for i, a := range req.CardAllocations {
		card, err := q.GetCardByID(ctx, a.CardID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusNotFound, "card not found: "+a.CardID.String())
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "get card: "+err.Error())
		}
		if memberIDForCardCheck != uuid.Nil && card.MemberID != memberIDForCardCheck {
			return echo.NewHTTPError(http.StatusBadRequest, "card does not belong to this member")
		}
		if card.Status != "active" {
			return echo.NewHTTPError(http.StatusBadRequest, "card is not active: "+card.Status)
		}
		if card.Balance.LessThan(a.Deduct) {
			return echo.NewHTTPError(http.StatusBadRequest, "insufficient balance on card "+card.CardTypeName)
		}
		allocs = append(allocs, allocLine{card, a.Deduct})
		if i == 0 {
			id := card.ID
			primaryCardID = &id
		}
	}
	// 多卡时 transactions.card_id 留空（后端靠 card_balance_logs 找关联）
	if len(allocs) > 1 {
		primaryCardID = nil
	}

	// 4. 构造 summary（如 "剪发、洗发"，最多 3 项，超出省略）
	summary := joinSummary(summaryParts, 3)
	notes := req.Notes
	txTime, err := parseTimestamp(req.TransactionTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transaction_time: "+err.Error())
	}
	if txTime.Valid {
		prefix := "[手动设置时间] "
		if notes != nil {
			v := prefix + *notes
			notes = &v
		} else {
			v := prefix
			notes = &v
		}
	}

	// 5. 建 transaction
	trx, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		TenantID:         t.ID,
		Kind:             "sale",
		PaymentMethodID:  req.PaymentMethodID,
		TotalAmount:      total,
		ActualPaidAmount: actualPaid,
		MemberID:         optUUID(req.MemberID),
		CustomerName:     req.CustomerName,
		StaffID:          optUUID(req.StaffID),
		CardID:           optUUID(primaryCardID),
		DiscountAmount:   decimal.NullDecimal{Decimal: discount, Valid: true},
		TransactionTime:  txTime,
		Summary:          strPtr(summary),
		Notes:            notes,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create transaction: "+err.Error())
	}

	// 6. 建 transaction_items
	for _, ir := range itemRows {
		if _, err := q.CreateTransactionItem(ctx, sqlc.CreateTransactionItemParams{
			TenantID:            t.ID,
			TransactionID:       trx.ID,
			ServiceID:           pgtype.UUID{Bytes: ir.svc.ID, Valid: true},
			ServiceNameSnapshot: ir.svc.Name,
			Price:               ir.svc.Price,
			Quantity:            ir.qty,
			CommissionAmount:    decimal.NullDecimal{Decimal: decimal.Zero, Valid: true},
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "create item: "+err.Error())
		}
	}

	// 7. 多卡扣款 + 流水
	for _, a := range allocs {
		before := a.card.Balance
		negDelta := a.deduct.Neg()
		updated, err := q.AdjustCardBalance(ctx, sqlc.AdjustCardBalanceParams{
			ID:      a.card.ID,
			Balance: negDelta,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "adjust balance: "+err.Error())
		}
		if _, err := q.CreateCardBalanceLog(ctx, sqlc.CreateCardBalanceLogParams{
			TenantID:      t.ID,
			CardID:        a.card.ID,
			ChangeType:    "consume",
			Delta:         negDelta,
			BalanceBefore: before,
			BalanceAfter:  updated.Balance,
			TransactionID: pgtype.UUID{Bytes: trx.ID, Valid: true},
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "log balance: "+err.Error())
		}
	}

	return c.JSON(http.StatusCreated, toDTO(trx))
}

// ============ POST /api/members/:memberId/cards/with-transaction  办卡 ============

type IssueCardRequest struct {
	CardTypeID        uuid.UUID        `json:"card_type_id"`
	FinalPrice        *decimal.Decimal `json:"final_price"`
	FinalDiscountRate *decimal.Decimal `json:"final_discount_rate"`
	PaymentMethodID   uuid.UUID        `json:"payment_method_id"`
	StaffID           *uuid.UUID       `json:"staff_id"`
	ExpiresAt         *string          `json:"expires_at"`
	Notes             *string          `json:"notes"`
	TransactionTime   *string          `json:"transaction_time"`
}

type IssueCardResponse struct {
	Card        any `json:"card"`
	Transaction DTO `json:"transaction"`
}

func IssueCard(c *echo.Context) error {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}
	var req IssueCardRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.CardTypeID == uuid.Nil || req.PaymentMethodID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "card_type_id and payment_method_id are required")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	ct, err := q.GetCardTypeByID(ctx, req.CardTypeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card_type not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get card_type: "+err.Error())
	}

	finalPrice := ct.Price
	if req.FinalPrice != nil {
		finalPrice = *req.FinalPrice
	}
	finalRate := ct.DiscountRate
	if req.FinalDiscountRate != nil {
		if req.FinalDiscountRate.LessThanOrEqual(decimal.Zero) || req.FinalDiscountRate.GreaterThan(decimal.NewFromInt(1)) {
			return echo.NewHTTPError(http.StatusBadRequest, "final_discount_rate must be in (0,1]")
		}
		finalRate = *req.FinalDiscountRate
	}

	expiresAt, err := parseTimestamp(req.ExpiresAt)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid expires_at: "+err.Error())
	}

	// 1. 建卡
	card, err := q.IssueCard(ctx, sqlc.IssueCardParams{
		TenantID:          t.ID,
		MemberID:          memberID,
		CardTypeID:        req.CardTypeID,
		FinalPrice:        finalPrice,
		FinalDiscountRate: finalRate,
		Balance:           decimal.NullDecimal{Decimal: finalPrice, Valid: true},
		ExpiresAt:         expiresAt,
		Notes:             req.Notes,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "issue card: "+err.Error())
	}

	// 2. 办卡交易
	summary := fmt.Sprintf("办理【%s %s折】", ct.Name, formatRate(finalRate))
	txTime, err := parseTimestamp(req.TransactionTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transaction_time: "+err.Error())
	}
	trx, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		TenantID:         t.ID,
		Kind:             "recharge",
		PaymentMethodID:  req.PaymentMethodID,
		TotalAmount:      finalPrice,
		ActualPaidAmount: finalPrice,
		MemberID:         pgtype.UUID{Bytes: memberID, Valid: true},
		StaffID:          optUUID(req.StaffID),
		CardID:           pgtype.UUID{Bytes: card.ID, Valid: true},
		DiscountAmount:   decimal.NullDecimal{Decimal: decimal.Zero, Valid: true},
		TransactionTime:  txTime,
		Summary:          strPtr(summary),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create recharge tx: "+err.Error())
	}

	// 3. 余额流水：issue
	if _, err := q.CreateCardBalanceLog(ctx, sqlc.CreateCardBalanceLogParams{
		TenantID:      t.ID,
		CardID:        card.ID,
		ChangeType:    "issue",
		Delta:         finalPrice,
		BalanceBefore: decimal.Zero,
		BalanceAfter:  finalPrice,
		TransactionID: pgtype.UUID{Bytes: trx.ID, Valid: true},
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "log balance: "+err.Error())
	}

	return c.JSON(http.StatusCreated, IssueCardResponse{
		Card:        card,
		Transaction: toDTO(trx),
	})
}

// ============ POST /api/members/:memberId/pending/:pendingId/settle  清单笔挂账 ============

type SettleRequest struct {
	PaymentMethodID uuid.UUID  `json:"payment_method_id"`
	CardID          *uuid.UUID `json:"card_id"` // 走会员卡清账时非空
	TransactionTime *string    `json:"transaction_time"`
}

func SettleCredit(c *echo.Context) error {
	memberID, err := uuid.Parse(c.Param("memberId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid member id")
	}
	pendingID, err := uuid.Parse(c.Param("pendingId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid pending id")
	}
	var req SettleRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.PaymentMethodID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payment_method_id required")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	credit, err := q.GetCreditByID(ctx, pendingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "pending credit not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get credit: "+err.Error())
	}
	if credit.MemberID != memberID {
		return echo.NewHTTPError(http.StatusBadRequest, "credit does not belong to this member")
	}
	if credit.SettledAt.Valid {
		return echo.NewHTTPError(http.StatusBadRequest, "credit already settled")
	}

	// 若走会员卡：扣卡
	var primaryCardID pgtype.UUID
	var cardBefore, cardAfter decimal.Decimal
	var card sqlc.GetCardByIDRow
	if req.CardID != nil {
		card, err = q.GetCardByID(ctx, *req.CardID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusNotFound, "card not found")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "get card: "+err.Error())
		}
		if card.MemberID != memberID {
			return echo.NewHTTPError(http.StatusBadRequest, "card does not belong to member")
		}
		if card.Status != "active" {
			return echo.NewHTTPError(http.StatusBadRequest, "card not active")
		}
		if card.Balance.LessThan(credit.Amount) {
			return echo.NewHTTPError(http.StatusBadRequest, "insufficient balance to settle credit")
		}
		cardBefore = card.Balance
		primaryCardID = pgtype.UUID{Bytes: card.ID, Valid: true}
	}

	summary := "清账"
	if credit.Summary != nil && *credit.Summary != "" {
		summary = "清账：" + *credit.Summary
	}

	txTime, err := parseTimestamp(req.TransactionTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid transaction_time: "+err.Error())
	}

	trx, err := q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		TenantID:         t.ID,
		Kind:             "credit_settlement",
		PaymentMethodID:  req.PaymentMethodID,
		TotalAmount:      credit.Amount,
		ActualPaidAmount: credit.Amount,
		MemberID:         pgtype.UUID{Bytes: memberID, Valid: true},
		CardID:           primaryCardID,
		TransactionTime:  txTime,
		Summary:          strPtr(summary),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create settle tx: "+err.Error())
	}

	// 走卡 → 扣款 + 流水
	if req.CardID != nil {
		updated, err := q.AdjustCardBalance(ctx, sqlc.AdjustCardBalanceParams{
			ID:      card.ID,
			Balance: credit.Amount.Neg(),
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "adjust balance: "+err.Error())
		}
		cardAfter = updated.Balance
		if _, err := q.CreateCardBalanceLog(ctx, sqlc.CreateCardBalanceLogParams{
			TenantID:      t.ID,
			CardID:        card.ID,
			ChangeType:    "consume",
			Delta:         credit.Amount.Neg(),
			BalanceBefore: cardBefore,
			BalanceAfter:  cardAfter,
			TransactionID: pgtype.UUID{Bytes: trx.ID, Valid: true},
			Note:          strPtr("clear pending credit"),
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "log balance: "+err.Error())
		}
	}

	// 标记挂账已清
	if _, err := q.MarkCreditSettled(ctx, sqlc.MarkCreditSettledParams{
		ID:               pendingID,
		SettledAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
		SettlementTxID:   pgtype.UUID{Bytes: trx.ID, Valid: true},
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "mark settled: "+err.Error())
	}

	return c.JSON(http.StatusCreated, toDTO(trx))
}

// ============ GET /api/transactions ============

type ListResponse struct {
	Items []DTO `json:"items"`
	Total int64 `json:"total"`
	Page  int32 `json:"page"`
	Limit int32 `json:"limit"`
}

func List(c *echo.Context) error {
	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))

	limit := parseInt32(c.QueryParam("limit"), 50, 200)
	page := parseInt32(c.QueryParam("page"), 1, 100000)
	offset := (page - 1) * limit

	start, err := parseTimestamp(strPtrOrNil(c.QueryParam("start_date")))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start_date")
	}
	end, err := parseTimestamp(strPtrOrNil(c.QueryParam("end_date")))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end_date")
	}
	var kind *string
	if v := c.QueryParam("kind"); v != "" {
		kind = &v
	}
	var includeVoided *bool
	if c.QueryParam("include_voided") == "1" {
		tr := true
		includeVoided = &tr
	}

	rows, err := q.ListTransactions(ctx, sqlc.ListTransactionsParams{
		Limit:         limit,
		Offset:        offset,
		StartDate:     start,
		EndDate:       end,
		Kind:          kind,
		IncludeVoided: includeVoided,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list: "+err.Error())
	}
	total, err := q.CountTransactionsBy(ctx, sqlc.CountTransactionsByParams{
		StartDate:     start,
		EndDate:       end,
		Kind:          kind,
		IncludeVoided: includeVoided,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "count: "+err.Error())
	}

	items := make([]DTO, 0, len(rows))
	txIDs := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		dto := toDTO(r.Transaction)
		dto.MemberName = r.MemberName
		dto.CardTypeName = r.CardTypeName
		dto.ItemQty = r.ItemQty
		items = append(items, dto)
		txIDs = append(txIDs, r.Transaction.ID)
	}

	// 批量取卡余额快照（多卡支付每张一条），附加到 DTO
	if len(txIDs) > 0 {
		snaps, snapErr := q.ListCardSnapshotsByTxIDs(ctx, txIDs)
		if snapErr != nil {
			// snapshot 仅为 UI 增强，失败不阻塞主列表，但需记录以便排查
			slog.Warn("list card snapshots failed", "err", snapErr, "tx_count", len(txIDs))
		} else {
			byTx := make(map[uuid.UUID][]CardSnapshot)
			for _, s := range snaps {
				if !s.TransactionID.Valid {
					continue
				}
				txID := uuid.UUID(s.TransactionID.Bytes)
				byTx[txID] = append(byTx[txID], CardSnapshot{
					CardID:        s.CardID,
					CardTypeName:  s.CardTypeName,
					BalanceBefore: s.BalanceBefore,
					BalanceAfter:  s.BalanceAfter,
					Delta:         s.Delta,
					ChangeType:    s.ChangeType,
				})
			}
			for i := range items {
				if list, ok := byTx[items[i].ID]; ok {
					items[i].CardSnapshots = list
				}
			}
		}
	}

	return c.JSON(http.StatusOK, ListResponse{
		Items: items, Total: total, Page: page, Limit: limit,
	})
}

// ============ GET /api/transactions/:id ============

type DetailResponse struct {
	Transaction DTO       `json:"transaction"`
	Items       []ItemDTO `json:"items"`
}

func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))
	trx, err := q.GetTransactionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "transaction not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	itemRows, err := q.ListTransactionItems(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "items: "+err.Error())
	}
	items := make([]ItemDTO, 0, len(itemRows))
	for _, it := range itemRows {
		dto := ItemDTO{
			ID:         it.ID,
			Name:       it.ServiceNameSnapshot,
			Price:      it.Price,
			Quantity:   it.Quantity,
			Commission: it.CommissionAmount,
		}
		if it.ServiceID.Valid {
			sid := uuid.UUID(it.ServiceID.Bytes)
			dto.ServiceID = &sid
		}
		items = append(items, dto)
	}
	return c.JSON(http.StatusOK, DetailResponse{Transaction: toDTO(trx), Items: items})
}

// ============ 工具 ============

func toDTO(m sqlc.Transaction) DTO {
	dto := DTO{
		ID:               m.ID,
		Kind:             m.Kind,
		Status:           m.Status,
		CustomerName:     m.CustomerName,
		PaymentMethodID:  m.PaymentMethodID,
		TotalAmount:      m.TotalAmount,
		ActualPaidAmount: m.ActualPaidAmount,
		DiscountAmount:   m.DiscountAmount,
		TransactionTime:  m.TransactionTime.Time,
		Summary:          m.Summary,
		Notes:            m.Notes,
		CreatedAt:        m.CreatedAt.Time,
	}
	if m.MemberID.Valid {
		v := uuid.UUID(m.MemberID.Bytes)
		dto.MemberID = &v
	}
	if m.StaffID.Valid {
		v := uuid.UUID(m.StaffID.Bytes)
		dto.StaffID = &v
	}
	if m.CardID.Valid {
		v := uuid.UUID(m.CardID.Bytes)
		dto.CardID = &v
	}
	if m.VoidedAt.Valid {
		v := m.VoidedAt.Time
		dto.VoidedAt = &v
		dto.VoidedByName = m.VoidedByName
		dto.VoidReason = m.VoidReason
	}
	return dto
}

func optUUID(p *uuid.UUID) pgtype.UUID {
	if p == nil || *p == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *p, Valid: true}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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

func parseInt32(s string, def, max int32) int32 {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	if int32(v) > max {
		return max
	}
	return int32(v)
}

func joinSummary(parts []string, maxN int) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) <= maxN {
		return strings.Join(parts, "、")
	}
	return strings.Join(parts[:maxN], "、") + fmt.Sprintf("等%d项", len(parts))
}

// formatRate 把 0.700 格式化为 "7"（去掉尾部 0）供 summary 使用
func formatRate(r decimal.Decimal) string {
	s := r.Mul(decimal.NewFromInt(10)).Truncate(1).String()
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}
