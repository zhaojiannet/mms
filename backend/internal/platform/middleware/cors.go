package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v5"
)

// CORS 中间件：白名单 origin + credentials
//   - 允许来源通过环境变量 CORS_ALLOWED_ORIGINS 配置（逗号分隔）
//   - 未配置时默认放行 http://localhost:3001 和 http://localhost:3000（开发便利）
//   - 响应 OPTIONS 预检
//   - 生产环境建议在反代层统一 CORS，后端可关闭此中间件
func CORS() echo.MiddlewareFunc {
	allowed := parseAllowedOrigins()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			if origin == "" {
				return next(c)
			}
			if !isAllowed(origin, allowed) {
				// 不在白名单：不设 Allow-Origin（浏览器会拦截跨域请求）
				if c.Request().Method == http.MethodOptions {
					return c.NoContent(http.StatusForbidden)
				}
				return next(c)
			}
			h := c.Response().Header()
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Vary", "Origin")

			if c.Request().Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-Slug")
				h.Set("Access-Control-Max-Age", "600")
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}

func parseAllowedOrigins() []string {
	v := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if v == "" {
		return []string{
			"http://localhost:3001",
			"http://localhost:3000",
		}
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// isAllowed 支持精确匹配，以及 "*.example.com" 通配匹配顶级子域
func isAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == origin {
			return true
		}
		if strings.HasPrefix(a, "*.") {
			suffix := a[1:] // ".example.com"
			if strings.HasSuffix(origin, suffix) && len(origin) > len(suffix) {
				return true
			}
		}
	}
	return false
}
