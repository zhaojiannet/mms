package notifications

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v5"

	mw "github.com/zhaojiannet/mms/backend/internal/platform/middleware"
	"github.com/zhaojiannet/mms/backend/sqlc"
)

// Item 统一通知条目，前端铃铛列表渲染
type Item struct {
	ID         string    `json:"id"`          // announcement 用 version、member 用 uuid、appt 用 uuid
	Type       string    `json:"type"`        // birthday | appointment | announcement
	Title      string    `json:"title"`
	Summary    string    `json:"summary"`
	Link       string    `json:"link,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
	Read       bool      `json:"read"` // 业务通知恒为 false；公告按用户已读状态
}

// ListResp 铃铛 GET /api/notifications 返回结构
type ListResp struct {
	Items       []Item `json:"items"`
	UnreadCount int    `json:"unread_count"` // 所有未读条（业务 + 公告未读）
}

// List GET /api/notifications
// 合并 3 类：今日生日会员 / 今日待确认预约 / 未读系统公告
// 业务通知走 TenantTx → RLS 自动按租户过滤；公告跨租户
func List(c *echo.Context) error {
	ctx := c.Request().Context()
	claims := mw.ClaimsFrom(c)
	q := sqlc.New(mw.TxFrom(c))

	items := make([]Item, 0, 16)
	now := time.Now()

	// 1. 今日生日：直接用 ReportTodayBirthdays，避免拉 15 天窗口再前端筛
	if births, err := q.ReportTodayBirthdays(ctx); err == nil {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		for _, m := range births {
			items = append(items, Item{
				ID:         m.ID.String(),
				Type:       "birthday",
				Title:      "会员生日",
				Summary:    fmt.Sprintf("今天是 %s 的生日", m.Name),
				Link:       "/members/" + m.ID.String(),
				OccurredAt: dayStart,
				Read:       false,
			})
		}
	}

	// 2. 今日待确认预约（status=pending，今日 00:00 ~ 明日 00:00）
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	statusPending := "pending"
	if appts, err := q.ListAppointments(ctx, sqlc.ListAppointmentsParams{
		StartDate: pgtype.Timestamptz{Time: dayStart, Valid: true},
		EndDate:   pgtype.Timestamptz{Time: dayEnd, Valid: true},
		Status:    &statusPending,
	}); err == nil {
		for _, a := range appts {
			name := "顾客"
			if a.CustomerName != "" {
				name = a.CustomerName
			}
			items = append(items, Item{
				ID:         a.ID.String(),
				Type:       "appointment",
				Title:      "预约待确认",
				Summary:    fmt.Sprintf("%s 于 %s 预约", name, a.AppointmentTime.Time.Format("15:04")),
				Link:       "/appointments",
				OccurredAt: a.AppointmentTime.Time,
				Read:       false,
			})
		}
	}

	// 3. 未读系统公告
	if announcements, err := q.ListUnreadAnnouncementsForUser(ctx, claims.UserID); err == nil {
		for _, a := range announcements {
			items = append(items, Item{
				ID:         a.Version,
				Type:       "announcement",
				Title:      a.Title,
				Summary:    a.Summary,
				Link:       "/changelog#v" + a.Version,
				OccurredAt: a.PublishedAt.Time,
				Read:       false,
			})
		}
	}

	// 按时间倒序
	sort.Slice(items, func(i, j int) bool {
		return items[i].OccurredAt.After(items[j].OccurredAt)
	})

	unread := 0
	for _, it := range items {
		if !it.Read {
			unread++
		}
	}

	return c.JSON(http.StatusOK, ListResp{Items: items, UnreadCount: unread})
}

// MarkRead POST /api/notifications/:version/read
// 只对 announcement 类型有效（业务通知是实时计算不入库）
func MarkRead(c *echo.Context) error {
	version := c.Param("version")
	if version == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "missing version")
	}
	claims := mw.ClaimsFrom(c)
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	if err := q.MarkAnnouncementReadByVersion(c.Request().Context(),
		sqlc.MarkAnnouncementReadByVersionParams{
			TenantID: t.ID,
			UserID:   claims.UserID,
			Version:  version,
		}); err != nil {
		return mw.InternalError(c, "mark read: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// MarkAllRead POST /api/notifications/read-all
// 把所有公告标为已读（业务通知自然消失无需操作）
func MarkAllRead(c *echo.Context) error {
	claims := mw.ClaimsFrom(c)
	t := mw.TenantFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	if err := q.MarkAllAnnouncementsRead(c.Request().Context(), sqlc.MarkAllAnnouncementsReadParams{
		TenantID: t.ID,
		UserID:   claims.UserID,
	}); err != nil {
		return mw.InternalError(c, "mark all: ", err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ListChangelog GET /api/changelog
// 完整公告列表（按时间倒序）+ 当前用户的已读状态，供 /changelog 页面渲染
func ListChangelog(c *echo.Context) error {
	claims := mw.ClaimsFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	rows, err := q.ListAllAnnouncementsWithReadStatus(c.Request().Context(), claims.UserID)
	if err != nil {
		return mw.InternalError(c, "list: ", err)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": rows})
}

// LatestVersion GET /api/notifications/latest-version
// 侧边栏版本号 + 红点：返回最新公告版本与当前用户是否已读
type LatestVersionResp struct {
	Version string `json:"version"`
	IsRead  bool   `json:"is_read"`
}

func LatestVersion(c *echo.Context) error {
	claims := mw.ClaimsFrom(c)
	q := sqlc.New(mw.TxFrom(c))
	row, err := q.GetLatestAnnouncementVersion(c.Request().Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusOK, LatestVersionResp{})
		}
		return mw.InternalError(c, "get version: ", err)
	}
	return c.JSON(http.StatusOK, LatestVersionResp{Version: row.Version, IsRead: row.IsRead})
}
