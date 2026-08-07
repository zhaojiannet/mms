package db

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
)

// TestPlatformSentinelDoesNotLeak 验证平台哨兵 app.platform_op 不会污染商户请求。
//
// 平台后台需要跨租户读 subscriptions，靠事务内 set_config('app.platform_op','on',true)
// 放行策略。风险在于：连接归还池子后若哨兵仍留在会话上，下一个商户请求就能读到
// 全租户订阅。本测试在同一连接上先跑平台事务再跑商户事务，确认不串。
func TestPlatformSentinelDoesNotLeak(t *testing.T) {
	if os.Getenv("DB_PASSWORD") == "" {
		t.Skip("DB_PASSWORD 未配置，跳过需要真实数据库的集成测试")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	// 独占一条连接，确保平台事务与后续商户事务跑在同一个后端会话上
	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer conn.Release()

	// 1. 平台事务：哨兵放行跨租户读
	txPlatform, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin platform tx: %v", err)
	}
	if _, err := txPlatform.Exec(ctx, `SELECT set_config('app.platform_op', 'on', true)`); err != nil {
		t.Fatalf("set platform sentinel: %v", err)
	}
	var platformVisible int
	if err := txPlatform.QueryRow(ctx, `SELECT count(*) FROM subscriptions`).Scan(&platformVisible); err != nil {
		t.Fatalf("平台事务读订阅: %v", err)
	}
	if err := txPlatform.Commit(ctx); err != nil {
		t.Fatalf("commit platform tx: %v", err)
	}

	// 2. 同一连接上的裸事务（模拟哨兵泄漏后的商户请求）：没有任何上下文时必须读到 0 行
	txAfter, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin follow-up tx: %v", err)
	}
	defer txAfter.Rollback(ctx)

	var sentinel string
	if err := txAfter.QueryRow(ctx,
		`SELECT COALESCE(current_setting('app.platform_op', true), '')`).Scan(&sentinel); err != nil {
		t.Fatalf("读哨兵: %v", err)
	}
	if sentinel == "on" {
		t.Errorf("哨兵泄漏到后续事务：app.platform_op=%q（set_config 第三参数必须为 true）", sentinel)
	}

	var leaked int
	if err := txAfter.QueryRow(ctx, `SELECT count(*) FROM subscriptions`).Scan(&leaked); err != nil {
		t.Fatalf("后续事务读订阅: %v", err)
	}
	if leaked != 0 {
		t.Errorf("无租户上下文的事务读到 %d 条订阅，RLS 未生效（平台事务可见 %d 条）", leaked, platformVisible)
	}
}

// TestPlatformTenantCtxClearsSentinel 验证平台事务切入单租户后哨兵被清除。
//
// 开通商户 / 重置密码会在平台事务内 SET app.current_tenant 后写 users、subscriptions。
// 若哨兵仍为 on，这些写操作后的任何查询都失去 RLS 兜底；清除后即便漏写租户过滤，
// 也只能看到当前租户的数据。
func TestPlatformTenantCtxClearsSentinel(t *testing.T) {
	if os.Getenv("DB_PASSWORD") == "" {
		t.Skip("DB_PASSWORD 未配置，跳过需要真实数据库的集成测试")
	}
	ctx := context.Background()
	pool, err := NewPool(ctx)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	suffix := uuid.NewString()[:8]
	tenantA := createTenant(ctx, t, pool, "pfrls-a-"+suffix)
	tenantB := createTenant(ctx, t, pool, "pfrls-b-"+suffix)
	defer func() {
		if _, err := pool.Exec(ctx,
			`UPDATE tenants SET status = 'deleted' WHERE id = ANY($1::uuid[])`,
			[]uuid.UUID{tenantA, tenantB}); err != nil {
			t.Errorf("标记测试租户 deleted 失败: %v", err)
		}
	}()

	memberA := createMember(ctx, t, pool, tenantA, "平台测试会员A")
	memberB := createMember(ctx, t, pool, tenantB, "平台测试会员B")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	// 平台事务起手：哨兵 on
	if _, err := tx.Exec(ctx, `SELECT set_config('app.platform_op', 'on', true)`); err != nil {
		t.Fatalf("set sentinel: %v", err)
	}
	// 切入 A 租户（与 core/platform.setTenantCtx 完全一致的语句）
	if _, err := tx.Exec(ctx,
		`SELECT set_config('app.current_tenant', $1, true), set_config('app.platform_op', '', true)`,
		tenantA.String()); err != nil {
		t.Fatalf("set tenant ctx: %v", err)
	}

	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM members WHERE id = $1`, memberA).Scan(&n); err != nil {
		t.Fatalf("读本租户会员: %v", err)
	}
	if n != 1 {
		t.Errorf("切入 A 后应能读到 A 的会员，got %d", n)
	}

	// 关键：哨兵已清，漏写租户过滤也读不到 B
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM members WHERE id = $1`, memberB).Scan(&n); err != nil {
		t.Fatalf("读跨租户会员: %v", err)
	}
	if n != 0 {
		t.Errorf("切入 A 的平台事务不应读到 B 租户会员，got %d（哨兵未清除？）", n)
	}
}
