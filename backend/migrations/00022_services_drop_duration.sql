-- +goose Up
-- +goose StatementBegin

-- 移除 services.duration_min
-- 阶段 1 不做按时段预约，此字段一直空置无人使用；老系统亦无此字段。
ALTER TABLE services DROP COLUMN IF EXISTS duration_min;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE services ADD COLUMN duration_min INT;
-- +goose StatementEnd
