package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BuildDSN 根据 env 生成 Postgres 连接字符串
//   - sslmode：默认 prefer（尝试 TLS，失败降级 plaintext）；生产跨网部署用 require/verify-full
//   - statement_timeout：5s，防慢查询卡死 pool（聚合报表若超 5s 需 handler 内显式 context 覆盖）
//   - application_name：mms，PG 日志/pg_stat_activity 里可区分来源
func BuildDSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s&statement_timeout=%s&application_name=mms",
		getenv("DB_USER", "mms"),
		url.QueryEscape(os.Getenv("DB_PASSWORD")),
		getenv("DB_HOST", "postgres-server"),
		getenv("DB_PORT", "5432"),
		getenv("DB_NAME", "mms"),
		getenv("DB_SSLMODE", "prefer"),
		getenv("DB_STATEMENT_TIMEOUT", "5000"), // 毫秒
	)
}

// NewPool 构建 mms 应用用户的 pgxpool
//   - 单用户模型（Z 方案）：迁移与运行时同用 mms，无 superuser 参与
//   - AfterConnect 注册 shopspring/decimal 类型解码器，让 sqlc 生成的 decimal.Decimal 能 scan
//   - 池大小通过 DB_MIN_CONNS / DB_MAX_CONNS env 配置（默认 2/20 对 500 商户规模够用）
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(BuildDSN())
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	cfg.MinConns = int32(getenvInt("DB_MIN_CONNS", 2))
	cfg.MaxConns = int32(getenvInt("DB_MAX_CONNS", 20))
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

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}
