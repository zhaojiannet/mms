-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (
    tenant_id, email, phone, password_hash, name, role
) VALUES (
    $1, $2, $3, $4, $5, COALESCE(sqlc.narg('role')::text, 'staff')
)
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE users
SET last_login_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: ListUsersByTenant :many
SELECT * FROM users
WHERE status = 'active'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
