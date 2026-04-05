BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS partner_ingestion_sources (
  source_id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                    UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  name                         TEXT NOT NULL,
  base_url                     TEXT NOT NULL,
  is_active                    BOOLEAN NOT NULL DEFAULT TRUE,
  paused_for_manual_review     BOOLEAN NOT NULL DEFAULT FALSE,
  manual_review_reason         TEXT,
  schedule_interval_minutes    INTEGER NOT NULL DEFAULT 60 CHECK (schedule_interval_minutes >= 5),
  schedule_jitter_seconds      INTEGER NOT NULL DEFAULT 120 CHECK (schedule_jitter_seconds >= 0 AND schedule_jitter_seconds <= 3600),
  rate_limit_per_minute        INTEGER NOT NULL DEFAULT 6 CHECK (rate_limit_per_minute >= 1 AND rate_limit_per_minute <= 120),
  request_timeout_seconds      INTEGER NOT NULL DEFAULT 20 CHECK (request_timeout_seconds >= 5 AND request_timeout_seconds <= 120),
  next_run_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_run_at                  TIMESTAMPTZ,
  created_by_user_id           UUID NOT NULL,
  updated_by_user_id           UUID,
  created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, source_id),
  UNIQUE (tenant_id, name),
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS partner_ingestion_proxies (
  proxy_id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                    UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  proxy_url                    TEXT NOT NULL,
  is_active                    BOOLEAN NOT NULL DEFAULT TRUE,
  failure_count                INTEGER NOT NULL DEFAULT 0,
  last_used_at                 TIMESTAMPTZ,
  created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, proxy_url)
);

CREATE TABLE IF NOT EXISTS partner_ingestion_user_agents (
  user_agent_id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                    UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  user_agent                   TEXT NOT NULL,
  is_active                    BOOLEAN NOT NULL DEFAULT TRUE,
  created_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, user_agent)
);

CREATE TABLE IF NOT EXISTS partner_portal_sessions (
  tenant_id                    UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  source_id                    UUID NOT NULL,
  cookies_json                 JSONB NOT NULL DEFAULT '{}'::jsonb,
  expires_at                   TIMESTAMPTZ,
  updated_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, source_id),
  FOREIGN KEY (tenant_id, source_id) REFERENCES partner_ingestion_sources(tenant_id, source_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS partner_ingestion_runs (
  run_id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                    UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  source_id                    UUID NOT NULL,
  trigger_type                 TEXT NOT NULL CHECK (trigger_type IN ('manual', 'scheduled')),
  status                       TEXT NOT NULL CHECK (status IN ('success', 'failed', 'paused_manual_review', 'rate_limited')),
  proxy_url                    TEXT,
  user_agent                   TEXT,
  http_status                  INTEGER,
  response_bytes               BIGINT,
  records_processed            INTEGER NOT NULL DEFAULT 0,
  error_message                TEXT,
  started_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at                 TIMESTAMPTZ,
  next_run_at                  TIMESTAMPTZ,
  UNIQUE (tenant_id, run_id),
  FOREIGN KEY (tenant_id, source_id) REFERENCES partner_ingestion_sources(tenant_id, source_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_partner_ingestion_sources_due
  ON partner_ingestion_sources (tenant_id, is_active, paused_for_manual_review, next_run_at);

CREATE INDEX IF NOT EXISTS idx_partner_ingestion_runs_recent
  ON partner_ingestion_runs (tenant_id, source_id, started_at DESC);

CREATE TABLE IF NOT EXISTS partner_ingested_records (
  record_id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                     UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  source_id                     UUID NOT NULL,
  external_id                   TEXT NOT NULL,
  document_id                   UUID,
  normalized_title              TEXT NOT NULL,
  normalized_category           TEXT NOT NULL,
  normalized_metadata           JSONB NOT NULL DEFAULT '{}'::jsonb,
  content_checksum              TEXT NOT NULL,
  ingested_at                   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, source_id, external_id),
  FOREIGN KEY (tenant_id, source_id) REFERENCES partner_ingestion_sources(tenant_id, source_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, document_id) REFERENCES documents(tenant_id, document_id) ON DELETE SET NULL
);

ALTER TABLE partner_ingestion_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE partner_ingestion_proxies ENABLE ROW LEVEL SECURITY;
ALTER TABLE partner_ingestion_user_agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE partner_portal_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE partner_ingestion_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE partner_ingested_records ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_partner_ingestion_sources ON partner_ingestion_sources;
CREATE POLICY tenant_isolation_partner_ingestion_sources ON partner_ingestion_sources
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_partner_ingestion_proxies ON partner_ingestion_proxies;
CREATE POLICY tenant_isolation_partner_ingestion_proxies ON partner_ingestion_proxies
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_partner_ingestion_user_agents ON partner_ingestion_user_agents;
CREATE POLICY tenant_isolation_partner_ingestion_user_agents ON partner_ingestion_user_agents
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_partner_portal_sessions ON partner_portal_sessions;
CREATE POLICY tenant_isolation_partner_portal_sessions ON partner_portal_sessions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_partner_ingestion_runs ON partner_ingestion_runs;
CREATE POLICY tenant_isolation_partner_ingestion_runs ON partner_ingestion_runs
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_partner_ingested_records ON partner_ingested_records;
CREATE POLICY tenant_isolation_partner_ingested_records ON partner_ingested_records
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
