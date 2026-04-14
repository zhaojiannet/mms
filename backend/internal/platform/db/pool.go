package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool 构建 mms 应用用户的 pgxpool
//   - 单用户模型（Z 方案）：迁移与运行时同用 mms，无 superuser 参与
//   - AfterConnect 注册 shopspring/decimal 类型解码器，让 sqlc 生成的 decimal.Decimal 能 scan
//   - 连接池大小按 500 商户规模 + 单 Go 实例估：min=2 max=20 已覆盖峰值
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&application_name=mms",
		getenv("DB_USER", "mms"),
		url.QueryEscape(os.Getenv("DB_PASSWORD")),
		getenv("DB_HOST", "postgres-server"),
		getenv("DB_PORT", "5432"),
		getenv("DB_NAME", "mms"),
	)

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	cfg.MinConns = 2
	cfg.MaxConns = 20
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pgxpool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pool: %w", err)
	}

	return pool, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
