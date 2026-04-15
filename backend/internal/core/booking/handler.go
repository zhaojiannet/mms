// Package booking 提供公开的预约 C 端接口（无需登录，但要 booking_code 校验）
//
// 对应老 demo routes/booking.js：
//   GET /api/booking/options?code=xxx      拿可选服务+员工
//   POST /api/booking?code=xxx             创建预约
//
// 所有路由挂在 api group（经过 TenantResolver + TxBegin，但不需要 RequireAuth）
package booking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// ------------- 速率限制（内存，per IP+phone 30s 一次） -------------

var (
	rateMu   sync.Mutex
	rateHits = map[string]time.Time{}
)

const rateLimitWindow = 30 * time.Second

func rateLimitCheck(key string) (allowed bool, retryAfter int) {
	rateMu.Lock()
	defer rateMu.Unlock()
	last, ok := rateHits[key]
	if ok && time.Since(last) < rateLimitWindow {
		return false, int((rateLimitWindow - time.Since(last)).Seconds()) + 1
	}
	rateHits[key] = time.Now()
	return true, 0
}

// ------------- 码校验 -------------

func verifyBookingCode(ctx context.Context, q *sqlc.Queries, tenantID uuid.UUID, code string) bool {
	if code == "" {
		return false
	}
	row, err := q.GetTenantSetting(ctx, sqlc.GetTenantSettingParams{TenantID: tenantID, Key: "booking_code"})
	if err != nil {
		return false
	}
	var stored *string
	if err := json.Unmarshal(row.Value, &stored); err != nil {
		return false
	}
	if stored == nil || *stored == "" {
		return false
	}
	return *stored == code
}

// ------------- 1. GET /api/booking/options?code=xxx -------------

type OptionsResponse struct {
	Services []ServiceOption `json:"services"`
	Staff    []StaffOption   `json:"staff"`
}

type ServiceOption struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Price       decimal.Decimal `json:"price"`
	DurationMin *int32          `json:"duration_min"`
}

type StaffOption struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func Options(c *echo.Context) error {
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	if !verifyBookingCode(ctx, q, t.ID, code) {
		return echo.NewHTTPError(http.StatusNotFound, "链接无效或已过期")
	}

	active := "active"
	services, err := q.ListServices(ctx, &active)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "services: "+err.Error())
	}
	staffActive, err := q.ListStaff(ctx, &active)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "staff: "+err.Error())
	}

	svcs := make([]ServiceOption, 0, len(services))
	for _, s := range services {
		svcs = append(svcs, ServiceOption{ID: s.ID, Name: s.Name, Price: s.Price, DurationMin: s.DurationMin})
	}
	stfs := make([]StaffOption, 0, len(staffActive))
	for _, s := range staffActive {
		stfs = append(stfs, StaffOption{ID: s.ID, Name: s.Name})
	}
	return c.JSON(http.StatusOK, OptionsResponse{Services: svcs, Staff: stfs})
}

// ------------- 2. POST /api/booking?code=xxx -------------

type CreateRequest struct {
	CustomerName    string      `json:"customer_name"`
	CustomerPhone   string      `json:"customer_phone"`
	AppointmentTime string      `json:"appointment_time"` // ISO RFC3339
	AssignedStaffID *uuid.UUID  `json:"assigned_staff_id"`
	ServiceIDs      []uuid.UUID `json:"service_ids"`
	Notes           *string     `json:"notes"`
}

var phoneRE = regexp.MustCompile(`^1[3-9]\d{9}$`)

