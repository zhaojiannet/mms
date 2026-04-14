-- name: GetMemberByID :one
SELECT * FROM members WHERE id = $1;

-- name: ListMembers :many
SELECT * FROM members
WHERE status = COALESCE(sqlc.narg('status')::text, status)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountMembers :one
SELECT count(*) FROM members
WHERE status = COALESCE(sqlc.narg('status')::text, status);

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
