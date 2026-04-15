-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 挂账台账 member_credits（应收账款）
--
-- 老 demo 用 Transaction.totalAmount<0 + transactionType='PENDING' + isPending 表达挂账：
--   * 负金额破坏语义（金额字段含义不一致）
--   * 清挂账靠 notes 文本 `CLEAR_PENDING:xxx` 关联，脆弱
--   * 报表需要绝对值 hack 才能算准
--
-- 新系统独立建表，与 transactions 解耦：
--   * amount 为正数（欠多少记多少）
--   * charged_tx_id：引起挂账的消费交易（可空，老数据无此关联）
--   * settled_at + settlement_tx_id：清挂账时间 + 清账交易
--   * 未清挂账 = WHERE settled_at IS NULL
--   * 按会员查欠款 / 按时间查走账 / 应收总额 都是直接聚合
-- ============================================================
CREATE TABLE member_credits (
  id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  member_id          UUID NOT NULL REFERENCES members(id) ON DELETE CASCADE,
  amount             NUMERIC(10,2) NOT NULL CHECK (amount > 0),
  summary            TEXT,                     -- "洗面 ¥38 折后 26.6"
  charged_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  charged_tx_id      UUID REFERENCES transactions(id) ON DELETE SET NULL,
  settled_at         TIMESTAMPTZ,
  settlement_tx_id   UUID REFERENCES transactions(id) ON DELETE SET NULL,
  legacy_id          TEXT,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- 一致性：settled_at 与 settlement_tx_id 要么都有要么都无
  CHECK ((settled_at IS NULL AND settlement_tx_id IS NULL)
      OR (settled_at IS NOT NULL AND settlement_tx_id IS NOT NULL))
);

CREATE INDEX idx_member_credits_tenant_member
  ON member_credits(tenant_id, member_id, charged_at DESC);
CREATE INDEX idx_member_credits_unsettled
  ON member_credits(tenant_id, member_id)
  WHERE settled_at IS NULL;
CREATE INDEX idx_member_credits_settlement_tx
  ON member_credits(settlement_tx_id)
  WHERE settlement_tx_id IS NOT NULL;
CREATE UNIQUE INDEX uq_member_credits_tenant_legacy_id
  ON member_credits(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN member_credits.legacy_id IS '老 demo PENDING Transaction.id 溯源';

ALTER TABLE member_credits ENABLE ROW LEVEL SECURITY;
ALTER TABLE member_credits FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_member_credits ON member_credits
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS member_credits CASCADE;
-- +goose StatementEnd
