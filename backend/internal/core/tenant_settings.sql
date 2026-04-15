-- name: GetTenantSetting :one
SELECT * FROM tenant_settings WHERE tenant_id = $1 AND key = $2;

-- name: ListTenantSettings :many
SELECT * FROM tenant_settings WHERE tenant_id = $1 ORDER BY key;

-- name: UpsertTenantSetting :one
INSERT INTO tenant_settings (tenant_id, key, value, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (tenant_id, key) DO UPDATE
SET value = EXCLUDED.value, updated_at = now()
RETURNING *;

-- name: DeleteTenantSetting :exec
DELETE FROM tenant_settings WHERE tenant_id = $1 AND key = $2;
