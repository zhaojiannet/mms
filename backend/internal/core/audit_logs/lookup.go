package auditlogs

// 批量 ID→名字解析端点
// 给操作日志对象列人名化用（members/transactions/appointments 量大，不适合 list 全量）
//
// 请求：GET /api/audit-logs/lookup?members=a,b,c&transactions=x,y&appointments=m
// 响应：{ members: {id: name}, transactions: {id: name}, appointments: {id: customer_name} }
//
// 权限：adminAndAbove（仅审计页会调用）
// RLS：SELECT 走当前 tenant tx，自动过滤跨租户

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type LookupResponse struct {
	Members      map[string]string `json:"members"`
	Transactions map[string]string `json:"transactions"`
	Appointments map[string]string `json:"appointments"`
}

// lookupMaxIDs 每组 UUID 列表的硬上限（防大数组 DoS / 慢查询）
const lookupMaxIDs = 200

// parseIDs 把 "a,b,c" 解析成 []uuid.UUID；忽略格式错误的；超出 lookupMaxIDs 直接截断
func parseIDs(s string) []uuid.UUID {
	out := make([]uuid.UUID, 0)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if u, err := uuid.Parse(part); err == nil {
			out = append(out, u)
			if len(out) >= lookupMaxIDs {
				break
			}
		}
	}
	return out
}

// Lookup GET /api/audit-logs/lookup
func Lookup(c *echo.Context) error {
	q := sqlc.New(mw.TxFrom(c))
	ctx := c.Request().Context()

	memberIDs := parseIDs(c.QueryParam("members"))
	txIDs := parseIDs(c.QueryParam("transactions"))
	apptIDs := parseIDs(c.QueryParam("appointments"))

	resp := LookupResponse{
		Members:      map[string]string{},
		Transactions: map[string]string{},
		Appointments: map[string]string{},
	}

	if len(memberIDs) > 0 {
		rows, err := q.LookupMembersByIDs(ctx, memberIDs)
		if err != nil {
			return mw.InternalError(c, "lookup members", err)
		}
		for _, r := range rows {
			resp.Members[r.ID.String()] = r.Name
		}
	}
	if len(txIDs) > 0 {
		rows, err := q.LookupTransactionsByIDs(ctx, txIDs)
		if err != nil {
			return mw.InternalError(c, "lookup transactions", err)
		}
		for _, r := range rows {
			// "张三 · ¥128.00"
			label := r.DisplayName
			if !r.TotalAmount.IsZero() {
				label = label + " · ¥" + r.TotalAmount.String()
			}
			resp.Transactions[r.ID.String()] = label
		}
	}
	if len(apptIDs) > 0 {
		rows, err := q.LookupAppointmentsByIDs(ctx, apptIDs)
		if err != nil {
			return mw.InternalError(c, "lookup appointments", err)
		}
		for _, r := range rows {
			resp.Appointments[r.ID.String()] = r.CustomerName
		}
	}

	return c.JSON(http.StatusOK, resp)
}
