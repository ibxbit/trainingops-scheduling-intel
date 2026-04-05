package calendar

import (
	"context"
	"database/sql"
	"errors"

	"trainingops/backend/internal/dbctx"
)

var (
	ErrNotFound        = errors.New("not found")
	ErrVersionConflict = errors.New("version conflict")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CheckAvailability(ctx context.Context, tenantID, sessionID string) (AvailabilityReason, error) {
	var startsAt, endsAt string
	var roomID string
	var instructor sql.NullString
	err := dbctx.QueryRowContext(ctx, r.db, `
SELECT starts_at::text, ends_at::text, room_id::text, instructor_user_id::text
FROM sessions
WHERE tenant_id::text = $1 AND session_id::text = $2
`, tenantID, sessionID).Scan(&startsAt, &endsAt, &roomID, &instructor)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}

	var ok bool
	err = dbctx.QueryRowContext(ctx, r.db, `
SELECT EXISTS (
  SELECT 1
  FROM academic_terms
  WHERE tenant_id::text = $1
    AND is_active = TRUE
    AND ($2::timestamptz)::date BETWEEN start_date AND end_date
)
`, tenantID, startsAt).Scan(&ok)
	if err != nil {
		return "", err
	}
	if !ok {
		return ReasonOutsideAcademicCalendar, nil
	}

	err = dbctx.QueryRowContext(ctx, r.db, `
SELECT EXISTS (
  SELECT 1
  FROM calendar_blackout_dates
  WHERE tenant_id::text = $1
    AND is_active = TRUE
    AND blackout_date = ($2::timestamptz)::date
    AND (room_id IS NULL OR room_id::text = $3)
)
`, tenantID, startsAt, roomID).Scan(&ok)
	if err != nil {
		return "", err
	}
	if ok {
		return ReasonBlackoutDate, nil
	}

	err = dbctx.QueryRowContext(ctx, r.db, `
SELECT EXISTS (
  SELECT 1
  FROM calendar_time_slot_rules
  WHERE tenant_id::text = $1
    AND is_active = TRUE
    AND weekday = EXTRACT(DOW FROM $2::timestamptz)
    AND slot_start = ($2::timestamptz)::time
    AND slot_end = ($3::timestamptz)::time
    AND (room_id IS NULL OR room_id::text = $4)
)
`, tenantID, startsAt, endsAt, roomID).Scan(&ok)
	if err != nil {
		return "", err
	}
	if !ok {
		return ReasonOutsideAllowedSlot, nil
	}

	err = dbctx.QueryRowContext(ctx, r.db, `
SELECT EXISTS (
  SELECT 1
  FROM sessions s
  WHERE s.tenant_id::text = $1
    AND s.session_id::text <> $2
    AND s.room_id::text = $3
    AND tstzrange(s.starts_at, s.ends_at, '[)') && tstzrange($4::timestamptz, $5::timestamptz, '[)')
)
`, tenantID, sessionID, roomID, startsAt, endsAt).Scan(&ok)
	if err != nil {
		return "", err
	}
	if ok {
		return ReasonRoomOccupied, nil
	}

	if instructor.Valid {
		err = dbctx.QueryRowContext(ctx, r.db, `
SELECT EXISTS (
  SELECT 1
  FROM sessions s
  WHERE s.tenant_id::text = $1
    AND s.session_id::text <> $2
    AND s.instructor_user_id::text = $3
    AND tstzrange(s.starts_at, s.ends_at, '[)') && tstzrange($4::timestamptz, $5::timestamptz, '[)')
)
`, tenantID, sessionID, instructor.String, startsAt, endsAt).Scan(&ok)
		if err != nil {
			return "", err
		}
		if ok {
			return ReasonInstructorUnavailable, nil
		}
	}

	err = dbctx.QueryRowContext(ctx, r.db, `
SELECT (
  (SELECT COUNT(*)
   FROM bookings b
   WHERE b.tenant_id::text = $1
     AND b.session_id::text = $2
     AND (b.state IN ('confirmed', 'rescheduled', 'checked_in')
       OR (b.state = 'held' AND b.hold_expires_at > NOW()))
  ) >=
  (SELECT capacity FROM sessions WHERE tenant_id::text = $1 AND session_id::text = $2)
)
`, tenantID, sessionID).Scan(&ok)
	if err != nil {
		return "", err
	}
	if ok {
		return ReasonCapacityReached, nil
	}

	return ReasonAvailable, nil
}

