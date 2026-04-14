-- name: GetTenantByID :one
SELECT * FROM tenants WHERE id = $1;

-- name: GetTenantBySlug :one
SELECT * FROM tenants WHERE slug = $1;

-- name: CreateTenant :one
INSERT INTO tenants (
    slug, name, status
) VALUES (
    $1, $2, COALESCE(sqlc.narg('status')::text, 'active')
)
RETURNING *;

-- name: UpdateTenantStatus :one
UPDATE tenants
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;
