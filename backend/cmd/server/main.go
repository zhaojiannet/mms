package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v5"
	echomw "github.com/labstack/echo/v5/middleware"
	"github.com/pressly/goose/v3"

	"github.com/zhaojiannet/mms/backend/internal/core/appointments"
	auditlogs "github.com/zhaojiannet/mms/backend/internal/core/audit_logs"
	"github.com/zhaojiannet/mms/backend/internal/core/auth"
	"github.com/zhaojiannet/mms/backend/internal/core/booking"
	cardtypes "github.com/zhaojiannet/mms/backend/internal/core/card_types"
	"github.com/zhaojiannet/mms/backend/internal/core/cards"
	membercredits "github.com/zhaojiannet/mms/backend/internal/core/member_credits"
	"github.com/zhaojiannet/mms/backend/internal/core/members"
	"github.com/zhaojiannet/mms/backend/internal/core/notifications"
	paymentmethods "github.com/zhaojiannet/mms/backend/internal/core/payment_methods"
	platformcore "github.com/zhaojiannet/mms/backend/internal/core/platform"
	"github.com/zhaojiannet/mms/backend/internal/core/reports"
	"github.com/zhaojiannet/mms/backend/internal/core/services"
	"github.com/zhaojiannet/mms/backend/internal/core/staff"
	tenantsettings "github.com/zhaojiannet/mms/backend/internal/core/tenant_settings"
	"github.com/zhaojiannet/mms/backend/internal/core/transactions"
	"github.com/zhaojiannet/mms/backend/internal/core/uploads"
	users2 "github.com/zhaojiannet/mms/backend/internal/core/users"
	"github.com/zhaojiannet/mms/backend/internal/platform/bootstrap"
	dbpkg "github.com/zhaojiannet/mms/backend/internal/platform/db"
	"github.com/zhaojiannet/mms/backend/internal/platform/events"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 启动硬校验：关键环境变量必须是非默认值，否则拒绝启动
	if err := validateCriticalEnv(); err != nil {
		slog.Error("env validation failed", "err", err)
		os.Exit(1)
	}

	if err := runMigrations(); err != nil {
		slog.Error("migration failed", "err", err)
		os.Exit(1)
	}

	pool, err := dbpkg.NewPool(context.Background())
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

	// 幂等 bootstrap：平台操作员（PLATFORM_ADMIN_* 齐全时创建，运营后台登录用）
	if err := bootstrap.EnsureOperator(context.Background(), pool); err != nil {
		slog.Error("bootstrap operator failed", "err", err)
		os.Exit(1)
	}

	// 系统公告 seed：从 announcements.json upsert 入库；失败只 warn 不中断
	notifications.SeedAnnouncements(context.Background(), pool)

	// 注入 pool 到 auth 包：登录失败计数的独立 tx 会用（绕过业务 handler tx 的 rollback）
	if err := auth.Init(pool); err != nil {
		slog.Error("auth init failed", "err", err)
		os.Exit(1)
	}

	// 注入 pool 到 platform 包：平台登录与公开申请不经过事务中间件
	if err := platformcore.Init(pool); err != nil {
		slog.Error("platform init failed", "err", err)
		os.Exit(1)
	}

	e := echo.New()

	// 全局 body 上限 2MB：防存储型 DoS（notes / name 塞巨大字符串）
	// logo 上传走独立 multipart 路径，handler 内自校验 2MB 限制
	e.Use(echomw.BodyLimit(2 << 20))

	// 全局 CORS（dev 用，生产由反代统一处理）
	e.Use(mw.CORS())

	// 全局安全响应头（defense-in-depth，反代可叠加/覆盖）
	hostedMode := getenv("DEPLOYMENT_MODE", "self-hosted") == "hosted"
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			// HSTS：仅 hosted 模式启用（自建者可能走 HTTP 或自签证书期，HSTS 会锁死浏览器）
			if hostedMode {
				h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			}
			return next(c)
		}
	})

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

	// 审计 worker pool：固定 4 worker 消费写队列；满时 block（不丢审计）
	mw.StartAuditWorkers(pool)

	// --------------- 平台层（运营后台 + 公开申请）：不经过租户中间件 ---------------
	// 公开验证码：主站申请表单与运营后台登录用（无租户上下文，与商户 /api/captcha 共享 store）
	publicCaptchaLimit := mw.RateLimit(20, time.Minute)
	e.GET("/api/public/captcha", auth.CaptchaHandler, publicCaptchaLimit)

	// 商户开通申请（公开表单提交，验证码必填）
	signupLimit := mw.RateLimit(5, time.Minute)
	e.POST("/api/signup/applications", platformcore.SubmitApplication, signupLimit)

	// 平台操作员登录（独立身份体系，issuer=mms-platform；仅 admin 子域可达）
	platformLoginLimit := mw.RateLimit(5, time.Minute)
	e.POST("/api/platform/login", platformcore.Login, mw.RequirePlatformHost(), platformLoginLimit)

	// 平台受保护路由：Host 限制 + 平台事务（app.platform_op 哨兵）+ 操作员鉴权
	pf := e.Group("/api/platform")
	pf.Use(mw.RequirePlatformHost())
	pf.Use(mw.PlatformTx(pool))
	pf.Use(mw.RequireOperator())
	pf.GET("/me", platformcore.Me)
	pf.GET("/applications", platformcore.ListApplications)
	pf.POST("/applications/:id/approve", platformcore.ApproveApplication)
	pf.POST("/applications/:id/reject", platformcore.RejectApplication)
	pf.GET("/tenants", platformcore.ListTenants)
	pf.POST("/tenants", platformcore.CreateTenant)
	pf.POST("/tenants/:id/suspend", platformcore.SuspendTenant)
	pf.POST("/tenants/:id/resume", platformcore.ResumeTenant)
	pf.POST("/tenants/:id/subscription", platformcore.SetSubscription)
	pf.POST("/tenants/:id/reset-admin-password", platformcore.ResetAdminPassword)
	pf.GET("/editions", platformcore.ListEditions)
	pf.PUT("/editions/:code", platformcore.UpdateEdition)

	// 多租户 API：所有 /api/* 必须经过 tenant resolver + 事务中间件
	api := e.Group("/api")
	api.Use(mw.TenantResolver(pool))
	api.Use(mw.TenantTx(pool))
	api.Use(mw.AuditLog(pool))

	// 登录端点：用 tenant 上下文 + RLS 验证 users 表
	// per-IP 限流：每分钟 10 次（防暴力穷举）
	loginLimit := mw.RateLimit(10, time.Minute)
	api.POST("/login", auth.LoginHandler, loginLimit)

	// 图形验证码：登录失败多次后前端显示，也用于锁定账号自救
	// per-IP 限流：每分钟 20 次（防刷消耗内存）
	captchaLimit := mw.RateLimit(20, time.Minute)
	api.GET("/captcha", auth.CaptchaHandler, captchaLimit)

	// 公开预约端（含 booking_code 校验，不需要 JWT）
	// per-IP 限流：防 booking_code 暴力枚举 + 垃圾预约（code 错误也消耗配额）
	bookingLimit := mw.RateLimit(30, time.Minute)
	api.GET("/booking/options", booking.Options, bookingLimit)
	api.POST("/booking", booking.Create, bookingLimit)

	// 公开店铺展示信息（登录页等未登录场景使用）
	api.GET("/store/info", tenantsettings.StoreInfo)

	// 静态文件服务：店铺 logo 等上传资源（公开访问）
	// 路径映射 /uploads/* → ./uploads/*
	// 安全响应头：禁止 MIME sniff + 严格 CSP + inline 展示（防 polyglot XSS）
	uploadsGroup := e.Group("/uploads", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'")
			h.Set("Content-Disposition", "inline")
			h.Set("X-Frame-Options", "DENY")
			return next(c)
		}
	})
	uploadsGroup.Static("", uploads.UploadRoot)

	// --------------- 三级权限分组 ---------------
	// secured        : 所有登录用户（super_admin + admin + staff）
	// adminAndAbove  : super_admin + admin（staff 不可）
	// superOnly      : 仅 super_admin
	secured := api.Group("")
	secured.Use(mw.RequireAuth())

	adminAndAbove := secured.Group("")
	adminAndAbove.Use(mw.RequireAtLeastAdmin())

	superOnly := secured.Group("")
	superOnly.Use(mw.RequireSuperAdmin())

	pwdLimit := mw.RateLimit(5, time.Minute)
	// 员工查询限流：仅 role=staff 生效，admin/super_admin 不受限
	// 覆盖会员/交易 list + detail，防止通过 :id 逐个枚举绕过 List 的 search≥3 限制
	staffReadLimit := mw.RateLimitStaffOnly(30, time.Minute)

	// --------------- 会员 ---------------
	// staff 可查、可创建（录入新会员）；编辑/删除需 admin（防离职员工批量改 phone 劫持续费通知）
	secured.GET("/members", members.List, staffReadLimit)
	secured.POST("/members", members.Create)
	secured.GET("/members/:id", members.Get, staffReadLimit)
	adminAndAbove.PUT("/members/:id", members.Update)
	adminAndAbove.DELETE("/members/:id", members.Delete)

	// --------------- 基础配置（仅 admin 及以上可写） ---------------
	secured.GET("/payment-methods", paymentmethods.List)
	secured.GET("/payment-methods/:id", paymentmethods.Get)
	adminAndAbove.POST("/payment-methods", paymentmethods.Create)
	adminAndAbove.PUT("/payment-methods/:id", paymentmethods.Update)
	adminAndAbove.DELETE("/payment-methods/:id", paymentmethods.Delete)

	secured.GET("/services", services.List)
	secured.GET("/services/:id", services.Get)
	adminAndAbove.POST("/services", services.Create)
	adminAndAbove.PUT("/services/:id", services.Update)
	adminAndAbove.DELETE("/services/:id", services.Delete)

	secured.GET("/staff", staff.List)
	secured.GET("/staff/:id", staff.Get)
	adminAndAbove.POST("/staff", staff.Create)
	adminAndAbove.PUT("/staff/:id", staff.Update)
	adminAndAbove.DELETE("/staff/:id", staff.Delete)

	secured.GET("/card-types", cardtypes.List)
	secured.GET("/card-types/:id", cardtypes.Get)
	adminAndAbove.POST("/card-types", cardtypes.Create)
	adminAndAbove.PUT("/card-types/:id", cardtypes.Update)
	adminAndAbove.DELETE("/card-types/:id", cardtypes.Delete)

	// --------------- 会员卡 ---------------
	// 只读列表；写操作唯一入口是下方 /cards/with-transaction（办卡+收款+流水联动）。
	// 裸发卡 / 卡级 GET/PUT/DELETE 已移除：裸发卡可凭空充值（无交易无流水），
	// 删卡会级联销毁 card_balance_logs 资金流水，且均无前端调用
	secured.GET("/members/:memberId/cards", cards.ListByMember)

	// --------------- 挂账 ---------------
	// staff 可建挂账、可清账（业务需要）；撤消（删挂账）需 admin（防员工消除痕迹）
	secured.GET("/members/:memberId/pending", membercredits.ListPending)
	secured.POST("/members/:memberId/pending", membercredits.Create)
	adminAndAbove.DELETE("/members/:memberId/pending/:pendingId", membercredits.Delete)
	secured.POST("/members/:memberId/pending/:pendingId/settle", transactions.SettleCredit)

	// --------------- 交易 ---------------
	secured.GET("/transactions", transactions.List, staffReadLimit)
	secured.POST("/transactions", transactions.Create)
	secured.GET("/transactions/:id", transactions.Get, staffReadLimit)
	// 交易撤销：仅 super_admin（敏感操作）
	superOnly.POST("/transactions/:id/void", transactions.Void)
	secured.POST("/members/:memberId/cards/with-transaction", transactions.IssueCard)
	secured.POST("/members/:memberId/pending/settle-all", transactions.SettleAllCredits)

	// --------------- 预约 ---------------
	// staff 可查+建预约；修改状态/删除需 admin（防改 completed 骗佣金 / 删他人预约）
	secured.GET("/appointments", appointments.List)
	secured.POST("/appointments", appointments.Create)
	secured.GET("/appointments/count/today", appointments.CountToday)
	secured.GET("/appointments/:id", appointments.Get)
	adminAndAbove.PATCH("/appointments/:id/status", appointments.UpdateStatus)
	adminAndAbove.DELETE("/appointments/:id", appointments.Delete)

	// --------------- 租户设置 ---------------
	secured.GET("/tenant-settings", tenantsettings.List)
	secured.GET("/tenant-settings/:key", tenantsettings.Get)
	adminAndAbove.PUT("/tenant-settings/:key", tenantsettings.Upsert)
	adminAndAbove.DELETE("/tenant-settings/:key", tenantsettings.Delete)
	// 开启/关闭撤单 + 设置超级密码：仅 super_admin
	// enable-void 校验超级密码，必须与其他密码类端点同档限流（防暴力穷举）
	superOnly.POST("/tenant-settings/enable-void", tenantsettings.EnableVoid, pwdLimit)
	superOnly.POST("/tenant-settings/disable-void", tenantsettings.DisableVoid, pwdLimit)
	adminAndAbove.POST("/tenant-settings/booking-code", tenantsettings.RegenerateBookingCode)
	superOnly.POST("/tenant-settings/super-password", tenantsettings.SetSuperPassword, pwdLimit)
	secured.GET("/tenant-settings/super-password/status", tenantsettings.SuperPasswordStatus)

	// --------------- 用户账号（仅 admin 及以上可见，仅 super_admin 可管） ---------------
	// staff 不应能列举员工账号（姓名/邮箱/role/最后登录时间）
	adminAndAbove.GET("/users", users2.List)
	adminAndAbove.GET("/users/:id", users2.Get)
	superOnly.POST("/users", users2.Create)
	superOnly.PUT("/users/:id", users2.Update)
	superOnly.POST("/users/:id/reset-password", users2.ResetPassword)
	superOnly.DELETE("/users/:id", users2.Delete)

	// --------------- 操作日志（admin 及以上） ---------------
	adminAndAbove.GET("/audit-logs", auditlogs.List)
	// 批量 ID→名字解析：给日志对象列显示"会员·张三 / 交易·¥128 / 预约·王女士"
	adminAndAbove.GET("/audit-logs/lookup", auditlogs.Lookup)

	// --------------- 自助修改密码 / 续签 / 登出 ---------------
	secured.POST("/auth/change-password", auth.ChangePasswordHandler, pwdLimit)
	secured.POST("/auth/refresh", auth.RefreshHandler)
	// 登出 = token_version+1，吊销该用户所有已签发 token（含 remember 长 token）
	secured.POST("/auth/logout", auth.LogoutHandler)

	// --------------- 店铺 logo（admin 及以上） ---------------
	adminAndAbove.POST("/store/logo", uploads.UploadLogo)
	adminAndAbove.DELETE("/store/logo", uploads.DeleteLogo)

	// --------------- 套餐订阅（admin 及以上，到期提示条数据源） ---------------
	adminAndAbove.GET("/store/subscription", tenantsettings.StoreSubscription)

	// --------------- 通知与公告（所有登录用户） ---------------
	secured.GET("/notifications", notifications.List)
	secured.POST("/notifications/:version/read", notifications.MarkRead)
	secured.POST("/notifications/read-all", notifications.MarkAllRead)
	secured.GET("/notifications/latest-version", notifications.LatestVersion)
	secured.GET("/changelog", notifications.ListChangelog)

	// --------------- 报表（admin 及以上） ---------------
	adminAndAbove.GET("/reports/business", reports.Business)
	adminAndAbove.GET("/reports/card-pool", reports.CardPool)
	adminAndAbove.GET("/reports/service-ranking", reports.ServiceRanking)
	adminAndAbove.GET("/reports/sleeping-members", reports.SleepingMembers)
	adminAndAbove.GET("/reports/member-ranking", reports.MemberRanking)
	adminAndAbove.GET("/reports/birthday-reminders", reports.BirthdayReminders)
	adminAndAbove.GET("/reports/payment-summary", reports.PaymentSummary)
	adminAndAbove.GET("/reports/card-sales-summary", reports.CardSalesSummary)
	adminAndAbove.GET("/reports/pending-stats", reports.PendingStats)

	addr := ":" + getenv("APP_PORT", "8080")
	slog.Info("mms server starting", "addr", addr)

	// 优雅关闭：SIGINT / SIGTERM → ctx cancel → Echo StartConfig 自动 drain 活跃请求
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := echo.StartConfig{
		Address:         addr,
		GracefulTimeout: 15 * time.Second,
	}
	if err := cfg.Start(ctx, e); err != nil && err != http.ErrServerClosed {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
	// 等待所有审计 goroutine 落盘（最多 8s）
	mw.AuditWait(8 * time.Second)
	slog.Info("mms server exited")
}

