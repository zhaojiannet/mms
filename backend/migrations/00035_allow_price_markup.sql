-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 放开手动定价上限：允许实付高于标价（加价）
--
-- 00031 加的 discount_amount >= 0 约束当时就标了 NOT VALID，因为老系统
-- 迁移来的历史交易本就存在实付高于标价的真实记录。现在手动定价按产品
-- 决定放开上限，新写入同样会产生 discount_amount < 0，约束整个移除。
-- ============================================================

-- ALTER 要拿 transactions 的 ACCESS EXCLUSIVE 锁，可能撞上 pg_dump 备份
-- 窗口（持 ACCESS SHARE 数十秒）：解除 DSN 带来的 5s 语句超时，改为最长
-- 等锁 60s，等过备份而不是超时进崩溃重启循环
SET LOCAL statement_timeout = 0;
SET LOCAL lock_timeout = '60s';

ALTER TABLE transactions
  DROP CONSTRAINT IF EXISTS transactions_discount_non_negative;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- 不重加约束：放开后产生的负 discount 行会让之后任何 UPDATE（如撤单只改
-- status 也整行重估 CHECK）直接失败。本仓库的回滚路径是 pg_dump 恢复，
-- goose down 只回退版本号
SELECT 1;
-- +goose StatementEnd
