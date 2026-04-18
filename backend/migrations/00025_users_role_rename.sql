-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- users.role 由 (admin / manager / staff) 改为 (super_admin / admin / staff)
--
-- 迁移映射：
--   admin   → super_admin  （原 admin 是全权，保留最高权限）
--   manager → admin        （原中间层 manager 合并到 admin）
--   staff   → staff        （不变）
--
-- 执行顺序关键：先把 admin 升为 super_admin，再把 manager 降为 admin
-- 如果反过来会混淆（manager 先变 admin 后会被第二步再升为 super_admin）
-- ============================================================

-- RLS 下 migration 身份 mms 被 FORCE RLS 限制，临时关掉
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

-- 删旧 CHECK
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

-- 第 1 步：原 admin → super_admin
UPDATE users SET role = 'super_admin' WHERE role = 'admin';

-- 第 2 步：原 manager → admin
UPDATE users SET role = 'admin' WHERE role = 'manager';

-- 装上新 CHECK
ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('super_admin', 'admin', 'staff'));

-- 显式保持默认值 staff
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'staff';

-- 恢复 RLS
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;

-- 回滚：super_admin → admin（原 admin）
-- 原 manager 在 Up 时变成了 admin，现在无法区分"本来就是 admin（被升）"还是"原 manager（被降）"
-- 采取：全部 admin 都降回 manager；super_admin 升回 admin
-- 这会把原本的 admin 也降成 manager，但这是不可逆的信息损失——Down 仅用于开发期
UPDATE users SET role = 'manager' WHERE role = 'admin';
UPDATE users SET role = 'admin'   WHERE role = 'super_admin';

ALTER TABLE users ADD CONSTRAINT users_role_check
  CHECK (role IN ('admin', 'manager', 'staff'));

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- +goose StatementEnd
