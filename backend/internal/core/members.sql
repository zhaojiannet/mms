-- name: GetMemberByID :one
SELECT * FROM members WHERE id = $1;

-- name: GetMemberStats :one
-- 单查会员的聚合字段，口径与 ListMembers 一致（active 卡余额 + 未清挂账）
SELECT
  COALESCE((SELECT SUM(c.balance) FROM cards c WHERE c.member_id = sqlc.arg('member_id') AND c.status = 'active'), 0)::numeric AS total_balance,
  COALESCE((SELECT SUM(mc.amount) FROM member_credits mc WHERE mc.member_id = sqlc.arg('member_id') AND mc.settled_at IS NULL), 0)::numeric AS total_pending,
  (SELECT COUNT(*) FROM cards c WHERE c.member_id = sqlc.arg('member_id') AND c.status = 'active')::bigint AS card_count,
  (SELECT COUNT(*) FROM cards c WHERE c.member_id = sqlc.arg('member_id') AND c.status = 'active' AND c.balance > 0)::bigint AS active_card_count;

-- name: ListMembers :many
SELECT
  m.id, m.tenant_id, m.name, m.phone, m.gender, m.birthday, m.notes, m.status,
  m.created_at, m.updated_at, m.legacy_id,
  COALESCE(cb.total_balance, 0)::numeric AS total_balance,
  COALESCE(pc.total_pending, 0)::numeric AS total_pending,
  COALESCE(cb.card_count, 0)::bigint     AS card_count,
  COALESCE(cb.active_card_count, 0)::bigint AS active_card_count
FROM members m
LEFT JOIN (
  SELECT member_id,
         SUM(balance)::numeric AS total_balance,
         COUNT(*)::bigint AS card_count,
         COUNT(*) FILTER (WHERE balance > 0)::bigint AS active_card_count
  FROM cards WHERE status = 'active' GROUP BY member_id
) cb ON cb.member_id = m.id
LEFT JOIN (
  SELECT member_id, SUM(amount)::numeric AS total_pending
  FROM member_credits WHERE settled_at IS NULL GROUP BY member_id
) pc ON pc.member_id = m.id
WHERE (sqlc.narg('status')::text IS NULL OR m.status = sqlc.narg('status')::text)
  AND (sqlc.narg('search')::text IS NULL
       OR m.name  ILIKE '%' || sqlc.narg('search')::text || '%'
       OR m.phone LIKE  '%' || sqlc.narg('search')::text || '%')
ORDER BY m.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountMembers :one
SELECT count(*) FROM members
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
  AND (sqlc.narg('search')::text IS NULL
       OR name  ILIKE '%' || sqlc.narg('search')::text || '%'
       OR phone LIKE  '%' || sqlc.narg('search')::text || '%');

-- name: TenantHasPendingCredits :one
-- 是否存在任何未清挂账（租户级，RLS 限定）：前端会员表按此决定挂账列显隐，
-- 不能从已加载的分页切片推导——未加载页的欠款会被整列藏掉
SELECT EXISTS(SELECT 1 FROM member_credits WHERE settled_at IS NULL);

-- name: TenantHasMemberNotes :one
SELECT EXISTS(SELECT 1 FROM members WHERE notes IS NOT NULL AND notes <> '');

-- name: CreateMember :one
INSERT INTO members (tenant_id, name, phone, gender, birthday, notes)
VALUES ($1, $2, $3, COALESCE(sqlc.narg('gender')::text, 'unknown'), $4, $5)
RETURNING *;

-- name: UpdateMember :one
UPDATE members
SET name       = COALESCE(sqlc.narg('name')::text,    name),
    phone      = COALESCE(sqlc.narg('phone')::text,   phone),
    gender     = COALESCE(sqlc.narg('gender')::text,  gender),
    birthday   = COALESCE(sqlc.narg('birthday')::date, birthday),
    notes      = COALESCE(sqlc.narg('notes')::text,   notes),
    status     = COALESCE(sqlc.narg('status')::text,  status),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteMember :exec
DELETE FROM members WHERE id = $1;

-- name: LookupMembersByIDs :many
-- 按一组 ID 批量返回 id→name（用于操作日志对象列人名化）
SELECT id, name FROM members WHERE id = ANY($1::uuid[]);

-- name: ExistsMemberByPhone :one
-- 手机号唯一性检查（占位号 '00000000000' 允许重复，由 handler 跳过）
-- exclude_id：编辑时排除自己
SELECT EXISTS (
  SELECT 1 FROM members
  WHERE phone = sqlc.arg('phone')::text
    AND (sqlc.narg('exclude_id')::uuid IS NULL OR id <> sqlc.narg('exclude_id')::uuid)
) AS "exists";
