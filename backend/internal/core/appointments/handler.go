package appointments

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/events"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/pgtypex"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/timex"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type DTO struct {
	ID              uuid.UUID  `json:"id"`
	MemberID        *uuid.UUID `json:"member_id"`
	CustomerName    string     `json:"customer_name"`
	CustomerPhone   string     `json:"customer_phone"`
	AppointmentTime time.Time  `json:"appointment_time"`
	AssignedStaffID *uuid.UUID `json:"assigned_staff_id"`
	Status          string     `json:"status"`
	TransactionID   *uuid.UUID `json:"transaction_id"`
	Source          string     `json:"source"`
	Notes           *string    `json:"notes"`
	ServiceIDs      []uuid.UUID `json:"service_ids,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CreateRequest struct {
	MemberID        *uuid.UUID  `json:"member_id"`
	CustomerName    string      `json:"customer_name"`
	CustomerPhone   string      `json:"customer_phone"`
	AppointmentTime string      `json:"appointment_time"` // ISO RFC3339
	AssignedStaffID *uuid.UUID  `json:"assigned_staff_id"`
	ServiceIDs      []uuid.UUID `json:"service_ids"`
	Notes           *string     `json:"notes"`
	Status          *string     `json:"status"` // 默认 pending
}

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// List GET /api/appointments?start=&end=&staff_id=&status=
func List(c *echo.Context) error {
	q := sqlc.New(mw.TxFrom(c))
	start, err := timex.ParseRFC3339TzStr(c.QueryParam("start_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start_date")
	}
	end, err := timex.ParseRFC3339TzStr(c.QueryParam("end_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid end_date")
	}
	var staffID pgtype.UUID
	if s := c.QueryParam("staff_id"); s != "" {
		u, err := uuid.Parse(s)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid staff_id")
		}
		staffID = pgtype.UUID{Bytes: u, Valid: true}
	}
	var status *string
	if s := c.QueryParam("status"); s != "" {
		status = &s
	}

	rows, err := q.ListAppointments(c.Request().Context(), sqlc.ListAppointmentsParams{
		StartDate: start, EndDate: end, StaffID: staffID, Status: status,
	})
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}
	items := make([]DTO, 0, len(rows))
	for _, r := range rows {
		items = append(items, toDTO(r, nil))
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

// Get GET /api/appointments/:id （含关联服务）
func Get(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))

	appt, err := q.GetAppointmentByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "appointment not found")
		}
		return mw.InternalError(c, "get: ", err)
	}
	svcs, err := q.ListAppointmentServices(ctx, id)
	if err != nil {
		return mw.InternalError(c, "list services: ", err)
	}
	svcIDs := make([]uuid.UUID, 0, len(svcs))
	for _, s := range svcs {
		svcIDs = append(svcIDs, s.ID)
	}
	return c.JSON(http.StatusOK, toDTO(appt, svcIDs))
}

// Create POST /api/appointments
func Create(c *echo.Context) error {
	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.CustomerName == "" || req.CustomerPhone == "" || req.AppointmentTime == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "customer_name / customer_phone / appointment_time required")
	}
	if len(req.ServiceIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "service_ids required")
	}
	if len(req.ServiceIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "最多选择 20 项服务")
	}
	apptTime, err := time.Parse(time.RFC3339, req.AppointmentTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid appointment_time: "+err.Error())
	}
	// 创建只允许排程态：completed/cancelled 属状态变更，走 adminAndAbove 的
	// PATCH /appointments/:id/status，否则 staff 可绕过该守卫直接建"已完成"预约
	if req.Status != nil && *req.Status != "" && *req.Status != "pending" && *req.Status != "confirmed" {
		return echo.NewHTTPError(http.StatusBadRequest, "status 只能是 pending 或 confirmed")
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	// 归属校验走 RLS 范围内的 SELECT：FK 检查不受 RLS 约束，
	// 跨租户 UUID 能通过 FK 但查不到行 → 这里报 400
	if req.MemberID != nil {
		if _, err := q.GetMemberByID(ctx, *req.MemberID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusBadRequest, "member not found")
			}
			return mw.InternalError(c, "check member: ", err)
		}
	}
	if req.AssignedStaffID != nil {
		if _, err := q.GetStaffByID(ctx, *req.AssignedStaffID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusBadRequest, "staff not found")
			}
			return mw.InternalError(c, "check staff: ", err)
		}
	}
	svcRows, err := q.GetServicesByIDs(ctx, req.ServiceIDs)
	if err != nil {
		return mw.InternalError(c, "check services: ", err)
	}
	if len(svcRows) != len(uniqueIDs(req.ServiceIDs)) {
		return echo.NewHTTPError(http.StatusBadRequest, "所选服务不存在")
	}

	appt, err := q.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		TenantID:        t.ID,
		CustomerName:    req.CustomerName,
		CustomerPhone:   req.CustomerPhone,
		AppointmentTime: pgtype.Timestamptz{Time: apptTime, Valid: true},
		MemberID:        pgtypex.OptUUID(req.MemberID),
		AssignedStaffID: pgtypex.OptUUID(req.AssignedStaffID),
		Status:          req.Status,
		Source:          nil, // 默认 'staff'
		Notes:           req.Notes,
	})
	if err != nil {
		return mw.InternalError(c, "create: ", err)
	}

	if err := q.AddAppointmentServicesBulk(ctx, sqlc.AddAppointmentServicesBulkParams{
		TenantID:      t.ID,
		AppointmentID: appt.ID,
		ServiceIds:    req.ServiceIDs,
	}); err != nil {
		return mw.InternalError(c, "add services: ", err)
	}

	events.DefaultBus.Publish(events.TopicAppointmentCreated, map[string]any{
		"appointment_id": appt.ID, "tenant_id": t.ID, "source": "staff",
	})

	return c.JSON(http.StatusCreated, toDTO(appt, req.ServiceIDs))
}

// uniqueIDs 去重（入参允许重复 service_id，与 AddAppointmentServicesBulk 的 ON CONFLICT 行为一致）
func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// UpdateStatus PATCH /api/appointments/:id/status
func UpdateStatus(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var req UpdateStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	switch req.Status {
	case "pending", "confirmed", "completed", "cancelled", "no_show":
	case "":
		return echo.NewHTTPError(http.StatusBadRequest, "status required")
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid status: "+req.Status)
	}

	q := sqlc.New(mw.TxFrom(c))
	appt, err := q.UpdateAppointmentStatus(c.Request().Context(), sqlc.UpdateAppointmentStatusParams{
		ID: id, Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "appointment not found")
		}
		return mw.InternalError(c, "update status: ", err)
	}
	return c.JSON(http.StatusOK, toDTO(appt, nil))
}

// Delete DELETE /api/appointments/:id
func Delete(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	ctx := c.Request().Context()
	q := sqlc.New(mw.TxFrom(c))
	if err := q.ClearAppointmentServices(ctx, id); err != nil {
		return mw.InternalError(c, "clear services: ", err)
	}
	if err := q.DeleteAppointment(ctx, id); err != nil {
		return mw.InternalError(c, "delete: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// CountToday GET /api/appointments/count/today
func CountToday(c *echo.Context) error {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.Add(24 * time.Hour)
	q := sqlc.New(mw.TxFrom(c))
	n, err := q.CountAppointmentsInRange(c.Request().Context(), sqlc.CountAppointmentsInRangeParams{
		AppointmentTime:   pgtype.Timestamptz{Time: start, Valid: true},
		AppointmentTime_2: pgtype.Timestamptz{Time: end, Valid: true},
	})
	if err != nil {
		return mw.InternalError(c, "count: ", err)
	}
	return c.JSON(http.StatusOK, map[string]int64{"count": n})
}

// --- 工具 ---

func toDTO(a sqlc.Appointment, svcIDs []uuid.UUID) DTO {
	dto := DTO{
		ID:              a.ID,
		CustomerName:    a.CustomerName,
		CustomerPhone:   a.CustomerPhone,
		AppointmentTime: a.AppointmentTime.Time,
		Status:          a.Status,
		Source:          a.Source,
		Notes:           a.Notes,
		CreatedAt:       a.CreatedAt.Time,
		ServiceIDs:      svcIDs,
	}
	if a.MemberID.Valid {
		v := uuid.UUID(a.MemberID.Bytes)
		dto.MemberID = &v
	}
	if a.AssignedStaffID.Valid {
		v := uuid.UUID(a.AssignedStaffID.Bytes)
		dto.AssignedStaffID = &v
	}
	if a.TransactionID.Valid {
		v := uuid.UUID(a.TransactionID.Bytes)
		dto.TransactionID = &v
	}
	return dto
}

