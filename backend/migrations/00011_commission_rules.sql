-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 提成规则 commission_rules
--
--   - 老 demo 有 staff.countsCommission 字段但无规则表（只有归属，无计算）
--   - 新系统提成计算优先级（transaction_items.commission_amount 写入时）：
--       1. commission_rules(staff_id, service_id) — 员工+项目定制
--       2. commission_rules(staff_id, service_id=NULL) — 员工通用
--       3. staff.default_commission_rate — 员工默认
--       4. 0
--     staff.counts_commission=FALSE 时直接跳过（结果始终为 0）
--   - rate 和 fixed_amount 二选一（不能同时有，但至少一个）
-- ============================================================
CREATE TABLE commission_rules (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  staff_id      UUID NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
  service_id    UUID REFERENCES services(id) ON DELETE CASCADE,   -- NULL = 该员工的通用规则
  rate          NUMERIC(4,3) CHECK (rate IS NULL OR (rate >= 0 AND rate <= 1)),
  fixed_amount  NUMERIC(10,2) CHECK (fixed_amount IS NULL OR fixed_amount >= 0),
  note          TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- 必须提供 rate 或 fixed_amount 之一，且不能同时提供
  CHECK ((rate IS NOT NULL)::INT + (fixed_amount IS NOT NULL)::INT = 1)
);

-- 同一员工对同一服务只能有一条规则
CREATE UNIQUE INDEX uq_commission_rules_staff_service
  ON commission_rules(tenant_id, staff_id, service_id)
  WHERE service_id IS NOT NULL;
-- 同一员工只能有一条"通用"规则
CREATE UNIQUE INDEX uq_commission_rules_staff_default
  ON commission_rules(tenant_id, staff_id)
  WHERE service_id IS NULL;

CREATE INDEX idx_commission_rules_staff
  ON commission_rules(tenant_id, staff_id);

COMMENT ON COLUMN commission_rules.rate         IS '提成率 [0,1]，与 fixed_amount 二选一';
COMMENT ON COLUMN commission_rules.fixed_amount IS '固定金额，与 rate 二选一';
COMMENT ON COLUMN commission_rules.service_id   IS 'NULL = 该员工所有服务的通用规则';

ALTER TABLE commission_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE commission_rules FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_commission_rules ON commission_rules
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS commission_rules CASCADE;
-- +goose StatementEnd
