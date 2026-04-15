-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 卡余额变化流水 card_balance_logs
--
-- 老 demo 用 Transaction.balanceSnapshot longtext JSON 记录：
--   * 只有 10% 交易有快照（老代码后加，历史数据不全）
--   * JSON 不便聚合/查询单卡历史
--   * 撤单时靠 notes 文本正则解析多卡支付，容错差
--
-- 新设计：每次卡余额变动都写一行（流水账）
--   * 单卡消费/多卡支付：每张被扣的卡各一行
--   * 办卡：change_type='issue'，delta=+final_price
--   * 撤单：change_type='void_restore'，delta 为正（还原金额）
--   * 手动调整：change_type='manual_adjust'（老系统无此功能，新增）
--
-- 收益：
--   * 单卡余额变化曲线随时查
--   * 撤单无需解析 notes，直接插反向记录
--   * 卡余额 = cards.balance（物化字段）= SUM(delta) 校验一致性
-- ============================================================
CREATE TABLE card_balance_logs (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  card_id           UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
  transaction_id    UUID REFERENCES transactions(id) ON DELETE SET NULL,
  change_type       TEXT NOT NULL
                    CHECK (change_type IN ('issue','consume','void_restore','manual_adjust','expire','freeze','unfreeze')),
  delta             NUMERIC(10,2) NOT NULL,           -- 正增负减
  balance_before    NUMERIC(10,2) NOT NULL,
  balance_after     NUMERIC(10,2) NOT NULL,
  operator_user_id  UUID REFERENCES users(id) ON DELETE SET NULL,
  note              TEXT,
  occurred_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- 一致性：before + delta = after
  CHECK (balance_before + delta = balance_after)
);

CREATE INDEX idx_card_balance_logs_card
  ON card_balance_logs(card_id, occurred_at DESC);
CREATE INDEX idx_card_balance_logs_tenant_time
  ON card_balance_logs(tenant_id, occurred_at DESC);
CREATE INDEX idx_card_balance_logs_tx
  ON card_balance_logs(transaction_id)
  WHERE transaction_id IS NOT NULL;

COMMENT ON COLUMN card_balance_logs.change_type IS 'issue=办卡, consume=消费, void_restore=撤单还原, manual_adjust=手动调整, expire=过期归零, freeze=冻结, unfreeze=解冻';
COMMENT ON COLUMN card_balance_logs.delta       IS '变化量，正为增加余额，负为减少';

ALTER TABLE card_balance_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE card_balance_logs FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_card_balance_logs ON card_balance_logs
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS card_balance_logs CASCADE;
-- +goose StatementEnd
