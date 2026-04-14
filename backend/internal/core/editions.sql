-- name: GetEditionByID :one
SELECT * FROM editions WHERE id = $1;

-- name: GetEditionByCode :one
SELECT * FROM editions WHERE code = $1;

-- name: ListActiveEditions :many
SELECT * FROM editions
WHERE active = TRUE
ORDER BY sort_order;
