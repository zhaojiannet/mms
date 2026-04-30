package transactions

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/pgtypex"
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
	if httpErr := checkVoidEnabled(c, q, t.ID, claims.UserID); httpErr != nil {
		return httpErr
	}

	// 2. 查交易 + 时效校验（FOR UPDATE 锁行：防双击并发反向扣款 2 倍余额回补）
	trx, err := q.LockTransactionForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "transaction not found")
		}
		return mw.InternalError(c, "void.get_tx", err)
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
		if err := reverseSale(c, q, t.ID, trx, claims.UserID); err != nil {
			return err
		}
	case "recharge":
		if err := reverseRecharge(c, q, t.ID, trx, claims.UserID); err != nil {
			return err
		}
	case "credit_settlement":
		if err := reverseSettlement(c, q, t.ID, trx, claims.UserID); err != nil {
			return err
		}
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "unknown transaction kind: "+trx.Kind)
	}

	// 4. 标记交易为 voided
	if err := q.VoidTransaction(ctx, sqlc.VoidTransactionParams{
		ID:             id,
		VoidedByUserID: pgtype.UUID{Bytes: claims.UserID, Valid: true},
		VoidedByName:   pgtypex.StrPtr(claims.Email),
		Reason:         &req.Reason,
	}); err != nil {
		return mw.InternalError(c, "void.mark_voided", err)
	}

	return c.JSON(http.StatusOK, map[string]any{"message": "transaction voided", "id": id})
}

// checkVoidEnabled 校验 enable_transaction_void=true 且未过期（10 分钟）+ 当前操作者 == 开启者
//
// 安全模型：
//
//	撤单需要先通过超级密码开启 10 分钟窗口，期间只有开启者本人能撤
//	防止多超管租户时 A 开启后 B 趁窗口期撤任意历史交易（见安全审查报告攻击链 C）
func checkVoidEnabled(c *echo.Context, q *sqlc.Queries, tenantID, currentUserID uuid.UUID) error {
	ctx := c.Request().Context()
	rows, err := q.GetTenantSettingsByKeys(ctx, sqlc.GetTenantSettingsByKeysParams{
		TenantID: tenantID,
		Keys:     []string{"enable_transaction_void", "void_enabled_at", "void_enabled_by"},
	})
	if err != nil {
		return mw.InternalError(c, "void.get_settings", err)
	}
	values := make(map[string][]byte, 3)
	for _, r := range rows {
		values[r.Key] = r.Value
	}

	// JSONB 损坏 → fail-secure（默认 false），warn 帮运维定位
	var enabled bool
	if raw, ok := values["enable_transaction_void"]; ok {
		if err := json.Unmarshal(raw, &enabled); err != nil {
			slog.Warn("void: enable_transaction_void unmarshal failed", "tenant_id", tenantID, "err", err)
		}
	}
	if !enabled {
		return echo.NewHTTPError(http.StatusForbidden, "交易撤销功能未启用")
	}

	var ts *time.Time
	if raw, ok := values["void_enabled_at"]; ok {
		var s *string
		if err := json.Unmarshal(raw, &s); err != nil {
			slog.Warn("void: void_enabled_at unmarshal failed", "tenant_id", tenantID, "err", err)
		} else if s != nil && *s != "" {
			if t, err := time.Parse(time.RFC3339, *s); err == nil {
				ts = &t
			}
		}
	}
	if ts == nil || time.Since(*ts) > 10*time.Minute {
		return echo.NewHTTPError(http.StatusForbidden, "撤销功能已过期，请重新开启")
	}

	// fail-closed：enabled_by 缺失/解析失败一律拒绝。
	// 旧数据残留（升级前未写 enabled_by）必须先重新开启窗口才能撤单，
	// 防止 A 开启 B 撤单的多管理员越权（见函数顶部安全模型注释）。
	var enablerStr string
	if raw, ok := values["void_enabled_by"]; ok {
		var s *string
		if err := json.Unmarshal(raw, &s); err != nil {
			slog.Warn("void: void_enabled_by unmarshal failed", "tenant_id", tenantID, "err", err)
		} else if s != nil {
			enablerStr = *s
		}
	}
	if enablerStr == "" {
		slog.Warn("void: enabled_by missing; rejecting (legacy or corrupted setting)", "tenant_id", tenantID)
		return echo.NewHTTPError(http.StatusForbidden, "撤销窗口异常，请重新开启")
	}
	enabler, err := uuid.Parse(enablerStr)
	if err != nil {
		slog.Warn("void: enabled_by uuid parse failed", "tenant_id", tenantID, "value", enablerStr, "err", err)
		return echo.NewHTTPError(http.StatusForbidden, "撤销窗口异常，请重新开启")
	}
	if enabler != currentUserID {
		return echo.NewHTTPError(http.StatusForbidden, "只有开启撤销窗口的管理员本人可以撤销交易")
	}
	return nil
}

