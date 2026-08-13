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

// 审计写入用 worker pool 模式：
//   - 固定 N 个 worker 从 channel 消费 job
//   - DB 连接占用上限 = N（不再随请求并发爆）
//   - 队列满时中间件 block，直到有 worker 空闲；不 drop（审计是合规要求）
//   - 关闭：StartAuditWorkers 后调 AuditWait → close(channel) + WG.Wait
const (
	auditWorkerCount = 4
	auditQueueSize   = 1024
)

var (
	auditWG    sync.WaitGroup
	auditQueue chan auditJob // 由 StartAuditWorkers 初始化；nil 时 fallback 启动单 goroutine 模式
)

// StartAuditWorkers 启动后台 worker pool。在 main.go 注册路由前调用。
func StartAuditWorkers(pool *pgxpool.Pool) {
	auditQueue = make(chan auditJob, auditQueueSize)
	for i := 0; i < auditWorkerCount; i++ {
		auditWG.Add(1)
		go func() {
			defer auditWG.Done()
			for job := range auditQueue {
				writeAuditJob(pool, job)
			}
		}()
	}
}

// AuditWait 由 main.go 在关闭前调用：close(queue) → workers drain → WG.Wait → 超时兜底
func AuditWait(timeout time.Duration) {
	if auditQueue != nil {
		close(auditQueue)
	}
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
//	- 异步写入：worker pool 消费 auditQueue，不阻塞 handler 响应
//	- 队列满时中间件 block：审计是合规要求，宁可让请求慢一点也不丢日志
//	- 独立 context + 5s 超时：原 req ctx 返回后会 cancel
//	- audit_logs 表 RLS：worker 内部 SET LOCAL app.current_tenant 后再 INSERT
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
				if data, err := io.ReadAll(req.Body); err != nil {
					slog.Warn("audit read body failed", "err", err, "path", path)
				} else {
					bodySnapshot = data
				}
				req.Body = io.NopCloser(bytes.NewBuffer(bodySnapshot))
			}

			err := next(c)

			// next 后才能拿到 claims（RequireAuth 在下游中间件设置）
			if !shouldRecord && !shouldAuditStaffRead(c, method, path) {
				return err
			}

			// 必须在入队前从 Context 提取所有值（Echo handler 返回后 Context 被 Reset）
			job := buildAuditJob(c, method, path, bodySnapshot, err)
			if auditQueue != nil {
				auditQueue <- job // 满了 block；不丢审计
			} else {
				// fallback：未调 StartAuditWorkers（如测试场景），同步写
				writeAuditJob(pool, job)
			}
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
	// 挂账定价预览是只读查询（POST 仅为带 body），购物车每次变动都调，
	// 全量入审计会用键击级噪音淹没 append-only 审计表里的真实写操作
	if strings.HasSuffix(path, "/transactions/pending-preview") {
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
	if data, err := json.Marshal(payload); err != nil {
		slog.Warn("audit marshal payload failed", "err", err, "action", job.action)
	} else {
		job.payload = data
	}

	// 与限流同一条信任链：直取 RemoteAddr 在容器部署下记到的是 Docker 网桥地址
	if s := ClientIP(c.Request()); s != "" {
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
