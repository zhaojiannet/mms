// Package transactions 提供三类交易的核心接口：消费(sale) / 办卡(recharge) / 清挂账(credit_settlement)
// 所有事务都在 mw.TxFrom(c) 事务里跑，handler 不 panic 即 COMMIT，panic 或 return err 即 ROLLBACK
package transactions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/echox"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/pgtypex"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/timex"
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
	PendingMode      *string          `json:"pending_mode"`     // "full"=全额挂账 "use_balance"=利用余额+差额挂账
}

func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if len(req.Items) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "items is required")
	}
	if len(req.Items) > 50 {
		return echo.NewHTTPError(http.StatusBadRequest, "最多支持 50 个商品项")
	}
	if len(req.CardAllocations) > 10 {
		return echo.NewHTTPError(http.StatusBadRequest, "最多支持 10 张卡组合支付")
	}
	if req.PaymentMethodID == uuid.Nil {
		return echo.NewHTTPError(http.StatusBadRequest, "payment_method_id is required")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	tx := mw.TxFrom(c)
	q := sqlc.New(tx)

	// 0. 归属校验：FK 检查不受 RLS 约束，跨租户 UUID 能过 FK；
	// 必须用 RLS 范围内的 SELECT 显式确认归属本租户
	if err := checkRefsBelongToTenant(c, q, req.PaymentMethodID, req.StaffID, req.MemberID); err != nil {
		return err
	}

	// 1. 查 services + 算 total_amount（总应收，按标价）
	// 一次性批量取所有 service，避免 N+1：原 for 循环 GetServiceByID
	if len(req.Items) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "items required")
	}
	svcIDs := make([]uuid.UUID, 0, len(req.Items))
	for _, it := range req.Items {
		if it.Quantity <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "quantity must be > 0")
		}
		svcIDs = append(svcIDs, it.ServiceID)
	}
	svcRows, err := q.GetServicesByIDs(ctx, svcIDs)
	if err != nil {
		return mw.InternalError(c, "transactions.get_services", err)
	}
	svcByID := make(map[uuid.UUID]sqlc.Service, len(svcRows))
	for _, s := range svcRows {
		svcByID[s.ID] = s
	}

	total := decimal.Zero
	itemRows := make([]struct {
		svc sqlc.Service
		qty int32
	}, 0, len(req.Items))
	summaryParts := make([]string, 0, len(req.Items))
	for _, it := range req.Items {
		svc, ok := svcByID[it.ServiceID]
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound, "service not found: "+it.ServiceID.String())
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
	pendingMode := ""
	if req.PendingMode != nil {
		pendingMode = *req.PendingMode
	}
	if pendingMode != "" && pendingMode != "full" && pendingMode != "use_balance" {
		return echo.NewHTTPError(http.StatusBadRequest, "pending_mode 只能是 full 或 use_balance")
	}
	if pendingMode != "" && req.MemberID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "挂账必须指定会员")
	}
	// 互斥：manual_price 接管实收后若仍按 allocations 扣卡，账面实收与实际扣卡可任意背离
	if req.ManualPrice != nil && (len(req.CardAllocations) > 0 || pendingMode != "") {
		return echo.NewHTTPError(http.StatusBadRequest, "手动定价不能与卡支付或挂账同时使用")
	}
	if len(req.CardAllocations) > 0 && req.MemberID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "卡支付必须指定会员")
	}

	actualPaid := total
	if pendingMode == "full" {
		actualPaid = decimal.Zero
		req.CardAllocations = nil
	} else if req.ManualPrice != nil {
		// 防恶意洗账：ManualPrice 必须 >= 0（不允许负数）且 <= total（不允许加价）
		if req.ManualPrice.IsNegative() {
			return echo.NewHTTPError(http.StatusBadRequest, "手动定价不能为负数")
		}
		if req.ManualPrice.GreaterThan(total) {
			return echo.NewHTTPError(http.StatusBadRequest, "手动定价不能超过应收金额")
		}
		if !req.ManualPrice.Equal(req.ManualPrice.Round(2)) {
			return echo.NewHTTPError(http.StatusBadRequest, "手动定价最多两位小数")
		}
		actualPaid = *req.ManualPrice
	} else if len(req.CardAllocations) > 0 {
		actualPaid = decimal.Zero
		for _, a := range req.CardAllocations {
			if a.Deduct.IsNegative() {
				return echo.NewHTTPError(http.StatusBadRequest, "card_allocations.deduct must be non-negative")
			}
			if !a.Deduct.Equal(a.Deduct.Round(2)) {
				return echo.NewHTTPError(http.StatusBadRequest, "扣卡金额最多两位小数")
			}
			actualPaid = actualPaid.Add(a.Deduct)
		}
		if actualPaid.GreaterThan(total) {
			return echo.NewHTTPError(http.StatusBadRequest, "卡支付总额不能超过应收金额")
		}
	}
	if actualPaid.IsNegative() {
		return echo.NewHTTPError(http.StatusBadRequest, "actual_paid cannot be negative")
	}
	discount := total.Sub(actualPaid)

	// 3. 扣卡前置校验
	// 批量 FOR UPDATE 锁卡：避免 N 张卡 N 次 RTT；ORDER BY id 保证多事务统一加锁顺序，防死锁
	var primaryCardID *uuid.UUID
	type allocLine struct {
		card   sqlc.LockCardsForUpdateRow
		deduct decimal.Decimal
	}
	allocs := make([]allocLine, 0, len(req.CardAllocations))
	memberIDForCardCheck := uuid.Nil
	if req.MemberID != nil {
		memberIDForCardCheck = *req.MemberID
	}

	if len(req.CardAllocations) > 0 {
		ids := make([]uuid.UUID, len(req.CardAllocations))
		for i, a := range req.CardAllocations {
			ids[i] = a.CardID
		}
		// 排序传入也无所谓（PG 端 ORDER BY 也保证），这里冗余排序便于调试日志一致
		sort.Slice(ids, func(i, j int) bool { return bytes.Compare(ids[i][:], ids[j][:]) < 0 })

		rows, err := q.LockCardsForUpdate(ctx, ids)
		if err != nil {
			return mw.InternalError(c, "lock cards", err)
		}
		cardByID := make(map[uuid.UUID]sqlc.LockCardsForUpdateRow, len(rows))
		for _, r := range rows {
			cardByID[r.ID] = r
		}

		// 按 req 原顺序遍历做业务校验 + 保留 primaryCardID 语义（首张卡）
		for i, a := range req.CardAllocations {
			card, ok := cardByID[a.CardID]
			if !ok {
				return echo.NewHTTPError(http.StatusNotFound, "card not found: "+a.CardID.String())
			}
			if memberIDForCardCheck != uuid.Nil && card.MemberID != memberIDForCardCheck {
				return echo.NewHTTPError(http.StatusBadRequest, "card does not belong to this member")
			}
			if card.Status != "active" {
				return echo.NewHTTPError(http.StatusBadRequest, "card is not active: "+card.Status)
			}
			if card.ExpiresAt.Valid && card.ExpiresAt.Time.Before(time.Now()) {
				return echo.NewHTTPError(http.StatusBadRequest, "卡已过期："+card.CardTypeName)
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
	}
	// 多卡时 transactions.card_id 留空（后端靠 card_balance_logs 找关联）
	if len(allocs) > 1 {
		primaryCardID = nil
	}

	// 4. 构造 summary（如 "剪发、洗发"，最多 3 项，超出省略）
	summary := joinSummary(summaryParts, 3)
	notes := req.Notes
	txTime, err := timex.ParseRFC3339Tz(req.TransactionTime)
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
		MemberID:         pgtypex.OptUUID(req.MemberID),
		CustomerName:     req.CustomerName,
		StaffID:          pgtypex.OptUUID(req.StaffID),
		CardID:           pgtypex.OptUUID(primaryCardID),
		DiscountAmount:   decimal.NullDecimal{Decimal: discount, Valid: true},
		TransactionTime:  txTime,
		Summary:          pgtypex.StrPtr(summary),
		Notes:            notes,
	})
	if err != nil {
		return mw.InternalError(c, "create transaction: ", err)
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
			return mw.InternalError(c, "create item: ", err)
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
			return mw.InternalError(c, "adjust balance: ", err)
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
			return mw.InternalError(c, "log balance: ", err)
		}
	}

	// 8. 挂账模式：创建 member_credits 记录
	if pendingMode != "" && req.MemberID != nil {
		pendingAmount := total.Sub(actualPaid)
		if pendingAmount.GreaterThan(decimal.Zero) {
			if _, err := q.CreateCredit(ctx, sqlc.CreateCreditParams{
				TenantID:    t.ID,
				MemberID:    *req.MemberID,
				Amount:      pendingAmount,
				Summary:     pgtypex.StrPtr(summary),
				ChargedAt:   txTime,
				ChargedTxID: pgtype.UUID{Bytes: trx.ID, Valid: true},
			}); err != nil {
				return mw.InternalError(c, "create pending: ", err)
			}
			slog.Info("pending credit created",
				"member_id", req.MemberID,
				"amount", pendingAmount,
				"tx_id", trx.ID,
				"mode", pendingMode,
			)
		}
	}

	return c.JSON(http.StatusCreated, toDTO(trx))
}

