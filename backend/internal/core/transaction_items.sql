-- name: CreateTransactionItem :one
INSERT INTO transaction_items (
  tenant_id, transaction_id, service_id, service_name_snapshot,
  price, quantity, commission_amount
) VALUES (
  $1, $2,
  sqlc.narg('service_id')::uuid,
  $3, $4, $5,
  COALESCE(sqlc.narg('commission_amount')::numeric, 0)
) RETURNING *;

-- name: ListTransactionItems :many
SELECT * FROM transaction_items
WHERE transaction_id = $1
ORDER BY created_at ASC;
