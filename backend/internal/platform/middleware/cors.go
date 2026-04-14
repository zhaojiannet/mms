package middleware

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// CORS 最简 CORS 中间件（开发用）
//   - 回显 Origin + credentials，支持任何携带 token 的前端开发来源
//   - 响应 OPTIONS 预检请求
//   - 生产环境应由反代（OpenResty / Cloudflare）统一策略，后端可关闭此中间件
func CORS() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			origin := c.Request().Header.Get("Origin")
			if origin == "" {
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