// checkRefsBelongToTenant 用 RLS 范围内的 SELECT 校验外键引用归属本租户。
// FK 约束检查绕过 RLS：跨租户的 payment_method/staff/member UUID 能通过 FK
// 落库，造成跨租户引用污染，必须在写入前显式查一次
func checkRefsBelongToTenant(c *echo.Context, q *sqlc.Queries, paymentMethodID uuid.UUID, staffID, memberID *uuid.UUID) error {
	ctx := c.Request().Context()
	if _, err := q.GetPaymentMethodByID(ctx, paymentMethodID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusBadRequest, "payment method not found")
		}
		return mw.InternalError(c, "check payment method: ", err)
	}
	if staffID != nil {
		if _, err := q.GetStaffByID(ctx, *staffID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusBadRequest, "staff not found")
			}
			return mw.InternalError(c, "check staff: ", err)
		}
	}
	if memberID != nil {
		if _, err := q.GetMemberByID(ctx, *memberID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusBadRequest, "member not found")
			}
			return mw.InternalError(c, "check member: ", err)
		}
	}
	return nil
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

	if err := checkRefsBelongToTenant(c, q, req.PaymentMethodID, req.StaffID, &memberID); err != nil {
		return err
	}

	ct, err := q.GetCardTypeByID(ctx, req.CardTypeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "card_type not found")
		}
		return mw.InternalError(c, "get card_type: ", err)
	}

	finalPrice := ct.Price
	if req.FinalPrice != nil {
		// 防篡改：final_price 必须 > 0；非 super_admin 不能低于模板 5 折（防 0.01 办 6999 卡）
		if !req.FinalPrice.IsPositive() {
			return echo.NewHTTPError(http.StatusBadRequest, "final_price 必须大于 0")
		}
		minFloor := ct.Price.Mul(decimal.NewFromFloat(0.5))
		claims := mw.ClaimsFrom(c)
		if claims.Role != mw.RoleSuperAdmin && req.FinalPrice.LessThan(minFloor) {
			return echo.NewHTTPError(http.StatusBadRequest,
				"final_price 不能低于模板价的 50%（如需更低折扣，请联系超级管理员）")
		}
		finalPrice = *req.FinalPrice
	}
	finalRate := ct.DiscountRate
	if req.FinalDiscountRate != nil {
		if req.FinalDiscountRate.LessThanOrEqual(decimal.Zero) || req.FinalDiscountRate.GreaterThan(decimal.NewFromInt(1)) {
			return echo.NewHTTPError(http.StatusBadRequest, "final_discount_rate must be in (0,1]")
		}
		finalRate = *req.FinalDiscountRate
	}

	expiresAt, err := timex.ParseRFC3339Tz(req.ExpiresAt)
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
		return mw.InternalError(c, "issue card: ", err)
	}

	// 2. 办卡交易
	summary := fmt.Sprintf("办理【%s %s折】", ct.Name, formatRate(finalRate))
	txTime, err := timex.ParseRFC3339Tz(req.TransactionTime)
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
		StaffID:          pgtypex.OptUUID(req.StaffID),
		CardID:           pgtype.UUID{Bytes: card.ID, Valid: true},
		DiscountAmount:   decimal.NullDecimal{Decimal: decimal.Zero, Valid: true},
		TransactionTime:  txTime,
		Summary:          pgtypex.StrPtr(summary),
	})
	if err != nil {
		return mw.InternalError(c, "create recharge tx: ", err)
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
		return mw.InternalError(c, "log balance: ", err)
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

	if err := checkRefsBelongToTenant(c, q, req.PaymentMethodID, nil, nil); err != nil {
		return err
	}

	// FOR UPDATE 锁定本条挂账：防并发双清
	credit, err := q.LockCreditForUpdate(ctx, pendingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "pending credit not found")
		}
		return mw.InternalError(c, "get credit: ", err)
	}
	if credit.MemberID != memberID {
		return echo.NewHTTPError(http.StatusBadRequest, "credit does not belong to this member")
	}
	if credit.SettledAt.Valid {
		return echo.NewHTTPError(http.StatusBadRequest, "credit already settled")
	}

	// 若走会员卡：扣卡（同样 FOR UPDATE 锁）
	var primaryCardID pgtype.UUID
	var cardBefore, cardAfter decimal.Decimal
	var card sqlc.LockCardForUpdateRow
	if req.CardID != nil {
		card, err = q.LockCardForUpdate(ctx, *req.CardID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusNotFound, "card not found")
			}
			return mw.InternalError(c, "get card: ", err)
		}
		if card.MemberID != memberID {
			return echo.NewHTTPError(http.StatusBadRequest, "card does not belong to member")
		}
		if card.Status != "active" {
			return echo.NewHTTPError(http.StatusBadRequest, "card not active")
		}
		if card.ExpiresAt.Valid && card.ExpiresAt.Time.Before(time.Now()) {
			return echo.NewHTTPError(http.StatusBadRequest, "卡已过期")
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

	txTime, err := timex.ParseRFC3339Tz(req.TransactionTime)
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
		Summary:          pgtypex.StrPtr(summary),
	})
	if err != nil {
		return mw.InternalError(c, "create settle tx: ", err)
	}

	// 走卡 → 扣款 + 流水
	if req.CardID != nil {
		updated, err := q.AdjustCardBalance(ctx, sqlc.AdjustCardBalanceParams{
			ID:      card.ID,
			Balance: credit.Amount.Neg(),
		})
		if err != nil {
			return mw.InternalError(c, "adjust balance: ", err)
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
			Note:          pgtypex.StrPtr("clear pending credit"),
		}); err != nil {
			return mw.InternalError(c, "log balance: ", err)
		}
	}

	// 标记挂账已清
	if _, err := q.MarkCreditSettled(ctx, sqlc.MarkCreditSettledParams{
		ID:               pendingID,
		SettledAt:        pgtype.Timestamptz{Time: time.Now(), Valid: true},
		SettlementTxID:   pgtype.UUID{Bytes: trx.ID, Valid: true},
	}); err != nil {
		return mw.InternalError(c, "mark settled: ", err)
	}

	return c.JSON(http.StatusCreated, toDTO(trx))
}

