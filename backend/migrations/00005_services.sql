-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 服务项目 services
--   - name 租户内唯一（老 demo 是全局 unique，新系统改为 tenant 隔离）
--   - no_discount：布尔标记"不参与折扣"（老 Service.noDiscount），POS 扣卡时按原价 1:1 扣
--   - duration_min：项目时长分钟，阶段 3 预约模块会用（老系统无此字段，新增）
--   - category：服务分类（老系统无，新增，字符串自由填）
--   - price 用 NUMERIC(10,2)，配合 shopspring/decimal 处理（禁用 float64）
-- ============================================================
CREATE TABLE services (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name          TEXT NOT NULL,
  price         NUMERIC(10,2) NOT NULL,
  duration_min  INT,
  category      TEXT,
  description   TEXT,
  no_discount   BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order    INT  NOT NULL DEFAULT 99,
  status        TEXT NOT NULL DEFAULT 'active'
                CHECK (status IN ('active','inactive')),
  legacy_id     TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_services_tenant_name ON services(tenant_id, name);
CREATE INDEX idx_services_tenant_sort ON services(tenant_id, sort_order, name);
CREATE UNIQUE INDEX uq_services_tenant_legacy_id
  ON services(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN services.no_discount IS '不参与会员卡折扣（商品/耗材类项目，按原价扣卡）';
COMMENT ON COLUMN services.legacy_id  IS '老 demo Service.id 溯源';

ALTER TABLE services ENABLE ROW LEVEL SECURITY;
ALTER TABLE services FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_services ON services
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS services CASCADE;
-- +goose StatementEnd
