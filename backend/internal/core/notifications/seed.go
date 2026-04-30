// Package notifications 提供系统公告 seed、通知 API。
//
// 公告内容来源 = 本包内的 announcements.json（embed）。
// 每次发版在 PR 里追加或修改 JSON 条目，后端启动时 upsert 到 system_announcements 表。
// 用 version 作唯一键，ON CONFLICT DO UPDATE 允许修订已发公告的标题 / 摘要 / 正文。
package notifications

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zhaojiannet/mms/backend/sqlc"
)

//go:embed announcements.json
var announcementsRaw []byte

type announcementEntry struct {
	Version      string    `json:"version"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Summary      string    `json:"summary"`
	BodyMarkdown string    `json:"body_markdown"`
	PublishedAt  time.Time `json:"published_at"`
}

// SeedAnnouncements 启动时调用，幂等。
// 失败只 log warn，不中断启动（通知系统是增强而非核心依赖）。
func SeedAnnouncements(ctx context.Context, pool *pgxpool.Pool) {
	var entries []announcementEntry
	if err := json.Unmarshal(announcementsRaw, &entries); err != nil {
		slog.Warn("announcements.json parse failed", "err", err)
		return
	}

	q := sqlc.New(pool)
	for _, e := range entries {
		if e.Version == "" {
			continue
		}
		err := q.UpsertSystemAnnouncement(ctx, sqlc.UpsertSystemAnnouncementParams{
			Version:      e.Version,
			Type:         e.Type,
			Title:        e.Title,
			Summary:      e.Summary,
			BodyMarkdown: e.BodyMarkdown,
			PublishedAt:  pgtype.Timestamptz{Time: e.PublishedAt, Valid: true},
		})
		if err != nil {
			slog.Warn("seed announcement failed", "version", e.Version, "err", err)
			continue
		}
	}
	slog.Info(fmt.Sprintf("announcements seeded: %d entries", len(entries)))
}