func Create(c *echo.Context) error {
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()

	code := c.QueryParam("code")
	if !verifyBookingCode(ctx, q, t.ID, code) {
		return echo.NewHTTPError(http.StatusNotFound, "链接无效或已过期")
	}

	var req CreateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request: "+err.Error())
	}
	if req.CustomerPhone == "" || req.AppointmentTime == "" || len(req.ServiceIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "请填写完整信息")
	}
	if !phoneRE.MatchString(req.CustomerPhone) {
		return echo.NewHTTPError(http.StatusBadRequest, "手机号格式不正确")
	}

	// 速率限制（IP + 手机号）
	clientIP := c.Request().RemoteAddr
	if xff := c.Request().Header.Get("X-Forwarded-For"); xff != "" {
		clientIP = xff
	}
	key := "booking:" + clientIP + ":" + req.CustomerPhone
	if ok, retry := rateLimitCheck(key); !ok {
		c.Response().Header().Set("Retry-After", string(rune(retry)))
		return echo.NewHTTPError(http.StatusTooManyRequests, "请稍后再试")
	}

	apptTime, err := time.Parse(time.RFC3339, req.AppointmentTime)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid appointment_time: "+err.Error())
	}
	if !apptTime.After(time.Now()) {
		return echo.NewHTTPError(http.StatusBadRequest, "预约时间必须在当前时间之后")
	}

	// 冲突检测（±1h 内）
	windowStart := pgtype.Timestamptz{Time: apptTime.Add(-1 * time.Hour), Valid: true}
	windowEnd := pgtype.Timestamptz{Time: apptTime.Add(1 * time.Hour), Valid: true}

	phoneCount, err := q.CountPhoneConflicts(ctx, sqlc.CountPhoneConflictsParams{
		CustomerPhone:     req.CustomerPhone,
		AppointmentTime:   windowStart,
		AppointmentTime_2: windowEnd,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "phone conflict check: "+err.Error())
	}
	if phoneCount > 0 {
		return echo.NewHTTPError(http.StatusConflict, "您在该时段已有预约")
	}

	if req.AssignedStaffID != nil {
		staffCount, err := q.CountStaffConflicts(ctx, sqlc.CountStaffConflictsParams{
			AssignedStaffID:   pgtype.UUID{Bytes: *req.AssignedStaffID, Valid: true},
			AppointmentTime:   windowStart,
			AppointmentTime_2: windowEnd,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "staff conflict check: "+err.Error())
		}
		if staffCount > 0 {
			return echo.NewHTTPError(http.StatusConflict, "该员工在此时段已有预约")
		}
	}

	// 校验服务 id 都存在且 active
	for _, sid := range req.ServiceIDs {
		s, err := q.GetServiceByID(ctx, sid)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusBadRequest, "所选服务不存在")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "check service: "+err.Error())
		}
		if s.Status != "active" {
			return echo.NewHTTPError(http.StatusBadRequest, "所选服务已下架")
		}
	}

	customerName := req.CustomerName
	if customerName == "" {
		customerName = "用户"
	}

	sourceBooking := "booking"
	appt, err := q.CreateAppointment(ctx, sqlc.CreateAppointmentParams{
		TenantID:        t.ID,
		CustomerName:    customerName,
		CustomerPhone:   req.CustomerPhone,
		AppointmentTime: pgtype.Timestamptz{Time: apptTime, Valid: true},
		AssignedStaffID: optUUID(req.AssignedStaffID),
		Source:          &sourceBooking,
		Notes:           req.Notes,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create: "+err.Error())
	}
	for _, sid := range req.ServiceIDs {
		if err := q.AddAppointmentService(ctx, sqlc.AddAppointmentServiceParams{
			TenantID: t.ID, AppointmentID: appt.ID, ServiceID: sid,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "add service: "+err.Error())
		}
	}

	// 微信推送（阶段 3 notifications 模块接入后再实现，现在只留 hook）
	go notifyMerchant(appt)

	return c.JSON(http.StatusCreated, map[string]any{
		"success":        true,
		"message":        "预约成功，我们会尽快与您确认",
		"appointment_id": appt.ID,
	})
}

// notifyMerchant 留作将来的通知接入点（注意：这是 goroutine，事务已结束）
func notifyMerchant(_ sqlc.Appointment) {
	// TODO: 阶段 3 notifications 模块完成后接入
	// 当前产线 demo 用 wxpush，迁移期可以继续让老 demo 通知（独立部署）
}

func optUUID(p *uuid.UUID) pgtype.UUID {
	if p == nil || *p == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *p, Valid: true}
}
