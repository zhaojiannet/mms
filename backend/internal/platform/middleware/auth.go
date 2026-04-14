package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
)

const CtxKeyClaims ctxKey = "claims"

// RequireAuth 解析 Authorization: Bearer <token>，验签并核对 tenant 一致性
//   - 必须在 TenantResolver 之后挂载（要拿 context 里的 tenant）
//   - 拒绝 token.tenant_id ≠ host.tenant_id（防止跨租户 token 串用）
func RequireAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			raw := c.Request().Header.Get("Authorization")
			if raw == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}
			parts := strings.SplitN(raw, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization scheme")
			}

			claims, err := auth.ParseAccessToken(parts[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token: "+err.Error())
			}

			tenant := TenantFrom(c)
			if claims.TenantID != tenant.ID {
				return echo.NewHTTPError(http.StatusForbidden, "token tenant does not match host")
			}

			c.Set(string(CtxKeyClaims), claims)
			return next(c)
		}
	}
}

// ClaimsFrom 取当前 JWT claims
func ClaimsFrom(c *echo.Context) *auth.Claims {
	v := c.Get(string(CtxKeyClaims))
	claims, ok := v.(*auth.Claims)
	if !ok {
		panic("claims context missing; ensure RequireAuth middleware ran before")
	}
	return claims
}
