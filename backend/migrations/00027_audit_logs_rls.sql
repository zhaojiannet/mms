-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- audit_logs 增强 RLS：defense-in-depth 防止跨租户日志泄露
--
-- 历史原因：audit_logs 原设计为"全局表"无 RLS，依赖 Go 层的 CreateAuditLog
-- 把 tenant_id 作为参数显式写入，List 接口的 SQL 手动过滤 tenant_id。
--
-- 风险：任何一个新加的手写查询忘传 tenant_id 过滤 → 跨租户审计泄露
--
-- 现在改为：
--   启用 FORCE RLS + policy：tenant_id IS NULL OR tenant_id = current_tenant_id()
--   NULL tenant 保留给未来 system 级事件（现在没有）
-- ============================================================

ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs FORCE ROW LEVEL SECURITY;

-- 策略：当前租户能看到自己的日志 + NULL tenant 的系统日志
CREATE POLICY tenant_isolation_audit_logs ON audit_logs
  USING (tenant_id IS NULL OR tenant_id = current_tenant_id())
  WITH CHECK (tenant_id IS NULL OR tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP POLICY IF EXISTS tenant_isolation_audit_logs ON audit_logs;
ALTER TABLE audit_logs NO FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
-- +goose StatementEnd
