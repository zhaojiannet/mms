-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
  tenant_id, actor_id, actor_type, action, target_type, target_id, payload, ip_address, user_agent
) VALUES (
  $1, $2, $3, $4,
  sqlc.narg('target_type')::text,
  sqlc.narg('target_id')::text,
  $5,
  sqlc.narg('ip_address')::inet,
  sqlc.narg('user_agent')::text
);

-- name: ListAuditLogs :many
SELECT
  a.*,
  u.name AS actor_name
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_id
WHERE (sqlc.narg('tenant_id')::uuid IS NULL OR a.tenant_id = sqlc.narg('tenant_id')::uuid)
ORDER BY a.created_at DESC
LIMIT $1 OFFSET $2;