// validateCriticalEnv 启动时检查关键环境变量，拦截默认/示例值
// 避免 JWT_SECRET 忘改导致"所有人可伪造任意 JWT"等灾难
func validateCriticalEnv() error {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET too short (need >= 32 bytes, got %d); use `openssl rand -hex 64`", len(secret))
	}
	if strings.Contains(secret, "change_me") {
		return fmt.Errorf("JWT_SECRET still has default placeholder 'change_me_*'; please set a real value")
	}

	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if strings.Contains(dbPass, "change_me") {
		return fmt.Errorf("DB_PASSWORD still has default placeholder 'change_me_*'; please set a real value")
	}

	// BOOTSTRAP_ADMIN_PASSWORD：仅在非空时校验（空=已有 admin 跳过 bootstrap）
	// 避免默认 "change_me_on_first_login" 成为事实默认密码
	bootstrapPwd := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if bootstrapPwd != "" && strings.Contains(bootstrapPwd, "change_me") {
		return fmt.Errorf("BOOTSTRAP_ADMIN_PASSWORD contains 'change_me' placeholder; set a real initial password or leave empty")
	}

	// DEPLOYMENT_MODE 拼错会让套餐限额静默失效（quota 只认 "hosted"），启动时把关
	switch os.Getenv("DEPLOYMENT_MODE") {
	case "", "hosted", "self-hosted", "enterprise", "dev":
	default:
		return fmt.Errorf("DEPLOYMENT_MODE must be one of hosted / self-hosted / enterprise (got %q)", os.Getenv("DEPLOYMENT_MODE"))
	}
	return nil
}

// runMigrations 使用应用用户 mms 跑 goose 迁移
// mms 是 mms 数据库的 owner，且 pgcrypto/citext/pg_trgm 均为 trusted extension，
// 无需 superuser 即可完成 CREATE EXTENSION / CREATE TABLE / RLS 策略创建
func runMigrations() error {
	// 与运行时 pool 共用 DSN 构造器（sslmode / statement_timeout 统一走 env）
	db, err := sql.Open("pgx", dbpkg.BuildDSN())
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

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
