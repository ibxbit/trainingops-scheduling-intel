package dashboard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"trainingops/backend/internal/dbctx"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) StartRefresh(ctx context.Context, tenantID, userID, metricDate string) (string, error) {
	var refreshID string
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO analytics_refresh_runs (tenant_id, metric_date, status, triggered_by_user_id)
VALUES ($1::uuid, $2::date, 'running', $3::uuid)
RETURNING refresh_id::text
`, tenantID, metricDate, userID).Scan(&refreshID)
	return refreshID, err
}

func (r *Repository) FinishRefresh(ctx context.Context, tenantID, refreshID string, errMsg *string) error {
	status := "success"
	if errMsg != nil {
		status = "failed"
	}
	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE analytics_refresh_runs
SET status = $3, error_message = $4, finished_at = NOW()
WHERE tenant_id::text = $1 AND refresh_id::text = $2
`, tenantID, refreshID, status, errMsg)
	return err
}

func (r *Repository) Precompute(ctx context.Context, tenantID, metricDate string) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
INSERT INTO analytics_dashboard_daily_summary (tenant_id, metric_date, todays_sessions, pending_approvals, computed_at)
VALUES (
  $1::uuid,
  $2::date,
  (SELECT COUNT(*) FROM sessions s WHERE s.tenant_id::text = $1 AND s.starts_at::date = $2::date),
  (SELECT COUNT(*) FROM approval_requests a WHERE a.tenant_id::text = $1 AND a.status = 'pending'),
  NOW()
)
ON CONFLICT (tenant_id, metric_date) DO UPDATE SET
  todays_sessions = EXCLUDED.todays_sessions,
  pending_approvals = EXCLUDED.pending_approvals,
  computed_at = NOW()
`, tenantID, metricDate)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM analytics_occupancy_heatmap_daily WHERE tenant_id::text = $1 AND metric_date = $2::date`, tenantID, metricDate)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO analytics_occupancy_heatmap_daily (
  tenant_id, metric_date, hour_bucket, room_id, sessions_count, booked_seats, total_seats, occupancy_rate, computed_at
)
SELECT
  s.tenant_id,
  $2::date,
  EXTRACT(HOUR FROM s.starts_at)::smallint AS hour_bucket,
  s.room_id,
  COUNT(DISTINCT s.session_id) AS sessions_count,
  COALESCE(SUM(booked.booked_count), 0) AS booked_seats,
  COALESCE(SUM(s.capacity), 0) AS total_seats,
  CASE WHEN COALESCE(SUM(s.capacity), 0) = 0 THEN 0
       ELSE COALESCE(SUM(booked.booked_count), 0)::double precision / SUM(s.capacity)::double precision END AS occupancy_rate,
  NOW()
FROM sessions s
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS booked_count
  FROM bookings b
  WHERE b.tenant_id = s.tenant_id
    AND b.session_id = s.session_id
    AND (b.state IN ('confirmed', 'rescheduled', 'checked_in') OR (b.state = 'held' AND b.hold_expires_at > NOW()))
) booked ON true
WHERE s.tenant_id::text = $1
  AND s.starts_at::date = $2::date
