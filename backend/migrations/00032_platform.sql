-- +goose Up
-- 平台层（运营后台 + 商户自助申请）：
--   - platform_operators / signup_applications 为平台全局表，无 tenant_id、不启用 RLS
--   - 平台身份与商户 users 完全隔离：独立表、独立 JWT issuer，互不可登
--   - 跨租户读 subscriptions 走哨兵策略（app.platform_op），DB 用户保持无 BYPASSRLS

-- 平台操作员（运营人员，当前单账号，由 PLATFORM_ADMIN_* env 幂等 bootstrap）
CREATE TABLE platform_operators (
  id                    UUID PRIMARY KEY DEFAULT uuidv7(),
  email                 CITEXT NOT NULL UNIQUE,
  password_hash         TEXT NOT NULL,
  name                  TEXT NOT NULL DEFAULT '运营',
  status                TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'disabled')),
  token_version         INT NOT NULL DEFAULT 1,
  password_changed_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  failed_login_attempts INT NOT NULL DEFAULT 0,
  locked_until          TIMESTAMPTZ,
  last_login_at         TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 商户开通申请（主站 /apply 表单落库，运营后台审批）
CREATE TABLE signup_applications (
  id            UUID PRIMARY KEY DEFAULT uuidv7(),
  store_name    TEXT NOT NULL,
  industry      TEXT NOT NULL,
  contact_name  TEXT NOT NULL,
  phone         TEXT NOT NULL,
  desired_slug  TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'approved', 'rejected')),
  reject_reason TEXT,
  -- 审批通过后指向开通的租户
  tenant_id     UUID REFERENCES tenants(id),
  reviewed_at   TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_signup_applications_status ON signup_applications (status, created_at DESC);

-- 平台哨兵策略：平台事务内 SET LOCAL app.platform_op = 'on' 后可跨租户读订阅
-- （运营后台商户列表需要；写操作仍逐租户 SET app.current_tenant 走原隔离策略）
CREATE POLICY platform_read_subscriptions ON subscriptions FOR SELECT
  USING (current_setting('app.platform_op', true) = 'on');

-- 重置商户管理员密码 / 审批建号需要平台读写 users
CREATE POLICY platform_read_users ON users FOR SELECT
  USING (current_setting('app.platform_op', true) = 'on');

-- +goose Down
DROP POLICY IF EXISTS platform_read_users ON users;
DROP POLICY IF EXISTS platform_read_subscriptions ON subscriptions;
DROP TABLE IF EXISTS signup_applications;
DROP TABLE IF EXISTS platform_operators;
