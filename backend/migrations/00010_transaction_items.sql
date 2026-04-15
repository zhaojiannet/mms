-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 交易明细 transaction_items
--
--   - service_id ON DELETE SET NULL：服务被删除不影响历史明细
--     * 老系统限制"有交易引用的服务不可删"很严，新系统靠 service_name_snapshot 兜底
--   - service_name_snapshot：项目名快照，防项目改名后历史报表错乱（老无此字段）
--   - commission_amount：下单时按规则算定的提成快照
--     * 规则后续改动不影响历史（老系统无提成功能，新增）
--   - 报表"项目销售排行" = SUM(price * quantity)（老系统 bug 未乘 quantity，新修正）
-- ============================================================
CREATE TABLE transaction_items (
  id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  transaction_id         UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
  service_id             UUID REFERENCES services(id) ON DELETE SET NULL,
  service_name_snapshot  TEXT NOT NULL,
  price                  NUMERIC(10,2) NOT NULL,           -- 单价
  quantity               INT NOT NULL DEFAULT 1
                         CHECK (quantity > 0),
  commission_amount      NUMERIC(10,2) NOT NULL DEFAULT 0
                         CHECK (commission_amount >= 0),
  legacy_id              TEXT,
  created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_transaction_items_tx
  ON transaction_items(transaction_id);
CREATE INDEX idx_transaction_items_tenant_service
  ON transaction_items(tenant_id, service_id)
  WHERE service_id IS NOT NULL;
CREATE UNIQUE INDEX uq_transaction_items_tenant_legacy_id
  ON transaction_items(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN transaction_items.service_name_snapshot IS '项目名快照，service 改名或删除后报表仍可读';
COMMENT ON COLUMN transaction_items.commission_amount     IS '下单时按 commission_rules + staff.default 算定的提成（快照，规则变化不影响历史）';

ALTER TABLE transaction_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE transaction_items FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_transaction_items ON transaction_items
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_items CASCADE;
-- +goose StatementEnd
