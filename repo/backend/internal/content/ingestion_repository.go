package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"trainingops/backend/internal/dbctx"
)

func (r *Repository) CreateIngestionSource(ctx context.Context, tenantID, userID, name, baseURL string, intervalMinutes, jitterSeconds, rateLimitPerMinute, timeoutSeconds int) (*IngestionSource, error) {
	s := &IngestionSource{}
	var reason sql.NullString
	var last sql.NullTime
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO partner_ingestion_sources (
  tenant_id, name, base_url, schedule_interval_minutes, schedule_jitter_seconds,
  rate_limit_per_minute, request_timeout_seconds, next_run_at, created_by_user_id, updated_by_user_id
)
VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, NOW(), $8::uuid, $8::uuid)
RETURNING source_id::text, name, base_url, is_active, paused_for_manual_review, manual_review_reason,
  schedule_interval_minutes, schedule_jitter_seconds, rate_limit_per_minute, request_timeout_seconds,
  next_run_at, last_run_at, created_by_user_id::text
`, tenantID, name, baseURL, intervalMinutes, jitterSeconds, rateLimitPerMinute, timeoutSeconds, userID).Scan(
		&s.SourceID,
		&s.Name,
		&s.BaseURL,
		&s.IsActive,
		&s.PausedForManualReview,
		&reason,
		&s.ScheduleIntervalMinutes,
		&s.ScheduleJitterSeconds,
		&s.RateLimitPerMinute,
		&s.RequestTimeoutSeconds,
		&s.NextRunAt,
		&last,
		&s.CreatedByUserID,
	)
	if err != nil {
		return nil, err
	}
	if reason.Valid {
		s.ManualReviewReason = &reason.String
	}
	if last.Valid {
		t := last.Time
		s.LastRunAt = &t
	}
	return s, nil
}

func (r *Repository) ListIngestionSources(ctx context.Context, tenantID string, limit int) ([]IngestionSource, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT source_id::text, name, base_url, is_active, paused_for_manual_review, manual_review_reason,
  schedule_interval_minutes, schedule_jitter_seconds, rate_limit_per_minute, request_timeout_seconds,
  next_run_at, last_run_at, created_by_user_id::text
FROM partner_ingestion_sources
WHERE tenant_id::text = $1
ORDER BY updated_at DESC
LIMIT $2
`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]IngestionSource, 0)
	for rows.Next() {
		var s IngestionSource
		var reason sql.NullString
		var last sql.NullTime
		if err := rows.Scan(
			&s.SourceID,
			&s.Name,
			&s.BaseURL,
			&s.IsActive,
			&s.PausedForManualReview,
			&reason,
			&s.ScheduleIntervalMinutes,
			&s.ScheduleJitterSeconds,
			&s.RateLimitPerMinute,
			&s.RequestTimeoutSeconds,
			&s.NextRunAt,
			&last,
			&s.CreatedByUserID,
		); err != nil {
			return nil, err
		}
		if reason.Valid {
			s.ManualReviewReason = &reason.String
		}
		if last.Valid {
			t := last.Time
			s.LastRunAt = &t
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) AddIngestionProxy(ctx context.Context, tenantID, proxyURL string) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO partner_ingestion_proxies (tenant_id, proxy_url, is_active)
VALUES ($1::uuid, $2, TRUE)
ON CONFLICT (tenant_id, proxy_url)
DO UPDATE SET is_active = TRUE
`, tenantID, proxyURL)
	return err
}

func (r *Repository) AddIngestionUserAgent(ctx context.Context, tenantID, userAgent string) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO partner_ingestion_user_agents (tenant_id, user_agent, is_active)
VALUES ($1::uuid, $2, TRUE)
ON CONFLICT (tenant_id, user_agent)
DO UPDATE SET is_active = TRUE
`, tenantID, userAgent)
	return err
}

func (r *Repository) DueIngestionSources(ctx context.Context, tenantID string, now time.Time, limit int) ([]IngestionSource, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT source_id::text, name, base_url, is_active, paused_for_manual_review, manual_review_reason,
  schedule_interval_minutes, schedule_jitter_seconds, rate_limit_per_minute, request_timeout_seconds,
  next_run_at, last_run_at, created_by_user_id::text
FROM partner_ingestion_sources
WHERE tenant_id::text = $1
  AND is_active = TRUE
  AND paused_for_manual_review = FALSE
  AND next_run_at <= $2
ORDER BY next_run_at
LIMIT $3
`, tenantID, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]IngestionSource, 0)
	for rows.Next() {
		var s IngestionSource
		var reason sql.NullString
		var last sql.NullTime
		if err := rows.Scan(
			&s.SourceID,
			&s.Name,
			&s.BaseURL,
			&s.IsActive,
			&s.PausedForManualReview,
			&reason,
			&s.ScheduleIntervalMinutes,
			&s.ScheduleJitterSeconds,
			&s.RateLimitPerMinute,
			&s.RequestTimeoutSeconds,
			&s.NextRunAt,
			&last,
			&s.CreatedByUserID,
		); err != nil {
			return nil, err
		}
		if reason.Valid {
			s.ManualReviewReason = &reason.String
		}
		if last.Valid {
			t := last.Time
			s.LastRunAt = &t
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) IngestionSourceByID(ctx context.Context, tenantID, sourceID string) (*IngestionSource, error) {
	var s IngestionSource
	var reason sql.NullString
	var last sql.NullTime
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT source_id::text, name, base_url, is_active, paused_for_manual_review, manual_review_reason,
  schedule_interval_minutes, schedule_jitter_seconds, rate_limit_per_minute, request_timeout_seconds,
  next_run_at, last_run_at, created_by_user_id::text
FROM partner_ingestion_sources
WHERE tenant_id::text = $1 AND source_id::text = $2
`, tenantID, sourceID).Scan(
		&s.SourceID,
		&s.Name,
		&s.BaseURL,
		&s.IsActive,
		&s.PausedForManualReview,
		&reason,
		&s.ScheduleIntervalMinutes,
		&s.ScheduleJitterSeconds,
		&s.RateLimitPerMinute,
		&s.RequestTimeoutSeconds,
		&s.NextRunAt,
		&last,
		&s.CreatedByUserID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if reason.Valid {
		s.ManualReviewReason = &reason.String
	}
	if last.Valid {
		t := last.Time
		s.LastRunAt = &t
	}
	return &s, nil
}

func (r *Repository) SetManualReview(ctx context.Context, tenantID, sourceID string, paused bool, reason *string, actorUserID string) error {
	res, err := dbctx.ExecContext(ctx, r.db, `
UPDATE partner_ingestion_sources
SET paused_for_manual_review = $3,
    manual_review_reason = NULLIF($4, ''),
    updated_by_user_id = $5::uuid,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND source_id::text = $2
`, tenantID, sourceID, paused, nullable(reason), actorUserID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) PickIngestionProxy(ctx context.Context, tenantID string) (*string, error) {
	var v sql.NullString
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT proxy_url
FROM partner_ingestion_proxies
WHERE tenant_id::text = $1 AND is_active = TRUE
ORDER BY random()
LIMIT 1
`, tenantID).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if !v.Valid {
		return nil, nil
	}
	return &v.String, nil
}

func (r *Repository) PickIngestionUserAgent(ctx context.Context, tenantID string) (*string, error) {
	var v sql.NullString
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT user_agent
FROM partner_ingestion_user_agents
WHERE tenant_id::text = $1 AND is_active = TRUE
ORDER BY random()
LIMIT 1
`, tenantID).Scan(&v)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if !v.Valid {
		return nil, nil
	}
	return &v.String, nil
}

func (r *Repository) LoadPortalSessionCookies(ctx context.Context, tenantID, sourceID string) (map[string]string, error) {
	var raw []byte
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT cookies_json
FROM partner_portal_sessions
WHERE tenant_id::text = $1 AND source_id::text = $2
  AND (expires_at IS NULL OR expires_at > NOW())
`, tenantID, sourceID).Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	cookies := map[string]string{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &cookies)
	}
	return cookies, nil
}

func (r *Repository) SavePortalSessionCookies(ctx context.Context, tenantID, sourceID string, cookies map[string]string, expiresAt *time.Time) error {
	b, _ := json.Marshal(cookies)
	_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO partner_portal_sessions (tenant_id, source_id, cookies_json, expires_at, updated_at)
VALUES ($1::uuid, $2::uuid, $3::jsonb, $4, NOW())
ON CONFLICT (tenant_id, source_id)
DO UPDATE SET cookies_json = EXCLUDED.cookies_json, expires_at = EXCLUDED.expires_at, updated_at = NOW()
`, tenantID, sourceID, string(b), expiresAt)
	return err
}

