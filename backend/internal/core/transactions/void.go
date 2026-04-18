package transactions

import (
	"context"
	"encoding/json"
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

type VoidRequest struct {
	Reason string `json:"reason"`
}

// Void POST /api/transactions/:id/void
//
// 业务规则（与老 demo 对齐）：
//   - 必须先开启 tenant_settings.enable_transaction_void = true
//   - 开启后 10 分钟自动过期（由 settings.void_enabled_at 判定）
//   - 仅允许 7 天内的交易撤单
//   - 必填 reason
//
// 反向操作（按 kind 不同）：
//   - sale：对该 tx 所有 card_balance_logs 插反向 void_restore 流水 + cards.balance 回补
//   - recharge（办卡）：卡改 status=frozen + balance 归 0 + 反向流水（保留审计，不删卡）
//   - credit_settlement（清挂账）：相关 member_credits 恢复 settled_at=NULL + 若走卡补卡
//
// 最后 transactions.status='voided'（软撤单，保留完整审计）
func Void(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req VoidRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.Reason == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "reason is required")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	claims := mw.ClaimsFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	// 1. 撤单功能开关检查（同时校验当前操作者是开启者本人）
	if httpErr := checkVoidEnabled(ctx, q, t.ID, claims.UserID); httpErr != nil {
		return httpErr
	}

	// 2. 查交易 + 时效校验（FOR UPDATE 锁行：防双击并发反向扣款 2 倍余额回补）
	trx, err := q.LockTransactionForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "transaction not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get tx: : "+err.Error())
	}
	if trx.Status != "completed" {
		return echo.NewHTTPError(http.StatusBadRequest, "transaction is not in completed status")
	}
	if time.Since(trx.TransactionTime.Time) > 7*24*time.Hour {
		return echo.NewHTTPError(http.StatusBadRequest, "transaction is older than 7 days")
	}

	// 3. 按 kind 反向
	switch trx.Kind {
	case "sale":
		if err := reverseSale(ctx, q, t.ID, trx, claims.UserID); err != nil {
			return err
		}
	case "recharge":
		if err := reverseRecharge(ctx, q, t.ID, trx, claims.UserID); err != nil {
			return err
		}
	case "credit_settlement":
		if err := reverseSettlement(ctx, q, t.ID, trx, claims.UserID); err != nil {
			return err
		}
	default:
		return echo.NewHTTPError(http.StatusInternalServerError, "unknown transaction kind: "+trx.Kind)
	}

	// 4. 标记交易为 voided
	if err := q.VoidTransaction(ctx, sqlc.VoidTransactionParams{
		ID:             id,
		VoidedByUserID: pgtype.UUID{Bytes: claims.UserID, Valid: true},
		VoidedByName:   strPtr(claims.Email),
		Reason:         &req.Reason,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "mark voided: : "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]any{"message": "transaction voided", "id": id})
}

// checkVoidEnabled 校验 enable_transaction_void=true 且未过期（10 分钟）+ 当前操作者 == 开启者
//
// 安全模型：
//   撤单需要先通过超级密码开启 10 分钟窗口，期间只有开启者本人能撤
//   防止多超管租户时 A 开启后 B 趁窗口期撤任意历史交易（见安全审查报告攻击链 C）
func checkVoidEnabled(ctx context.Context, q *sqlc.Queries, tenantID, currentUserID uuid.UUID) error {
	enabled, err := getBoolSetting(ctx, q, tenantID, "enable_transaction_void")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "get setting: : "+err.Error())
	}
	if !enabled {
		return echo.NewHTTPError(http.StatusForbidden, "交易撤销功能未启用")
	}
	ts, err := getTimestampSetting(ctx, q, tenantID, "void_enabled_at")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "get void_enabled_at: : "+err.Error())
	}
	if ts == nil || time.Since(*ts) > 10*time.Minute {
		return echo.NewHTTPError(http.StatusForbidden, "撤销功能已过期，请重新开启")
	}
	enablerStr, err := getStringSetting(ctx, q, tenantID, "void_enabled_by")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "get void_enabled_by: : "+err.Error())
	}
	// enablerStr 空视为旧数据，放宽（避免现有租户受影响）；生产强制可改为 Forbidden
	if enablerStr != "" {
		enabler, err := uuid.Parse(enablerStr)
		if err != nil || enabler != currentUserID {
			return echo.NewHTTPError(http.StatusForbidden, "只有开启撤销窗口的管理员本人可以撤销交易")
		}
	}
	return nil
}

