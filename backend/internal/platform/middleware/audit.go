package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v5"

	"github.com/zhaojiannet/mms/backend/internal/platform/auth"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// auditWG 进程级 WaitGroup：main 优雅关闭时调用 AuditWait，等所有在途 goroutine 写完再退出
// 避免 SIGTERM 瞬间丢失审计日志（尤其是"谁触发了停机"这种关键信息）
var auditWG sync.WaitGroup

// AuditWait 由 main.go 在关闭前调用；带超时防挂死
// 超时：后台 goroutine 本身已有 5s DB 超时，此处留 8s 总兜底
func AuditWait(timeout time.Duration) {
	done := make(chan struct{})
	go func() {
		auditWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(timeout):
		slog.Warn("audit drain timed out; some records may be lost")
	}
}

// AuditLog 记录 /api/* 的写操作及 staff 的敏感读到 audit_logs 表
//
// 关键安全约束（禁止在 goroutine 内直接访问 *echo.Context）：
//
//	Echo v5 的 Context 实例由 sync.Pool 管理，handler 返回后会 Reset 并放回池，
//	下一个并发请求很可能拿到同一个实例。若 goroutine 仍持有 c 指针，
//	会读到下一个请求的 tenant/claims/header，审计日志张冠李戴。
//	→ 所有 context 依赖值必须在 goroutine 启动前提取到 auditJob 值类型里传递。
//
//	- 异步写入：不阻塞 handler 响应
//	- 独立 context + 5s 超时：原 req ctx 返回后会 cancel
//	- audit_logs 表无 RLS（全局表），用独立 pool 连接写
func AuditLog(pool *pgxpool.Pool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			method := req.Method
			path := req.URL.Path

			// 判断是否需要记录
			shouldRecord, recordBody := shouldAuditRequest(method, path)

			var bodySnapshot []byte
			if recordBody && req.Body != nil {
				bodySnapshot, _ = io.ReadAll(req.Body)
				req.Body = io.NopCloser(bytes.NewBuffer(bodySnapshot))
			}

			err := next(c)

			// next 后才能拿到 claims（RequireAuth 在下游中间件设置）
			if !shouldRecord && !shouldAuditStaffRead(c, method, path) {
				return err
			}

			// 必须在 goroutine 启动前从 Context 提取所有值
			// 此时 c 还属于当前请求，Echo 尚未 Reset
			job := buildAuditJob(c, method, path, bodySnapshot, err)
			auditWG.Add(1)
			go func() {
				defer auditWG.Done()
				writeAuditJob(pool, job)
			}()
			return err
		}
	}
}

// shouldAuditRequest 判断非 GET 请求是否需要审计（通用规则）
func shouldAuditRequest(method, path string) (record, readBody bool) {
	if method == "GET" || method == "OPTIONS" || method == "HEAD" {
		return false, false
	}
	if strings.HasSuffix(path, "/login") || strings.HasSuffix(path, "/captcha") ||
		strings.HasSuffix(path, "/refresh") || strings.HasSuffix(path, "/logout") {
		return false, false
	}
	return true, true
}

// shouldAuditStaffRead staff 的"敏感读"也记录：防止悄悄批量看会员
var staffReadAuditPrefixes = []string{
	"/api/members",
	"/api/transactions",
}