func (r *Repository) StartIngestionRun(ctx context.Context, tenantID, sourceID, triggerType string, proxyURL, userAgent *string) (string, error) {
	var runID string
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO partner_ingestion_runs (tenant_id, source_id, trigger_type, status, proxy_url, user_agent, started_at)
VALUES ($1::uuid, $2::uuid, $3, 'failed', NULLIF($4, ''), NULLIF($5, ''), NOW())
RETURNING run_id::text
`, tenantID, sourceID, triggerType, nullable(proxyURL), nullable(userAgent)).Scan(&runID)
	if err != nil {
		return "", err
	}
	return runID, nil
}

func (r *Repository) CompleteIngestionRun(ctx context.Context, tenantID, runID, status string, httpStatus *int, responseBytes *int64, processed int, errMessage *string, nextRunAt *time.Time) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE partner_ingestion_runs
SET status = $3,
    http_status = $4,
    response_bytes = $5,
    records_processed = $6,
    error_message = NULLIF($7, ''),
    next_run_at = $8,
    completed_at = NOW()
WHERE tenant_id::text = $1 AND run_id::text = $2
`, tenantID, runID, status, nullableInt(httpStatus), nullableInt64(responseBytes), processed, nullable(errMessage), nextRunAt)
	return err
}

func (r *Repository) AdvanceIngestionSchedule(ctx context.Context, tenantID, sourceID string, nextRunAt, lastRunAt time.Time) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE partner_ingestion_sources
SET next_run_at = $3,
    last_run_at = $4,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND source_id::text = $2
