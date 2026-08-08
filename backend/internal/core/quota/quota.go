// Package quota 套餐限额执行：hosted 模式下按租户当前套餐的 quotas 拦截新增
package quota

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
)

var (
	hostedOnce sync.Once
	hosted     bool
)

// enforced 限额仅 hosted 生效；self-hosted / enterprise 视为无限（既定约束）
// DEPLOYMENT_MODE 的取值合法性由 main.validateCriticalEnv 启动时把关，拼错不会静默放行
func enforced() bool {
	hostedOnce.Do(func() {
		hosted = os.Getenv("DEPLOYMENT_MODE") == "hosted"
		slog.Info("quota enforcement", "enabled", hosted)
	})
	return hosted
}

// kinds 可执行的限额项；只统计在册（active）行，归档不占额度
var kinds = map[string]struct {
	countSQL  string // $1 = 排除的行 id（uuid.Nil 表示不排除）
	statusSQL string // 取单行当前状态，判断本次更新是否真的新增 active 名额
	label     string
}{
	"max_members": {
		"SELECT count(*) FROM members WHERE status = 'active' AND id != $1",
		"SELECT status FROM members WHERE id = $1",
		"会员",
	},
	"max_staff": {
		"SELECT count(*) FROM staff WHERE status = 'active' AND id != $1",
		"SELECT status FROM staff WHERE id = $1",
		"员工",
	},
}

// Enforce 创建路径调用：超限返回 403 业务错误，未超限返回 nil
func Enforce(c *echo.Context, kind string) error {
	return enforce(c, kind, uuid.Nil)
}

// EnforceOnActivate 更新路径带 status=active 时调用，堵"归档→新建→恢复"绕过
//
// 只有原状态不是 active 才计限额：前端编辑表单总会回传当前 status，
// 若一律检查，则限额下调或商户降档后，存量已超限的记录连改个手机号都会被拦
// （README 承诺"已有数据不追溯"）。行不存在时放过，交由后续 UPDATE 返回 404。
func EnforceOnActivate(c *echo.Context, kind string, id uuid.UUID) error {
	if !enforced() {
		return nil
	}
	spec, ok := kinds[kind]
	if !ok {
		return fmt.Errorf("quota: unknown kind %q", kind)
	}
	var current string
	if err := mw.TxFrom(c).QueryRow(c.Request().Context(), spec.statusSQL, id).Scan(&current); err != nil {
		return nil
	}
	if current == "active" {
		return nil
	}
	return enforce(c, kind, id)
}

// enforce 并发安全：先 FOR NO KEY UPDATE 锁租户行，同租户并发创建被序列化、
// 限额边界不超发；不用 FOR UPDATE——它与子表 FK 校验的 FOR KEY SHARE 冲突，
// 会把该租户全部并发 INSERT（交易/预约/审计）都挡在锁后
// 限额来源：该租户最新一条订阅的套餐 quotas；无订阅行按 free 档当时值。
// quotas 里键缺失或值 <= 0 视为不限（套餐配置的显式语义，由运营后台维护）。
func enforce(c *echo.Context, kind string, excludeID uuid.UUID) error {
	if !enforced() {
		return nil
	}
	spec, ok := kinds[kind]
	if !ok {
		return fmt.Errorf("quota: unknown kind %q", kind)
	}

	ctx := c.Request().Context()
	tx := mw.TxFrom(c)
	tenant := mw.TenantFrom(c)

	if _, err := tx.Exec(ctx, "SELECT 1 FROM tenants WHERE id = $1 FOR NO KEY UPDATE", tenant.ID); err != nil {
		return fmt.Errorf("quota: lock tenant: %w", err)
	}

	var quotasRaw []byte
	err := tx.QueryRow(ctx, `
		SELECT e.quotas FROM subscriptions s
		JOIN editions e ON e.id = s.edition_id
		WHERE s.tenant_id = $1
		ORDER BY s.created_at DESC LIMIT 1
	`, tenant.ID).Scan(&quotasRaw)
	if err != nil {
		// 无订阅行：按 free 档执行（安全缺省，防"无记录 = 无限量"）
		if err := tx.QueryRow(ctx,
			"SELECT quotas FROM editions WHERE code = 'free'",
		).Scan(&quotasRaw); err != nil {
			return fmt.Errorf("quota: load free edition: %w", err)
		}
	}

	var quotas map[string]int64
	if err := json.Unmarshal(quotasRaw, &quotas); err != nil {
		return fmt.Errorf("quota: parse quotas: %w", err)
	}
	limit, ok := quotas[kind]
	if !ok || limit <= 0 {
		return nil
	}

	var current int64
	if err := tx.QueryRow(ctx, spec.countSQL, excludeID).Scan(&current); err != nil {
		return fmt.Errorf("quota: count: %w", err)
	}
	if current >= limit {
		return echo.NewHTTPError(http.StatusForbidden,
			fmt.Sprintf("当前套餐%s数已达上限（%d），请联系升级套餐", spec.label, limit))
	}
	return nil
}
