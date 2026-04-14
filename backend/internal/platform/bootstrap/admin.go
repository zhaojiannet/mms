// Package bootstrap 提供启动期幂等初始化逻辑
// 当前仅负责在 BOOTSTRAP_ADMIN_* 环境变量完整且目标租户无同名用户时创建一条 admin 账户
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
)

// EnsureAdmin 幂等确保 BOOTSTRAP_TENANT_SLUG 下存在 BOOTSTRAP_ADMIN_EMAIL 的 admin
//   - 三个 env 任意为空则跳过（AGPL 用户可自行关闭 bootstrap）
//   - 目标 tenant 不存在则跳过（不应由本函数建 tenant，tenant 须由 migration seed 或运营后台创建）
//   - 已存在同 email 的用户则跳过
func EnsureAdmin(ctx context.Context, pool *pgxpool.Pool) error {
	slug := os.Getenv("BOOTSTRAP_TENANT_SLUG")
	email := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	password := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	name := os.Getenv("BOOTSTRAP_ADMIN_NAME")
	if name == "" {
		name = "管理员"
	}
	if slug == "" || email == "" || password == "" {
		slog.Info("bootstrap admin skipped (env not fully set)")
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var tenantID uuid.UUID
	err = tx.QueryRow(ctx, "SELECT id FROM tenants WHERE slug = $1", slug).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("bootstrap admin: tenant not found", "slug", slug)
			return nil
		}
		return fmt.Errorf("query tenant: %w", err)
	}

	// 进入 tenant 上下文，后续 users 操作受 RLS 保护
	if _, err := tx.Exec(ctx,
		"SELECT set_config('app.current_tenant', $1, true)",
		tenantID.String(),
	); err != nil {
		return fmt.Errorf("set tenant ctx: %w", err)
	}

	var existing int
	if err := tx.QueryRow(ctx,
		"SELECT count(*) FROM users WHERE email = $1",
		email,
	).Scan(&existing); err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if existing > 0 {
		slog.Info("bootstrap admin already exists", "tenant", slug, "email", email)
		return tx.Commit(ctx)
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, name, role)
		VALUES ($1, $2, $3, $4, 'admin')
	`, tenantID, email, hash, name); err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	slog.Info("bootstrap admin created", "tenant", slug, "email", email)
	return tx.Commit(ctx)
}
