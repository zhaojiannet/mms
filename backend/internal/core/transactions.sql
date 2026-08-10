-- name: GetTransactionByID :one
SELECT * FROM transactions WHERE id = $1;

-- name: LockTransactionForUpdate :one
-- 撤单路径：防止双击重复反向扣款
SELECT * FROM transactions WHERE id = $1 FOR UPDATE;

-- name: LookupTransactionsByIDs :many
-- 批量返回 id + 简要信息（给操作日志对象列显示用）
SELECT t.id,
       t.total_amount,
       COALESCE(m.name, t.customer_name, '非会员')::text AS display_name
FROM transactions t
LEFT JOIN members m ON m.id = t.member_id
WHERE t.id = ANY($1::uuid[]);

-- name: CreateTransaction :one
-- 统一入口：消费 / 办卡 / 清挂账 都走这里，kind 区分
-- discount_amount 可空默认 0；transaction_time 可空默认 now()
INSERT INTO transactions (
  tenant_id, kind, status,
  member_id, customer_name, staff_id,
  payment_method_id, card_id,
  total_amount, actual_paid_amount, discount_amount,
  transaction_time, summary, notes
) VALUES (
  $1, $2, 'completed',
  sqlc.narg('member_id')::uuid,
  sqlc.narg('customer_name')::text,
  sqlc.narg('staff_id')::uuid,
  $3,
  sqlc.narg('card_id')::uuid,
  $4, $5,
  COALESCE(sqlc.narg('discount_amount')::numeric, 0),
  COALESCE(sqlc.narg('transaction_time')::timestamptz, now()),
  sqlc.narg('summary')::text,
  sqlc.narg('notes')::text
) RETURNING *;

-- name: ListTransactions :many
-- 列出交易，附带会员名 / 卡类型名 / 项目数 (POS 收银底部 / 流水页展示)
--   status:         可选精确过滤（'completed' / 'voided'）
--   include_voided: 兼容参数，true 时等同不限 status；否则仅 completed（当 status 未指定时生效）
SELECT
  sqlc.embed(t),
  m.name  AS member_name,
  -- 卡展示名：自定义面值卡按实际面值展示并带折扣后缀（迁移把 isCustomCard 塌缩为
  -- final_price，与卡型标价不同即为自定义），老板在老系统看惯的「自定义面值卡(¥N) n折」不丢
  COALESCE(CASE WHEN c.final_price <> ct.price
             THEN '自定义面值卡(¥' || to_char(c.final_price, 'FM9999999990.00') || ')'
             ELSE ct.name END
        || CASE WHEN c.final_discount_rate < 1
             THEN ' ' || rtrim(rtrim(to_char(c.final_discount_rate * 10, 'FM90.9'), '0'), '.') || '折'
             ELSE '' END, '')::text AS card_type_name,
  COALESCE((SELECT SUM(quantity)::int FROM transaction_items WHERE transaction_id = t.id), 0)::int AS item_qty,
  -- 明细标价合计：迁移把加价单 total 抬平到实收，原应收只在明细单价里，前端靠它显示「加 ¥N」
  COALESCE((SELECT SUM(ti.price * ti.quantity) FROM transaction_items ti WHERE ti.transaction_id = t.id), 0)::numeric(10,2) AS items_total,
  -- 挂账登记交易（0 实收）关联的挂账金额：流水里显示「挂账 ¥N」而不是令人困惑的 ¥0
  COALESCE((SELECT mc.amount FROM member_credits mc WHERE mc.charged_tx_id = t.id LIMIT 1), 0)::numeric(10,2) AS credit_amount,
  -- summary 为空时（老系统迁入的交易普遍如此）前端回退显示明细聚合，服务项目列不再一片「—」
  (SELECT string_agg(ti.service_name_snapshot
            || CASE WHEN ti.quantity > 1 THEN '×' || ti.quantity ELSE '' END, '、')
   FROM transaction_items ti WHERE ti.transaction_id = t.id) AS items_summary
FROM transactions t
LEFT JOIN members m     ON m.id = t.member_id
LEFT JOIN cards c       ON c.id = t.card_id
LEFT JOIN card_types ct ON ct.id = c.card_type_id
WHERE (sqlc.narg('start_date')::timestamptz IS NULL OR t.transaction_time >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz   IS NULL OR t.transaction_time <  sqlc.narg('end_date')::timestamptz)
  AND (sqlc.narg('kind')::text              IS NULL OR t.kind             =  sqlc.narg('kind')::text)
  AND (sqlc.narg('member_id')::uuid         IS NULL OR t.member_id       =  sqlc.narg('member_id')::uuid)
  -- 搜索范围与老系统对齐并扩展：会员姓名 / 手机号 / 非会员姓名 / 项目摘要 /
  -- 明细服务名；纯数字输入再按金额精确匹配（应收或实收）
  AND (sqlc.narg('search')::text IS NULL
       OR m.name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR m.phone ILIKE '%' || sqlc.narg('search')::text || '%'
       OR t.customer_name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR t.summary ILIKE '%' || sqlc.narg('search')::text || '%'
       OR EXISTS (SELECT 1 FROM transaction_items ti2
                  WHERE ti2.transaction_id = t.id
                    AND ti2.service_name_snapshot ILIKE '%' || sqlc.narg('search')::text || '%')
       OR (sqlc.narg('search')::text ~ '^[0-9]+(\.[0-9]{1,2})?$'
           AND (t.total_amount = sqlc.narg('search')::numeric
                OR t.actual_paid_amount = sqlc.narg('search')::numeric)))
  AND (
    sqlc.narg('status')::text IS NOT NULL AND t.status = sqlc.narg('status')::text
    OR sqlc.narg('status')::text IS NULL AND (sqlc.narg('include_voided')::bool IS TRUE OR t.status = 'completed')
  )
