BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'approval_status') THEN
    CREATE TYPE approval_status AS ENUM ('pending', 'approved', 'rejected');
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS approval_requests (
  approval_request_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  request_type          TEXT NOT NULL,
  reference_id          UUID,
  status                approval_status NOT NULL DEFAULT 'pending',
  submitted_by_user_id  UUID,
  reviewed_by_user_id   UUID,
  submitted_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reviewed_at           TIMESTAMPTZ,
  UNIQUE (tenant_id, approval_request_id),
  FOREIGN KEY (tenant_id, submitted_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL,
  FOREIGN KEY (tenant_id, reviewed_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS study_time_logs (
  study_log_id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  user_id               UUID NOT NULL,
  session_id            UUID,
  minutes_logged        INTEGER NOT NULL CHECK (minutes_logged > 0),
  logged_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, study_log_id),
  FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, user_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, session_id) REFERENCES sessions(tenant_id, session_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS content_engagement_events (
  engagement_id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  user_id               UUID,
  document_id           UUID,
  event_type            TEXT NOT NULL,
  occurred_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, engagement_id),
  FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL,
  FOREIGN KEY (tenant_id, document_id) REFERENCES documents(tenant_id, document_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS community_activity_events (
  activity_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  user_id               UUID,
  activity_type         TEXT NOT NULL,
  occurred_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, activity_id),
  FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS analytics_dashboard_daily_summary (
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  metric_date           DATE NOT NULL,
  todays_sessions       INTEGER NOT NULL DEFAULT 0,
  pending_approvals     INTEGER NOT NULL DEFAULT 0,
  computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, metric_date)
);

CREATE TABLE IF NOT EXISTS analytics_dashboard_kpi_daily (
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  metric_date           DATE NOT NULL,
  metric_key            TEXT NOT NULL,
  metric_value          DOUBLE PRECISION NOT NULL DEFAULT 0,
  numerator             DOUBLE PRECISION NOT NULL DEFAULT 0,
  denominator           DOUBLE PRECISION NOT NULL DEFAULT 0,
  computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, metric_date, metric_key)
);

CREATE TABLE IF NOT EXISTS analytics_occupancy_heatmap_daily (
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  metric_date           DATE NOT NULL,
  hour_bucket           SMALLINT NOT NULL CHECK (hour_bucket BETWEEN 0 AND 23),
  room_id               UUID,
  sessions_count        INTEGER NOT NULL DEFAULT 0,
  booked_seats          INTEGER NOT NULL DEFAULT 0,
  total_seats           INTEGER NOT NULL DEFAULT 0,
  occupancy_rate        DOUBLE PRECISION NOT NULL DEFAULT 0,
  computed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, metric_date, hour_bucket, room_id),
  FOREIGN KEY (tenant_id, room_id) REFERENCES rooms(tenant_id, room_id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS analytics_refresh_runs (
  refresh_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id             UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  metric_date           DATE NOT NULL,
  started_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at           TIMESTAMPTZ,
  status                TEXT NOT NULL,
  error_message         TEXT,
  triggered_by_user_id  UUID,
  UNIQUE (tenant_id, refresh_id),
  FOREIGN KEY (tenant_id, triggered_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

ALTER TABLE approval_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE study_time_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE content_engagement_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE community_activity_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_dashboard_daily_summary ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_dashboard_kpi_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_occupancy_heatmap_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_refresh_runs ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_approval_requests ON approval_requests;
CREATE POLICY tenant_isolation_approval_requests ON approval_requests
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_study_time_logs ON study_time_logs;
CREATE POLICY tenant_isolation_study_time_logs ON study_time_logs
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_content_engagement_events ON content_engagement_events;
CREATE POLICY tenant_isolation_content_engagement_events ON content_engagement_events
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_community_activity_events ON community_activity_events;
CREATE POLICY tenant_isolation_community_activity_events ON community_activity_events
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_dashboard_daily_summary ON analytics_dashboard_daily_summary;
CREATE POLICY tenant_isolation_analytics_dashboard_daily_summary ON analytics_dashboard_daily_summary
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_dashboard_kpi_daily ON analytics_dashboard_kpi_daily;
CREATE POLICY tenant_isolation_analytics_dashboard_kpi_daily ON analytics_dashboard_kpi_daily
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_occupancy_heatmap_daily ON analytics_occupancy_heatmap_daily;
CREATE POLICY tenant_isolation_analytics_occupancy_heatmap_daily ON analytics_occupancy_heatmap_daily
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_refresh_runs ON analytics_refresh_runs;
CREATE POLICY tenant_isolation_analytics_refresh_runs ON analytics_refresh_runs
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
