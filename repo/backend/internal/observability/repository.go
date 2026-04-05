package observability

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"trainingops/backend/internal/dashboard"
	"trainingops/backend/internal/dbctx"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ApplyRetention(ctx context.Context, days int) error {
	if days <= 0 {
		days = 90
	}
	_, err := dbctx.ExecContext(ctx, r.db, `
DELETE FROM workflow_logs WHERE occurred_at < NOW() - ($1::text || ' days')::interval;
DELETE FROM scraping_errors WHERE occurred_at < NOW() - ($1::text || ' days')::interval;
DELETE FROM anomaly_events WHERE created_at < NOW() - ($1::text || ' days')::interval;
DELETE FROM report_exports WHERE generated_at < NOW() - ($1::text || ' days')::interval;
`, days)
	return err
}

func (r *Repository) InsertWorkflowLog(ctx context.Context, tenantID string, userID *string, workflowName, resourceID, outcome string, statusCode, latencyMS int, details map[string]any) error {
	b, _ := json.Marshal(details)
	_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO workflow_logs (tenant_id, actor_user_id, workflow_name, resource_id, outcome, status_code, latency_ms, details)
VALUES ($1::uuid, NULLIF($2, '')::uuid, $3, NULLIF($4, ''), $5, $6, $7, $8::jsonb)
`, tenantID, nullable(userID), workflowName, resourceID, outcome, statusCode, latencyMS, string(b))
	return err
}

func (r *Repository) ListWorkflowLogs(ctx context.Context, tenantID string, limit int) ([]WorkflowLog, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT workflow_log_id::text, workflow_name, coalesce(resource_id, ''), outcome, coalesce(status_code, 0), coalesce(latency_ms, 0), occurred_at
FROM workflow_logs
WHERE tenant_id::text = $1
ORDER BY occurred_at DESC
LIMIT $2
`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]WorkflowLog, 0)
	for rows.Next() {
		var w WorkflowLog
		if err := rows.Scan(&w.WorkflowLogID, &w.WorkflowName, &w.ResourceID, &w.Outcome, &w.StatusCode, &w.LatencyMS, &w.OccurredAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) InsertScrapingError(ctx context.Context, tenantID, sourceName, errorCode, errorMessage string, metadata map[string]any) error {
	b, _ := json.Marshal(metadata)
	_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO scraping_errors (tenant_id, source_name, error_code, error_message, metadata)
VALUES ($1::uuid, $2, NULLIF($3, ''), $4, $5::jsonb)
`, tenantID, sourceName, errorCode, errorMessage, string(b))
	return err
}

func (r *Repository) DetectAnomalies(ctx context.Context, tenantID, date string) (int, error) {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	inserted := 0

	var failedToday, failedBaseline sql.NullFloat64
	err = tx.QueryRowContext(ctx, `
WITH daily AS (
  SELECT occurred_at::date AS d, COUNT(*)::double precision AS n
  FROM workflow_logs
  WHERE tenant_id::text = $1
    AND workflow_name LIKE 'booking.%'
    AND outcome = 'failed'
    AND occurred_at::date BETWEEN ($2::date - INTERVAL '7 days')::date AND $2::date
  GROUP BY occurred_at::date
)
SELECT
  COALESCE((SELECT n FROM daily WHERE d = $2::date), 0),
  COALESCE((SELECT AVG(n) FROM daily WHERE d < $2::date), 0)
`, tenantID, date).Scan(&failedToday, &failedBaseline)
	if err != nil {
		return 0, err
	}
	failedThreshold := failedBaseline.Float64 * 2
	if failedThreshold < 5 {
		failedThreshold = 5
	}
	if failedToday.Float64 > failedThreshold {
		_, err = tx.ExecContext(ctx, `
INSERT INTO anomaly_events (tenant_id, anomaly_date, anomaly_type, severity, observed_value, baseline_value, threshold_value, details)
VALUES ($1::uuid, $2::date, 'failed_booking_spike', 'high', $3, $4, $5, '{}'::jsonb)
ON CONFLICT (tenant_id, anomaly_date, anomaly_type)
DO UPDATE SET observed_value = EXCLUDED.observed_value, baseline_value = EXCLUDED.baseline_value, threshold_value = EXCLUDED.threshold_value, created_at = NOW()
`, tenantID, date, failedToday.Float64, failedBaseline.Float64, failedThreshold)
		if err != nil {
			return 0, err
		}
		inserted++
	}

	var scrapeToday, scrapeBaseline sql.NullFloat64
	err = tx.QueryRowContext(ctx, `
WITH daily AS (
  SELECT occurred_at::date AS d, COUNT(*)::double precision AS n
  FROM scraping_errors
  WHERE tenant_id::text = $1
    AND occurred_at::date BETWEEN ($2::date - INTERVAL '7 days')::date AND $2::date
  GROUP BY occurred_at::date
)
SELECT
  COALESCE((SELECT n FROM daily WHERE d = $2::date), 0),
  COALESCE((SELECT AVG(n) FROM daily WHERE d < $2::date), 0)
`, tenantID, date).Scan(&scrapeToday, &scrapeBaseline)
	if err != nil {
		return 0, err
	}
	scrapeThreshold := scrapeBaseline.Float64 * 2
	if scrapeThreshold < 3 {
		scrapeThreshold = 3
	}
	if scrapeToday.Float64 > scrapeThreshold {
		_, err = tx.ExecContext(ctx, `
INSERT INTO anomaly_events (tenant_id, anomaly_date, anomaly_type, severity, observed_value, baseline_value, threshold_value, details)
VALUES ($1::uuid, $2::date, 'scraping_error_spike', 'medium', $3, $4, $5, '{}'::jsonb)
ON CONFLICT (tenant_id, anomaly_date, anomaly_type)
DO UPDATE SET observed_value = EXCLUDED.observed_value, baseline_value = EXCLUDED.baseline_value, threshold_value = EXCLUDED.threshold_value, created_at = NOW()
`, tenantID, date, scrapeToday.Float64, scrapeBaseline.Float64, scrapeThreshold)
		if err != nil {
			return 0, err
		}
		inserted++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (r *Repository) ListAnomalies(ctx context.Context, tenantID, date string, limit int) ([]AnomalyEvent, error) {
	q := `
SELECT anomaly_event_id::text, anomaly_date::text, anomaly_type, severity, observed_value, baseline_value, threshold_value, created_at
FROM anomaly_events
WHERE tenant_id::text = $1`
	args := []any{tenantID}
	if date != "" {
		q += ` AND anomaly_date = $2::date`
		args = append(args, date)
	}
	q += ` ORDER BY anomaly_date DESC, created_at DESC LIMIT $` + itoa(len(args)+1)
	args = append(args, limit)

	rows, err := dbctx.QueryContext(ctx, r.db, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]AnomalyEvent, 0)
	for rows.Next() {
		var a AnomalyEvent
		if err := rows.Scan(&a.AnomalyEventID, &a.AnomalyDate, &a.AnomalyType, &a.Severity, &a.ObservedValue, &a.BaselineValue, &a.ThresholdValue, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) CreateSchedule(ctx context.Context, tenantID, userID, name, format, frequency, folder string, nextRunAt time.Time) (*ReportSchedule, error) {
	s := &ReportSchedule{}
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO report_schedules (tenant_id, name, format, frequency, output_folder, next_run_at, created_by_user_id)
VALUES ($1::uuid, $2, $3::report_format, $4::report_frequency, $5, $6, $7::uuid)
RETURNING schedule_id::text, name, format::text, frequency::text, output_folder, is_active, next_run_at
`, tenantID, name, format, frequency, folder, nextRunAt, userID).Scan(
		&s.ScheduleID,
		&s.Name,
		&s.Format,
		&s.Frequency,
		&s.OutputFolder,
		&s.IsActive,
		&s.NextRunAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *Repository) DueSchedules(ctx context.Context, tenantID string, now time.Time) ([]ReportSchedule, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT schedule_id::text, name, format::text, frequency::text, output_folder, is_active, next_run_at
FROM report_schedules
WHERE tenant_id::text = $1 AND is_active = TRUE AND next_run_at <= $2
ORDER BY next_run_at
`, tenantID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ReportSchedule, 0)
	for rows.Next() {
		var s ReportSchedule
		if err := rows.Scan(&s.ScheduleID, &s.Name, &s.Format, &s.Frequency, &s.OutputFolder, &s.IsActive, &s.NextRunAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) ScheduleByID(ctx context.Context, tenantID, scheduleID string) (*ReportSchedule, error) {
	s := &ReportSchedule{}
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT schedule_id::text, name, format::text, frequency::text, output_folder, is_active, next_run_at
FROM report_schedules
WHERE tenant_id::text = $1 AND schedule_id::text = $2
`, tenantID, scheduleID).Scan(
		&s.ScheduleID,
		&s.Name,
		&s.Format,
		&s.Frequency,
		&s.OutputFolder,
		&s.IsActive,
		&s.NextRunAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

func (r *Repository) InsertExport(ctx context.Context, tenantID string, scheduleID *string, reportDate, format, status string, filePath *string, size *int64, errMsg *string) (*ReportExport, error) {
	e := &ReportExport{}
	var sid sql.NullString
	var fp sql.NullString
	var sz sql.NullInt64
	var em sql.NullString
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO report_exports (tenant_id, schedule_id, report_date, format, file_path, file_size_bytes, status, error_message)
VALUES ($1::uuid, NULLIF($2, '')::uuid, $3::date, $4::report_format, NULLIF($5, ''), $6, $7, NULLIF($8, ''))
RETURNING export_id::text, schedule_id::text, report_date::text, format::text, file_path, file_size_bytes, status, error_message, created_at
`, tenantID, nullable(scheduleID), reportDate, format, nullable(filePath), nullableInt64(size), status, nullable(errMsg)).Scan(
		&e.ExportID, &sid, &e.ReportDate, &e.Format, &fp, &sz, &e.Status, &em, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if sid.Valid {
		e.ScheduleID = &sid.String
	}
	if fp.Valid {
		e.FilePath = &fp.String
	}
	if sz.Valid {
		e.FileSizeBytes = &sz.Int64
	}
	if em.Valid {
		e.ErrorMessage = &em.String
	}
	return e, nil
}

func (r *Repository) AdvanceScheduleNextRun(ctx context.Context, tenantID, scheduleID, frequency string, now time.Time) error {
	next := now
	if frequency == "weekly" {
		next = next.Add(7 * 24 * time.Hour)
	} else {
		next = next.Add(24 * time.Hour)
	}
	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE report_schedules
SET next_run_at = $3, updated_at = NOW()
WHERE tenant_id::text = $1 AND schedule_id::text = $2
`, tenantID, scheduleID, next)
	return err
}

func (r *Repository) ListExports(ctx context.Context, tenantID string, limit int) ([]ReportExport, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT export_id::text, schedule_id::text, report_date::text, format::text, file_path, file_size_bytes, status, error_message, created_at
FROM report_exports
WHERE tenant_id::text = $1
ORDER BY created_at DESC
LIMIT $2
`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]ReportExport, 0)
	for rows.Next() {
		var e ReportExport
		var sid sql.NullString
		var fp sql.NullString
		var sz sql.NullInt64
		var em sql.NullString
		if err := rows.Scan(&e.ExportID, &sid, &e.ReportDate, &e.Format, &fp, &sz, &e.Status, &em, &e.CreatedAt); err != nil {
			return nil, err
		}
		if sid.Valid {
			e.ScheduleID = &sid.String
		}
		if fp.Valid {
			e.FilePath = &fp.String
		}
		if sz.Valid {
			e.FileSizeBytes = &sz.Int64
		}
		if em.Valid {
			e.ErrorMessage = &em.String
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) DashboardDataForReport(ctx context.Context, tenantID, date string) (dashboard.DailySummary, []dashboard.KPI, []dashboard.HeatmapCell, error) {
	var s dashboard.DailySummary
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT metric_date::text, todays_sessions, pending_approvals
FROM analytics_dashboard_daily_summary
WHERE tenant_id::text = $1 AND metric_date = $2::date
`, tenantID, date).Scan(&s.MetricDate, &s.TodaysSessions, &s.PendingApprovals)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return dashboard.DailySummary{}, nil, nil, ErrNotFound
		}
		return dashboard.DailySummary{}, nil, nil, err
	}

	kRows, err := dbctx.QueryContext(ctx, r.db, `
SELECT metric_key, metric_value, numerator, denominator
FROM analytics_dashboard_kpi_daily
WHERE tenant_id::text = $1 AND metric_date = $2::date
ORDER BY metric_key
`, tenantID, date)
	if err != nil {
		return dashboard.DailySummary{}, nil, nil, err
	}
	defer kRows.Close()
	kpis := make([]dashboard.KPI, 0)
	for kRows.Next() {
		var k dashboard.KPI
		if err := kRows.Scan(&k.MetricKey, &k.MetricValue, &k.Numerator, &k.Denominator); err != nil {
			return dashboard.DailySummary{}, nil, nil, err
		}
		kpis = append(kpis, k)
	}
	if err := kRows.Err(); err != nil {
		return dashboard.DailySummary{}, nil, nil, err
	}

	hRows, err := dbctx.QueryContext(ctx, r.db, `
SELECT hour_bucket, room_id::text, sessions_count, booked_seats, total_seats, occupancy_rate
FROM analytics_occupancy_heatmap_daily
WHERE tenant_id::text = $1 AND metric_date = $2::date
ORDER BY hour_bucket, room_id
`, tenantID, date)
	if err != nil {
		return dashboard.DailySummary{}, nil, nil, err
	}
	defer hRows.Close()
	hm := make([]dashboard.HeatmapCell, 0)
	for hRows.Next() {
		var c dashboard.HeatmapCell
		var room sql.NullString
		if err := hRows.Scan(&c.HourBucket, &room, &c.SessionsCount, &c.BookedSeats, &c.TotalSeats, &c.OccupancyRate); err != nil {
			return dashboard.DailySummary{}, nil, nil, err
		}
		if room.Valid {
			c.RoomID = &room.String
		}
		hm = append(hm, c)
	}
	if err := hRows.Err(); err != nil {
		return dashboard.DailySummary{}, nil, nil, err
	}

	return s, kpis, hm, nil
}

func nullable(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := false
	if v < 0 {
		neg = true
		v = -v
	}
	b := make([]byte, 0, 12)
	for v > 0 {
		d := v % 10
		b = append([]byte{byte('0' + d)}, b...)
		v /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
