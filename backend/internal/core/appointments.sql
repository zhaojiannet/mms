-- name: ListAppointments :many
SELECT * FROM appointments
WHERE (sqlc.narg('start_date')::timestamptz IS NULL OR appointment_time >= sqlc.narg('start_date')::timestamptz)
  AND (sqlc.narg('end_date')::timestamptz   IS NULL OR appointment_time <  sqlc.narg('end_date')::timestamptz)
  AND (sqlc.narg('staff_id')::uuid          IS NULL OR assigned_staff_id = sqlc.narg('staff_id')::uuid)
  AND (sqlc.narg('status')::text            IS NULL OR status            = sqlc.narg('status')::text)
ORDER BY appointment_time ASC;

-- name: GetAppointmentByID :one
SELECT * FROM appointments WHERE id = $1;

-- name: CreateAppointment :one
INSERT INTO appointments (
  tenant_id, member_id, customer_name, customer_phone,
  appointment_time, assigned_staff_id, status, source, notes
) VALUES (
  $1,
  sqlc.narg('member_id')::uuid,
  $2, $3, $4,
  sqlc.narg('assigned_staff_id')::uuid,
  COALESCE(sqlc.narg('status')::text, 'pending'),
  COALESCE(sqlc.narg('source')::text, 'staff'),
  sqlc.narg('notes')::text
) RETURNING *;

-- name: UpdateAppointmentStatus :one
UPDATE appointments
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteAppointment :exec
DELETE FROM appointments WHERE id = $1;

-- name: CountAppointmentsInRange :one
-- 今日/范围内预约计数（仅 pending+confirmed）
SELECT count(*) FROM appointments
WHERE appointment_time >= $1 AND appointment_time < $2
  AND status IN ('pending', 'confirmed');

-- name: AddAppointmentService :exec
INSERT INTO appointment_services (tenant_id, appointment_id, service_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: ClearAppointmentServices :exec
DELETE FROM appointment_services WHERE appointment_id = $1;

-- name: ListAppointmentServices :many
SELECT s.id, s.name, s.price, s.duration_min
FROM appointment_services aps
JOIN services s ON s.id = aps.service_id
WHERE aps.appointment_id = $1
ORDER BY s.sort_order ASC, s.name ASC;

-- name: CountPhoneConflicts :one
-- 同手机号在时间窗内已有 pending/confirmed 预约
SELECT count(*) FROM appointments
WHERE customer_phone = $1
  AND appointment_time >= $2
  AND appointment_time <= $3
  AND status IN ('pending', 'confirmed');

-- name: CountStaffConflicts :one
SELECT count(*) FROM appointments
WHERE assigned_staff_id = $1
  AND appointment_time >= $2
  AND appointment_time <= $3
  AND status IN ('pending', 'confirmed');