GROUP BY s.tenant_id, EXTRACT(HOUR FROM s.starts_at), s.room_id
`, tenantID, metricDate)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM analytics_dashboard_kpi_daily WHERE tenant_id::text = $1 AND metric_date = $2::date`, tenantID, metricDate)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
WITH current_enrollment AS (
  SELECT COUNT(DISTINCT b.learner_user_id) AS n
  FROM bookings b
  JOIN sessions s ON s.tenant_id = b.tenant_id AND s.session_id = b.session_id
  WHERE b.tenant_id::text = $1
    AND s.starts_at::date BETWEEN ($2::date - INTERVAL '6 days')::date AND $2::date
    AND b.state IN ('confirmed', 'rescheduled', 'checked_in')
),
previous_enrollment AS (
  SELECT COUNT(DISTINCT b.learner_user_id) AS n
  FROM bookings b
  JOIN sessions s ON s.tenant_id = b.tenant_id AND s.session_id = b.session_id
  WHERE b.tenant_id::text = $1
    AND s.starts_at::date BETWEEN ($2::date - INTERVAL '13 days')::date AND ($2::date - INTERVAL '7 days')::date
    AND b.state IN ('confirmed', 'rescheduled', 'checked_in')
),
repeat_attendance AS (
  SELECT COUNT(*)::double precision AS numerator,
         NULLIF((SELECT COUNT(DISTINCT learner_user_id) FROM bookings WHERE tenant_id::text = $1 AND state = 'checked_in' AND created_at::date <= $2::date), 0)::double precision AS denominator
  FROM (
    SELECT learner_user_id
    FROM bookings
    WHERE tenant_id::text = $1
      AND state = 'checked_in'
      AND created_at::date BETWEEN ($2::date - INTERVAL '30 days')::date AND $2::date
    GROUP BY learner_user_id
    HAVING COUNT(*) >= 2
  ) t
),
study_time AS (
  SELECT COALESCE(SUM(minutes_logged), 0)::double precision AS total
  FROM study_time_logs
  WHERE tenant_id::text = $1
    AND logged_at::date BETWEEN ($2::date - INTERVAL '6 days')::date AND $2::date
),
content_conv AS (
  SELECT
    COALESCE(SUM(CASE WHEN event_type = 'download' THEN 1 ELSE 0 END), 0)::double precision AS downloads,
    NULLIF(COALESCE(SUM(CASE WHEN event_type = 'preview' THEN 1 ELSE 0 END), 0), 0)::double precision AS previews
  FROM content_engagement_events
  WHERE tenant_id::text = $1
    AND occurred_at::date BETWEEN ($2::date - INTERVAL '30 days')::date AND $2::date
),
community AS (
  SELECT COUNT(*)::double precision AS total
  FROM community_activity_events
  WHERE tenant_id::text = $1
    AND occurred_at::date BETWEEN ($2::date - INTERVAL '6 days')::date AND $2::date
)
INSERT INTO analytics_dashboard_kpi_daily (tenant_id, metric_date, metric_key, metric_value, numerator, denominator, computed_at)
SELECT $1::uuid, $2::date, metric_key, metric_value, numerator, denominator, NOW()
FROM (
  SELECT
    'enrollment_growth'::text AS metric_key,
    CASE WHEN pe.n = 0 THEN 0 ELSE ((ce.n - pe.n)::double precision / pe.n::double precision) END AS metric_value,
    ce.n::double precision AS numerator,
    NULLIF(pe.n, 0)::double precision AS denominator
  FROM current_enrollment ce, previous_enrollment pe

  UNION ALL

  SELECT
    'repeat_attendance',
    CASE WHEN ra.denominator IS NULL THEN 0 ELSE (ra.numerator / ra.denominator) END,
    ra.numerator,
    COALESCE(ra.denominator, 0)
  FROM repeat_attendance ra

  UNION ALL

  SELECT 'study_time_logged', st.total, st.total, 1 FROM study_time st

  UNION ALL

  SELECT
    'content_conversion',
    CASE WHEN cc.previews IS NULL THEN 0 ELSE (cc.downloads / cc.previews) END,
    cc.downloads,
    COALESCE(cc.previews, 0)
  FROM content_conv cc

  UNION ALL

  SELECT 'community_activity', c.total, c.total, 1 FROM community c
) q
`, tenantID, metricDate)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) Overview(ctx context.Context, tenantID, metricDate string) (*Overview, error) {
	out := &Overview{}
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT metric_date::text, todays_sessions, pending_approvals
FROM analytics_dashboard_daily_summary
WHERE tenant_id::text = $1 AND metric_date = $2::date
`, tenantID, metricDate).Scan(&out.Summary.MetricDate, &out.Summary.TodaysSessions, &out.Summary.PendingApprovals)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	kRows, err := dbctx.QueryContext(ctx, r.db, `
SELECT metric_key, metric_value, numerator, denominator
FROM analytics_dashboard_kpi_daily
WHERE tenant_id::text = $1 AND metric_date = $2::date
ORDER BY metric_key
`, tenantID, metricDate)
	if err != nil {
		return nil, err
	}
	defer kRows.Close()
	out.KPIs = make([]KPI, 0)
	for kRows.Next() {
		var k KPI
		if err := kRows.Scan(&k.MetricKey, &k.MetricValue, &k.Numerator, &k.Denominator); err != nil {
			return nil, err
		}
		out.KPIs = append(out.KPIs, k)
	}
	if err := kRows.Err(); err != nil {
		return nil, err
	}

	hRows, err := dbctx.QueryContext(ctx, r.db, `
SELECT hour_bucket, room_id::text, sessions_count, booked_seats, total_seats, occupancy_rate
FROM analytics_occupancy_heatmap_daily
WHERE tenant_id::text = $1 AND metric_date = $2::date
ORDER BY hour_bucket, room_id
`, tenantID, metricDate)
	if err != nil {
		return nil, err
	}
	defer hRows.Close()
	out.Heatmap = make([]HeatmapCell, 0)
	for hRows.Next() {
		var c HeatmapCell
		var room sql.NullString
		if err := hRows.Scan(&c.HourBucket, &room, &c.SessionsCount, &c.BookedSeats, &c.TotalSeats, &c.OccupancyRate); err != nil {
			return nil, err
		}
		if room.Valid {
			c.RoomID = &room.String
		}
		out.Heatmap = append(out.Heatmap, c)
	}
	if err := hRows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (r *Repository) StartFeatureBatch(ctx context.Context, tenantID, userID, featureDate string, windowDays int) (string, error) {
	var batchID string
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO analytics_feature_batch_runs (tenant_id, feature_date, window_days, status, triggered_by_user_id)
VALUES ($1::uuid, $2::date, $3, 'running', $4::uuid)
RETURNING batch_id::text
`, tenantID, featureDate, windowDays, userID).Scan(&batchID)
	return batchID, err
}

func (r *Repository) FinishFeatureBatch(ctx context.Context, tenantID, batchID string, errMsg *string) error {
	status := "success"
	if errMsg != nil {
		status = "failed"
	}
	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE analytics_feature_batch_runs
SET status = $3, error_message = $4, finished_at = NOW()
WHERE tenant_id::text = $1 AND batch_id::text = $2
`, tenantID, batchID, status, errMsg)
	return err
}

