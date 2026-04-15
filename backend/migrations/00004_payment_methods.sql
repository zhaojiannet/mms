-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 支付方式 payment_methods
--   - 完全由商户自定义（非枚举）：可建"现金/微信/支付宝/老板微信/POS"等任意归类
--   - 与"是否扣会员卡"解耦：扣卡逻辑靠 transactions.card_id 判定
--   - 新建租户 bootstrap 时 seed 8 种常用方式（见 migration 00010 预留），商户可自由改名增减
--   - 不在 transactions 外键 ON DELETE CASCADE（删支付方式不应删交易）
-- ============================================================
CREATE TABLE payment_methods (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  sort_order  INT  NOT NULL DEFAULT 99,
  is_active   BOOLEAN NOT NULL DEFAULT TRUE,
  legacy_id   TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_payment_methods_tenant_name ON payment_methods(tenant_id, name);
CREATE INDEX idx_payment_methods_tenant_sort ON payment_methods(tenant_id, sort_order, name);
CREATE UNIQUE INDEX uq_payment_methods_tenant_legacy_id
  ON payment_methods(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON TABLE payment_methods IS '商户自定义的支付方式分类（非支付通道集成）';
COMMENT ON COLUMN payment_methods.legacy_id IS '老 demo PaymentMethod 枚举名（CASH/WECHAT_PAY/...），迁移溯源用';

ALTER TABLE payment_methods ENABLE ROW LEVEL SECURITY;
ALTER TABLE payment_methods FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_payment_methods ON payment_methods
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payment_methods CASCADE;
-- +goose StatementEnd
