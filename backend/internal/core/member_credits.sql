-- name: ListPendingCreditsByMember :many
-- 未清挂账
SELECT * FROM member_credits
WHERE member_id = $1 AND settled_at IS NULL
ORDER BY charged_at DESC;

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

-- name: LockCreditForUpdate :one
-- 单笔清账路径：防双击重复创建 settlement 交易
SELECT * FROM member_credits WHERE id = $1 FOR UPDATE;

-- name: LockPendingCreditsByMember :many
-- 批量清账路径：一次锁定该会员所有未清挂账；再有并发请求会阻塞到本 tx commit
SELECT * FROM member_credits
WHERE member_id = $1 AND settled_at IS NULL
ORDER BY charged_at DESC
FOR UPDATE;

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

-- name: MarkCreditsSettledByIDs :exec
-- 批量清挂账：只更新调用方已 FOR UPDATE 锁定的那批 id。
-- 不能按 member_id+settled_at 谓词全量更新：READ COMMITTED 下锁定与
-- 更新之间并发插入的新挂账会被一并标记已清，金额却不在结算总额里
UPDATE member_credits
SET settled_at       = sqlc.arg('settled_at'),
    settlement_tx_id = sqlc.arg('settlement_tx_id'),
    updated_at       = now()
WHERE id = ANY(sqlc.arg('ids')::uuid[]) AND settled_at IS NULL;

-- name: ListCreditsBySettlementTx :many
-- 撤销清账交易时，查哪些挂账被这笔清账清掉（用于恢复）
SELECT * FROM member_credits
WHERE settlement_tx_id = $1;

-- name: ListCreditsByChargedTx :many
-- 撤销消费交易时，查这笔交易产生的挂账（未清的删除，已清的阻止撤单）
SELECT * FROM member_credits
WHERE charged_tx_id = $1;

-- name: UnsettleCredit :exec
-- 撤销挂账清账：把 settled_at / settlement_tx_id 重置回 NULL
UPDATE member_credits
SET settled_at       = NULL,
    settlement_tx_id = NULL,
    updated_at       = now()
WHERE id = $1;