func (r *Repository) ComputeFeatureWindow(ctx context.Context, tenantID, featureDate string, windowDays int) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
DELETE FROM analytics_learner_features_daily
WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3
`, tenantID, featureDate, windowDays)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
WITH bounds AS (
  SELECT ($2::date - (($3 - 1) * INTERVAL '1 day'))::date AS start_date,
         $2::date AS end_date
),
booked AS (
  SELECT b.learner_user_id,
         COUNT(*)::int AS sessions_booked,
         COUNT(*) FILTER (WHERE b.state = 'checked_in')::int AS sessions_attended
  FROM bookings b
  JOIN sessions s ON s.tenant_id = b.tenant_id AND s.session_id = b.session_id
  JOIN bounds bd ON true
  WHERE b.tenant_id::text = $1
    AND s.starts_at::date BETWEEN bd.start_date AND bd.end_date
    AND b.state IN ('confirmed', 'rescheduled', 'checked_in')
  GROUP BY b.learner_user_id
),
study AS (
  SELECT st.user_id AS learner_user_id,
         COALESCE(SUM(st.minutes_logged), 0)::int AS study_minutes,
         COUNT(DISTINCT st.logged_at::date)::int AS active_days
  FROM study_time_logs st
  JOIN bounds bd ON true
  WHERE st.tenant_id::text = $1
    AND st.logged_at::date BETWEEN bd.start_date AND bd.end_date
  GROUP BY st.user_id
),
content AS (
  SELECT ce.user_id AS learner_user_id,
         COUNT(*) FILTER (WHERE ce.event_type = 'preview')::int AS content_previews,
         COUNT(*) FILTER (WHERE ce.event_type = 'download')::int AS content_downloads
  FROM content_engagement_events ce
  JOIN bounds bd ON true
  WHERE ce.tenant_id::text = $1
    AND ce.occurred_at::date BETWEEN bd.start_date AND bd.end_date
  GROUP BY ce.user_id
),
community AS (
  SELECT ca.user_id AS learner_user_id,
         COUNT(*)::int AS community_events
  FROM community_activity_events ca
  JOIN bounds bd ON true
  WHERE ca.tenant_id::text = $1
    AND ca.occurred_at::date BETWEEN bd.start_date AND bd.end_date
  GROUP BY ca.user_id
),
keys AS (
  SELECT learner_user_id FROM booked
  UNION
  SELECT learner_user_id FROM study
  UNION
  SELECT learner_user_id FROM content
  UNION
  SELECT learner_user_id FROM community
)
INSERT INTO analytics_learner_features_daily (
  tenant_id, feature_date, window_days, learner_user_id,
  sessions_booked, sessions_attended, attendance_rate, active_days,
  study_minutes, content_previews, content_downloads, community_events,
  engagement_score, segment, computed_at
)
SELECT
  $1::uuid,
  $2::date,
  $3,
  k.learner_user_id,
  COALESCE(b.sessions_booked, 0),
  COALESCE(b.sessions_attended, 0),
  CASE WHEN COALESCE(b.sessions_booked, 0) = 0 THEN 0
       ELSE COALESCE(b.sessions_attended, 0)::double precision / b.sessions_booked::double precision END,
  COALESCE(st.active_days, 0),
  COALESCE(st.study_minutes, 0),
  COALESCE(c.content_previews, 0),
  COALESCE(c.content_downloads, 0),
  COALESCE(cm.community_events, 0),
  (
    (CASE WHEN COALESCE(b.sessions_booked, 0) = 0 THEN 0 ELSE COALESCE(b.sessions_attended, 0)::double precision / b.sessions_booked::double precision END) * 0.45
    + LEAST(COALESCE(st.study_minutes, 0)::double precision / 300.0, 1.0) * 0.30
    + LEAST((COALESCE(c.content_previews, 0) + COALESCE(c.content_downloads, 0))::double precision / 30.0, 1.0) * 0.15
    + LEAST(COALESCE(cm.community_events, 0)::double precision / 20.0, 1.0) * 0.10
  ) AS engagement_score,
  CASE
    WHEN (
      (CASE WHEN COALESCE(b.sessions_booked, 0) = 0 THEN 0 ELSE COALESCE(b.sessions_attended, 0)::double precision / b.sessions_booked::double precision END) >= 0.80
      AND COALESCE(st.study_minutes, 0) >= 300
    ) THEN 'high_engagement'
    WHEN (
      (CASE WHEN COALESCE(b.sessions_booked, 0) = 0 THEN 0 ELSE COALESCE(b.sessions_attended, 0)::double precision / b.sessions_booked::double precision END) >= 0.50
      OR COALESCE(st.study_minutes, 0) >= 120
      OR COALESCE(c.content_previews, 0) >= 8
    ) THEN 'steady'
    WHEN COALESCE(b.sessions_booked, 0) > 0 THEN 'at_risk'
    ELSE 'inactive'
  END AS segment,
  NOW()
FROM keys k
LEFT JOIN booked b ON b.learner_user_id = k.learner_user_id
LEFT JOIN study st ON st.learner_user_id = k.learner_user_id
LEFT JOIN content c ON c.learner_user_id = k.learner_user_id
LEFT JOIN community cm ON cm.learner_user_id = k.learner_user_id
`, tenantID, featureDate, windowDays)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
DELETE FROM analytics_cohort_features_daily
WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3
`, tenantID, featureDate, windowDays)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
WITH cohort_rollup AS (
  SELECT
    m.cohort_id,
    COUNT(*)::int AS members_count,
    COUNT(*) FILTER (WHERE lf.sessions_attended > 0 OR lf.study_minutes > 0)::int AS active_learners,
    COALESCE(AVG(lf.attendance_rate), 0)::double precision AS avg_attendance_rate,
    COALESCE(AVG(lf.study_minutes), 0)::double precision AS avg_study_minutes,
    COALESCE(AVG(lf.engagement_score), 0)::double precision AS avg_engagement_score
  FROM learner_cohort_memberships m
  LEFT JOIN analytics_learner_features_daily lf
    ON lf.tenant_id::text = m.tenant_id::text
   AND lf.feature_date = $2::date
   AND lf.window_days = $3
   AND lf.learner_user_id::text = m.learner_user_id::text
  WHERE m.tenant_id::text = $1
    AND m.joined_at::date <= $2::date
    AND (m.left_at IS NULL OR m.left_at::date > $2::date)
  GROUP BY m.cohort_id
),
segment_dist AS (
  SELECT
    m.cohort_id,
    lf.segment,
    COUNT(*)::int AS n
  FROM learner_cohort_memberships m
  JOIN analytics_learner_features_daily lf
    ON lf.tenant_id::text = m.tenant_id::text
   AND lf.feature_date = $2::date
   AND lf.window_days = $3
   AND lf.learner_user_id::text = m.learner_user_id::text
  WHERE m.tenant_id::text = $1
    AND m.joined_at::date <= $2::date
    AND (m.left_at IS NULL OR m.left_at::date > $2::date)
  GROUP BY m.cohort_id, lf.segment
),
segment_json AS (
  SELECT cohort_id, jsonb_object_agg(segment, n) AS d
  FROM segment_dist
  GROUP BY cohort_id
)
INSERT INTO analytics_cohort_features_daily (
  tenant_id, feature_date, window_days, cohort_id,
  members_count, active_learners, avg_attendance_rate,
  avg_study_minutes, avg_engagement_score, segment_distribution, computed_at
)
SELECT
  $1::uuid,
  $2::date,
  $3,
  cr.cohort_id,
  cr.members_count,
  cr.active_learners,
  cr.avg_attendance_rate,
  cr.avg_study_minutes,
  cr.avg_engagement_score,
  COALESCE(sj.d, '{}'::jsonb),
  NOW()
FROM cohort_rollup cr
LEFT JOIN segment_json sj ON sj.cohort_id = cr.cohort_id
`, tenantID, featureDate, windowDays)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
DELETE FROM analytics_reporting_metrics_daily
WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3
`, tenantID, featureDate, windowDays)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
WITH l AS (
  SELECT *
  FROM analytics_learner_features_daily
  WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3
),
c AS (
  SELECT *
  FROM analytics_cohort_features_daily
  WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3
)
INSERT INTO analytics_reporting_metrics_daily (
  tenant_id, feature_date, window_days, metric_key, metric_value, numerator, denominator, computed_at
)
SELECT $1::uuid, $2::date, $3, metric_key, metric_value, numerator, denominator, NOW()
FROM (
  SELECT
    'learner_count'::text AS metric_key,
    COUNT(*)::double precision AS metric_value,
    COUNT(*)::double precision AS numerator,
    1::double precision AS denominator
  FROM l

  UNION ALL

  SELECT
    'active_learner_rate',
    CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(*) FILTER (WHERE sessions_attended > 0 OR study_minutes > 0)::double precision / COUNT(*)::double precision END,
    COUNT(*) FILTER (WHERE sessions_attended > 0 OR study_minutes > 0)::double precision,
    NULLIF(COUNT(*), 0)::double precision
  FROM l

  UNION ALL

  SELECT
    'high_engagement_rate',
    CASE WHEN COUNT(*) = 0 THEN 0 ELSE COUNT(*) FILTER (WHERE segment = 'high_engagement')::double precision / COUNT(*)::double precision END,
    COUNT(*) FILTER (WHERE segment = 'high_engagement')::double precision,
    NULLIF(COUNT(*), 0)::double precision
  FROM l

  UNION ALL

  SELECT
    'avg_attendance_rate',
    COALESCE(AVG(attendance_rate), 0)::double precision,
    COALESCE(AVG(attendance_rate), 0)::double precision,
    1::double precision
  FROM l

  UNION ALL

  SELECT
    'cohort_count',
    COUNT(*)::double precision,
    COUNT(*)::double precision,
    1::double precision
  FROM c
) q
`, tenantID, featureDate, windowDays)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) LearnerFeatures(ctx context.Context, tenantID, featureDate string, windowDays, limit int, segment string) ([]LearnerFeature, error) {
	q := `
SELECT feature_date::text, window_days, learner_user_id::text,
  sessions_booked, sessions_attended, attendance_rate, active_days,
  study_minutes, content_previews, content_downloads, community_events,
  engagement_score, segment
FROM analytics_learner_features_daily
WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3`
	args := []any{tenantID, featureDate, windowDays}
	if segment != "" {
		q += ` AND segment = $4`
		args = append(args, segment)
		q += ` ORDER BY engagement_score DESC LIMIT $5`
		args = append(args, limit)
	} else {
		q += ` ORDER BY engagement_score DESC LIMIT $4`
		args = append(args, limit)
	}

	rows, err := dbctx.QueryContext(ctx, r.db, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]LearnerFeature, 0)
	for rows.Next() {
		var v LearnerFeature
		if err := rows.Scan(
			&v.FeatureDate,
			&v.WindowDays,
			&v.LearnerUserID,
			&v.SessionsBooked,
			&v.SessionsAttended,
			&v.AttendanceRate,
			&v.ActiveDays,
			&v.StudyMinutes,
			&v.ContentPreviews,
			&v.ContentDownloads,
			&v.CommunityEvents,
			&v.EngagementScore,
			&v.Segment,
		); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) CohortFeatures(ctx context.Context, tenantID, featureDate string, windowDays, limit int) ([]CohortFeature, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT feature_date::text, window_days, cohort_id::text,
  members_count, active_learners, avg_attendance_rate, avg_study_minutes,
  avg_engagement_score, segment_distribution
FROM analytics_cohort_features_daily
WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3
ORDER BY avg_engagement_score DESC, members_count DESC
LIMIT $4
`, tenantID, featureDate, windowDays, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]CohortFeature, 0)
	for rows.Next() {
		var v CohortFeature
		var raw []byte
		if err := rows.Scan(
			&v.FeatureDate,
			&v.WindowDays,
			&v.CohortID,
			&v.MembersCount,
			&v.ActiveLearners,
			&v.AvgAttendanceRate,
			&v.AvgStudyMinutes,
			&v.AvgEngagementScore,
			&raw,
		); err != nil {
			return nil, err
		}
		v.SegmentDistribution = map[string]int{}
		_ = json.Unmarshal(raw, &v.SegmentDistribution)
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) ReportingMetrics(ctx context.Context, tenantID, featureDate string, windowDays int) ([]ReportingMetric, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT feature_date::text, window_days, metric_key, metric_value, numerator, denominator
FROM analytics_reporting_metrics_daily
WHERE tenant_id::text = $1 AND feature_date = $2::date AND window_days = $3
ORDER BY metric_key
`, tenantID, featureDate, windowDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ReportingMetric, 0)
	for rows.Next() {
		var m ReportingMetric
		var num sql.NullFloat64
		var den sql.NullFloat64
		if err := rows.Scan(&m.FeatureDate, &m.WindowDays, &m.MetricKey, &m.MetricValue, &num, &den); err != nil {
			return nil, err
		}
		if num.Valid {
			v := num.Float64
			m.Numerator = &v
		}
		if den.Valid {
			v := den.Float64
			m.Denominator = &v
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) TodaySessions(ctx context.Context, tenantID, metricDate string, limit int) ([]TodaySession, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT
  s.session_id::text,
  s.title,
  s.starts_at,
  s.ends_at,
  s.room_id::text,
  s.capacity,
  COALESCE(booked.booked_count, 0) AS booked_seats,
  CASE WHEN s.capacity = 0 THEN 0 ELSE COALESCE(booked.booked_count, 0)::double precision / s.capacity::double precision END AS occupancy_rate,
  s.instructor_user_id::text
FROM sessions s
LEFT JOIN LATERAL (
  SELECT COUNT(*) AS booked_count
  FROM bookings b
  WHERE b.tenant_id = s.tenant_id
    AND b.session_id = s.session_id
    AND (b.state IN ('confirmed', 'rescheduled', 'checked_in') OR (b.state = 'held' AND b.hold_expires_at > NOW()))
) booked ON true
WHERE s.tenant_id::text = $1
  AND s.starts_at::date = $2::date
ORDER BY s.starts_at ASC
LIMIT $3
`, tenantID, metricDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TodaySession, 0)
	for rows.Next() {
		var item TodaySession
		var instructor sql.NullString
		if err := rows.Scan(
			&item.SessionID,
			&item.Title,
			&item.StartsAt,
			&item.EndsAt,
			&item.RoomID,
			&item.Capacity,
			&item.BookedSeats,
			&item.OccupancyRate,
			&instructor,
		); err != nil {
			return nil, err
		}
		if instructor.Valid {
			item.InstructorUser = &instructor.String
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func todayDateUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}
