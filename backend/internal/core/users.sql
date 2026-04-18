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
    updated_at = now(),
    failed_login_attempts = 0,
    locked_until = NULL
WHERE id = $1;

-- name: RecordLoginFailure :one
UPDATE users
SET failed_login_attempts = failed_login_attempts + 1,
    locked_until = CASE
        WHEN failed_login_attempts + 1 >= sqlc.arg('max_attempts')::int
        THEN now() + (sqlc.arg('lock_minutes')::int || ' minutes')::interval
        ELSE locked_until
    END,
    updated_at = now()
WHERE id = $1
RETURNING failed_login_attempts, locked_until;

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
-- 改密同时递增 token_version + 更新 password_changed_at：旧 token 立即失效
UPDATE users
SET password_hash       = $2,
    token_version       = token_version + 1,
    password_changed_at = now(),
    updated_at          = now()
WHERE id = $1;

-- name: IncrementUserTokenVersion :exec
-- 角色/状态变更后递增版本，旧 token 失效（不改密码）
UPDATE users SET token_version = token_version + 1, updated_at = now() WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: LockActiveSuperAdmins :many
-- 锁定当前租户所有 active super_admin 行，防 TOCTOU 竞态
-- 调用方必须在事务中使用；RLS 自动按 current_tenant 过滤
-- 用于"最后一个超级管理员不可删/不可降级/不可停用"保护
SELECT id FROM users
WHERE role = 'super_admin' AND status = 'active'
ORDER BY id
FOR UPDATE;
