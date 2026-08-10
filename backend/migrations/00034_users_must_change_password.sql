-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- users 表增加强制首次改密标志
--
-- 密码由他人设定的三个时刻（运营后台开通商户、管理员建员工、任一方重置
-- 密码）都置 true：那个密码是生成出来经人手转达的，不改掉就等于设密码的
-- 人长期持有这个账号。置 true 后除改密端点外一律 403，前端也直接跳改密页。
-- 本人改密时置回 false。
--
-- 已有账号一律按 false 处理：它们的密码是否本人所设无从判断，强制全员改密
-- 会把现有会话全部打断，收益不抵代价。
-- ============================================================

ALTER TABLE users
  ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN users.must_change_password IS '密码由他人设定，登录后必须先改密；期间除改密端点外一律 403';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
  DROP COLUMN IF EXISTS must_change_password;
-- +goose StatementEnd