func shouldAuditStaffRead(c *echo.Context, method, path string) bool {
	if method != "GET" {
		return false
	}
	v := c.Get(string(CtxKeyClaims))
	if v == nil {
		return false
	}
	cl, ok := v.(*auth.Claims)
	if !ok || cl.Role != RoleStaff {
		return false
	}
	for _, p := range staffReadAuditPrefixes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

// auditJob 审计日志任务的值类型：所有字段均为值拷贝/独立分配，goroutine 安全
type auditJob struct {
	tenantID  pgtype.UUID
	actorID   pgtype.UUID
	actorType string
	action    string
	payload   []byte
	ipAddr    *netip.Addr
	userAgent string
}

// buildAuditJob 必须在主 goroutine 内（Echo 还没 Reset 前）调用
func buildAuditJob(c *echo.Context, method, path string, body []byte, handlerErr error) auditJob {
	job := auditJob{
		actorType: "anonymous",
		action:    method + " " + path,
	}

	if v := c.Get(string(CtxKeyTenant)); v != nil {
		if t, ok := v.(Tenant); ok {
			job.tenantID = pgtype.UUID{Bytes: t.ID, Valid: true}
		}
	}
	if v := c.Get(string(CtxKeyClaims)); v != nil {
		if cl, ok := v.(*auth.Claims); ok {
			job.actorID = pgtype.UUID{Bytes: cl.UserID, Valid: true}
			job.actorType = "user"
		}
	}

	// payload 构建 + 截断
	const maxPayload = 4096
	if len(body) > maxPayload {
		body = body[:maxPayload]
	}
	outcome := "ok"
	if handlerErr != nil {
		outcome = "error"
	}
	payload := map[string]any{"outcome": outcome}
	if handlerErr != nil {
		payload["err"] = handlerErr.Error()
	}
	if len(body) > 0 {
		var raw any
		if err := json.Unmarshal(body, &raw); err == nil {
			payload["body"] = redactSensitive(raw)
		} else {
			payload["raw"] = string(body)
		}
	}
	job.payload, _ = json.Marshal(payload)

	// IP（从 RemoteAddr 提取；X-Forwarded-For 处理见 clientIP()）
	if s := c.Request().RemoteAddr; s != "" {
		if i := strings.LastIndex(s, ":"); i > 0 {
			s = s[:i]
		}
		if a, err := netip.ParseAddr(s); err == nil {
			job.ipAddr = &a
		}
	}
	// UA：string 在 Go 是值类型，赋值即拷贝
	job.userAgent = c.Request().Header.Get("User-Agent")

	return job
}

func writeAuditJob(pool *pgxpool.Pool, job auditJob) {
	defer func() { _ = recover() }() // 审计失败不影响业务

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// audit_logs 启用了 RLS，必须在独立 tx 里 SET LOCAL tenant 才能 INSERT
	// tenant_id 为 NULL（匿名/公开端点）时也允许（policy: tenant_id IS NULL OR = current_tenant_id()）
	tx, err := pool.Begin(ctx)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if job.tenantID.Valid {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)",
			uuidToString(job.tenantID)); err != nil {
			return
		}
	}

	ua := job.userAgent
	if err := sqlc.New(tx).CreateAuditLog(ctx, sqlc.CreateAuditLogParams{
		TenantID:  job.tenantID,
		ActorID:   job.actorID,
		ActorType: job.actorType,
		Action:    job.action,
		Payload:   job.payload,
		IpAddress: job.ipAddr,
		UserAgent: &ua,
	}); err != nil {
		return
	}
	_ = tx.Commit(ctx)
}

// uuidToString pgtype.UUID 转字符串（for SET LOCAL）
func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// sensitiveKeys 在审计日志 payload 中必须被屏蔽的字段名（小写匹配）
var sensitiveKeys = map[string]bool{
	"password":         true,
	"old_password":     true,
	"new_password":     true,
	"current_password": true,
	"super_password":   true,
	"token":            true,
	"access_token":     true,
	"refresh_token":    true,
	"authorization":    true,
	"secret":           true,
	"api_key":          true,
	"apikey":           true,
	"webhook_secret":   true,
	"otp":              true,
	"pin":              true,
	"session_id":       true,
	"sessionid":        true,
}

// redactSensitive 递归遍历 JSON 结构，把敏感字段的值替换为 "***"
func redactSensitive(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(vv))
		for k, val := range vv {
			if sensitiveKeys[strings.ToLower(k)] {
				out[k] = "***"
			} else {
				out[k] = redactSensitive(val)
			}
		}
		return out
	case []any:
		for i := range vv {
			vv[i] = redactSensitive(vv[i])
		}
		return vv
	default:
		return v
	}
}
