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

	"github.com/zhaojiannet/mms/backend/internal/core/auth"
	"github.com/zhaojiannet/mms/backend/internal/core/members"
	"github.com/zhaojiannet/mms/backend/internal/platform/bootstrap"
	"github.com/zhaojiannet/mms/backend/internal/platform/db"
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

	// 多租户 API：所有 /api/* 必须经过 tenant resolver + 事务中间件
	api := e.Group("/api")
	api.Use(mw.TenantResolver(pool))
	api.Use(mw.TenantTx(pool))

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

	// 需要 JWT 的业务路由组（JWT 中 tenant_id 必须与 Host 解析到的一致）
	secured := api.Group("")
	secured.Use(mw.RequireAuth())
	secured.GET("/members", members.List)
	secured.POST("/members", members.Create)
	secured.GET("/members/:id", members.Get)
	secured.PUT("/members/:id", members.Update)
	secured.DELETE("/members/:id", members.Delete)

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
