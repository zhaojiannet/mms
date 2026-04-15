-- name: ReportBusiness :one
-- 营业报表：区间内的综合统计（对齐老系统口径）
--   sale_revenue     : kind='sale' 实收合计（含挂账场景）
--   credit_charged   : 区间内新增的挂账金额（老口径"挂账也算营业"）
--   card_consumption : sale 中走会员卡部分
--   recharge_amount  : kind='recharge' 实收合计（办卡/充值）
--   customer_count   : 客数（有 member_id 按会员去重；无 member_id 按 tx id 去重）
SELECT
  COALESCE(SUM(CASE WHEN kind='sale'     AND status='completed' THEN actual_paid_amount ELSE 0 END), 0)::numeric AS sale_revenue,
  COALESCE(SUM(CASE WHEN kind='recharge' AND status='completed' THEN actual_paid_amount ELSE 0 END), 0)::numeric AS recharge_amount,
  COALESCE(SUM(CASE WHEN kind='sale' AND status='completed' AND card_id IS NOT NULL THEN actual_paid_amount ELSE 0 END), 0)::numeric AS card_consumption,
  COUNT(DISTINCT CASE WHEN kind='sale' AND status='completed' THEN COALESCE(member_id::text, id::text) END)::bigint AS customer_count
FROM transactions
WHERE transaction_time >= $1 AND transaction_time < $2;

-- name: ReportCreditsCharged :one
-- 区间内新增的挂账（营业报表要加上）
SELECT COALESCE(SUM(amount), 0)::numeric AS total
FROM member_credits
WHERE charged_at >= $1 AND charged_at < $2;

-- name: ReportServiceRanking :many
SELECT
  s.id           AS service_id,
  s.name         AS service_name,
  SUM(ti.price * ti.quantity)::numeric AS total_sales,
  SUM(ti.quantity)::bigint              AS total_count
FROM transaction_items ti
JOIN services     s ON s.id = ti.service_id
JOIN transactions t ON t.id = ti.transaction_id
WHERE t.status = 'completed'
  AND t.kind = 'sale'
  AND t.transaction_time >= $1
  AND t.transaction_time <  $2
GROUP BY s.id, s.name
ORDER BY total_sales DESC
LIMIT $3 OFFSET $4;

-- name: ReportSleepingMembers :many
-- 90 天无 sale 交易的活跃会员
SELECT
  m.id, m.name, m.phone, m.created_at,
  MAX(t.transaction_time) AS last_visit
FROM members m
LEFT JOIN transactions t
  ON t.member_id = m.id AND t.kind = 'sale' AND t.status = 'completed'
WHERE m.status = 'active'
GROUP BY m.id, m.name, m.phone, m.created_at
HAVING MAX(t.transaction_time) IS NULL OR MAX(t.transaction_time) < $1
ORDER BY MAX(t.transaction_time) ASC NULLS FIRST
LIMIT $2 OFFSET $3;

-- name: ReportMemberRanking :many
SELECT
  m.id, m.name, m.phone,
  SUM(t.actual_paid_amount)::numeric AS total_consumption
FROM transactions t
JOIN members m ON m.id = t.member_id
WHERE t.kind = 'sale'
  AND t.status = 'completed'
  AND t.transaction_time >= $1
  AND t.transaction_time <  $2
GROUP BY m.id, m.name, m.phone
ORDER BY total_consumption DESC
LIMIT $3 OFFSET $4;

-- name: ReportBirthdayReminders :many
-- 未来 15 天内生日（不看年份，按月日）
SELECT id, name, phone, birthday
FROM members
WHERE status = 'active'
  AND birthday IS NOT NULL
  AND to_char(birthday, 'MM-DD') IN (
    SELECT to_char(CURRENT_DATE + (gs || ' days')::interval, 'MM-DD')
    FROM generate_series(0, 15) AS gs
  )
ORDER BY to_char(birthday, 'MM-DD');

-- name: ReportPaymentSummary :many
SELECT
  pm.id   AS payment_method_id,
  pm.name AS payment_method_name,
  COUNT(t.id)::bigint              AS tx_count,
  SUM(t.actual_paid_amount)::numeric AS total
FROM transactions t
JOIN payment_methods pm ON pm.id = t.payment_method_id
WHERE t.status = 'completed'
  AND t.transaction_time >= $1
  AND t.transaction_time <  $2
GROUP BY pm.id, pm.name
ORDER BY total DESC;

-- name: ReportCardSalesSummary :many
SELECT
  ct.id   AS card_type_id,
  ct.name AS card_type_name,
  COUNT(c.id)::bigint        AS card_count,
  SUM(c.final_price)::numeric AS total
FROM cards c
JOIN card_types ct ON ct.id = c.card_type_id
WHERE c.issued_at >= $1 AND c.issued_at < $2
GROUP BY ct.id, ct.name
ORDER BY total DESC;

-- name: ReportPendingStats :one
SELECT
  COUNT(*)::bigint              AS record_count,
  COUNT(DISTINCT member_id)::bigint AS member_count,
  COALESCE(SUM(amount), 0)::numeric AS total_amount
FROM member_credits
WHERE settled_at IS NULL;

-- name: ReportPendingByMember :many
SELECT
  m.id, m.name, m.phone,
  COUNT(mc.id)::bigint         AS record_count,
  SUM(mc.amount)::numeric       AS total_pending,
  MIN(mc.charged_at)            AS earliest,
  MAX(mc.charged_at)            AS latest
FROM member_credits mc
JOIN members m ON m.id = mc.member_id
WHERE mc.settled_at IS NULL
GROUP BY m.id, m.name, m.phone
ORDER BY total_pending DESC
LIMIT $1 OFFSET $2;
