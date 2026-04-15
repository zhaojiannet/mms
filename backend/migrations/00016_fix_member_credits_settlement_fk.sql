-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 修 00012 的外键一致性 bug
--   原 member_credits.settlement_tx_id 为 ON DELETE SET NULL
--   但同表 CHECK 要求 (settled_at IS NULL) = (settlement_tx_id IS NULL)
--   → 硬删清账 transaction 会触发 CHECK 违反
--
-- 改为 ON DELETE RESTRICT：
--   * 清账交易是账目凭证，业务上本不应该硬删
--   * 撤单走 status='voided' 软撤，不触发外键
--   * 保证账目一致性
-- ============================================================
ALTER TABLE member_credits
  DROP CONSTRAINT IF EXISTS member_credits_settlement_tx_id_fkey;

ALTER TABLE member_credits
  ADD CONSTRAINT member_credits_settlement_tx_id_fkey
  FOREIGN KEY (settlement_tx_id) REFERENCES transactions(id) ON DELETE RESTRICT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE member_credits
  DROP CONSTRAINT IF EXISTS member_credits_settlement_tx_id_fkey;

ALTER TABLE member_credits
  ADD CONSTRAINT member_credits_settlement_tx_id_fkey
  FOREIGN KEY (settlement_tx_id) REFERENCES transactions(id) ON DELETE SET NULL;
-- +goose StatementEnd
