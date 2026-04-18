package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

const CtxKeyClaims ctxKey = "claims"

// RequireAuth 解析 Authorization: Bearer <token>，验签并核对 tenant 一致性
//   - 必须在 TenantResolver 之后挂载（要拿 context 里的 tenant）
//   - 拒绝 token.tenant_id ≠ host.tenant_id（防止跨租户 token 串用）
//   - 对比 DB 的 token_version 和 password_changed_at：被盗 token 在改密/降级后立即失效
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

			// token 版本 + 密码修改时间校验：防改密/降级后旧 token 仍有效
			u, err := sqlc.New(TxFrom(c)).GetUserByID(c.Request().Context(), claims.UserID)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "账号不存在")
			}
			if u.Status != "active" {
				return echo.NewHTTPError(http.StatusForbidden, "账号已停用")
			}
			if claims.Ver != u.TokenVersion {
				return echo.NewHTTPError(http.StatusUnauthorized, "会话已失效，请重新登录")
			}
			// iat 早于 password_changed_at → token 在改密前签发，拒绝
			if claims.IssuedAt != nil && u.PasswordChangedAt.Valid &&
				claims.IssuedAt.Time.Before(u.PasswordChangedAt.Time) {
				return echo.NewHTTPError(http.StatusUnauthorized, "会话已失效，请重新登录")
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

// role 常量
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleStaff      = "staff"
)

// currentUserFromDB 从 DB 重取用户的 role 和 status
// 用于关键权限守卫：防止账号被 disable/降级后旧 JWT 仍能通过 claims 中的陈旧 role
// 依赖 RequireAuth 已挂 claims + TenantTx 已开 tx
func currentUserFromDB(c *echo.Context) (sqlc.User, error) {
	claims := ClaimsFrom(c)
	q := sqlc.New(TxFrom(c))
	return q.GetUserByID(c.Request().Context(), claims.UserID)
}

// RequireSuperAdmin 仅 super_admin 可通过
//   - 必须在 RequireAuth 之后挂载（依赖 claims + tx）
//   - 从 DB 重取 role/status：账号被 disable 或降级后立即生效，无需等 JWT 过期
func RequireSuperAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			u, err := currentUserFromDB(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "账号不存在")
			}
			if u.Status != "active" {
				return echo.NewHTTPError(http.StatusForbidden, "账号已停用")
			}
			if u.Role != RoleSuperAdmin {
				return echo.NewHTTPError(http.StatusForbidden, "需要超级管理员权限")
			}
			return next(c)
		}
	}
}

// RequireAtLeastAdmin super_admin 或 admin 可通过（拒绝 staff）
//   - 从 DB 重取 role/status，语义同 RequireSuperAdmin
func RequireAtLeastAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			u, err := currentUserFromDB(c)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "账号不存在")
			}
			if u.Status != "active" {
				return echo.NewHTTPError(http.StatusForbidden, "账号已停用")
			}
			if u.Role != RoleSuperAdmin && u.Role != RoleAdmin {
				return echo.NewHTTPError(http.StatusForbidden, "需要管理员及以上权限")
			}
			return next(c)
		}
	}
}
