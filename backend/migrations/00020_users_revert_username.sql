-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 撤销 00017：email 列本身是 CITEXT，可以接受任意账号字符串
--             "email" 只是列名，不强制要求合法邮箱格式，无需额外 username 列
--
-- 撤销步骤：
--   1. 把 email IS NULL 但 username 非空的行，email 回填 username 值
--   2. 删除 users_login_identity_check（email/username 二选一）CHECK
--   3. email 恢复 NOT NULL
--   4. 删掉 username 列和其唯一索引
-- ============================================================

-- 临时关 RLS（migration 身份 mms 是表 owner，FORCE RLS 下 UPDATE 会被当前 tenant 过滤空）
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

-- 回填 email：若 email 为空则用 username 填入
UPDATE users
SET email = username
WHERE email IS NULL AND username IS NOT NULL;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_login_identity_check;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
DROP INDEX IF EXISTS uq_users_tenant_username;
ALTER TABLE users DROP COLUMN IF EXISTS username;

-- 恢复 RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 恢复 00017 的状态（仅供开发期回滚）
ALTER TABLE users ADD COLUMN username CITEXT;
CREATE UNIQUE INDEX uq_users_tenant_username ON users(tenant_id, username)
  WHERE username IS NOT NULL;
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_login_identity_check
  CHECK (email IS NOT NULL OR username IS NOT NULL);
-- +goose StatementEnd
