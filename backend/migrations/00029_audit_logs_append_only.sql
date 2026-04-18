-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- audit_logs 改为 append-only：防 super_admin 通过 SQL 注入或
-- 未来误加的 DELETE/UPDATE handler 删改审计历史
--
-- 为什么用 trigger 而非 REVOKE：
--   mms 既是表 owner 又是 app user，REVOKE 对 owner 无效
--   trigger 对所有调用方（含 owner）生效
--
-- 运维操作：未来若需 schema 变更 / 清理旧日志，DROP TRIGGER → 操作 → CREATE TRIGGER
-- 这个"不便"正是审计完整性的代价
-- ============================================================

CREATE OR REPLACE FUNCTION audit_logs_reject_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs is append-only; UPDATE/DELETE/TRUNCATE rejected (op=%)', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_no_update
    BEFORE UPDATE ON audit_logs
    FOR EACH STATEMENT
    EXECUTE FUNCTION audit_logs_reject_mutation();

CREATE TRIGGER audit_logs_no_delete
    BEFORE DELETE ON audit_logs
    FOR EACH STATEMENT
    EXECUTE FUNCTION audit_logs_reject_mutation();

CREATE TRIGGER audit_logs_no_truncate
    BEFORE TRUNCATE ON audit_logs
    FOR EACH STATEMENT
    EXECUTE FUNCTION audit_logs_reject_mutation();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS audit_logs_no_truncate ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_no_delete ON audit_logs;
DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_reject_mutation();
-- +goose StatementEnd
