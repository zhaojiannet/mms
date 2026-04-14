-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 会员表 members（阶段 2 核心业务第 1 张表）
--   - 多租户：tenant_id 列 + RLS + WITH CHECK
--   - 复合索引 (tenant_id, ...) 保证 RLS 过滤不被 seq scan 拖垮
--   - phone 按 (tenant_id, phone) 去重，同一租户内手机号唯一（允许 NULL）
-- ============================================================
CREATE TABLE members (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  phone        TEXT,
  gender       TEXT NOT NULL DEFAULT 'unknown'
               CHECK (gender IN ('male','female','unknown')),
  birthday     DATE,
  notes        TEXT,
  status       TEXT NOT NULL DEFAULT 'active'
               CHECK (status IN ('active','inactive')),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_members_tenant_created ON members(tenant_id, created_at DESC);
CREATE UNIQUE INDEX uq_members_tenant_phone ON members(tenant_id, phone)
  WHERE phone IS NOT NULL;

ALTER TABLE members ENABLE ROW LEVEL SECURITY;
ALTER TABLE members FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_members ON members
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS members CASCADE;
-- +goose StatementEnd
