-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- users 表增加账号锁定字段：失败登录累计 N 次锁定 M 分钟
--
-- 字段：
--   failed_login_attempts  INT       累计失败次数，成功登录清零
--   locked_until           TIMESTAMPTZ 锁定到此时刻；NULL 表示未锁定
-- ============================================================

ALTER TABLE users
  ADD COLUMN failed_login_attempts INT NOT NULL DEFAULT 0,
  ADD COLUMN locked_until TIMESTAMPTZ;

COMMENT ON COLUMN users.failed_login_attempts IS '连续失败登录次数，成功登录清零';
COMMENT ON COLUMN users.locked_until IS '账号锁定截止时刻（UTC），NULL 表示未锁定';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
  DROP COLUMN IF EXISTS locked_until,
  DROP COLUMN IF EXISTS failed_login_attempts;
-- +goose StatementEnd
