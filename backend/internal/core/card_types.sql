-- name: ListCardTypes :many
SELECT * FROM card_types
WHERE status = COALESCE(sqlc.narg('status')::text, status)
ORDER BY sort_order ASC, price ASC;

-- name: GetCardTypeByID :one
SELECT * FROM card_types WHERE id = $1;

-- name: CreateCardType :one
INSERT INTO card_types (tenant_id, name, price, discount_rate, sort_order)
VALUES ($1, $2, $3, $4, COALESCE(sqlc.narg('sort_order')::int, 99))
RETURNING *;

-- name: UpdateCardType :one
UPDATE card_types
SET name          = COALESCE(sqlc.narg('name')::text,          name),
    price         = COALESCE(sqlc.narg('price')::numeric,      price),
    discount_rate = COALESCE(sqlc.narg('discount_rate')::numeric, discount_rate),
    sort_order    = COALESCE(sqlc.narg('sort_order')::int,     sort_order),
    status        = COALESCE(sqlc.narg('status')::text,        status),
    updated_at    = now()
WHERE id = $1
RETURNING *;

-- name: DeleteCardType :exec
DELETE FROM card_types WHERE id = $1;

-- name: CountCardTypeUsage :one
SELECT count(*) FROM cards WHERE card_type_id = $1;
