// Package booking 提供公开的预约 C 端接口（无需登录，但要 booking_code 校验）
//
// 对应老 demo routes/booking.js：
//
//	GET /api/booking/options?code=xxx      拿可选服务+员工
//	POST /api/booking?code=xxx             创建预约
//
// 所有路由挂在 api group（经过 TenantResolver + TxBegin，但不需要 RequireAuth）
package booking

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"

	"github.com/zhaojiannet/mms/backend/internal/platform/events"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/internal/platform/util/pgtypex"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// ------------- 速率限制（内存，per IP+phone 30s 一次） -------------
//
// 与外层 mw.RateLimit(30/min/IP) 互补：
//   外层挡"同 IP 刷所有手机号"，本层挡"同 IP 针对同一手机号连续提交"
//
// 加了 GC loop + 硬上限防内存无限增长

var (
	rateMu   sync.Mutex
	rateHits = map[string]time.Time{}
)

const (
	rateLimitWindow   = 30 * time.Second
	rateMapMaxSize    = 10_000 // 满则拒绝新 key（攻击者已经很难穿透外层 RateLimit 了）
	rateMapGCInterval = 5 * time.Minute
)

func init() {
	go func() {
		t := time.NewTicker(rateMapGCInterval)
		defer t.Stop()
		for range t.C {
			rateMu.Lock()
			now := time.Now()
			for k, v := range rateHits {
				if now.Sub(v) > rateLimitWindow*2 {
					delete(rateHits, k)
				}
			}
			rateMu.Unlock()
		}
	}()
}

func rateLimitCheck(key string) (allowed bool, retryAfter int) {
	rateMu.Lock()
	defer rateMu.Unlock()
	last, ok := rateHits[key]
	if ok && time.Since(last) < rateLimitWindow {
		return false, int((rateLimitWindow - time.Since(last)).Seconds()) + 1
	}
	// 容量保护：满了就拒绝，等下轮 GC
	if len(rateHits) >= rateMapMaxSize && !ok {
		return false, int(rateLimitWindow.Seconds())
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
	// constant-time 比较防时序泄露：subtle.ConstantTimeCompare 对等长串做 O(n)
	// 不等长直接返 false（长度泄露低风险，code 本身是固定 8 位）
	a := []byte(*stored)
	b := []byte(code)
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

// ------------- 1. GET /api/booking/options?code=xxx -------------

type OptionsResponse struct {
	Services []ServiceOption `json:"services"`
	Staff    []StaffOption   `json:"staff"`
}

type ServiceOption struct {
	ID    uuid.UUID       `json:"id"`
	Name  string          `json:"name"`
	Price decimal.Decimal `json:"price"`
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
		return mw.InternalError(c, "booking.options.services", err)
	}
	staffActive, err := q.ListStaff(ctx, &active)
	if err != nil {
		return mw.InternalError(c, "booking.options.staff", err)
	}

	svcs := make([]ServiceOption, 0, len(services))
	for _, s := range services {
		svcs = append(svcs, ServiceOption{ID: s.ID, Name: s.Name, Price: s.Price})
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
	if len(req.ServiceIDs) > 20 {
		return echo.NewHTTPError(http.StatusBadRequest, "最多选择 20 项服务")
	}
	if !phoneRE.MatchString(req.CustomerPhone) {
		return echo.NewHTTPError(http.StatusBadRequest, "手机号格式不正确")
	}

	// 速率限制（IP + 手机号），IP 复用 middleware.ClientIP 的 TRUSTED_PROXIES 信任链
	// 避免无条件信任 X-Forwarded-For 被伪造绕过
	key := "booking:" + mw.ClientIP(c.Request()) + ":" + req.CustomerPhone
	if ok, retry := rateLimitCheck(key); !ok {
		c.Response().Header().Set("Retry-After", strconv.Itoa(retry))
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
		return mw.InternalError(c, "booking.phone-conflict", err)
	}
	if phoneCount > 0 {
		return echo.NewHTTPError(http.StatusConflict, "您在该时段已有预约")
	}

	if req.AssignedStaffID != nil {
		// 归属校验走 RLS 范围内的 SELECT：FK 不受 RLS 约束，
		// 跨租户 staff UUID 能过 FK，这里必须显式查一次拦下
		if _, err := q.GetStaffByID(ctx, *req.AssignedStaffID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return echo.NewHTTPError(http.StatusBadRequest, "所选员工不存在")
			}
			return mw.InternalError(c, "booking.check-staff", err)
		}
		staffCount, err := q.CountStaffConflicts(ctx, sqlc.CountStaffConflictsParams{
			AssignedStaffID:   pgtype.UUID{Bytes: *req.AssignedStaffID, Valid: true},
			AppointmentTime:   windowStart,
			AppointmentTime_2: windowEnd,
		})
		if err != nil {
			return mw.InternalError(c, "booking.staff-conflict", err)
		}
		if staffCount > 0 {
			return echo.NewHTTPError(http.StatusConflict, "该员工在此时段已有预约")
		}
	}

	// 校验服务 id 都存在且 active（批量取，避免 N+1）
	svcRows, err := q.GetServicesByIDs(ctx, req.ServiceIDs)
	if err != nil {
		return mw.InternalError(c, "booking.check-service", err)
	}
	svcByID := make(map[uuid.UUID]sqlc.Service, len(svcRows))
	for _, s := range svcRows {
		svcByID[s.ID] = s
	}
	for _, sid := range req.ServiceIDs {
		s, ok := svcByID[sid]
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "所选服务不存在")
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
		AssignedStaffID: pgtypex.OptUUID(req.AssignedStaffID),
		Source:          &sourceBooking,
		Notes:           req.Notes,
	})
	if err != nil {
		return mw.InternalError(c, "booking.create", err)
	}
	if err := q.AddAppointmentServicesBulk(ctx, sqlc.AddAppointmentServicesBulkParams{
		TenantID: t.ID, AppointmentID: appt.ID, ServiceIds: req.ServiceIDs,
	}); err != nil {
		return mw.InternalError(c, "booking.add-services", err)
	}

	events.DefaultBus.Publish(events.TopicAppointmentCreated, map[string]any{
		"appointment_id": appt.ID, "tenant_id": t.ID, "source": "booking",
	})

	return c.JSON(http.StatusCreated, map[string]any{
		"success":        true,
		"message":        "预约成功，我们会尽快与您确认",
		"appointment_id": appt.ID,
	})
}
