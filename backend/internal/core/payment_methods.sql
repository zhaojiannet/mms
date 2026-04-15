-- name: ListPaymentMethods :many
SELECT * FROM payment_methods
WHERE is_active = COALESCE(sqlc.narg('is_active')::bool, is_active)
ORDER BY sort_order ASC, name ASC;

-- name: GetPaymentMethodByID :one
SELECT * FROM payment_methods WHERE id = $1;

-- name: CreatePaymentMethod :one
INSERT INTO payment_methods (tenant_id, name, sort_order, is_active)
VALUES ($1, $2, COALESCE(sqlc.narg('sort_order')::int, 99), TRUE)
RETURNING *;

-- name: UpdatePaymentMethod :one
UPDATE payment_methods
SET name       = COALESCE(sqlc.narg('name')::text,       name),
    sort_order = COALESCE(sqlc.narg('sort_order')::int,  sort_order),
    is_active  = COALESCE(sqlc.narg('is_active')::bool,  is_active),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeletePaymentMethod :exec
DELETE FROM payment_methods WHERE id = $1;

-- name: CountPaymentMethodUsage :one
-- 删除前检查：这个支付方式是否被 transactions 引用
SELECT count(*) FROM transactions WHERE payment_method_id = $1;
