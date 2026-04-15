-- name: ListPendingCreditsByMember :many
-- 未清挂账
SELECT * FROM member_credits
WHERE member_id = $1 AND settled_at IS NULL
ORDER BY charged_at DESC;

-- name: ListCreditsByMember :many
SELECT * FROM member_credits
WHERE member_id = $1
ORDER BY charged_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateCredit :one
-- 新增挂账（消费后没付款）
INSERT INTO member_credits (
  tenant_id, member_id, amount, summary, charged_at, charged_tx_id
)
VALUES (
  $1, $2, $3,
  sqlc.narg('summary')::text,
  COALESCE(sqlc.narg('charged_at')::timestamptz, now()),
  sqlc.narg('charged_tx_id')::uuid
)
RETURNING *;

-- name: GetCreditByID :one
SELECT * FROM member_credits WHERE id = $1;

-- name: MarkCreditSettled :one
-- 标记单笔挂账已清（由 handler 先建 credit_settlement 交易，再调此函数）
UPDATE member_credits
SET settled_at       = $2,
    settlement_tx_id = $3,
    updated_at       = now()
WHERE id = $1 AND settled_at IS NULL
RETURNING *;

-- name: DeleteCredit :exec
-- 撤销错误录入的挂账（仅未清的可以直接删）
DELETE FROM member_credits WHERE id = $1 AND settled_at IS NULL;

-- name: SumUnsettledByMember :one
SELECT COALESCE(SUM(amount), 0)::numeric AS total_unsettled
FROM member_credits
WHERE member_id = $1 AND settled_at IS NULL;

-- name: MarkCreditsSettledByMember :exec
-- 批量清挂账：把会员当前所有未清挂账一次性标记已清
UPDATE member_credits
SET settled_at       = $2,
    settlement_tx_id = $3,
    updated_at       = now()
WHERE member_id = $1 AND settled_at IS NULL;

-- name: ListCreditsBySettlementTx :many
-- 撤销清账交易时，查哪些挂账被这笔清账清掉（用于恢复）
SELECT * FROM member_credits
WHERE settlement_tx_id = $1;

-- name: UnsettleCredit :exec
-- 撤销挂账清账：把 settled_at / settlement_tx_id 重置回 NULL
UPDATE member_credits
SET settled_at       = NULL,
    settlement_tx_id = NULL,
    updated_at       = now()
WHERE id = $1;
