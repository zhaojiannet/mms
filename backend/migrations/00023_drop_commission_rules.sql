-- +goose Up
-- +goose StatementBegin

-- 删除 commission_rules 表
-- 阶段 1 决定：简化提成模型，仅保留 staff.default_commission_rate 级别
-- 未来若需细粒度规则（按员工×服务定制）再加回
DROP TABLE IF EXISTS commission_rules CASCADE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

CREATE TABLE commission_rules (
  id          UUID PRIMARY KEY DEFAULT uuidv7(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  staff_id    UUID NOT NULL REFERENCES staff(id)   ON DELETE CASCADE,
  service_id  UUID REFERENCES services(id)        ON DELETE CASCADE,
  rate        NUMERIC(5,4),
  fixed_amount NUMERIC(10,2),
  note        TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE commission_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE commission_rules FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_commission_rules ON commission_rules
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd
