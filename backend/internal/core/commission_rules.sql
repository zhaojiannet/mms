-- name: ListCommissionRules :many
-- 可按 staff_id 过滤
SELECT
  cr.id, cr.tenant_id, cr.staff_id, cr.service_id, cr.rate, cr.fixed_amount, cr.note,
  cr.created_at, cr.updated_at,
  s.name AS staff_name,
  sv.name AS service_name
FROM commission_rules cr
JOIN staff s ON s.id = cr.staff_id
LEFT JOIN services sv ON sv.id = cr.service_id
WHERE (sqlc.narg('staff_id')::uuid IS NULL OR cr.staff_id = sqlc.narg('staff_id')::uuid)
ORDER BY s.sort_order ASC, s.name ASC, sv.sort_order ASC NULLS FIRST;

-- name: GetCommissionRuleByID :one
SELECT * FROM commission_rules WHERE id = $1;

-- name: CreateCommissionRule :one
INSERT INTO commission_rules (
  tenant_id, staff_id, service_id, rate, fixed_amount, note
) VALUES (
  $1, $2,
  sqlc.narg('service_id')::uuid,
  sqlc.narg('rate')::numeric,
  sqlc.narg('fixed_amount')::numeric,
  sqlc.narg('note')::text
) RETURNING *;

-- name: UpdateCommissionRule :one
UPDATE commission_rules
SET rate         = COALESCE(sqlc.narg('rate')::numeric,         rate),
    fixed_amount = COALESCE(sqlc.narg('fixed_amount')::numeric, fixed_amount),
    note         = COALESCE(sqlc.narg('note')::text,            note),
    updated_at   = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCommissionRule :exec
DELETE FROM commission_rules WHERE id = $1;