// reverseSale：对每条 card_balance_log 反向还原
func reverseSale(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, trx sqlc.Transaction, operatorID uuid.UUID) error {
	logs, err := q.ListCardBalanceLogsByTx(ctx, pgtype.UUID{Bytes: trx.ID, Valid: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list logs: : "+err.Error())
	}
	for _, l := range logs {
		if l.ChangeType != "consume" {
			continue // 只反向消费型扣款
		}
		restore := l.Delta.Neg() // consume 是负数，反向为正
		card, err := q.AdjustCardBalance(ctx, sqlc.AdjustCardBalanceParams{
			ID:      l.CardID,
			Balance: restore,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "restore balance: : "+err.Error())
		}
		if _, err := q.CreateCardBalanceLog(ctx, sqlc.CreateCardBalanceLogParams{
			TenantID:       tenantID,
			CardID:         l.CardID,
			ChangeType:     "void_restore",
			Delta:          restore,
			BalanceBefore:  l.BalanceAfter,
			BalanceAfter:   card.Balance,
			TransactionID:  pgtype.UUID{Bytes: trx.ID, Valid: true},
			OperatorUserID: pgtype.UUID{Bytes: operatorID, Valid: true},
			Note:           strPtr("void sale"),
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "log restore: : "+err.Error())
		}
	}
	return nil
}

// reverseRecharge：冻结卡 + 清零 + 反向 issue 流水
// 不硬删卡，保留历史便于审计
func reverseRecharge(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, trx sqlc.Transaction, operatorID uuid.UUID) error {
	if !trx.CardID.Valid {
		return nil // 没关联卡，跳过
	}
	cardID := uuid.UUID(trx.CardID.Bytes)
	card, err := q.LockCardForUpdate(ctx, cardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // 卡已被删，跳过
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "get card: : "+err.Error())
	}
	if !card.Balance.Equal(trx.TotalAmount) {
		return echo.NewHTTPError(http.StatusBadRequest, "卡已被使用（余额与办卡金额不符），不能撤销办卡")
	}
	// 清零余额
	before := card.Balance
	cleared, err := q.AdjustCardBalance(ctx, sqlc.AdjustCardBalanceParams{
		ID:      cardID,
		Balance: before.Neg(),
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "zero balance: : "+err.Error())
	}
	// 反向流水
	if _, err := q.CreateCardBalanceLog(ctx, sqlc.CreateCardBalanceLogParams{
		TenantID:       tenantID,
		CardID:         cardID,
		ChangeType:     "void_restore",
		Delta:          before.Neg(),
		BalanceBefore:  before,
		BalanceAfter:   cleared.Balance,
		TransactionID:  pgtype.UUID{Bytes: trx.ID, Valid: true},
		OperatorUserID: pgtype.UUID{Bytes: operatorID, Valid: true},
		Note:           strPtr("void recharge"),
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "log: : "+err.Error())
	}
	// 冻结卡
	frozen := "frozen"
	if _, err := q.UpdateCard(ctx, sqlc.UpdateCardParams{
		ID:     cardID,
		Status: &frozen,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "freeze card: : "+err.Error())
	}
	return nil
}

// reverseSettlement：恢复挂账未清 + 若走卡再回补
func reverseSettlement(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, trx sqlc.Transaction, operatorID uuid.UUID) error {
	// 恢复所有被这笔 settlement 清掉的挂账
	credits, err := q.ListCreditsBySettlementTx(ctx, pgtype.UUID{Bytes: trx.ID, Valid: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list credits: : "+err.Error())
	}
	for _, cr := range credits {
		if err := q.UnsettleCredit(ctx, cr.ID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "unsettle: : "+err.Error())
		}
	}

	// 如果清账时走的会员卡，回补余额 + 反向流水
	if trx.CardID.Valid {
		cardID := uuid.UUID(trx.CardID.Bytes)
		logs, err := q.ListCardBalanceLogsByTx(ctx, pgtype.UUID{Bytes: trx.ID, Valid: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "list logs: : "+err.Error())
		}
		for _, l := range logs {
			if l.ChangeType != "consume" {
				continue
			}
			restore := l.Delta.Neg()
			card, err := q.AdjustCardBalance(ctx, sqlc.AdjustCardBalanceParams{
				ID:      l.CardID,
				Balance: restore,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "restore: : "+err.Error())
			}
			_ = cardID // 静默未用
			if _, err := q.CreateCardBalanceLog(ctx, sqlc.CreateCardBalanceLogParams{
				TenantID:       tenantID,
				CardID:         l.CardID,
				ChangeType:     "void_restore",
				Delta:          restore,
				BalanceBefore:  l.BalanceAfter,
				BalanceAfter:   card.Balance,
				TransactionID:  pgtype.UUID{Bytes: trx.ID, Valid: true},
				OperatorUserID: pgtype.UUID{Bytes: operatorID, Valid: true},
				Note:           strPtr("void settlement"),
			}); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "log: : "+err.Error())
			}
		}
	}
	return nil
}

// --- tenant_settings helper ---

func getStringSetting(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, key string) (string, error) {
	row, err := q.GetTenantSetting(ctx, sqlc.GetTenantSettingParams{TenantID: tenantID, Key: key})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	var s *string
	if err := json.Unmarshal(row.Value, &s); err != nil || s == nil {
		return "", nil
	}
	return *s, nil
}

func getBoolSetting(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, key string) (bool, error) {
	row, err := q.GetTenantSetting(ctx, sqlc.GetTenantSettingParams{TenantID: tenantID, Key: key})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	var b bool
	if err := json.Unmarshal(row.Value, &b); err != nil {
		return false, nil
	}
	return b, nil
}

func getTimestampSetting(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, key string) (*time.Time, error) {
	row, err := q.GetTenantSetting(ctx, sqlc.GetTenantSettingParams{TenantID: tenantID, Key: key})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	// JSON 可能是 null / "2026-04-15T10:00:00Z"
	var s *string
	if err := json.Unmarshal(row.Value, &s); err != nil {
		return nil, err
	}
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// 静默未用变量
var _ = decimal.Zero