`, tenantID, sourceID, nextRunAt, lastRunAt)
	return err
}

func (r *Repository) ListIngestionRuns(ctx context.Context, tenantID string, limit int) ([]IngestionRun, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT run_id::text, source_id::text, trigger_type, status, proxy_url, user_agent, http_status,
  response_bytes, records_processed, error_message, started_at, completed_at, next_run_at
FROM partner_ingestion_runs
WHERE tenant_id::text = $1
ORDER BY started_at DESC
LIMIT $2
`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]IngestionRun, 0)
	for rows.Next() {
		var run IngestionRun
		var proxy sql.NullString
		var ua sql.NullString
		var status sql.NullInt32
		var size sql.NullInt64
		var msg sql.NullString
		var completed sql.NullTime
		var next sql.NullTime
		if err := rows.Scan(&run.RunID, &run.SourceID, &run.TriggerType, &run.Status, &proxy, &ua, &status, &size, &run.RecordsProcessed, &msg, &run.StartedAt, &completed, &next); err != nil {
			return nil, err
		}
		if proxy.Valid {
			run.ProxyURL = &proxy.String
		}
		if ua.Valid {
			run.UserAgent = &ua.String
		}
		if status.Valid {
			v := int(status.Int32)
			run.HTTPStatus = &v
		}
		if size.Valid {
			run.ResponseBytes = &size.Int64
		}
		if msg.Valid {
			run.ErrorMessage = &msg.String
		}
		if completed.Valid {
			t := completed.Time
			run.CompletedAt = &t
		}
		if next.Valid {
			t := next.Time
			run.NextRunAt = &t
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) UpsertIngestedRecord(ctx context.Context, tenantID, sourceID, actorUserID, storagePath string, fileSize int64, item IngestedRecord) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var existingDoc sql.NullString
	err = tx.QueryRowContext(ctx, `
SELECT document_id::text
FROM partner_ingested_records
WHERE tenant_id::text = $1 AND source_id::text = $2 AND external_id = $3
FOR UPDATE
`, tenantID, sourceID, item.ExternalID).Scan(&existingDoc)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	docID := ""
	if existingDoc.Valid {
		docID = existingDoc.String
		_, err = tx.ExecContext(ctx, `
UPDATE documents
SET title = $3,
    summary = $4,
    difficulty = $5,
    duration_minutes = $6,
    updated_by_user_id = $7::uuid,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND document_id::text = $2
`, tenantID, docID, item.Title, item.Summary, item.Difficulty, item.DurationMins, actorUserID)
		if err != nil {
			return err
		}
	} else {
		err = tx.QueryRowContext(ctx, `
INSERT INTO documents (tenant_id, title, summary, difficulty, duration_minutes, created_by_user_id, updated_by_user_id)
VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid, $6::uuid)
RETURNING document_id::text
`, tenantID, item.Title, item.Summary, item.Difficulty, item.DurationMins, actorUserID).Scan(&docID)
		if err != nil {
			return err
		}
	}

	var versionNo int
	err = tx.QueryRowContext(ctx, `
UPDATE documents
SET current_version_no = current_version_no + 1,
    updated_by_user_id = $3::uuid,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND document_id::text = $2
RETURNING current_version_no
`, tenantID, docID, actorUserID).Scan(&versionNo)
	if err != nil {
		return err
	}

	fileName := sanitizeStorageName(item.Title)
	if fileName == "" {
		fileName = "partner_content"
	}
	fileName += ".txt"

	_, err = tx.ExecContext(ctx, `
INSERT INTO document_versions (
  tenant_id, document_id, version_no, file_name, storage_path, mime_type,
  file_size_bytes, sha256_checksum, extracted_text, created_by_user_id
)
VALUES ($1::uuid, $2::uuid, $3, $4, $5, 'text/plain', $6, $7, $8, $9::uuid)
`, tenantID, docID, versionNo, fileName, storagePath, fileSize, item.Checksum, item.BodyText, actorUserID)
	if err != nil {
		return err
	}

	categoryID, err := upsertCategoryID(ctx, tx, tenantID, item.Category)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO document_categories (tenant_id, document_id, category_id)
VALUES ($1::uuid, $2::uuid, $3::uuid)
ON CONFLICT DO NOTHING
`, tenantID, docID, categoryID)
	if err != nil {
		return err
	}

	for _, tag := range item.Tags {
		tagID, err := upsertTagID(ctx, tx, tenantID, tag)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO document_tags (tenant_id, document_id, tag_id)
VALUES ($1::uuid, $2::uuid, $3::uuid)
ON CONFLICT DO NOTHING
`, tenantID, docID, tagID)
		if err != nil {
			return err
		}
	}

	b, _ := json.Marshal(item.Metadata)
	_, err = tx.ExecContext(ctx, `
INSERT INTO partner_ingested_records (
  tenant_id, source_id, external_id, document_id, normalized_title,
  normalized_category, normalized_metadata, content_checksum, ingested_at
)
VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5, $6, $7::jsonb, $8, NOW())
ON CONFLICT (tenant_id, source_id, external_id)
DO UPDATE SET
  document_id = EXCLUDED.document_id,
  normalized_title = EXCLUDED.normalized_title,
  normalized_category = EXCLUDED.normalized_category,
  normalized_metadata = EXCLUDED.normalized_metadata,
  content_checksum = EXCLUDED.content_checksum,
  ingested_at = NOW()
`, tenantID, sourceID, item.ExternalID, docID, item.Title, item.Category, string(b), item.Checksum)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func upsertCategoryID(ctx context.Context, tx *sql.Tx, tenantID, name string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
WITH ins AS (
  INSERT INTO categories (tenant_id, parent_category_id, name)
  VALUES ($1::uuid, NULL, $2)
  ON CONFLICT (tenant_id, parent_category_id, name) DO NOTHING
  RETURNING category_id::text
)
SELECT category_id FROM ins
UNION ALL
SELECT category_id::text FROM categories WHERE tenant_id::text = $1 AND parent_category_id IS NULL AND name = $2
LIMIT 1
`, tenantID, name).Scan(&id)
	return id, err
}

func upsertTagID(ctx context.Context, tx *sql.Tx, tenantID, name string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `
WITH ins AS (
  INSERT INTO tags (tenant_id, name)
  VALUES ($1::uuid, $2)
  ON CONFLICT (tenant_id, name) DO NOTHING
  RETURNING tag_id::text
)
SELECT tag_id FROM ins
UNION ALL
SELECT tag_id::text FROM tags WHERE tenant_id::text = $1 AND name = $2
LIMIT 1
`, tenantID, name).Scan(&id)
	return id, err
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableInt64(v *int64) any {
	if v == nil {
		return nil
	}
	return *v
}
