-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 交易主表 transactions（核心）
--
-- 老 demo 的几处"hack"在新设计里干净化：
--   1. 挂账不再用负金额 + transactionType='PENDING'，独立到 member_credits 表
--      * 新挂账交易：kind='sale' + actual_paid_amount < total_amount，差额入 member_credits
--      * 纯挂账（完全没收钱）：actual_paid_amount=0，total 为应收
--   2. 清挂账独立语义：kind='credit_settlement'，与普通消费 kind='sale' 区分
--   3. 办卡交易：kind='recharge'（老靠 summary startswith '办理【' 识别，脆弱）
--   4. 撤单不再 DELETE + 建 VoidLog，改为 status='voided' 软撤单 + 审计字段
--      * 老 VoidLog 的查询改为 SELECT * FROM transactions WHERE status='voided'
--   5. 支付方式不再枚举，外键到用户自定义的 payment_methods
--   6. 扣会员卡的判定：card_id IS NOT NULL（与 payment_method 解耦）
-- ============================================================
CREATE TABLE transactions (
  id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

  -- 交易性质
  kind                 TEXT NOT NULL DEFAULT 'sale'
                       CHECK (kind IN ('sale','recharge','credit_settlement')),
  status               TEXT NOT NULL DEFAULT 'completed'
                       CHECK (status IN ('completed','voided')),

  -- 客户（可为散客）
  member_id            UUID REFERENCES members(id) ON DELETE SET NULL,
  customer_name        TEXT,         -- 散客姓名 或 会员物理删除后的留痕

  -- 员工归属（可空，不强制）
  staff_id             UUID REFERENCES staff(id) ON DELETE SET NULL,

  -- 支付
  payment_method_id    UUID NOT NULL REFERENCES payment_methods(id),
  card_id              UUID REFERENCES cards(id) ON DELETE SET NULL,

  -- 金额（NUMERIC 10,2 + shopspring/decimal.Decimal）
  total_amount         NUMERIC(10,2) NOT NULL,           -- 应收
  actual_paid_amount   NUMERIC(10,2) NOT NULL,           -- 实收（<total 时差额进 member_credits）
  discount_amount      NUMERIC(10,2) NOT NULL DEFAULT 0, -- 优惠

  -- 发生时间（可手动指定，对应老 customTransactionTime）
  transaction_time     TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- 撤单审计（status='voided' 时填充）
  voided_at            TIMESTAMPTZ,
  voided_by_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
  voided_by_name       TEXT,         -- 快照，防用户改名/删除后审计信息缺失
  void_reason          TEXT,

  -- 摘要 / 备注
  summary              TEXT,         -- 自动生成的概览，如 "洗发、剪发" / "办理【500元卡 7折】"
  notes                TEXT,

  legacy_id            TEXT,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- 一致性：voided 必须有 voided_at
  CHECK ((status = 'completed' AND voided_at IS NULL)
      OR (status = 'voided'    AND voided_at IS NOT NULL))
);

CREATE INDEX idx_transactions_tenant_time
  ON transactions(tenant_id, transaction_time DESC);
CREATE INDEX idx_transactions_tenant_member_time
  ON transactions(tenant_id, member_id, transaction_time DESC)
  WHERE member_id IS NOT NULL;
CREATE INDEX idx_transactions_tenant_staff_time
  ON transactions(tenant_id, staff_id, transaction_time DESC)
  WHERE staff_id IS NOT NULL;
CREATE INDEX idx_transactions_card
  ON transactions(tenant_id, card_id)
  WHERE card_id IS NOT NULL;
CREATE INDEX idx_transactions_payment_method
  ON transactions(tenant_id, payment_method_id);
CREATE INDEX idx_transactions_kind_status
  ON transactions(tenant_id, kind, status, transaction_time DESC);
CREATE INDEX idx_transactions_voided
  ON transactions(tenant_id, voided_at DESC)
  WHERE status = 'voided';
CREATE UNIQUE INDEX uq_transactions_tenant_legacy_id
  ON transactions(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

COMMENT ON COLUMN transactions.kind               IS 'sale=消费, recharge=办卡/充值, credit_settlement=清挂账';
COMMENT ON COLUMN transactions.status             IS 'completed/voided；撤单不删数据，改状态';
COMMENT ON COLUMN transactions.card_id            IS '非空即走扣会员卡（与 payment_method 解耦）';
COMMENT ON COLUMN transactions.actual_paid_amount IS '实收；<total 时差额入 member_credits（挂账）';

ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE transactions FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_transactions ON transactions
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transactions CASCADE;
-- +goose StatementEnd
