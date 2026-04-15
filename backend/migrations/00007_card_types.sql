-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 会员卡模板 card_types
--   - 模板只定义"默认值"：price 和 discount_rate
--   - 实例化到 cards 表时可以"定制"：cards.final_price / final_discount_rate 覆盖模板
--   - 不再保留老系统 isCustomCard 标记（定制与否由 cards 字段是否偏离模板推导）
--   - name 租户内唯一（老全局 unique，新改 tenant 隔离）
-- ============================================================
CREATE TABLE card_types (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name           TEXT NOT NULL,
  price          NUMERIC(10,2) NOT NULL,
  discount_rate  NUMERIC(4,3) NOT NULL DEFAULT 1.000
                 CHECK (discount_rate > 0 AND discount_rate <= 1),
  sort_order     INT  NOT NULL DEFAULT 99,
  status         TEXT NOT NULL DEFAULT 'active'
                 CHECK (status IN ('active','inactive')),
  legacy_id      TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_card_types_tenant_name ON card_types(tenant_id, name);
CREATE INDEX idx_card_types_tenant_sort ON card_types(tenant_id, sort_order, price);
CREATE UNIQUE INDEX uq_card_types_tenant_legacy_id
  ON card_types(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN card_types.price         IS '模板默认售价（cards.final_price 可覆盖）';
COMMENT ON COLUMN card_types.discount_rate IS '模板默认折扣率，例 0.700=7折（cards.final_discount_rate 可覆盖）';

ALTER TABLE card_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE card_types FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_card_types ON card_types
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS card_types CASCADE;
-- +goose StatementEnd
