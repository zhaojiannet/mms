-- name: GetTenantSetting :one
SELECT * FROM tenant_settings WHERE tenant_id = $1 AND key = $2;

-- name: GetTenantSettingsByKeys :many
-- 批量按 key 取多条 setting，避免连续 N 次 GetTenantSetting 往返
SELECT * FROM tenant_settings
WHERE tenant_id = $1 AND key = ANY(sqlc.arg('keys')::text[]);

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