// buildListETag 把 filter 指纹 + max(updated_at) + 分页 hash 成弱 ETag
//
// filter 任意一项变 → ETag 变（避免不同 filter 共用旧 304）
// 范围内有任何写入 → max_updated_at 变 → ETag 变
func buildListETag(
	start, end pgtype.Timestamptz,
	kind, status *string,
	includeVoided *bool,
	memberID pgtype.UUID,
	page, limit int32,
	maxUpdatedAt pgtype.Timestamptz,
) string {
	var b strings.Builder
	if start.Valid {
		b.WriteString(start.Time.UTC().Format(time.RFC3339Nano))
	}
	b.WriteByte('|')
	if end.Valid {
		b.WriteString(end.Time.UTC().Format(time.RFC3339Nano))
	}
	b.WriteByte('|')
	if kind != nil {
		b.WriteString(*kind)
	}
	b.WriteByte('|')
	if status != nil {
		b.WriteString(*status)
	}
	b.WriteByte('|')
	if includeVoided != nil && *includeVoided {
		b.WriteByte('1')
	}
	b.WriteByte('|')
	if memberID.Valid {
		b.WriteString(uuid.UUID(memberID.Bytes).String())
	}
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(int(page)))
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(int(limit)))
	b.WriteByte('|')
	if maxUpdatedAt.Valid {
		b.WriteString(strconv.FormatInt(maxUpdatedAt.Time.UnixNano(), 10))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return `W/"` + hex.EncodeToString(sum[:16]) + `"`
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

	limit := echox.ParseInt32(c.QueryParam("limit"), 50, 200)
	page := echox.ParseInt32(c.QueryParam("page"), 1, 100000)
	offset := (page - 1) * limit

	start, err := timex.ParseRFC3339Tz(pgtypex.StrPtr(c.QueryParam("start_date")))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start_date")
	}
	end, err := timex.ParseRFC3339Tz(pgtypex.StrPtr(c.QueryParam("end_date")))
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
	var status *string
	if v := c.QueryParam("status"); v != "" {
		status = &v
	}
	var memberID pgtype.UUID
	if v := c.QueryParam("member_id"); v != "" {
		mid, err := uuid.Parse(v)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid member_id")
		}
		memberID = pgtype.UUID{Bytes: mid, Valid: true}
	}

	// staff 强制仅今日 + 强制 status=completed：忽略客户端 start_date/end_date/status/include_voided
	// 不让员工看撤销记录（敏感操作，admin 通过操作日志查）
	claims := mw.ClaimsFrom(c)
	if claims.Role == mw.RoleStaff {
		now := time.Now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		todayEnd := todayStart.Add(24 * time.Hour)
		start = pgtype.Timestamptz{Time: todayStart, Valid: true}
		end = pgtype.Timestamptz{Time: todayEnd, Valid: true}
		completedStatus := "completed"
		status = &completedStatus
		includeVoided = nil
		if limit > 50 {
			limit = 50
		}
	}

	// ETag：filter 指纹 + 当前 filter 范围的 MAX(updated_at) + page/limit
	// 命中 If-None-Match → 304，避开主查询 + JSON marshal 开销（POS 60s 轮询时大幅省带宽）
	maxUpdatedAt, err := q.MaxTransactionsUpdatedAt(ctx, sqlc.MaxTransactionsUpdatedAtParams{
		StartDate:     start,
		EndDate:       end,
		Kind:          kind,
		MemberID:      memberID,
		IncludeVoided: includeVoided,
		Status:        status,
	})
	if err != nil {
		return mw.InternalError(c, "list.max_updated_at", err)
	}
	etag := buildListETag(start, end, kind, status, includeVoided, memberID, page, limit, maxUpdatedAt)
	if match := c.Request().Header.Get("If-None-Match"); match != "" && match == etag {
		c.Response().Header().Set("ETag", etag)
		return c.NoContent(http.StatusNotModified)
	}
	c.Response().Header().Set("ETag", etag)

	rows, err := q.ListTransactions(ctx, sqlc.ListTransactionsParams{
		Limit:         limit,
		Offset:        offset,
		StartDate:     start,
		EndDate:       end,
		Kind:          kind,
		MemberID:      memberID,
		IncludeVoided: includeVoided,
		Status:        status,
	})
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}
	total, err := q.CountTransactionsBy(ctx, sqlc.CountTransactionsByParams{
		StartDate:     start,
		EndDate:       end,
		Kind:          kind,
		MemberID:      memberID,
		IncludeVoided: includeVoided,
		Status:        status,
	})
	if err != nil {
		return mw.InternalError(c, "count: ", err)
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
		return mw.InternalError(c, "get: ", err)
	}
	itemRows, err := q.ListTransactionItems(ctx, id)
	if err != nil {
		return mw.InternalError(c, "items: ", err)
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
