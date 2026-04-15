-- name: CreateCardBalanceLog :one
INSERT INTO card_balance_logs (
  tenant_id, card_id, transaction_id,
  change_type, delta, balance_before, balance_after,
  operator_user_id, note
) VALUES (
  $1, $2,
  sqlc.narg('transaction_id')::uuid,
  $3, $4, $5, $6,
  sqlc.narg('operator_user_id')::uuid,
  sqlc.narg('note')::text
) RETURNING *;

-- name: AdjustCardBalance :one
-- 原子地给 cards.balance 加上 delta（负数即扣款），并带乐观检查（不许扣成负数）
UPDATE cards
SET balance = balance + $2,
    updated_at = now()
WHERE id = $1
  AND balance + $2 >= 0
RETURNING *;

-- name: ListCardBalanceLogs :many
SELECT * FROM card_balance_logs
WHERE card_id = $1
ORDER BY occurred_at DESC;

-- name: ListCardBalanceLogsByTx :many
SELECT * FROM card_balance_logs
WHERE transaction_id = $1
ORDER BY occurred_at ASC;

-- name: ListCardSnapshotsByTxIDs :many
-- 批量取多笔交易的卡余额快照（POS 今日记录 hover 显示用）
-- 同时 join 卡类型名，前端不需二次查询
SELECT
  l.transaction_id,
  l.card_id,
  l.balance_before,
  l.balance_after,
  l.delta,
  l.change_type,
  ct.name AS card_type_name
FROM card_balance_logs l
JOIN cards c       ON c.id = l.card_id
JOIN card_types ct ON ct.id = c.card_type_id
WHERE l.transaction_id = ANY($1::uuid[])
ORDER BY l.transaction_id, l.occurred_at ASC;
