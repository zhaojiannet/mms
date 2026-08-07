package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
)

const CtxKeyOperator ctxKey = "operator"

// Operator 是 RequireOperator 注入 context 的平台操作员摘要
type Operator struct {
	ID    uuid.UUID
	Email string
	Name  string
}

// RequirePlatformHost 平台路由只在 admin 子域可达（本地回环放行便于开发与运维排查）
// 商户子域的反代同样把 /api/ 转到本后端，不加 Host 限制则运营后台入口在每个商户域名暴露
func RequirePlatformHost() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			host := c.Request().Host
			if i := strings.Index(host, ":"); i >= 0 {
				host = host[:i]
			}
			if strings.HasPrefix(host, "admin.") || host == "localhost" || host == "127.0.0.1" {
				return next(c)
			}
			// 404 而非 403：不向商户域名的探测者确认平台入口存在
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
	}
}

// PlatformTx 平台请求事务：SET LOCAL app.platform_op 哨兵，放行跨租户读策略
//   - 与 TenantTx 互斥使用（平台路由不经过 TenantResolver）
//   - 单租户写操作由 handler 自行再 SET LOCAL app.current_tenant，走原隔离策略
//   - 复用 CtxKeyTx，下游 TxFrom 通用
func PlatformTx(pool *pgxpool.Pool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx := c.Request().Context()
			tx, err := pool.Begin(ctx)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "begin tx: "+err.Error())
			}
			if _, err := tx.Exec(ctx,
				"SELECT set_config('app.platform_op', 'on', true)",
			); err != nil {
				_ = tx.Rollback(ctx)
				return echo.NewHTTPError(http.StatusInternalServerError, "set platform ctx: "+err.Error())
			}
			c.Set(string(CtxKeyTx), tx)

			panicked := true
			defer func() {
				if panicked {
					_ = tx.Rollback(ctx)
				}
			}()
			if err := next(c); err != nil {
				_ = tx.Rollback(ctx)
				panicked = false
				return err
			}
			panicked = false
			return tx.Commit(ctx)
		}
	}
}

// RequireOperator 解析平台操作员 token 并核对 DB 行（状态 / token 版本 / 改密时间）
// 必须挂在 PlatformTx 之后
func RequireOperator() echo.MiddlewareFunc {
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
			claims, err := auth.ParseOperatorToken(parts[1])
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token: "+err.Error())
			}

			var (
				op                Operator
				status            string
				ver               int32
				passwordChangedAt time.Time
			)
			err = TxFrom(c).QueryRow(c.Request().Context(), `
				SELECT id, email, name, status, token_version, password_changed_at
				FROM platform_operators WHERE id = $1
			`, claims.OperatorID).Scan(&op.ID, &op.Email, &op.Name, &status, &ver, &passwordChangedAt)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "账号不存在")
			}
			if status != "active" {
				return echo.NewHTTPError(http.StatusForbidden, "账号已停用")
			}
			if claims.Ver != ver {
				return echo.NewHTTPError(http.StatusUnauthorized, "会话已失效，请重新登录")
			}
			if claims.IssuedAt != nil && claims.IssuedAt.Time.Before(passwordChangedAt) {
				return echo.NewHTTPError(http.StatusUnauthorized, "会话已失效，请重新登录")
			}

			c.Set(string(CtxKeyOperator), op)
			return next(c)
		}
	}
}

// OperatorFrom 取当前平台操作员
func OperatorFrom(c *echo.Context) Operator {
	v := c.Get(string(CtxKeyOperator))
	op, ok := v.(Operator)
	if !ok {
		panic("operator context missing; ensure RequireOperator middleware ran before")
	}
	return op
}
