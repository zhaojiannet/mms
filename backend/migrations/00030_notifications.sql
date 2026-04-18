-- +goose Up
-- 系统公告：全局表（所有租户共享同一份），不启用 RLS / tenant_id
-- 内容来源：代码仓库内的 announcements.json，启动时 seed 入库（ON CONFLICT DO UPDATE）
CREATE TABLE system_announcements (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  version       VARCHAR(32) NOT NULL UNIQUE,   -- 如 "26.4"、"26.4.1"
  type          VARCHAR(16) NOT NULL,          -- release | maintenance | policy
  title         TEXT NOT NULL,
  summary       TEXT NOT NULL,                 -- 铃铛列表显示的短摘要
  body_markdown TEXT NOT NULL,                 -- /changelog 页展开的完整内容
  published_at  TIMESTAMPTZ NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_system_announcements_published ON system_announcements (published_at DESC);

-- 每 user 对每条公告的已读状态（多店员彼此独立未读）
CREATE TABLE system_announcement_reads (
  user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  announcement_id UUID NOT NULL REFERENCES system_announcements(id) ON DELETE CASCADE,
  read_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, announcement_id)
);

-- +goose Down
DROP TABLE IF EXISTS system_announcement_reads;
DROP TABLE IF EXISTS system_announcements;
