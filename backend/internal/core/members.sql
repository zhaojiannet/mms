-- name: GetMemberByID :one
SELECT * FROM members WHERE id = $1;

-- name: ListMembers :many
SELECT
  m.id, m.tenant_id, m.name, m.phone, m.gender, m.birthday, m.notes, m.status,
  m.created_at, m.updated_at, m.legacy_id,
  COALESCE(cb.total_balance, 0)::numeric AS total_balance,
  COALESCE(pc.total_pending, 0)::numeric AS total_pending,
  COALESCE(cb.card_count, 0)::bigint     AS card_count
FROM members m
LEFT JOIN (
  SELECT member_id, SUM(balance)::numeric AS total_balance, COUNT(*)::bigint AS card_count
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
