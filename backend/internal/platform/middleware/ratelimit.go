package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
)

// trustedProxies 环境变量 TRUSTED_PROXIES（逗号分隔 CIDR 列表）
// 只有 RemoteAddr 属于这些 CIDR 的请求才允许读 X-Forwarded-For / X-Real-IP
// 默认空 → 不信任任何代理（本地开发 / 未反代直连）
var trustedProxies = loadTrustedProxies()

func loadTrustedProxies() []netip.Prefix {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return nil
	}
	var out []netip.Prefix
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			// 单 IP 兼容：转成 /32 或 /128
			if addr, err2 := netip.ParseAddr(s); err2 == nil {
				bits := 32
				if addr.Is6() {
					bits = 128
				}
				p = netip.PrefixFrom(addr, bits)
			} else {
				continue
			}
		}
		out = append(out, p)
	}
	return out
}

func isTrustedProxy(remoteAddr string) bool {
	if len(trustedProxies) == 0 {
		return false
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	for _, p := range trustedProxies {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

// RateLimit 内存令牌桶：per-IP 在 window 时间窗口内最多允许 max 次请求
//
// 说明：
//   - 单实例部署下足够抵挡暴力穷举；多实例要换成 Redis/Valkey
//   - 维护一个简单 map[ip]*ipState，sync.Mutex 保护；空闲 > 10*window 的 key 由后台 GC 清理
//   - 键基于 client IP（优先 X-Forwarded-For 首段），不关心路径；若想 per-user 细分请在调用端先登录解析
//   - 超限返回 429 + Retry-After
func RateLimit(max int, window time.Duration) echo.MiddlewareFunc {
	buckets := &rateBuckets{
		data:   make(map[string]*rateState),
		window: window,
		max:    max,
	}
	go buckets.gcLoop()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ip := ClientIP(c.Request())
			if !buckets.allow(ip) {
				c.Response().Header().Set("Retry-After", strSeconds(window))
				return echo.NewHTTPError(http.StatusTooManyRequests, "请求过于频繁，请稍后再试")
			}
			return next(c)
		}
	}
}

// RateLimitStaffOnly 仅对 role=staff 用户做 per-user 限流（admin/super_admin 不受限）
//   - 必须在 RequireAuth 之后挂载（依赖 claims）
//   - 键使用 user_id（比 IP 更精准，防止员工换浏览器绕过）
func RateLimitStaffOnly(max int, window time.Duration) echo.MiddlewareFunc {
	buckets := &rateBuckets{
		data:   make(map[string]*rateState),
		window: window,
		max:    max,
	}
	go buckets.gcLoop()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			claims := ClaimsFrom(c)
			if claims.Role != RoleStaff {
				return next(c)
			}
			key := "user:" + claims.UserID.String()
			if !buckets.allow(key) {
				c.Response().Header().Set("Retry-After", strSeconds(window))
				return echo.NewHTTPError(http.StatusTooManyRequests, "查询过于频繁，请稍后再试")
			}
			return next(c)
		}
	}
}

type rateState struct {
	count    int
	windowAt time.Time
	lastAt   time.Time
}

type rateBuckets struct {
	mu     sync.Mutex
	data   map[string]*rateState
	window time.Duration
	max    int
}

func (b *rateBuckets) allow(ip string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	s, ok := b.data[ip]
	if !ok || now.Sub(s.windowAt) >= b.window {
		b.data[ip] = &rateState{count: 1, windowAt: now, lastAt: now}
		return true
	}
	s.lastAt = now
	if s.count >= b.max {
		return false
	}
	s.count++
	return true
}

func (b *rateBuckets) gcLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		b.mu.Lock()
		cutoff := time.Now().Add(-10 * b.window)
		for k, v := range b.data {
			if v.lastAt.Before(cutoff) {
				delete(b.data, k)
			}
		}
		b.mu.Unlock()
	}
}

// ClientIP 返回客户端真实 IP
//
// 信任链：只有当 RemoteAddr 属于 TRUSTED_PROXIES 时，才读 XFF/X-Real-IP
// 否则直接用 RemoteAddr（防伪造：攻击者发 X-Forwarded-For: 1.2.3.4 来绕过限流）
// 公开导出供其他模块（如 booking）复用同一套信任策略
func ClientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}
	if isTrustedProxy(r.RemoteAddr) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// XFF 格式：client, proxy1, proxy2 —— 取最左（最初的客户端）
			if i := strings.Index(xff, ","); i > 0 {
				return strings.TrimSpace(xff[:i])
			}
			return strings.TrimSpace(xff)
		}
		if rip := r.Header.Get("X-Real-IP"); rip != "" {
			return strings.TrimSpace(rip)
		}
	}
	return remoteHost
}

func strSeconds(d time.Duration) string {
	s := max(int(d.Seconds()), 1)
	return itoa(s)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
