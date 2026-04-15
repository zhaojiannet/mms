-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 会员卡实例 cards
--   - 老 demo 4 个定制字段（isCustomCard/customAmount/customDiscountRate/discountSource）
--     塌缩为 2 个字段：
--       * final_price：开卡时的实际金额（可定制，也可取 cardType.price）
--       * final_discount_rate：实际折扣率（可定制，也可取 cardType.discount_rate）
--     是否"定制"由 (final_price, final_discount_rate) 是否等于 cardType 默认值推导
--   - balance：当前余额（扣卡时减少，办卡时=final_price）
--   - 卡余额变化流水在 card_balance_logs 表（阶段 3 建）—— 比老 balanceSnapshot JSON 更可查
--   - status：active / frozen / expired / depleted（4 态保留，老 100% active 但代码引用 frozen）
-- ============================================================
CREATE TABLE cards (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  member_id              UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  card_type_id           UUID NOT NULL REFERENCES card_types(id),
  final_price            NUMERIC(10,2) NOT NULL
                         CHECK (final_price >= 0),
  final_discount_rate    NUMERIC(4,3) NOT NULL
                         CHECK (final_discount_rate > 0 AND final_discount_rate <= 1),
  balance                NUMERIC(10,2) NOT NULL DEFAULT 0
                         CHECK (balance >= 0),
  issued_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at             TIMESTAMPTZ,
  status                 TEXT NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active','frozen','expired','depleted')),
  notes                  TEXT,
  legacy_id              TEXT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cards_tenant_member        ON cards(tenant_id, member_id);
CREATE INDEX idx_cards_member_status        ON cards(member_id, status);
CREATE INDEX idx_cards_tenant_balance       ON cards(tenant_id, balance DESC);
CREATE INDEX idx_cards_tenant_issued        ON cards(tenant_id, issued_at DESC);
CREATE UNIQUE INDEX uq_cards_tenant_legacy_id
  ON cards(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN cards.final_price         IS '开卡实际金额（塌缩老 customAmount/cardType.initialPrice）';
COMMENT ON COLUMN cards.final_discount_rate IS '实际折扣率（塌缩老 customDiscountRate/cardType.discountRate）';
COMMENT ON COLUMN cards.balance             IS '当前可用余额；变化流水在 card_balance_logs';

ALTER TABLE cards ENABLE ROW LEVEL SECURITY;
ALTER TABLE cards FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_cards ON cards
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS cards CASCADE;
-- +goose StatementEnd
