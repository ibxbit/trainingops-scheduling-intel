BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS learner_cohorts (
  cohort_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  cohort_code         TEXT NOT NULL,
  name                TEXT NOT NULL,
  is_active           BOOLEAN NOT NULL DEFAULT TRUE,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, cohort_id),
  UNIQUE (tenant_id, cohort_code)
);

CREATE TABLE IF NOT EXISTS learner_cohort_memberships (
  tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  cohort_id           UUID NOT NULL,
  learner_user_id     UUID NOT NULL,
  joined_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  left_at             TIMESTAMPTZ,
  PRIMARY KEY (tenant_id, cohort_id, learner_user_id),
  FOREIGN KEY (tenant_id, cohort_id) REFERENCES learner_cohorts(tenant_id, cohort_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, learner_user_id) REFERENCES users(tenant_id, user_id) ON DELETE CASCADE,
  CHECK (left_at IS NULL OR left_at >= joined_at)
);

CREATE TABLE IF NOT EXISTS analytics_feature_batch_runs (
  batch_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  feature_date        DATE NOT NULL,
  window_days         INTEGER NOT NULL CHECK (window_days IN (7, 30, 90)),
  status              TEXT NOT NULL CHECK (status IN ('running', 'success', 'failed')),
  error_message       TEXT,
  triggered_by_user_id UUID,
  started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at         TIMESTAMPTZ,
  UNIQUE (tenant_id, batch_id)
);

CREATE TABLE IF NOT EXISTS analytics_learner_features_daily (
  tenant_id                UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  feature_date             DATE NOT NULL,
  window_days              INTEGER NOT NULL CHECK (window_days IN (7, 30, 90)),
  learner_user_id          UUID NOT NULL,
  sessions_booked          INTEGER NOT NULL DEFAULT 0,
  sessions_attended        INTEGER NOT NULL DEFAULT 0,
  attendance_rate          DOUBLE PRECISION NOT NULL DEFAULT 0,
  active_days              INTEGER NOT NULL DEFAULT 0,
  study_minutes            INTEGER NOT NULL DEFAULT 0,
  content_previews         INTEGER NOT NULL DEFAULT 0,
  content_downloads        INTEGER NOT NULL DEFAULT 0,
  community_events         INTEGER NOT NULL DEFAULT 0,
  engagement_score         DOUBLE PRECISION NOT NULL DEFAULT 0,
  segment                  TEXT NOT NULL,
  computed_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, feature_date, window_days, learner_user_id),
  FOREIGN KEY (tenant_id, learner_user_id) REFERENCES users(tenant_id, user_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_analytics_learner_features_segment
  ON analytics_learner_features_daily (tenant_id, feature_date, window_days, segment);

CREATE TABLE IF NOT EXISTS analytics_cohort_features_daily (
  tenant_id                 UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  feature_date              DATE NOT NULL,
  window_days               INTEGER NOT NULL CHECK (window_days IN (7, 30, 90)),
  cohort_id                 UUID NOT NULL,
  members_count             INTEGER NOT NULL DEFAULT 0,
  active_learners           INTEGER NOT NULL DEFAULT 0,
  avg_attendance_rate       DOUBLE PRECISION NOT NULL DEFAULT 0,
  avg_study_minutes         DOUBLE PRECISION NOT NULL DEFAULT 0,
  avg_engagement_score      DOUBLE PRECISION NOT NULL DEFAULT 0,
  segment_distribution      JSONB NOT NULL DEFAULT '{}'::jsonb,
  computed_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, feature_date, window_days, cohort_id),
  FOREIGN KEY (tenant_id, cohort_id) REFERENCES learner_cohorts(tenant_id, cohort_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS analytics_reporting_metrics_daily (
  tenant_id                 UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  feature_date              DATE NOT NULL,
  window_days               INTEGER NOT NULL CHECK (window_days IN (7, 30, 90)),
  metric_key                TEXT NOT NULL,
  metric_value              DOUBLE PRECISION NOT NULL,
  numerator                 DOUBLE PRECISION,
  denominator               DOUBLE PRECISION,
  computed_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, feature_date, window_days, metric_key)
);

ALTER TABLE learner_cohorts ENABLE ROW LEVEL SECURITY;
ALTER TABLE learner_cohort_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_feature_batch_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_learner_features_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_cohort_features_daily ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_reporting_metrics_daily ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_learner_cohorts ON learner_cohorts;
CREATE POLICY tenant_isolation_learner_cohorts ON learner_cohorts
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_learner_cohort_memberships ON learner_cohort_memberships;
CREATE POLICY tenant_isolation_learner_cohort_memberships ON learner_cohort_memberships
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_feature_batch_runs ON analytics_feature_batch_runs;
CREATE POLICY tenant_isolation_analytics_feature_batch_runs ON analytics_feature_batch_runs
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_learner_features_daily ON analytics_learner_features_daily;
CREATE POLICY tenant_isolation_analytics_learner_features_daily ON analytics_learner_features_daily
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_cohort_features_daily ON analytics_cohort_features_daily;
CREATE POLICY tenant_isolation_analytics_cohort_features_daily ON analytics_cohort_features_daily
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_analytics_reporting_metrics_daily ON analytics_reporting_metrics_daily;
CREATE POLICY tenant_isolation_analytics_reporting_metrics_daily ON analytics_reporting_metrics_daily
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
