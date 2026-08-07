-- +goose Up
-- 平台层收尾加固（评审产物）：

-- 并发提交同 slug 申请的唯一约束（应用层 check-then-insert 有竞态窗口）
CREATE UNIQUE INDEX uniq_signup_applications_pending_slug
  ON signup_applications (desired_slug) WHERE status = 'pending';

-- 撤掉无消费者的全租户 users 读策略：重置密码/审批建号都走
-- SET app.current_tenant 后的原隔离策略，常开跨租户读 users（含密码哈希）纯属多余暴露
DROP POLICY IF EXISTS platform_read_users ON users;

-- 统一"不限"哨兵为 0：种子里 ultra 档的 -1 与运营后台的 0 两套语义并存，
-- 读取端 (<=0 视为不限) 两者等价，写入端只收非负数，归一到 0
UPDATE editions
SET quotas = (
  SELECT COALESCE(jsonb_object_agg(key, CASE WHEN value::bigint < 0 THEN '0'::jsonb ELSE value END), '{}'::jsonb)
  FROM jsonb_each(quotas)
)
WHERE EXISTS (SELECT 1 FROM jsonb_each(quotas) WHERE value::bigint < 0);

-- +goose Down
CREATE POLICY platform_read_users ON users FOR SELECT
  USING (current_setting('app.platform_op', true) = 'on');
DROP INDEX IF EXISTS uniq_signup_applications_pending_slug;
