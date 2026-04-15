-- +goose Up
-- +goose StatementBegin

-- users 表补 legacy_id（同其他业务表，用于从老 demo 追溯原始 cuid）
ALTER TABLE users ADD COLUMN legacy_id TEXT;

CREATE UNIQUE INDEX uq_users_tenant_legacy_id
  ON users(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN users.legacy_id IS '老 demo User.id（cuid），迁移溯源用';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_users_tenant_legacy_id;
ALTER TABLE users DROP COLUMN IF EXISTS legacy_id;
-- +goose StatementEnd
