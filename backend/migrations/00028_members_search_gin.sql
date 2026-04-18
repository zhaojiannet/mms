-- +goose Up
-- +goose StatementBegin

-- pg_trgm GIN 索引：members 搜索性能优化
-- 原 ILIKE '%xx%' 走全表 seq scan，上万会员后搜索慢
-- GIN trgm 索引让 LIKE '%xx%' 走索引，大幅加速

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_members_name_trgm
  ON members USING GIN (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_members_phone_trgm
  ON members USING GIN (phone gin_trgm_ops);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_members_phone_trgm;
DROP INDEX IF EXISTS idx_members_name_trgm;
-- pg_trgm extension 保留，多处可能复用
-- +goose StatementEnd
