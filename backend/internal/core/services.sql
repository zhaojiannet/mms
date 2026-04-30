-- name: ListServices :many
SELECT * FROM services
WHERE status = COALESCE(sqlc.narg('status')::text, status)
ORDER BY sort_order ASC, name ASC;

-- name: CountServices :one
SELECT count(*) FROM services;

-- name: GetServiceByID :one
SELECT * FROM services WHERE id = $1;

-- name: GetServicesByIDs :many
-- 批量查多个服务（用于 transactions / booking / appointments 创建时一次性校验 + 取价）
-- 返回顺序与入参 ids 顺序无关，调用方自行 build map
SELECT * FROM services WHERE id = ANY(sqlc.arg('ids')::uuid[]);

-- name: CreateService :one
INSERT INTO services (
  tenant_id, name, price, category, description,
  no_discount, sort_order
)
VALUES (
  $1, $2, $3,
  sqlc.narg('category')::text,
  sqlc.narg('description')::text,
  COALESCE(sqlc.narg('no_discount')::bool, FALSE),
  COALESCE(sqlc.narg('sort_order')::int, 99)
)
RETURNING *;

-- name: UpdateService :one
UPDATE services
SET name         = COALESCE(sqlc.narg('name')::text,         name),
    price        = COALESCE(sqlc.narg('price')::numeric,     price),
    category     = COALESCE(sqlc.narg('category')::text,     category),
    description  = COALESCE(sqlc.narg('description')::text,  description),
    no_discount  = COALESCE(sqlc.narg('no_discount')::bool,  no_discount),
    sort_order   = COALESCE(sqlc.narg('sort_order')::int,    sort_order),
    status       = COALESCE(sqlc.narg('status')::text,       status),
    updated_at   = now()
WHERE id = $1
RETURNING *;

-- name: DeleteService :exec
DELETE FROM services WHERE id = $1;

-- name: CountServiceUsage :one
-- 删除前检查：是否被 transaction_items 或 appointment_services 引用
SELECT
  (SELECT count(*) FROM transaction_items  ti WHERE ti.service_id = $1)::bigint AS tx_item_count,
  (SELECT count(*) FROM appointment_services aps WHERE aps.service_id = $1)::bigint AS appt_count;