// reverseSale：对每条 card_balance_log 反向还原
func reverseSale(c *echo.Context, q *sqlc.Queries, tenantID uuid.UUID, trx sqlc.Transaction, operatorID uuid.UUID) error {
	ctx := c.Request().Context()
	logs, err := q.ListCardBalanceLogsByTx(ctx, pgtype.UUID{Bytes: trx.ID, Valid: true})
	if err != nil {
		return mw.InternalError(c, "void.sale.list_logs", err)
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
			return mw.InternalError(c, "void.sale.restore_balance", err)
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
			Note:           pgtypex.StrPtr("void sale"),
		}); err != nil {
			return mw.InternalError(c, "void.sale.log_restore", err)
		}
	}
	return nil
}

// reverseRecharge：冻结卡 + 清零 + 反向 issue 流水
// 不硬删卡，保留历史便于审计
func reverseRecharge(c *echo.Context, q *sqlc.Queries, tenantID uuid.UUID, trx sqlc.Transaction, operatorID uuid.UUID) error {
	ctx := c.Request().Context()
	if !trx.CardID.Valid {
		return nil // 没关联卡，跳过
	}
	cardID := uuid.UUID(trx.CardID.Bytes)
	card, err := q.LockCardForUpdate(ctx, cardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // 卡已被删，跳过
		}
		return mw.InternalError(c, "void.recharge.get_card", err)
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
		return mw.InternalError(c, "void.recharge.zero_balance", err)
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
		Note:           pgtypex.StrPtr("void recharge"),
	}); err != nil {
		return mw.InternalError(c, "void.recharge.log", err)
	}
	// 冻结卡
	frozen := "frozen"
	if _, err := q.UpdateCard(ctx, sqlc.UpdateCardParams{
		ID:     cardID,
		Status: &frozen,
	}); err != nil {
		return mw.InternalError(c, "void.recharge.freeze_card", err)
	}
	return nil
}

// reverseSettlement：恢复挂账未清 + 若走卡再回补
func reverseSettlement(c *echo.Context, q *sqlc.Queries, tenantID uuid.UUID, trx sqlc.Transaction, operatorID uuid.UUID) error {
	ctx := c.Request().Context()
	// 恢复所有被这笔 settlement 清掉的挂账
	credits, err := q.ListCreditsBySettlementTx(ctx, pgtype.UUID{Bytes: trx.ID, Valid: true})
	if err != nil {
		return mw.InternalError(c, "void.settle.list_credits", err)
	}
	for _, cr := range credits {
		if err := q.UnsettleCredit(ctx, cr.ID); err != nil {
			return mw.InternalError(c, "void.settle.unsettle", err)
		}
	}

	// 如果清账时走的会员卡，回补余额 + 反向流水
	if trx.CardID.Valid {
		logs, err := q.ListCardBalanceLogsByTx(ctx, pgtype.UUID{Bytes: trx.ID, Valid: true})
		if err != nil {
			return mw.InternalError(c, "void.settle.list_logs", err)
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
				return mw.InternalError(c, "void.settle.restore", err)
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
				Note:           pgtypex.StrPtr("void settlement"),
			}); err != nil {
				return mw.InternalError(c, "void.settle.log", err)
			}
		}
	}
	return nil
}

