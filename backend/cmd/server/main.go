package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v5"
	"github.com/pressly/goose/v3"

	"github.com/zhaojiannet/mms/backend/internal/core/appointments"
	auditlogs "github.com/zhaojiannet/mms/backend/internal/core/audit_logs"
	"github.com/zhaojiannet/mms/backend/internal/core/auth"
	"github.com/zhaojiannet/mms/backend/internal/core/booking"
	cardtypes "github.com/zhaojiannet/mms/backend/internal/core/card_types"
	"github.com/zhaojiannet/mms/backend/internal/core/cards"
	commissionrules "github.com/zhaojiannet/mms/backend/internal/core/commission_rules"
	membercredits "github.com/zhaojiannet/mms/backend/internal/core/member_credits"
	"github.com/zhaojiannet/mms/backend/internal/core/members"
	paymentmethods "github.com/zhaojiannet/mms/backend/internal/core/payment_methods"
	"github.com/zhaojiannet/mms/backend/internal/core/reports"
	"github.com/zhaojiannet/mms/backend/internal/core/services"
	"github.com/zhaojiannet/mms/backend/internal/core/staff"
	tenantsettings "github.com/zhaojiannet/mms/backend/internal/core/tenant_settings"
	"github.com/zhaojiannet/mms/backend/internal/core/transactions"
	users2 "github.com/zhaojiannet/mms/backend/internal/core/users"
	"github.com/zhaojiannet/mms/backend/internal/platform/bootstrap"
	"github.com/zhaojiannet/mms/backend/internal/platform/db"
	"github.com/zhaojiannet/mms/backend/internal/platform/events"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := runMigrations(); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}

	pool, err := db.NewPool(context.Background())
	if err != nil {
		slog.Error("pgxpool init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 幂等 bootstrap：若 BOOTSTRAP_* 齐全且首次启动，则为目标 tenant 创建首个 admin
	if err := bootstrap.EnsureAdmin(context.Background(), pool); err != nil {
		slog.Error("bootstrap admin failed", "err", err)
		os.Exit(1)
	}

	e := echo.New()

	// 全局 CORS（dev 用，生产由反代统一处理）
	e.Use(mw.CORS())

	// 全局健康检查（不经过租户解析）
	e.GET("/health", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{
			"status":  "ok",
			"service": "mms",
			"mode":    getenv("DEPLOYMENT_MODE", "self-hosted"),
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	})

	// 订阅事件总线：简单日志记录（notifications 模块未来接入 wxpush 等 adapter）
	events.DefaultBus.Subscribe(events.TopicAppointmentCreated, func(p any) {
		slog.Info("event: appointment.created", "data", p)
	})
	events.DefaultBus.Subscribe(events.TopicTransactionVoided, func(p any) {
		slog.Info("event: transaction.voided", "data", p)
	})

	// 多租户 API：所有 /api/* 必须经过 tenant resolver + 事务中间件
	api := e.Group("/api")
	api.Use(mw.TenantResolver(pool))
	api.Use(mw.TenantTx(pool))
	api.Use(mw.AuditLog(pool))

	// 验证端点：返回当前请求解析到的租户
	api.GET("/me", func(c *echo.Context) error {
		t := mw.TenantFrom(c)
		return c.JSON(http.StatusOK, map[string]any{
			"tenant_id": t.ID,
			"slug":      t.Slug,
			"name":      t.Name,
			"status":    t.Status,
		})
	})

	// 登录端点：用 tenant 上下文 + RLS 验证 users 表
	api.POST("/login", auth.LoginHandler)

	// 公开预约端（含 booking_code 校验，不需要 JWT）
	api.GET("/booking/options", booking.Options)
	api.POST("/booking", booking.Create)

	// 需要 JWT 的业务路由组（JWT 中 tenant_id 必须与 Host 解析到的一致）
	secured := api.Group("")
	secured.Use(mw.RequireAuth())
	secured.GET("/members", members.List)
	secured.POST("/members", members.Create)
	secured.GET("/members/:id", members.Get)
	secured.PUT("/members/:id", members.Update)
	secured.DELETE("/members/:id", members.Delete)

	secured.GET("/payment-methods", paymentmethods.List)
	secured.POST("/payment-methods", paymentmethods.Create)
	secured.GET("/payment-methods/:id", paymentmethods.Get)
	secured.PUT("/payment-methods/:id", paymentmethods.Update)
	secured.DELETE("/payment-methods/:id", paymentmethods.Delete)

	secured.GET("/services", services.List)
	secured.POST("/services", services.Create)
	secured.GET("/services/:id", services.Get)
	secured.PUT("/services/:id", services.Update)
	secured.DELETE("/services/:id", services.Delete)

	secured.GET("/staff", staff.List)
	secured.POST("/staff", staff.Create)
	secured.GET("/staff/:id", staff.Get)
	secured.PUT("/staff/:id", staff.Update)
	secured.DELETE("/staff/:id", staff.Delete)

	secured.GET("/card-types", cardtypes.List)
	secured.POST("/card-types", cardtypes.Create)
	secured.GET("/card-types/:id", cardtypes.Get)
	secured.PUT("/card-types/:id", cardtypes.Update)
	secured.DELETE("/card-types/:id", cardtypes.Delete)

	secured.GET("/members/:memberId/cards", cards.ListByMember)
	secured.POST("/members/:memberId/cards", cards.Issue)
	secured.GET("/cards/:id", cards.Get)
	secured.PUT("/cards/:id", cards.Update)
	secured.DELETE("/cards/:id", cards.Delete)

	secured.GET("/members/:memberId/pending", membercredits.ListPending)
	secured.POST("/members/:memberId/pending", membercredits.Create)
	secured.DELETE("/members/:memberId/pending/:pendingId", membercredits.Delete)
	secured.POST("/members/:memberId/pending/:pendingId/settle", transactions.SettleCredit)

	secured.GET("/transactions", transactions.List)
	secured.POST("/transactions", transactions.Create)
	secured.GET("/transactions/:id", transactions.Get)
	secured.POST("/transactions/:id/void", transactions.Void)
	secured.POST("/members/:memberId/cards/with-transaction", transactions.IssueCard)
	secured.POST("/members/:memberId/pending/settle-all", transactions.SettleAllCredits)

	secured.GET("/appointments", appointments.List)
	secured.POST("/appointments", appointments.Create)
	secured.GET("/appointments/count/today", appointments.CountToday)
	secured.GET("/appointments/:id", appointments.Get)
	secured.PATCH("/appointments/:id/status", appointments.UpdateStatus)
	secured.DELETE("/appointments/:id", appointments.Delete)

	secured.GET("/tenant-settings", tenantsettings.List)
	secured.GET("/tenant-settings/:key", tenantsettings.Get)
	secured.PUT("/tenant-settings/:key", tenantsettings.Upsert)
	secured.DELETE("/tenant-settings/:key", tenantsettings.Delete)
	secured.POST("/tenant-settings/enable-void", tenantsettings.EnableVoid)
	secured.POST("/tenant-settings/disable-void", tenantsettings.DisableVoid)
	secured.POST("/tenant-settings/booking-code", tenantsettings.RegenerateBookingCode)

	secured.GET("/users",                  users2.List)
	secured.POST("/users",                 users2.Create)
	secured.GET("/users/:id",              users2.Get)
	secured.PUT("/users/:id",              users2.Update)
	secured.POST("/users/:id/reset-password", users2.ResetPassword)
	secured.DELETE("/users/:id",           users2.Delete)

	secured.GET("/audit-logs",              auditlogs.List)

	secured.GET("/commission-rules",        commissionrules.List)
	secured.POST("/commission-rules",       commissionrules.Create)
	secured.PUT("/commission-rules/:id",    commissionrules.Update)
	secured.DELETE("/commission-rules/:id", commissionrules.Delete)

	secured.GET("/reports/business",           reports.Business)
	secured.GET("/reports/service-ranking",    reports.ServiceRanking)
	secured.GET("/reports/sleeping-members",   reports.SleepingMembers)
	secured.GET("/reports/member-ranking",     reports.MemberRanking)
	secured.GET("/reports/birthday-reminders", reports.BirthdayReminders)
	secured.GET("/reports/payment-summary",    reports.PaymentSummary)
	secured.GET("/reports/card-sales-summary", reports.CardSalesSummary)
	secured.GET("/reports/pending-stats",      reports.PendingStats)

	addr := ":" + getenv("APP_PORT", "8080")
	slog.Info("mms server starting", "addr", addr)
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// runMigrations 使用应用用户 mms 跑 goose 迁移
// mms 是 mms 数据库的 owner，且 pgcrypto/citext/pg_trgm 均为 trusted extension，
// 无需 superuser 即可完成 CREATE EXTENSION / CREATE TABLE / RLS 策略创建
func runMigrations() error {
	dsn := buildDSN(
		getenv("DB_USER", "mms"),
		os.Getenv("DB_PASSWORD"),
		getenv("DB_HOST", "postgres-server"),
		getenv("DB_PORT", "5432"),
		getenv("DB_NAME", "mms"),
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration db: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	slog.Info("migrations applied")
	return nil
}

func buildDSN(user, pass, host, port, name string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, url.QueryEscape(pass), host, port, name)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