ORDER BY t.transaction_time DESC, t.id DESC
LIMIT $1 OFFSET $2;

-- name: MaxTransactionsUpdatedAt :one
-- 给 ETag 用：返回当前 filter 范围内最近一次写入的时间
-- 同 ListTransactions 的 filter 子句保持一致，否则 ETag 会跨 filter 错误命中
SELECT COALESCE(MAX(updated_at), '1970-01-01 00:00:00+00'::timestamptz)::timestamptz AS max_updated_at
FROM transactions
WHERE (sqlc.narg('start_date')::timestamptz IS NULL OR transaction_time >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz   IS NULL OR transaction_time <  sqlc.narg('end_date')::timestamptz)
  AND (sqlc.narg('kind')::text              IS NULL OR kind              =  sqlc.narg('kind')::text)
  AND (sqlc.narg('member_id')::uuid         IS NULL OR member_id        =  sqlc.narg('member_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL
       OR EXISTS (SELECT 1 FROM members mm WHERE mm.id = transactions.member_id
                  AND (mm.name ILIKE '%' || sqlc.narg('search')::text || '%'
                       OR mm.phone ILIKE '%' || sqlc.narg('search')::text || '%'))
       OR customer_name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR summary ILIKE '%' || sqlc.narg('search')::text || '%'
       OR EXISTS (SELECT 1 FROM transaction_items ti2
                  WHERE ti2.transaction_id = transactions.id
                    AND ti2.service_name_snapshot ILIKE '%' || sqlc.narg('search')::text || '%')
       OR (sqlc.narg('search')::text ~ '^[0-9]+(\.[0-9]{1,2})?$'
           AND (total_amount = sqlc.narg('search')::numeric
                OR actual_paid_amount = sqlc.narg('search')::numeric)))
  AND (
    sqlc.narg('status')::text IS NOT NULL AND status = sqlc.narg('status')::text
    OR sqlc.narg('status')::text IS NULL AND (sqlc.narg('include_voided')::bool IS TRUE OR status = 'completed')
  );

-- name: CountTransactionsBy :one
SELECT count(*) FROM transactions
WHERE (sqlc.narg('start_date')::timestamptz IS NULL OR transaction_time >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz   IS NULL OR transaction_time <  sqlc.narg('end_date')::timestamptz)
  AND (sqlc.narg('kind')::text              IS NULL OR kind              =  sqlc.narg('kind')::text)
  AND (sqlc.narg('member_id')::uuid         IS NULL OR member_id        =  sqlc.narg('member_id')::uuid)
  AND (sqlc.narg('search')::text IS NULL
       OR EXISTS (SELECT 1 FROM members mm WHERE mm.id = transactions.member_id
                  AND (mm.name ILIKE '%' || sqlc.narg('search')::text || '%'
                       OR mm.phone ILIKE '%' || sqlc.narg('search')::text || '%'))
       OR customer_name ILIKE '%' || sqlc.narg('search')::text || '%'
       OR summary ILIKE '%' || sqlc.narg('search')::text || '%'
       OR EXISTS (SELECT 1 FROM transaction_items ti2
                  WHERE ti2.transaction_id = transactions.id
                    AND ti2.service_name_snapshot ILIKE '%' || sqlc.narg('search')::text || '%')
       OR (sqlc.narg('search')::text ~ '^[0-9]+(\.[0-9]{1,2})?$'
           AND (total_amount = sqlc.narg('search')::numeric
                OR actual_paid_amount = sqlc.narg('search')::numeric)))
  AND (
    sqlc.narg('status')::text IS NOT NULL AND status = sqlc.narg('status')::text
    OR sqlc.narg('status')::text IS NULL AND (sqlc.narg('include_voided')::bool IS TRUE OR status = 'completed')
  );

-- name: VoidTransaction :exec
-- 软撤单（A4 实现完整逻辑；此 query 仅负责更新状态 + 审计字段）
UPDATE transactions
SET status            = 'voided',
    voided_at         = now(),
    voided_by_user_id = $2,
    voided_by_name    = $3,
    void_reason       = sqlc.narg('reason')::text,
    updated_at        = now()
WHERE id = $1 AND status = 'completed';
