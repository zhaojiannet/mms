-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 上线前加固（对应全面排查确认的 schema 层问题）
--
-- 1. audit_logs.actor_type 放行 'anonymous'：
--    审计中间件对未登录写操作（公开预约 POST /api/booking）写
--    actor_type='anonymous'，原 CHECK 不含该值导致这类审计被静默丢弃
--
-- 2. card_balance_logs.card_id 级联改 RESTRICT：
--    资金流水是对账依据（balance = SUM(delta)），删卡/删会员的
--    级联链不应连带销毁流水——有流水的卡只能冻结，不能物理删除
--
-- 3. transactions 金额符号 CHECK：
--    账目正确性原靠 handler 校验，缺 DB 层防御纵深
--
-- 4. system_announcement_reads 补 tenant_id + RLS：
--    原表既无 tenant_id 也无 RLS，是唯一脱离 RLS 保护的业务相关表
--
-- 5. staff 列注释去掉对已删除 commission_rules 表的引用
-- ============================================================

ALTER TABLE audit_logs DROP CONSTRAINT audit_logs_actor_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_actor_type_check
  CHECK (actor_type IN ('user','super_admin','system','anonymous'));

ALTER TABLE card_balance_logs DROP CONSTRAINT card_balance_logs_card_id_fkey;
ALTER TABLE card_balance_logs ADD CONSTRAINT card_balance_logs_card_id_fkey
  FOREIGN KEY (card_id) REFERENCES cards(id) ON DELETE RESTRICT;

ALTER TABLE transactions ADD CONSTRAINT transactions_amounts_non_negative
  CHECK (total_amount >= 0 AND actual_paid_amount >= 0);
-- 折扣约束单独 NOT VALID：老系统迁移来的历史交易存在 discount_amount < 0
-- （实付高于标价的真实历史记录），只约束今后的写入，不改写历史
ALTER TABLE transactions ADD CONSTRAINT transactions_discount_non_negative
  CHECK (discount_amount >= 0) NOT VALID;

ALTER TABLE system_announcement_reads ADD COLUMN tenant_id UUID;
-- 回填需要跨租户读 users，而迁移以 mms（owner，FORCE RLS）身份执行；
-- 事务内临时取消 FORCE 让 owner 旁路 RLS，回填完立即恢复（失败则整体回滚）
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
UPDATE system_announcement_reads r SET tenant_id = u.tenant_id
  FROM users u WHERE u.id = r.user_id;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE system_announcement_reads ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE system_announcement_reads ADD CONSTRAINT system_announcement_reads_tenant_id_fkey
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_system_announcement_reads_tenant
  ON system_announcement_reads (tenant_id, user_id);

ALTER TABLE system_announcement_reads ENABLE ROW LEVEL SECURITY;
ALTER TABLE system_announcement_reads FORCE  ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON system_announcement_reads
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

COMMENT ON COLUMN staff.default_commission_rate IS '默认提成率 [0,1]（提成核算功能未上线，字段预留）';
COMMENT ON COLUMN staff.counts_commission IS '是否参与提成核算（老板/前台可关闭；提成核算功能未上线，字段预留）';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
COMMENT ON COLUMN staff.default_commission_rate IS '默认提成率 [0,1]，commission_rules 无匹配时兜底';
COMMENT ON COLUMN staff.counts_commission IS '是否参与提成核算（老板/前台可关闭）';

DROP POLICY IF EXISTS tenant_isolation ON system_announcement_reads;
ALTER TABLE system_announcement_reads DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_system_announcement_reads_tenant;
ALTER TABLE system_announcement_reads DROP CONSTRAINT IF EXISTS system_announcement_reads_tenant_id_fkey;
ALTER TABLE system_announcement_reads DROP COLUMN IF EXISTS tenant_id;

ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_discount_non_negative;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_amounts_non_negative;

ALTER TABLE card_balance_logs DROP CONSTRAINT card_balance_logs_card_id_fkey;
ALTER TABLE card_balance_logs ADD CONSTRAINT card_balance_logs_card_id_fkey
  FOREIGN KEY (card_id) REFERENCES cards(id) ON DELETE CASCADE;

ALTER TABLE audit_logs DROP CONSTRAINT audit_logs_actor_type_check;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_actor_type_check
  CHECK (actor_type IN ('user','super_admin','system'));
-- +goose StatementEnd
