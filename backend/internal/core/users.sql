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
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status')::text)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET name   = COALESCE(sqlc.narg('name')::text,   name),
    phone  = COALESCE(sqlc.narg('phone')::text,  phone),
    role   = COALESCE(sqlc.narg('role')::text,   role),
    status = COALESCE(sqlc.narg('status')::text, status),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
