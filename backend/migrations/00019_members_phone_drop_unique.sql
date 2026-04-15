-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 移除 members.phone 的租户内唯一约束
--
-- 原因：老 demo 业务允许占位符号码 '00000000000' 重复（"非手机客户"场景）
-- 老代码：应用层检查 phone !== '00000000000' && findFirst(...)，DB 无约束
-- 新设计对齐老行为：DB 层不强制唯一，handler 层做同等检查
--
-- 用普通 btree 索引代替，仍然支持 WHERE phone = X 的快速查询
-- ============================================================
DROP INDEX IF EXISTS uq_members_tenant_phone;

CREATE INDEX idx_members_tenant_phone ON members(tenant_id, phone) WHERE phone IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_members_tenant_phone;
CREATE UNIQUE INDEX uq_members_tenant_phone
  ON members(tenant_id, phone) WHERE phone IS NOT NULL;
-- +goose StatementEnd
