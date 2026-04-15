-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 租户配置 tenant_settings（KV 化）
--
-- 老 demo SystemConfig 是单行宽表（每加一个配置项都要改 schema）：
--   * enableLoginCaptcha / enableTransactionVoid / voidEnabledAt
--   * bookingCode / bookingCodeUpdatedAt（代码有但 dump schema 无）
--
-- 新设计：(tenant_id, key) → JSONB value
--   * 每租户独立配置（多租户必备）
--   * 加新配置不用改 schema
--   * value 用 JSONB 支持任意结构（布尔、字符串、数字、对象）
--
-- 约定的 key（首批，更多可随时加）：
--   enable_login_captcha     bool
--   enable_transaction_void  bool
--   void_enabled_at          ISO timestamp / null
--   booking_code             string / null
--   booking_code_updated_at  ISO timestamp / null
-- ============================================================
CREATE TABLE tenant_settings (
  tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  key         TEXT NOT NULL,
  value       JSONB NOT NULL DEFAULT 'null'::jsonb,
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, key)
);

CREATE INDEX idx_tenant_settings_key ON tenant_settings(key);

COMMENT ON TABLE  tenant_settings IS '每租户 KV 配置表，替代老 SystemConfig 单行宽表';
COMMENT ON COLUMN tenant_settings.value IS 'JSONB，支持任意结构（bool/string/number/object/null）';

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_tenant_settings ON tenant_settings
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tenant_settings CASCADE;
-- +goose StatementEnd
