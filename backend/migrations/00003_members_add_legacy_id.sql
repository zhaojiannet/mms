-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- members 表补丁：增加 legacy_id（迁移溯源用）
--   - 所有业务表统一保留 legacy_id 以便从老 demo 追溯原始主键
--   - 不参与业务逻辑，仅用于数据追溯和问题排查
--   - 联合唯一索引（tenant_id, legacy_id）允许不同租户各自独立的历史 id 空间
-- ============================================================
ALTER TABLE members ADD COLUMN legacy_id TEXT;

CREATE UNIQUE INDEX uq_members_tenant_legacy_id
  ON members(tenant_id, legacy_id)
  WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN members.legacy_id IS '老 demo Member.id（nanoid），迁移溯源用，新系统不参与业务';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_members_tenant_legacy_id;
ALTER TABLE members DROP COLUMN IF EXISTS legacy_id;
-- +goose StatementEnd
