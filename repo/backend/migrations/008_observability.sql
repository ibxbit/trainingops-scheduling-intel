BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'report_format') THEN
    CREATE TYPE report_format AS ENUM ('csv', 'pdf');
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'report_frequency') THEN
    CREATE TYPE report_frequency AS ENUM ('daily', 'weekly');
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS workflow_logs (
  workflow_log_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  actor_user_id         UUID,
  workflow_name         TEXT NOT NULL,
  resource_id           TEXT,
  outcome               TEXT NOT NULL,
  status_code           INTEGER,
  latency_ms            INTEGER,
  details               JSONB NOT NULL DEFAULT '{}'::jsonb,
  occurred_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  FOREIGN KEY (tenant_id, actor_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_logs_tenant_time
  ON workflow_logs (tenant_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_logs_tenant_name
  ON workflow_logs (tenant_id, workflow_name, occurred_at DESC);

CREATE TABLE IF NOT EXISTS scraping_errors (
  scraping_error_id     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  source_name           TEXT NOT NULL,
  error_code            TEXT,
  error_message         TEXT NOT NULL,
  occurred_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  metadata              JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_scraping_errors_tenant_time
  ON scraping_errors (tenant_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS anomaly_events (
  anomaly_event_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  anomaly_date          DATE NOT NULL,
  anomaly_type          TEXT NOT NULL,
  severity              TEXT NOT NULL,
  observed_value        DOUBLE PRECISION NOT NULL,
  baseline_value        DOUBLE PRECISION NOT NULL,
  threshold_value       DOUBLE PRECISION NOT NULL,
  details               JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, anomaly_date, anomaly_type)
);

CREATE TABLE IF NOT EXISTS report_schedules (
  schedule_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  name                  TEXT NOT NULL,
  format                report_format NOT NULL,
  frequency             report_frequency NOT NULL,
  output_folder         TEXT NOT NULL,
  is_active             BOOLEAN NOT NULL DEFAULT TRUE,
  next_run_at           TIMESTAMPTZ NOT NULL,
  created_by_user_id    UUID,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, schedule_id),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_report_schedules_due
  ON report_schedules (tenant_id, is_active, next_run_at);

CREATE TABLE IF NOT EXISTS report_exports (
  export_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  schedule_id           UUID,
  report_date           DATE NOT NULL,
  format                report_format NOT NULL,
  file_path             TEXT,
  file_size_bytes       BIGINT,
  status                TEXT NOT NULL,
  error_message         TEXT,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  FOREIGN KEY (tenant_id, schedule_id) REFERENCES report_schedules(tenant_id, schedule_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_report_exports_tenant_date
  ON report_exports (tenant_id, report_date DESC, created_at DESC);

ALTER TABLE workflow_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE scraping_errors ENABLE ROW LEVEL SECURITY;
ALTER TABLE anomaly_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_schedules ENABLE ROW LEVEL SECURITY;
ALTER TABLE report_exports ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_workflow_logs ON workflow_logs;
CREATE POLICY tenant_isolation_workflow_logs ON workflow_logs
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_scraping_errors ON scraping_errors;
CREATE POLICY tenant_isolation_scraping_errors ON scraping_errors
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_anomaly_events ON anomaly_events;
CREATE POLICY tenant_isolation_anomaly_events ON anomaly_events
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_report_schedules ON report_schedules;
CREATE POLICY tenant_isolation_report_schedules ON report_schedules
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_report_exports ON report_exports;
CREATE POLICY tenant_isolation_report_exports ON report_exports
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
