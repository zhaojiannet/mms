package reports

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/db"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// TestReportMonthlyBusinessIgnoresSessionTimeZone 在 UTC 会话里按上海时区的月界统计，
// 月底最后一刻和月初第一刻的数据必须分别落在各自的月份，且只统计本租户。
// 走真实 PG，本地无库时跳过；全部写入在事务里，结束时回滚
func TestReportMonthlyBusinessIgnoresSessionTimeZone(t *testing.T) {
	if os.Getenv("DB_PASSWORD") == "" {
		t.Skip("DB_PASSWORD 未配置，跳过需要真实数据库的集成测试")
	}
	ctx := context.Background()
	pool, err := db.NewPool(ctx)
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SET LOCAL timezone = 'UTC'`); err != nil {
		t.Fatalf("设会话时区: %v", err)
	}

	sh := mustLoadLocation(t, "Asia/Shanghai")
	suffix := uuid.NewString()[:8]
	tenantA := createTenantWithMembers(ctx, t, tx, "monthly-a-"+suffix,
		time.Date(2026, 8, 31, 23, 59, 59, 0, sh),
		time.Date(2026, 9, 30, 23, 59, 59, 0, sh),
		time.Date(2026, 10, 1, 0, 0, 0, 0, sh),
		time.Date(2026, 10, 31, 23, 59, 59, 0, sh),
	)
	// B 租户在同几个月也有会员，A 的统计里不能出现
	createTenantWithMembers(ctx, t, tx, "monthly-b-"+suffix,
		time.Date(2026, 9, 15, 12, 0, 0, 0, sh),
		time.Date(2026, 10, 15, 12, 0, 0, 0, sh),
	)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantA.String()); err != nil {
		t.Fatalf("set_config: %v", err)
	}

	rows, err := sqlc.New(tx).ReportMonthlyBusiness(ctx, monthBounds(time.Date(2026, 10, 24, 12, 0, 0, 0, sh), 12))
	if err != nil {
		t.Fatalf("ReportMonthlyBusiness: %v", err)
	}
	if len(rows) != 12 {
		t.Fatalf("应返回 12 个月，got %d", len(rows))
	}

	wantNew := map[string]int32{"2026-08": 1, "2026-09": 1, "2026-10": 2}
	want := time.Date(2025, 11, 1, 0, 0, 0, 0, sh)
	for _, r := range rows {
		got := r.MStart.Time.In(sh)
		if !got.Equal(want) {
			t.Errorf("月初 = %v，应为 %v", got, want)
		}
		month := want.Format("2006-01")
		if r.NewMembers != wantNew[month] {
			t.Errorf("%s 新增会员 = %d，应为 %d", month, r.NewMembers, wantNew[month])
		}
		want = want.AddDate(0, 1, 0)
	}
}

// createTenantWithMembers 在 tx 里建租户并以该租户身份插入给定创建时间的会员，
// 结束时 app.current_tenant 停在这个租户上
func createTenantWithMembers(ctx context.Context, t *testing.T, tx pgx.Tx, slug string, createdAt ...time.Time) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ($1, $2) RETURNING id`,
		slug, "月度报表测试租户").Scan(&id); err != nil {
		t.Fatalf("建测试租户 %s: %v", slug, err)
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, id.String()); err != nil {
		t.Fatalf("set_config: %v", err)
	}
	for _, at := range createdAt {
		if _, err := tx.Exec(ctx, `INSERT INTO members (tenant_id, name, created_at) VALUES ($1, $2, $3)`,
			id, "会员 "+at.Format(time.DateTime), at); err != nil {
			t.Fatalf("建测试会员: %v", err)
		}
	}
	return id
}
