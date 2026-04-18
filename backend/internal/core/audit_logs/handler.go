// Package auditlogs 提供审计日志查询 API
package auditlogs

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

type DTO struct {
	ID         int64     `json:"id"`
	ActorID    *string   `json:"actor_id"`
	ActorName  *string   `json:"actor_name"`
	ActorType  string    `json:"actor_type"`
	Action     string    `json:"action"`
	TargetType *string   `json:"target_type"`
	TargetID   *string   `json:"target_id"`
	Payload    any       `json:"payload"`
	IpAddress  *string   `json:"ip_address"`
	UserAgent  *string   `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

func List(c *echo.Context) error {
	q := sqlc.New(mw.TxFrom(c))
	t := mw.TenantFrom(c)

	limit := int32(50)
	if s := c.QueryParam("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 && v <= 200 {
			limit = int32(v)
		}
	}
	offset := int32(0)
	if s := c.QueryParam("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	rows, err := q.ListAuditLogs(c.Request().Context(), sqlc.ListAuditLogsParams{
		Limit:    limit,
		Offset:   offset,
		TenantID: pgtype.UUID{Bytes: t.ID, Valid: true},
	})
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}

	items := make([]DTO, 0, len(rows))
	for _, r := range rows {
		dto := DTO{
			ID:         r.ID,
			ActorName:  r.ActorName,
			ActorType:  r.ActorType,
			Action:     r.Action,
			TargetType: r.TargetType,
			TargetID:   r.TargetID,
			UserAgent:  r.UserAgent,
			CreatedAt:  r.CreatedAt.Time,
		}
		if r.ActorID.Valid {
			s := uuid.UUID(r.ActorID.Bytes).String()
			dto.ActorID = &s
		}
		if r.IpAddress != nil {
			s := r.IpAddress.String()
			dto.IpAddress = &s
		}
		dto.Payload = r.Payload
		items = append(items, dto)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}
