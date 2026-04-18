package middleware

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

// InternalError 处理服务端内部错误：
//   - 内部 slog.Error 记录完整错误 + 上下文（含 request path / tenant / actor，便于排查）
//   - 对外返回固定文案，不暴露 SQL 错误 / schema / 实现细节
//
// 对比 `echo.NewHTTPError(500, "xxx: "+err.Error())` 的问题：
//   - pgx/pgconn 错误通常带表名、约束名、字段名（"violates foreign key constraint ..."）
//   - 暴露后攻击者可侦察 schema、SQL 结构、RLS 策略名，辅助后续攻击
//
// 使用：
//
//	if err := q.XX(ctx, ...); err != nil {
//	    return mw.InternalError(c, "xx.create", err)
//	}
func InternalError(c *echo.Context, label string, err error) error {
	if err == nil {
		return nil
	}
	attrs := []any{"err", err.Error(), "label", label, "path", c.Request().URL.Path}
	if v := c.Get(string(CtxKeyTenant)); v != nil {
		if t, ok := v.(Tenant); ok {
			attrs = append(attrs, "tenant", t.Slug)
		}
	}
	if v := c.Get(string(CtxKeyClaims)); v != nil {
		// 避免 import cycle：不具体取 Claims 类型，仅记录存在
		_ = v
	}
	slog.Error("handler internal error", attrs...)
	return echo.NewHTTPError(http.StatusInternalServerError, "系统繁忙，请稍后再试")
}
