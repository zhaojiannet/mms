package platform

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

// slugPattern 商户子域：小写字母开头，字母数字连字符，3-30 位，不以连字符结尾
var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,28}[a-z0-9]$`)

// ValidateSlug 校验期望子域的格式与保留字
func ValidateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return errors.New("子域格式不合法：小写字母开头，仅限字母数字连字符，3-30 位")
	}
	if mw.IsReservedSubdomain(slug) {
		return errors.New("该子域为系统保留，换一个吧")
	}
	return nil
}

type provisionInput struct {
	Slug       string
	Name       string
	Edition    string // editions.code
	Months     int
	Source     string // gift | manual | contract
	AdminEmail string
	AdminName  string
}

type provisionResult struct {
	TenantID      uuid.UUID
	AdminPassword string // 一次性明文，仅本次响应展示
}

// provisionTenant 开通商户：租户 + 订阅 + 管理员，一个事务内完成
// 调用方负责最外层事务的提交/回滚；本函数会切换 app.current_tenant
func provisionTenant(ctx context.Context, tx pgx.Tx, in provisionInput) (provisionResult, error) {
	var out provisionResult

	if err := ValidateSlug(in.Slug); err != nil {
		return out, err
	}
	if in.Name == "" || in.AdminEmail == "" {
		return out, errors.New("店铺名与管理员邮箱必填")
	}
	if in.Months < 1 || in.Months > 120 {
		return out, errors.New("期限须在 1-120 个月之间")
	}
	if in.Source == "" {
		in.Source = "manual"
	}
	if in.Source != "gift" && in.Source != "manual" && in.Source != "contract" {
		return out, errors.New("无效的订阅来源")
	}
	if in.AdminName == "" {
		in.AdminName = "管理员"
	}

	editionID, _, err := editionByCode(ctx, tx, in.Edition)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, errors.New("套餐不存在：" + in.Edition)
		}
		return out, fmt.Errorf("query edition: %w", err)
	}

	var exists int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM tenants WHERE slug = $1", in.Slug).Scan(&exists); err != nil {
		return out, fmt.Errorf("check slug: %w", err)
	}
	if exists > 0 {
		return out, errors.New("子域已被占用：" + in.Slug)
	}

	if err := tx.QueryRow(ctx, `
		INSERT INTO tenants (slug, name, status) VALUES ($1, $2, 'active') RETURNING id
	`, in.Slug, in.Name).Scan(&out.TenantID); err != nil {
		return out, fmt.Errorf("insert tenant: %w", err)
	}

	if err := setTenantCtx(ctx, tx, out.TenantID); err != nil {
		return out, fmt.Errorf("set tenant ctx: %w", err)
	}

	cycle := "monthly"
	switch {
	case in.Source == "gift":
		cycle = "gift"
	case in.Source == "contract":
		cycle = "contract"
	case in.Months >= 12:
		cycle = "yearly"
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO subscriptions (tenant_id, edition_id, source, billing_cycle, current_period_end, auto_renew)
		VALUES ($1, $2, $3, $4, $5, FALSE)
	`, out.TenantID, editionID, in.Source, cycle, time.Now().AddDate(0, in.Months, 0)); err != nil {
		return out, fmt.Errorf("insert subscription: %w", err)
	}

	password, err := genPassword()
	if err != nil {
		return out, fmt.Errorf("gen password: %w", err)
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return out, fmt.Errorf("hash password: %w", err)
	}
	// 商户老板给 super_admin：要能自管员工账号（users CRUD 是 superOnly 路由）
	if _, err := tx.Exec(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, name, role)
		VALUES ($1, $2, $3, $4, 'super_admin')
	`, out.TenantID, in.AdminEmail, hash, in.AdminName); err != nil {
		return out, fmt.Errorf("insert admin: %w", err)
	}

	out.AdminPassword = password
	return out, nil
}
