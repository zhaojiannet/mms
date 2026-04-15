package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/netip"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// AuditLog 记录所有 /api/* 的写操作（POST/PUT/PATCH/DELETE）到 audit_logs 表
//
// 说明：
//   - 只记录修改类请求（跳过 GET），跳过 /api/login /api/captcha 等公开端点
//   - payload 只存前 4KB 以避免日志膨胀
//   - 异步写入独立 tx（不占用业务事务），写入失败不阻塞业务
//   - audit_logs 表无 RLS（全局表），用独立 pool 连接写
func AuditLog(pool *pgxpool.Pool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()

			// 只审计修改类请求
			if req.Method == "GET" || req.Method == "OPTIONS" || req.Method == "HEAD" {
				return next(c)
			}
			// 跳过登录/验证码等公开端点
			path := req.URL.Path
			if strings.HasSuffix(path, "/login") || strings.HasSuffix(path, "/captcha") ||
				strings.HasSuffix(path, "/refresh") || strings.HasSuffix(path, "/logout") {
				return next(c)
			}

			// 读 body 存后回填，否则 handler 读不到
			var bodySnapshot []byte
			if req.Body != nil {
				bodySnapshot, _ = io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewBuffer(bodySnapshot))
			}

			err := next(c)

			// 异步写（不阻塞响应）
			go writeAudit(pool, c, bodySnapshot, err)

			return err
		}
	}
}

func writeAudit(pool *pgxpool.Pool, c *echo.Context, body []byte, handlerErr error) {
	defer func() { _ = recover() }() // 审计失败不影响业务

	ctx := c.Request().Context()
	var tenantID, actorID pgtype.UUID
	actorType := "anonymous"

	if v := c.Get(string(CtxKeyTenant)); v != nil {
		if t, ok := v.(Tenant); ok {
			tenantID = pgtype.UUID{Bytes: t.ID, Valid: true}
		}
	}
	if v := c.Get(string(CtxKeyClaims)); v != nil {
		if cl, ok := v.(*auth.Claims); ok {
			actorID = pgtype.UUID{Bytes: cl.UserID, Valid: true}
			actorType = "user"
		}
	}

	action := c.Request().Method + " " + c.Request().URL.Path
	outcome := "ok"
	if handlerErr != nil {
		outcome = "error"
	}

	// payload 截到 4KB
	const maxPayload = 4096
	if len(body) > maxPayload {
		body = body[:maxPayload]
	}
	payload := map[string]any{
		"outcome": outcome,
	}
	if handlerErr != nil {
		payload["err"] = handlerErr.Error()
	}
	if len(body) > 0 {
		// 尝试解析 JSON；失败就存原串
		var raw any
		if err := json.Unmarshal(body, &raw); err == nil {
			payload["body"] = raw
		} else {
			payload["raw"] = string(body)
		}
	}
	payloadJSON, _ := json.Marshal(payload)

	var ipAddr *netip.Addr
	if s := c.Request().RemoteAddr; s != "" {
		if i := strings.LastIndex(s, ":"); i > 0 {
			s = s[:i]
		}
		if a, err := netip.ParseAddr(s); err == nil {
			ipAddr = &a
		}
	}
	ua := c.Request().Header.Get("User-Agent")

	q := sqlc.New(pool)
	_ = q.CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		TenantID:  tenantID,
		ActorID:   actorID,
		ActorType: actorType,
		Action:    action,
		Payload:   payloadJSON,
		IpAddress: ipAddr,
		UserAgent: &ua,
	})
}
