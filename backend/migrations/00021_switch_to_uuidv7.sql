-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 主键 DEFAULT 从 gen_random_uuid()（UUID v4）切换到 uuidv7()
--
-- 原因：
--   - UUID v7 时间有序，B-tree 插入性能更好（不 rebalance）
--   - PG 18 原生支持 uuidv7() / uuid_extract_timestamp()
--   - 随机性仍足够（74 位随机，单 PG 每秒百万级不会撞）
--
-- 已有的 v4 数据保留不动（两种版本 UUID 在同表共存无副作用）
-- 只影响之后新 INSERT（走 DEFAULT）的行
-- ============================================================
ALTER TABLE tenants           ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE subscriptions     ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE users             ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE refresh_tokens    ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE members           ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE payment_methods   ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE services          ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE staff             ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE card_types        ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE cards             ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE transactions      ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE transaction_items ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE commission_rules  ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE member_credits    ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE card_balance_logs ALTER COLUMN id SET DEFAULT uuidv7();
ALTER TABLE appointments      ALTER COLUMN id SET DEFAULT uuidv7();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants           ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE subscriptions     ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE users             ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE refresh_tokens    ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE members           ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE payment_methods   ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE services          ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE staff             ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE card_types        ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE cards             ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE transactions      ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE transaction_items ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE commission_rules  ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE member_credits    ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE card_balance_logs ALTER COLUMN id SET DEFAULT gen_random_uuid();
ALTER TABLE appointments      ALTER COLUMN id SET DEFAULT gen_random_uuid();
-- +goose StatementEnd
