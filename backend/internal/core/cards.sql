-- name: ListCardsByMember :many
-- 会员名下的所有卡，附带卡型模板字段（前端显示用）
SELECT
  c.id, c.tenant_id, c.member_id, c.card_type_id,
  c.final_price, c.final_discount_rate, c.balance,
  c.issued_at, c.expires_at, c.status, c.notes,
  c.legacy_id, c.created_at, c.updated_at,
  ct.name          AS card_type_name,
  ct.price         AS card_type_price,
  ct.discount_rate AS card_type_discount_rate
FROM cards c
JOIN card_types ct ON ct.id = c.card_type_id
WHERE c.member_id = $1
ORDER BY c.issued_at DESC;

-- name: ListCardsByMemberForUpdate :many
-- 挂账定价路径：先锁会员全部卡再取折扣率，防并发把卡扣空/作废后仍按旧折扣入账
-- 带 member_id / card_type_name：挂账分支的扣卡校验直接复用锁行，不再二次 SELECT
-- ORDER BY c.id 与 LockCardsForUpdate 同序加锁，防死锁
SELECT c.id, c.member_id, c.final_discount_rate, c.balance, c.status, c.expires_at,
       ct.name AS card_type_name
FROM cards c
JOIN card_types ct ON ct.id = c.card_type_id
WHERE c.member_id = $1
ORDER BY c.id
FOR UPDATE OF c;

-- name: LockCardForUpdate :one
-- 扣款路径专用：SELECT FOR UPDATE 锁单行，防并发双扣
-- 注：JOIN 的 card_types 不加锁（只读关联）
SELECT
  c.id, c.tenant_id, c.member_id, c.card_type_id,
  c.final_price, c.final_discount_rate, c.balance,
  c.issued_at, c.expires_at, c.status, c.notes,
  c.legacy_id, c.created_at, c.updated_at,
  ct.name          AS card_type_name,
  ct.price         AS card_type_price,
  ct.discount_rate AS card_type_discount_rate
FROM cards c
JOIN card_types ct ON ct.id = c.card_type_id
WHERE c.id = $1
FOR UPDATE OF c;

-- name: LockCardsForUpdate :many
-- 多卡支付路径：一次性锁所有目标卡，避免 N+1 RTT
-- ORDER BY c.id 关键：所有事务用同序加锁，防死锁
-- 调用方需在传入 ids 前自行排序（或保证传入顺序确定）
SELECT
  c.id, c.tenant_id, c.member_id, c.card_type_id,
  c.final_price, c.final_discount_rate, c.balance,
  c.issued_at, c.expires_at, c.status, c.notes,
  c.legacy_id, c.created_at, c.updated_at,
  ct.name          AS card_type_name,
  ct.price         AS card_type_price,
  ct.discount_rate AS card_type_discount_rate
FROM cards c
JOIN card_types ct ON ct.id = c.card_type_id
WHERE c.id = ANY(sqlc.arg('ids')::uuid[])
ORDER BY c.id
FOR UPDATE OF c;

-- name: IssueCard :one
-- 最小单元：只建 card 不建 transaction（开卡+收款联动由 handler 层组合）
-- balance 默认 = final_price；调用方也可显式传（比如迁移时保留老余额）
INSERT INTO cards (
  tenant_id, member_id, card_type_id,
  final_price, final_discount_rate, balance,
  expires_at, notes
)
VALUES (
  $1, $2, $3, $4, $5,
  COALESCE(sqlc.narg('balance')::numeric, $4),
  sqlc.narg('expires_at')::timestamptz,
  sqlc.narg('notes')::text
)
RETURNING *;

-- name: UpdateCard :one
UPDATE cards
SET final_price         = COALESCE(sqlc.narg('final_price')::numeric,         final_price),
    final_discount_rate = COALESCE(sqlc.narg('final_discount_rate')::numeric, final_discount_rate),
    status              = COALESCE(sqlc.narg('status')::text,                 status),
    expires_at          = COALESCE(sqlc.narg('expires_at')::timestamptz,      expires_at),
    notes               = COALESCE(sqlc.narg('notes')::text,                  notes),
    updated_at          = now()
WHERE id = $1
RETURNING *;
