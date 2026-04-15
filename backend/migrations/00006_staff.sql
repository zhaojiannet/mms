-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 员工 staff
--   - position 必填（职位/头衔，如"发型师/美容师/前台"，自由文本）
--   - user_id：员工可关联一个登录账号（一对一，可空）—— 老系统反向 User.staffId，新放 staff 侧
--   - counts_commission：是否计提成（老板/前台可能不计）
--   - default_commission_rate：默认提成率（优先级最低，commission_rules 覆盖）
--   - phone 在租户内唯一（老系统全局唯一，新改为 tenant 隔离）
-- ============================================================
CREATE TABLE staff (
  id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id                  UUID REFERENCES users(id) ON DELETE SET NULL,
  name                     TEXT NOT NULL,
  position                 TEXT NOT NULL,
  phone                    TEXT,
  hire_date                DATE,
  counts_commission        BOOLEAN NOT NULL DEFAULT TRUE,
  default_commission_rate  NUMERIC(4,3) NOT NULL DEFAULT 0.000
                           CHECK (default_commission_rate >= 0 AND default_commission_rate <= 1),
  sort_order               INT  NOT NULL DEFAULT 99,
  status                   TEXT NOT NULL DEFAULT 'active'
                           CHECK (status IN ('active','inactive')),
  legacy_id                TEXT,
  created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_staff_tenant_sort ON staff(tenant_id, sort_order, name);
CREATE UNIQUE INDEX uq_staff_tenant_phone
  ON staff(tenant_id, phone) WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX uq_staff_user_id
  ON staff(user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX uq_staff_tenant_legacy_id
  ON staff(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN staff.default_commission_rate IS '默认提成率 [0,1]，commission_rules 无匹配时兜底';
COMMENT ON COLUMN staff.counts_commission IS '是否参与提成核算（老板/前台可关闭）';

ALTER TABLE staff ENABLE ROW LEVEL SECURITY;
ALTER TABLE staff FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_staff ON staff
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS staff CASCADE;
-- +goose StatementEnd
