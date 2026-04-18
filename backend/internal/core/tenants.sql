-- name: GetTenantBySlug :one
-- 供 middleware/tenant.go 解析 X-Tenant-Slug
SELECT * FROM tenants WHERE slug = $1;
