-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- 预约 appointments + 预约-服务关联 appointment_services
--
--   - customer_name / customer_phone 都 NOT NULL（散客也要留）
--   - member_id 可空（对应散客）
--   - assigned_staff_id 可空（前台先收单、后续指派）
--   - source: 管理端(staff) / 公开 booking(booking)，冲突检测规则可能不同
--   - transaction_id UNIQUE：预约完成后关联到消费交易（一对一）
-- ============================================================
CREATE TABLE appointments (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  member_id           UUID REFERENCES members(id) ON DELETE SET NULL,
  customer_name       TEXT NOT NULL,
  customer_phone      TEXT NOT NULL,
  appointment_time    TIMESTAMPTZ NOT NULL,
  assigned_staff_id   UUID REFERENCES staff(id) ON DELETE SET NULL,
  status              TEXT NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','confirmed','completed','cancelled','no_show')),
  transaction_id      UUID REFERENCES transactions(id) ON DELETE SET NULL,
  source              TEXT NOT NULL DEFAULT 'staff'
                      CHECK (source IN ('staff','booking')),
  notes               TEXT,
  legacy_id           TEXT,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_appointments_tenant_time
  ON appointments(tenant_id, appointment_time);
CREATE INDEX idx_appointments_tenant_status_time
  ON appointments(tenant_id, status, appointment_time);
CREATE INDEX idx_appointments_member
  ON appointments(tenant_id, member_id)
  WHERE member_id IS NOT NULL;
CREATE INDEX idx_appointments_staff_time
  ON appointments(tenant_id, assigned_staff_id, appointment_time)
  WHERE assigned_staff_id IS NOT NULL;
CREATE INDEX idx_appointments_phone_time
  ON appointments(tenant_id, customer_phone, appointment_time);
CREATE UNIQUE INDEX uq_appointments_transaction
  ON appointments(transaction_id) WHERE transaction_id IS NOT NULL;
CREATE UNIQUE INDEX uq_appointments_tenant_legacy_id
  ON appointments(tenant_id, legacy_id) WHERE legacy_id IS NOT NULL;

ALTER TABLE appointments ENABLE ROW LEVEL SECURITY;
ALTER TABLE appointments FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_appointments ON appointments
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- ============================================================
-- 预约 - 服务 多对多关联
--   - tenant_id 冗余写入，方便 RLS 直接隔离（无需经 appointments 间接查）
-- ============================================================
CREATE TABLE appointment_services (
  tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  appointment_id  UUID NOT NULL REFERENCES appointments(id) ON DELETE CASCADE,
  service_id      UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  PRIMARY KEY (appointment_id, service_id)
);

CREATE INDEX idx_appointment_services_service
  ON appointment_services(tenant_id, service_id);

ALTER TABLE appointment_services ENABLE ROW LEVEL SECURITY;
ALTER TABLE appointment_services FORCE  ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_appointment_services ON appointment_services
  USING (tenant_id = current_tenant_id())
  WITH CHECK (tenant_id = current_tenant_id());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS appointment_services CASCADE;
DROP TABLE IF EXISTS appointments CASCADE;
-- +goose StatementEnd
