package middleware

import (
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"
)

// TenantTx 为每个请求开事务并 SET LOCAL app.current_tenant
//   - 依赖 TenantResolver 先跑（从 context 取 Tenant）
//   - set_config(..., true) 的第三参数 true 表示 "只在当前事务内生效"，相当于 SET LOCAL
//   - handler 返回 nil → COMMIT；返回 error → ROLLBACK；panic → ROLLBACK + 继续 panic
//   - 业务代码从 context 取 pgx.Tx，不再访问原 pool，保证所有查询都经过 RLS
func TenantTx(pool *pgxpool.Pool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			t := TenantFrom(c)
			ctx := c.Request().Context()

			tx, err := pool.Begin(ctx)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "begin tx: "+err.Error())
			}

			if _, err := tx.Exec(ctx,
				"SELECT set_config('app.current_tenant', $1, true)",
				t.ID.String(),
			); err != nil {
				_ = tx.Rollback(ctx)
				return echo.NewHTTPError(http.StatusInternalServerError, "set tenant: "+err.Error())
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

// TxFrom 从 echo.Context 取当前事务
func TxFrom(c *echo.Context) pgx.Tx {
	v := c.Get(string(CtxKeyTx))
	tx, ok := v.(pgx.Tx)
	if !ok {
		panic("tx context missing; ensure TenantTx middleware ran before")
	}
	return tx
}
