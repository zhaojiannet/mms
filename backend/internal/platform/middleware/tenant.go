package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/sqlc"
)

// allowSlugHeaderFallback：是否允许 X-Tenant-Slug header 作为 Host 解析失败后的兜底
// 仅 self-hosted / dev 开发方便；hosted 和 enterprise 模式禁用，防运维漏配反代
var allowSlugHeaderFallback = func() bool {
	mode := os.Getenv("DEPLOYMENT_MODE")
	return mode == "" || mode == "self-hosted" || mode == "dev"
}()

// Context keys（不直接导出字符串，避免跨包字面量冲突）
type ctxKey string

const (
	CtxKeyTenant ctxKey = "tenant"
	CtxKeyTx     ctxKey = "tx"
)

// Tenant 是 tenant resolver 注入到 echo.Context 的租户摘要
type Tenant struct {
	ID     uuid.UUID
	Slug   string
	Status string
	Name   string
}

// reservedSubdomains 是不当作 tenant slug 的保留子域
var reservedSubdomains = map[string]struct{}{
	"":          {},
	"www":       {},
	"vip":       {},
	"dev":       {},
	"api":       {},
	"admin":     {},
	"mail":      {},
	"localhost": {},
	"127":       {},
}

// TenantResolver 从 Host 头（或 X-Tenant-Slug）解析 slug，查 tenants 公开目录，注入 Tenant 到 context
//   - 进程内 LRU+TTL 缓存减少数据库压力；5s TTL 权衡"status 变更响应速度"与"回表压力"
//     （500 商户 × 5s TTL ≈ 100 QPS 回表峰值，PG 轻松扛）
//   - 非 active 状态的 tenant 直接 403
//   - tenants 表不启用 RLS（公开目录设计，参考 Supabase/PostgREST），此查询无需 SET app.current_tenant
func TenantResolver(pool *pgxpool.Pool) echo.MiddlewareFunc {
	cache := newTenantCache(5 * time.Second)
	queries := sqlc.New(pool)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			slug := extractSlug(c.Request())
			if slug == "" {
				return echo.NewHTTPError(http.StatusBadRequest, "missing tenant slug (Host subdomain or X-Tenant-Slug header)")
			}

			if t, ok := cache.get(slug); ok {
				c.Set(string(CtxKeyTenant), t)
				return next(c)
			}

			row, err := queries.GetTenantBySlug(c.Request().Context(), slug)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
				}
				// DB 原始错误只进日志，不回传客户端（可能含连接串/表结构细节）
				slog.Error("tenant lookup failed", "slug", slug, "err", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "tenant lookup failed")
			}
			if row.Status != "active" {
				return echo.NewHTTPError(http.StatusForbidden, "tenant is "+row.Status)
			}

			t := Tenant{ID: row.ID, Slug: row.Slug, Status: row.Status, Name: row.Name}
			cache.set(slug, t)
			c.Set(string(CtxKeyTenant), t)
			return next(c)
		}
	}
}

// TenantFrom 从 echo.Context 取 Tenant，未设置时 panic（middleware 顺序错误）
func TenantFrom(c *echo.Context) Tenant {
	v := c.Get(string(CtxKeyTenant))
	t, ok := v.(Tenant)
	if !ok {
		panic("tenant context missing; ensure TenantResolver middleware ran before")
	}
	return t
}

// IsReservedSubdomain 判断 slug 是否为保留子域（signup 校验期望子域时用）
func IsReservedSubdomain(s string) bool {
	_, ok := reservedSubdomains[strings.ToLower(s)]
	return ok
}

// extractSlug 解析 Host 子域名作为 tenant slug
//   - 生产：<slug>.<app_public_domain> → slug
//   - 开发/自建 fallback：X-Tenant-Slug 请求头；hosted/enterprise 禁用以防反代漏配
func extractSlug(r *http.Request) string {
	host := r.Host
	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "."); idx > 0 {
		candidate := strings.ToLower(host[:idx])
		if _, reserved := reservedSubdomains[candidate]; !reserved {
			return candidate
		}
	}
	if allowSlugHeaderFallback {
		return r.Header.Get("X-Tenant-Slug")
	}
	return ""
}

// ---- 进程内 LRU+TTL 缓存（最简实现，500 商户容量足够） ----

type tenantCache struct {
	mu    sync.RWMutex
	items map[string]cachedTenant
	ttl   time.Duration
}

type cachedTenant struct {
	tenant    Tenant
	expiresAt time.Time
}

func newTenantCache(ttl time.Duration) *tenantCache {
	return &tenantCache{
		items: make(map[string]cachedTenant),
		ttl:   ttl,
	}
}

func (c *tenantCache) get(slug string) (Tenant, bool) {
	c.mu.RLock()
	it, ok := c.items[slug]
	c.mu.RUnlock()
	if !ok || time.Now().After(it.expiresAt) {
		return Tenant{}, false
	}
	return it.tenant, true
}

func (c *tenantCache) set(slug string, t Tenant) {
	c.mu.Lock()
	c.items[slug] = cachedTenant{tenant: t, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

