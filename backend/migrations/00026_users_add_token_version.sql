-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- users 加 token_version + password_changed_at：JWT 吊销机制
--
-- 用途：
--   token_version        - 每次改密/重置/role 或 status 变更时 +1；JWT claim 带此值；
--                          RequireAuth 对比 DB 值不等则拒绝
--   password_changed_at  - 改密时间戳；JWT iat 早于此值则拒绝（双重保险）
--
-- 效果：
--   密码被盗后改密 → 旧 token 立即失效
--   role 降级/账号停用 → 持有的旧 token 立即失效
--   refresh 攻击：refresh 会签带新 ver 的 token，但旧 ver token 失效
-- ============================================================

ALTER TABLE users
  ADD COLUMN token_version       INT         NOT NULL DEFAULT 0,
  ADD COLUMN password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now();

COMMENT ON COLUMN users.token_version IS 'JWT 版本号；改密/重置/role/status 变更时 +1，旧 token 立即失效';
COMMENT ON COLUMN users.password_changed_at IS '最近一次密码变更时刻；JWT iat 早于此则拒绝';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
  DROP COLUMN IF EXISTS password_changed_at,
  DROP COLUMN IF EXISTS token_version;
-- +goose StatementEnd
