BEGIN;

CREATE TABLE IF NOT EXISTS academic_terms (
  term_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  name           TEXT NOT NULL,
  start_date     DATE NOT NULL,
  end_date       DATE NOT NULL,
  is_active      BOOLEAN NOT NULL DEFAULT TRUE,
  lock_version   INTEGER NOT NULL DEFAULT 0,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (end_date >= start_date),
  UNIQUE (tenant_id, term_id)
);

CREATE INDEX IF NOT EXISTS idx_academic_terms_tenant_dates
  ON academic_terms (tenant_id, start_date, end_date)
  WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS calendar_time_slot_rules (
  rule_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  room_id         UUID,
  weekday         SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
  slot_start      TIME NOT NULL,
  slot_end        TIME NOT NULL,
  is_active       BOOLEAN NOT NULL DEFAULT TRUE,
  lock_version    INTEGER NOT NULL DEFAULT 0,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (slot_end > slot_start),
  UNIQUE (tenant_id, rule_id),
  FOREIGN KEY (tenant_id, room_id) REFERENCES rooms(tenant_id, room_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_slot_rules_tenant_weekday
  ON calendar_time_slot_rules (tenant_id, weekday, slot_start, slot_end)
  WHERE is_active = TRUE;

CREATE TABLE IF NOT EXISTS calendar_blackout_dates (
  blackout_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  room_id          UUID,
  blackout_date    DATE NOT NULL,
  reason           TEXT NOT NULL,
  is_active        BOOLEAN NOT NULL DEFAULT TRUE,
  lock_version     INTEGER NOT NULL DEFAULT 0,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, blackout_id),
  FOREIGN KEY (tenant_id, room_id) REFERENCES rooms(tenant_id, room_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_blackout_tenant_date
  ON calendar_blackout_dates (tenant_id, blackout_date)
  WHERE is_active = TRUE;

ALTER TABLE academic_terms ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar_time_slot_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE calendar_blackout_dates ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_academic_terms ON academic_terms;
CREATE POLICY tenant_isolation_academic_terms ON academic_terms
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_slot_rules ON calendar_time_slot_rules;
CREATE POLICY tenant_isolation_slot_rules ON calendar_time_slot_rules
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_blackouts ON calendar_blackout_dates;
CREATE POLICY tenant_isolation_blackouts ON calendar_blackout_dates
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
