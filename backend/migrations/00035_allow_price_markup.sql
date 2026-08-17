-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 放开手动定价上限：允许实付高于标价（加价）
--
-- 00031 加的 discount_amount >= 0 约束当时就标了 NOT VALID，因为老系统
-- 迁移来的历史交易本就存在实付高于标价的真实记录。现在手动定价按产品
-- 决定放开上限，新写入同样会产生 discount_amount < 0，约束整个移除。
-- ============================================================

ALTER TABLE transactions
  DROP CONSTRAINT IF EXISTS transactions_discount_non_negative;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions
  ADD CONSTRAINT transactions_discount_non_negative
  CHECK (discount_amount >= 0) NOT VALID;
-- +goose StatementEnd
