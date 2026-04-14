-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- PostgreSQL 扩展
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";     -- 大小写不敏感文本（账号/邮箱）
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- 模糊搜索

-- 注：mms 数据库与 mms 用户由运维在 PG 外预先创建（见 README bootstrap 流程）
-- 本迁移由 mms 用户自行跑：mms 是库 owner，pgcrypto/citext/pg_trgm 均为 trusted extension
-- 所以无需 superuser。表 owner=mms，FORCE RLS 对 owner 生效，业务表隔离真实起效。

-- ============================================================
-- 套餐定义表 editions
-- ============================================================
CREATE TABLE editions (
  id              SMALLSERIAL PRIMARY KEY,
  code            TEXT NOT NULL UNIQUE,
  name            TEXT NOT NULL,
  price_monthly   NUMERIC(10, 2),
  price_yearly    NUMERIC(10, 2),
  quotas          JSONB NOT NULL DEFAULT '{}',
  features        JSONB NOT NULL DEFAULT '{}',
  sort_order      SMALLINT NOT NULL DEFAULT 0,
  active          BOOLEAN NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON COLUMN editions.quotas IS '{"max_members":100, "max_staff":2, "max_branches":1, "push_monthly":100}, -1=unlimited';
COMMENT ON COLUMN editions.features IS 'feature flag 开关集';

INSERT INTO editions (code, name, price_monthly, price_yearly, quotas, features, sort_order) VALUES
('free',       '免费版',  0,     NULL,
  '{"max_members":100,  "max_staff":2,  "max_branches":1,  "push_monthly":100}',
  '{}', 1),
('plus',       'Plus',    9.9,   99,
  '{"max_members":1000, "max_staff":10, "max_branches":1,  "push_monthly":1000}',
  '{}', 2),
('pro',        'Pro',     39.9,  399,
  '{"max_members":5000, "max_staff":50, "max_branches":3,  "push_monthly":5000}',
  '{"api":true}', 3),
('ultra',      'Ultra',   199.9, 1999,
  '{"max_members":-1,   "max_staff":-1, "max_branches":-1, "push_monthly":-1}',
  '{"api":true,"industry_modules":true,"custom_workflow":true}', 4),
('enterprise', '企业版',  NULL,  NULL,
  '{}',
  '{"private_deploy":true,"custom_dev":true,"sla":true}', 5);

-- ============================================================
-- 租户表 tenants（精简版）
--   - 只保留租户身份识别的最小必要字段
--   - 品牌化字段（logo_url/primary_color/industry/brand_config）阶段 3 做主题化时
--     新建 tenant_themes 表承载，不往这里塞 JSON
--   - 配额/特性覆盖（quota_override/feature_override）阶段 4 admin 后台出现时再加
--   - isolation_mode（bridge/silo）真做多隔离模式时 schema 会变，不提前预留
--   - 当前有效套餐由 subscriptions 表查询，不在 tenants 表冗余 edition_id 指针
-- ============================================================
CREATE TABLE tenants (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug        CITEXT NOT NULL UNIQUE,
  name        TEXT NOT NULL,
  status      TEXT NOT NULL DEFAULT 'active'
              CHECK (status IN ('pending','active','suspended','deleted')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenants_status ON tenants(status) WHERE status != 'deleted';
COMMENT ON COLUMN tenants.slug IS '子域名前缀，如 demo → demo.<app_domain>';

-- ============================================================
-- 订阅表 subscriptions
-- ============================================================
CREATE TABLE subscriptions (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  edition_id     SMALLINT NOT NULL REFERENCES editions(id),
  status         TEXT NOT NULL DEFAULT 'active'
                 CHECK (status IN ('trialing','active','past_due','cancelled','expired')),
  billing_cycle  TEXT NOT NULL DEFAULT 'monthly'
                 CHECK (billing_cycle IN ('monthly','yearly','gift','contract')),
  source         TEXT NOT NULL DEFAULT 'manual'
                 CHECK (source IN ('gift','manual','contract')),
  started_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  current_period_start TIMESTAMPTZ NOT NULL DEFAULT now(),
  current_period_end   TIMESTAMPTZ,
  cancel_at      TIMESTAMPTZ,
  auto_renew     BOOLEAN NOT NULL DEFAULT TRUE,
  notes          TEXT,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_tenant ON subscriptions(tenant_id);
CREATE INDEX idx_subscriptions_expire ON subscriptions(current_period_end)
  WHERE status IN ('trialing','active');

-- ============================================================
-- 用户表 users
-- ============================================================
CREATE TABLE users (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  email           CITEXT NOT NULL,
  phone           TEXT,
  password_hash   TEXT NOT NULL,
  name            TEXT NOT NULL,
  role            TEXT NOT NULL DEFAULT 'staff'
                  CHECK (role IN ('admin','manager','staff')),
  status          TEXT NOT NULL DEFAULT 'active'
                  CHECK (status IN ('active','disabled')),
  last_login_at   TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, email)
);

CREATE INDEX idx_users_tenant ON users(tenant_id);

-- ============================================================
-- 刷新令牌表 refresh_tokens
-- ============================================================
CREATE TABLE refresh_tokens (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash    TEXT NOT NULL UNIQUE,
  expires_at    TIMESTAMPTZ NOT NULL,
  revoked_at    TIMESTAMPTZ,
  user_agent    TEXT,
  ip_address    INET,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expire ON refresh_tokens(expires_at) WHERE revoked_at IS NULL;

-- ============================================================
-- 审计日志表 audit_logs（运营级，非租户级）
-- ============================================================
CREATE TABLE audit_logs (
  id            BIGSERIAL PRIMARY KEY,
  tenant_id     UUID REFERENCES tenants(id) ON DELETE SET NULL,
  actor_id      UUID,
  actor_type    TEXT NOT NULL CHECK (actor_type IN ('user','super_admin','system')),
  action        TEXT NOT NULL,
  target_type   TEXT,
  target_id     TEXT,
  payload       JSONB NOT NULL DEFAULT '{}',
  ip_address    INET,
  user_agent    TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_actor  ON audit_logs(actor_id, created_at DESC);

-- ============================================================
-- Row Level Security（行级安全）策略
-- 应用用户通过 SET LOCAL app.current_tenant = '<uuid>' 提供上下文
--
-- 设计原则（参考 Supabase / PostgREST 业界实践）：
--   1. tenants 是"公开目录"（子域名 slug→id 映射），不启用 RLS，靠 GRANT 收敛权限
--   2. 业务表（subscriptions/users/refresh_tokens）启用 RLS + FORCE + USING + WITH CHECK
--      - USING 管 SELECT/UPDATE/DELETE 的读可见性
--      - WITH CHECK 管 INSERT 和 UPDATE 后新值的写合法性
--      - FORCE 让表 owner（postgres）自身也受 policy 约束，防止 owner 被误用
--   3. editions / audit_logs 是全局表，不启用 RLS，权限走 GRANT
-- ============================================================
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID
  LANGUAGE sql STABLE AS
$$
  SELECT NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID
$$;

-- ============================================================
-- Bootstrap 种子：demo 租户赠送 Plus 1 年（示例商户）
--   - 顺序说明：必须在 ENABLE/FORCE ROW LEVEL SECURITY 之前执行，
--     否则 WITH CHECK policy 会把本次 INSERT 挡回（mms 是 owner 但 FORCE 对 owner 生效）
--   - 自建用户可改 slug/name 或删除本段后再跑迁移
-- ============================================================
INSERT INTO tenants (slug, name) VALUES ('demo', 'Demo Store');

INSERT INTO subscriptions (tenant_id, edition_id, source, billing_cycle, current_period_end, auto_renew, notes)
SELECT t.id, e.id, 'gift', 'gift', now() + interval '1 year', FALSE, 'demo 示例商户，赠送 Plus 1 年'
FROM tenants t
CROSS JOIN editions e
WHERE t.slug = 'demo' AND e.code = 'plus';

ALTER TABLE subscriptions   ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions   FORCE  ROW LEVEL SECURITY;
ALTER TABLE users           ENABLE ROW LEVEL SECURITY;
ALTER TABLE users           FORCE  ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens  ENABLE ROW LEVEL SECURITY;
ALTER TABLE refresh_tokens  FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_subscriptions ON subscriptions
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY tenant_isolation_users ON users
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

CREATE POLICY tenant_isolation_refresh_tokens ON refresh_tokens
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- ============================================================
-- 权限模型
--   - mms 是库 owner 兼表 owner，天然拥有全部权限
--   - 应用层对 tenants / audit_logs 的"不写"靠 repo 层不暴露写方法 + sqlc 固定 SQL
--   - 未来引入运营后台时再新建 mms_admin 角色做 DDL/跨租户操作（参见 docs/decisions.md）
-- ============================================================

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TABLE IF EXISTS editions CASCADE;
DROP FUNCTION IF EXISTS current_tenant_id();
-- +goose StatementEnd