func (r *Repository) SuggestAlternatives(ctx context.Context, tenantID, sessionID string, limit int) ([]Alternative, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
WITH base AS (
  SELECT starts_at AS base_start
  FROM sessions
  WHERE tenant_id::text = $1 AND session_id::text = $2
)
SELECT s.session_id::text, s.room_id::text, s.starts_at, s.ends_at
FROM sessions s, base
WHERE s.tenant_id::text = $1
  AND s.session_id::text <> $2
  AND s.starts_at > NOW()
  AND EXISTS (
    SELECT 1 FROM academic_terms t
    WHERE t.tenant_id = s.tenant_id
      AND t.is_active = TRUE
      AND s.starts_at::date BETWEEN t.start_date AND t.end_date
  )
  AND EXISTS (
    SELECT 1 FROM calendar_time_slot_rules r
    WHERE r.tenant_id = s.tenant_id
      AND r.is_active = TRUE
      AND r.weekday = EXTRACT(DOW FROM s.starts_at)
      AND r.slot_start = s.starts_at::time
      AND r.slot_end = s.ends_at::time
      AND (r.room_id IS NULL OR r.room_id = s.room_id)
  )
  AND NOT EXISTS (
    SELECT 1 FROM calendar_blackout_dates b
    WHERE b.tenant_id = s.tenant_id
      AND b.is_active = TRUE
      AND b.blackout_date = s.starts_at::date
      AND (b.room_id IS NULL OR b.room_id = s.room_id)
  )
  AND (
    (SELECT COUNT(*)
     FROM bookings bk
     WHERE bk.tenant_id = s.tenant_id
       AND bk.session_id = s.session_id
       AND (bk.state IN ('confirmed', 'rescheduled', 'checked_in')
         OR (bk.state = 'held' AND bk.hold_expires_at > NOW()))
    ) < s.capacity
  )
ORDER BY ABS(EXTRACT(EPOCH FROM (s.starts_at - base.base_start))), s.starts_at
LIMIT $3
`, tenantID, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Alternative, 0, limit)
	for rows.Next() {
		var a Alternative
		if err := rows.Scan(&a.SessionID, &a.RoomID, &a.StartsAt, &a.EndsAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *Repository) CreateTimeSlotRule(ctx context.Context, tenantID string, in TimeSlotRule) (string, error) {
	var id string
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO calendar_time_slot_rules (tenant_id, room_id, weekday, slot_start, slot_end, is_active)
VALUES ($1::uuid, NULLIF($2, '')::uuid, $3, $4::time, $5::time, $6)
RETURNING rule_id::text
`, tenantID, nullableString(in.RoomID), in.Weekday, in.SlotStart, in.SlotEnd, in.IsActive).Scan(&id)
	return id, err
}

func (r *Repository) UpdateTimeSlotRule(ctx context.Context, tenantID, ruleID string, in TimeSlotRule) error {
	res, err := dbctx.ExecContext(ctx, r.db, `
UPDATE calendar_time_slot_rules
SET room_id = NULLIF($3, '')::uuid,
    weekday = $4,
    slot_start = $5::time,
    slot_end = $6::time,
    is_active = $7,
    lock_version = lock_version + 1,
    updated_at = NOW()
WHERE tenant_id::text = $1
  AND rule_id::text = $2
  AND lock_version = $8
`, tenantID, ruleID, nullableString(in.RoomID), in.Weekday, in.SlotStart, in.SlotEnd, in.IsActive, in.LockVersion)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrVersionConflict
	}
	return nil
}

func (r *Repository) CreateBlackoutDate(ctx context.Context, tenantID string, in BlackoutDate) (string, error) {
	var id string
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO calendar_blackout_dates (tenant_id, room_id, blackout_date, reason, is_active)
VALUES ($1::uuid, NULLIF($2, '')::uuid, $3::date, $4, $5)
RETURNING blackout_id::text
`, tenantID, nullableString(in.RoomID), in.BlackoutDate, in.Reason, in.IsActive).Scan(&id)
	return id, err
}

func (r *Repository) UpdateBlackoutDate(ctx context.Context, tenantID, blackoutID string, in BlackoutDate) error {
	res, err := dbctx.ExecContext(ctx, r.db, `
UPDATE calendar_blackout_dates
SET room_id = NULLIF($3, '')::uuid,
    blackout_date = $4::date,
    reason = $5,
    is_active = $6,
    lock_version = lock_version + 1,
    updated_at = NOW()
WHERE tenant_id::text = $1
  AND blackout_id::text = $2
  AND lock_version = $7
`, tenantID, blackoutID, nullableString(in.RoomID), in.BlackoutDate, in.Reason, in.IsActive, in.LockVersion)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrVersionConflict
	}
	return nil
}

func (r *Repository) CreateAcademicTerm(ctx context.Context, tenantID string, in AcademicTerm) (string, error) {
	var id string
	err := dbctx.QueryRowContext(ctx, r.db, `
INSERT INTO academic_terms (tenant_id, name, start_date, end_date, is_active)
VALUES ($1::uuid, $2, $3::date, $4::date, $5)
RETURNING term_id::text
`, tenantID, in.Name, in.StartDate, in.EndDate, in.IsActive).Scan(&id)
	return id, err
}

func (r *Repository) UpdateAcademicTerm(ctx context.Context, tenantID, termID string, in AcademicTerm) error {
	res, err := dbctx.ExecContext(ctx, r.db, `
UPDATE academic_terms
SET name = $3,
    start_date = $4::date,
    end_date = $5::date,
    is_active = $6,
    lock_version = lock_version + 1,
    updated_at = NOW()
WHERE tenant_id::text = $1
  AND term_id::text = $2
  AND lock_version = $7
`, tenantID, termID, in.Name, in.StartDate, in.EndDate, in.IsActive, in.LockVersion)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrVersionConflict
	}
	return nil
}

func nullableString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
