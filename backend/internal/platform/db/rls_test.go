package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestTenantRLSIsolation 验证多租户硬约束：A 租户读不到、也写不进 B 租户。
//
// 走真实 PG（复用 NewPool 的 env 连接配置，即容器内 .env），同时顺带验证
// NewPool 的 BYPASSRLS 启动校验放行普通角色。本地无库时跳过。
func TestTenantRLSIsolation(t *testing.T) {
	if os.Getenv("DB_PASSWORD") == "" {
		t.Skip("DB_PASSWORD 未配置，跳过需要真实数据库的集成测试")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	// tenants 是公开目录表（无 RLS），直接建两个一次性租户
	suffix := uuid.NewString()[:8]
	tenantA := createTenant(ctx, t, pool, "rlstest-a-"+suffix)
	tenantB := createTenant(ctx, t, pool, "rlstest-b-"+suffix)
	defer func() {
		// 不能 DELETE tenants：audit_logs 的 append-only 触发器是语句级的，
		// 级联动作即使影响 0 行也会被拒绝（审计不可变性优先，删租户是手动运维流程）。
		// 改为按租户上下文删测试 members + 把租户标记 deleted
		for _, tid := range []uuid.UUID{tenantA, tenantB} {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Errorf("清理 begin: %v", err)
				continue
			}
			_, _ = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tid.String())
			_, _ = tx.Exec(ctx, `DELETE FROM members WHERE tenant_id = $1`, tid)
			_ = tx.Commit(ctx)
		}
		if _, err := pool.Exec(ctx, `UPDATE tenants SET status = 'deleted' WHERE id = ANY($1::uuid[])`,
			[]uuid.UUID{tenantA, tenantB}); err != nil {
			t.Errorf("标记测试租户 deleted 失败: %v", err)
		}
	}()

	memberA := createMember(ctx, t, pool, tenantA, "会员A")
	memberB := createMember(ctx, t, pool, tenantB, "会员B")

	// 进入 A 租户上下文（与 TenantTx 中间件相同：事务内 set_config local）
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantA.String()); err != nil {
		t.Fatalf("set_config: %v", err)
	}

	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM members WHERE id = $1`, memberA).Scan(&n); err != nil {
		t.Fatalf("查本租户会员: %v", err)
	}
	if n != 1 {
		t.Errorf("A 上下文应能读到自己的会员，got count=%d", n)
	}

	if err := tx.QueryRow(ctx, `SELECT count(*) FROM members WHERE id = $1`, memberB).Scan(&n); err != nil {
		t.Fatalf("查跨租户会员: %v", err)
	}
	if n != 0 {
		t.Errorf("A 上下文不应读到 B 租户会员，got count=%d", n)
	}

	// 跨租户写入必须被 RLS 的 WITH CHECK 拒绝
	if _, err := tx.Exec(ctx,
		`INSERT INTO members (tenant_id, name) VALUES ($1, '越权写入')`, tenantB); err == nil {
		t.Errorf("A 上下文写入 B 租户的 INSERT 应被 RLS 拒绝，却成功了")
	}
}

func createTenant(ctx context.Context, t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING id`,
		slug, "RLS 隔离测试租户").Scan(&id); err != nil {
		t.Fatalf("建测试租户 %s: %v", slug, err)
	}
	return id
}

func createMember(ctx context.Context, t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID.String()); err != nil {
		t.Fatalf("set_config: %v", err)
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx,
		`INSERT INTO members (tenant_id, name) VALUES ($1, $2) RETURNING id`,
		tenantID, name).Scan(&id); err != nil {
		t.Fatalf("建测试会员: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return id
}
