package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
)

// EnsureOperator 幂等确保 PLATFORM_ADMIN_EMAIL 的平台操作员存在
//   - 任一 env 为空则跳过（自建部署可不启用平台后台）
//   - 已存在同 email 则跳过；建好后可清空 env
func EnsureOperator(ctx context.Context, pool *pgxpool.Pool) error {
	email := os.Getenv("PLATFORM_ADMIN_EMAIL")
	password := os.Getenv("PLATFORM_ADMIN_PASSWORD")
	name := os.Getenv("PLATFORM_ADMIN_NAME")
	if name == "" {
		name = "运营"
	}
	if email == "" || password == "" {
		slog.Info("bootstrap operator skipped (env not fully set)")
		return nil
	}

	var existing int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM platform_operators WHERE email = $1", email,
	).Scan(&existing); err != nil {
		return fmt.Errorf("count operators: %w", err)
	}
	if existing > 0 {
		slog.Info("bootstrap operator already exists", "email", email)
		return nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO platform_operators (email, password_hash, name)
		VALUES ($1, $2, $3)
	`, email, hash, name); err != nil {
		return fmt.Errorf("insert operator: %w", err)
	}
	slog.Info("bootstrap operator created", "email", email)
	return nil
}
