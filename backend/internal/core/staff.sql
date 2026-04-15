-- name: ListStaff :many
SELECT * FROM staff
WHERE status = COALESCE(sqlc.narg('status')::text, status)
ORDER BY sort_order ASC, name ASC;

-- name: GetStaffByID :one
SELECT * FROM staff WHERE id = $1;

-- name: CreateStaff :one
INSERT INTO staff (
  tenant_id, name, position, phone, hire_date,
  counts_commission, default_commission_rate, sort_order
)
VALUES (
  $1, $2, $3,
  sqlc.narg('phone')::text,
  sqlc.narg('hire_date')::date,
  COALESCE(sqlc.narg('counts_commission')::bool, TRUE),
  COALESCE(sqlc.narg('default_commission_rate')::numeric, 0.000),
  COALESCE(sqlc.narg('sort_order')::int, 99)
)
RETURNING *;

-- name: UpdateStaff :one
UPDATE staff
SET name                    = COALESCE(sqlc.narg('name')::text,      name),
    position                = COALESCE(sqlc.narg('position')::text,  position),
    phone                   = COALESCE(sqlc.narg('phone')::text,     phone),
    hire_date               = COALESCE(sqlc.narg('hire_date')::date, hire_date),
    status                  = COALESCE(sqlc.narg('status')::text,    status),
    counts_commission       = COALESCE(sqlc.narg('counts_commission')::bool, counts_commission),
    default_commission_rate = COALESCE(sqlc.narg('default_commission_rate')::numeric, default_commission_rate),
    sort_order              = COALESCE(sqlc.narg('sort_order')::int, sort_order),
    updated_at              = now()
WHERE id = $1
RETURNING *;

-- name: DeleteStaff :exec
DELETE FROM staff WHERE id = $1;

-- name: UnlinkStaffFromTransactions :exec
UPDATE transactions SET staff_id = NULL WHERE staff_id = $1;

-- name: UnlinkStaffFromAppointments :exec
UPDATE appointments SET assigned_staff_id = NULL WHERE assigned_staff_id = $1;
