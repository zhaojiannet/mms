package appointments

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
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
	start, err := parseTS(c.QueryParam("start_date"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid start_date")
	}
	end, err := parseTS(c.QueryParam("end_date"))
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
		return echo.NewHTTPError(http.StatusInternalServerError, "list: "+err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, "get: "+err.Error())
	}
	svcs, err := q.ListAppointmentServices(ctx, id)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "list services: "+err.Error())
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
	apptTime, err := time.Parse(time.RFC3339, req.AppointmentTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid appointment_time: "+err.Error())
	}

	ctx := c.Request().Context()
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	appt, err := q.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		TenantID:        t.ID,
		CustomerName:    req.CustomerName,
		CustomerPhone:   req.CustomerPhone,
		AppointmentTime: pgtype.Timestamptz{Time: apptTime, Valid: true},
		MemberID:        optUUID(req.MemberID),
		AssignedStaffID: optUUID(req.AssignedStaffID),
		Status:          req.Status,
		Source:          nil, // 默认 'staff'
		Notes:           req.Notes,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create: "+err.Error())
	}

	for _, sid := range req.ServiceIDs {
		if err := q.AddAppointmentService(ctx, sqlc.AddAppointmentServiceParams{
			TenantID:      t.ID,
			AppointmentID: appt.ID,
			ServiceID:     sid,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "add service: "+err.Error())
		}
	}

	return c.JSON(http.StatusCreated, toDTO(appt, req.ServiceIDs))
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
	if req.Status == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "status required")
	}

	q := sqlc.New(mw.TxFrom(c))
	appt, err := q.UpdateAppointmentStatus(c.Request().Context(), sqlc.UpdateAppointmentStatusParams{
		ID: id, Status: req.Status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, "appointment not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "update status: "+err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, "clear services: "+err.Error())
	}
	if err := q.DeleteAppointment(ctx, id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete: "+err.Error())
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
		return echo.NewHTTPError(http.StatusInternalServerError, "count: "+err.Error())
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

func optUUID(p *uuid.UUID) pgtype.UUID {
	if p == nil || *p == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *p, Valid: true}
}

func parseTS(s string) (pgtype.Timestamptz, error) {
	if s == "" {
		return pgtype.Timestamptz{Valid: false}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}
