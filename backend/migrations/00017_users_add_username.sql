-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- users 表补 username 列：登录支持 username 或 email
--
-- 老 demo 用 username（非邮箱格式，如 'demo@example.com'）登录
-- 新 MMS 原设计只有 email，但老数据迁入时 username 不一定是合法邮箱
-- 方案：users 同时保留 username（可空）+ email（主）
--   - 登录查询：WHERE email = $1 OR username = $1
--   - 迁移老 demo@example.com：填 username，email 留空（NULL）
--   - 应用层创建新用户：填 email，username 留空（或同步填 email 前缀）
-- ============================================================
ALTER TABLE users ADD COLUMN username CITEXT;

-- 租户内 username 唯一（NULL 不参与唯一约束）
CREATE UNIQUE INDEX uq_users_tenant_username
  ON users(tenant_id, username) WHERE username IS NOT NULL;

COMMENT ON COLUMN users.username IS '登录用户名（非邮箱），用于兼容老系统或非邮箱登录场景';

-- email 改为可空（老数据可能没有合法 email）
-- 但至少要有 username 或 email 之一
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;

ALTER TABLE users ADD CONSTRAINT users_login_identity_check
  CHECK (email IS NOT NULL OR username IS NOT NULL);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_login_identity_check;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
DROP INDEX IF EXISTS uq_users_tenant_username;
ALTER TABLE users DROP COLUMN IF EXISTS username;
-- +goose StatementEnd
