-- name: ListUnreadAnnouncementsForUser :many
-- 某用户未读的系统公告（铃铛通知合并用）
SELECT a.id, a.version, a.type, a.title, a.summary, a.body_markdown, a.published_at, a.created_at
FROM system_announcements a
WHERE NOT EXISTS (
  SELECT 1 FROM system_announcement_reads r
  WHERE r.announcement_id = a.id AND r.user_id = $1
)
ORDER BY a.published_at DESC;

-- name: ListAllAnnouncementsWithReadStatus :many
-- changelog 页使用：所有公告 + 当前 user 的已读状态
SELECT
  a.id, a.version, a.type, a.title, a.summary, a.body_markdown, a.published_at, a.created_at,
  (r.read_at IS NOT NULL)::boolean AS is_read
FROM system_announcements a
LEFT JOIN system_announcement_reads r
  ON r.announcement_id = a.id AND r.user_id = $1
ORDER BY a.published_at DESC;

-- name: MarkAnnouncementReadByVersion :exec
-- 按 version 字符串标记已读（前端传 version 更稳定，不用暴露内部 UUID）
INSERT INTO system_announcement_reads (user_id, announcement_id)
SELECT $1, id FROM system_announcements WHERE version = $2
ON CONFLICT DO NOTHING;

-- name: MarkAllAnnouncementsRead :exec
INSERT INTO system_announcement_reads (user_id, announcement_id)
SELECT $1, id FROM system_announcements
ON CONFLICT DO NOTHING;

-- name: GetLatestAnnouncementVersion :one
-- 侧边栏版本号红点：最新公告版本；若当前用户未读则前端显红点
SELECT a.version, (r.read_at IS NOT NULL)::boolean AS is_read
FROM system_announcements a
LEFT JOIN system_announcement_reads r
  ON r.announcement_id = a.id AND r.user_id = $1
ORDER BY a.published_at DESC
LIMIT 1;
