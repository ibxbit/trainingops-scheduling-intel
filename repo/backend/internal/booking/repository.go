package booking

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"trainingops/backend/internal/dbctx"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrCapacityReached    = errors.New("capacity reached")
	ErrRoomOccupied       = errors.New("room occupied")
	ErrInstructorBusy     = errors.New("instructor unavailable")
	ErrOutsideTerm        = errors.New("outside academic calendar")
	ErrBlackoutDate       = errors.New("blackout date")
	ErrOutsideSlot        = errors.New("outside allowed slot")
	ErrInvalidState       = errors.New("invalid booking state")
	ErrRescheduleLimit    = errors.New("reschedule limit reached")
	ErrCancellationCutoff = errors.New("cancellation cutoff exceeded")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Hold(ctx context.Context, tenantID, learnerUserID, sessionID, reason string, holdFor time.Duration) (*Booking, error) {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := cleanupExpiredHolds(ctx, tx, tenantID); err != nil {
		return nil, err
	}

	s, err := lockSession(ctx, tx, tenantID, sessionID)
	if err != nil {
		return nil, err
	}
	if err := detectScheduleConflicts(ctx, tx, s); err != nil {
		return nil, err
	}
	if err := enforceCalendarRules(ctx, tx, s); err != nil {
		return nil, err
	}

	activeCount, err := lockAndCountActiveBookings(ctx, tx, tenantID, sessionID)
	if err != nil {
		return nil, err
	}
	if activeCount >= s.Capacity {
		return nil, ErrCapacityReached
	}

	holdUntil := time.Now().UTC().Add(holdFor)
	b := &Booking{}
	err = tx.QueryRowContext(ctx, `
INSERT INTO bookings (
  tenant_id, session_id, learner_user_id, state, hold_expires_at,
  reschedule_count, created_by_user_id, updated_by_user_id
)
VALUES ($1::uuid, $2::uuid, $3::uuid, 'held', $4, 0, $3::uuid, $3::uuid)
RETURNING booking_id::text, tenant_id::text, session_id::text, learner_user_id::text, state, hold_expires_at, reschedule_count, created_at, updated_at
`, tenantID, sessionID, learnerUserID, holdUntil).Scan(
		&b.BookingID,
		&b.TenantID,
		&b.SessionID,
		&b.LearnerUserID,
		&b.State,
		&b.HoldExpiresAt,
		&b.RescheduleCount,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := insertTransition(ctx, tx, tenantID, b.BookingID, nil, StateHeld, learnerUserID, reasonOrDefault(reason, "hold")); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Repository) Confirm(ctx context.Context, tenantID, actorUserID, bookingID, reason string, tenantWide bool) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := cleanupExpiredHolds(ctx, tx, tenantID); err != nil {
		return err
	}

	b, err := lockBooking(ctx, tx, tenantID, bookingID, actorUserID, tenantWide)
	if err != nil {
		return err
	}
	if b.State != StateHeld {
		return ErrInvalidState
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE bookings
SET state = 'confirmed', hold_expires_at = NULL, updated_by_user_id = $3::uuid, updated_at = NOW()
WHERE tenant_id::text = $1 AND booking_id::text = $2
`, tenantID, bookingID, actorUserID); err != nil {
		return err
	}

	if err := insertTransition(ctx, tx, tenantID, bookingID, ptrState(StateHeld), StateConfirmed, actorUserID, reasonOrDefault(reason, "confirm")); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) Reschedule(ctx context.Context, tenantID, actorUserID, bookingID, newSessionID, reason string, tenantWide bool) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := cleanupExpiredHolds(ctx, tx, tenantID); err != nil {
		return err
	}

	b, err := lockBooking(ctx, tx, tenantID, bookingID, actorUserID, tenantWide)
	if err != nil {
		return err
	}
	if b.State != StateConfirmed && b.State != StateRescheduled {
		return ErrInvalidState
	}
	if b.RescheduleCount >= 2 {
		return ErrRescheduleLimit
	}

	s, err := lockSession(ctx, tx, tenantID, newSessionID)
	if err != nil {
		return err
	}
	if err := detectScheduleConflicts(ctx, tx, s); err != nil {
		return err
	}
	if err := enforceCalendarRules(ctx, tx, s); err != nil {
		return err
	}

	activeCount, err := lockAndCountActiveBookings(ctx, tx, tenantID, newSessionID)
	if err != nil {
		return err
	}
	if activeCount >= s.Capacity {
		return ErrCapacityReached
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE bookings
SET session_id = $3::uuid,
    state = 'rescheduled',
    hold_expires_at = NULL,
    reschedule_count = reschedule_count + 1,
    updated_by_user_id = $4::uuid,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND booking_id::text = $2
`, tenantID, bookingID, newSessionID, actorUserID); err != nil {
		return err
	}

	if err := insertTransition(ctx, tx, tenantID, bookingID, ptrState(b.State), StateRescheduled, actorUserID, reasonOrDefault(reason, "reschedule")); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) Cancel(ctx context.Context, tenantID, actorUserID, bookingID, reason string, cutoff time.Duration, tenantWide bool) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := cleanupExpiredHolds(ctx, tx, tenantID); err != nil {
		return err
	}

	b, err := lockBooking(ctx, tx, tenantID, bookingID, actorUserID, tenantWide)
	if err != nil {
		return err
	}
	if b.State == StateCanceled || b.State == StateCheckedIn {
		return ErrInvalidState
	}

	s, err := lockSession(ctx, tx, tenantID, b.SessionID)
	if err != nil {
		return err
	}
	if time.Now().UTC().After(s.StartsAt.Add(-cutoff)) {
		return ErrCancellationCutoff
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE bookings
SET state = 'canceled', hold_expires_at = NULL, updated_by_user_id = $3::uuid, updated_at = NOW()
WHERE tenant_id::text = $1 AND booking_id::text = $2
`, tenantID, bookingID, actorUserID); err != nil {
		return err
	}

	if err := insertTransition(ctx, tx, tenantID, bookingID, ptrState(b.State), StateCanceled, actorUserID, reasonOrDefault(reason, "cancel")); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) CheckIn(ctx context.Context, tenantID, actorUserID, bookingID, reason string) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := cleanupExpiredHolds(ctx, tx, tenantID); err != nil {
		return err
	}

	b, err := lockBookingByID(ctx, tx, tenantID, bookingID)
	if err != nil {
		return err
	}
	if b.State != StateConfirmed && b.State != StateRescheduled {
		return ErrInvalidState
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE bookings
SET state = 'checked_in', hold_expires_at = NULL, updated_by_user_id = $3::uuid, updated_at = NOW()
WHERE tenant_id::text = $1 AND booking_id::text = $2
`, tenantID, bookingID, actorUserID); err != nil {
		return err
	}

	if err := insertTransition(ctx, tx, tenantID, bookingID, ptrState(b.State), StateCheckedIn, actorUserID, reasonOrDefault(reason, "check_in")); err != nil {
		return err
	}

	return tx.Commit()
}

func lockSession(ctx context.Context, tx *sql.Tx, tenantID, sessionID string) (*Session, error) {
	s := &Session{}
	var instructor sql.NullString
	err := tx.QueryRowContext(ctx, `
SELECT session_id::text, tenant_id::text, room_id::text, instructor_user_id::text, capacity, starts_at, ends_at
FROM sessions
WHERE tenant_id::text = $1 AND session_id::text = $2
FOR UPDATE
`, tenantID, sessionID).Scan(
		&s.SessionID,
		&s.TenantID,
		&s.RoomID,
		&instructor,
		&s.Capacity,
		&s.StartsAt,
		&s.EndsAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if instructor.Valid {
		s.InstructorUserID = &instructor.String
	}
	return s, nil
}

func detectScheduleConflicts(ctx context.Context, tx *sql.Tx, s *Session) error {
	var roomConflict string
	err := tx.QueryRowContext(ctx, `
SELECT session_id::text
FROM sessions
WHERE tenant_id::text = $1
  AND session_id::text <> $2
  AND room_id::text = $3
  AND tstzrange(starts_at, ends_at, '[)') && tstzrange($4::timestamptz, $5::timestamptz, '[)')
LIMIT 1
`, s.TenantID, s.SessionID, s.RoomID, s.StartsAt, s.EndsAt).Scan(&roomConflict)
	if err == nil {
		return ErrRoomOccupied
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if s.InstructorUserID == nil {
		return nil
	}

	var instructorConflict string
	err = tx.QueryRowContext(ctx, `
SELECT session_id::text
FROM sessions
WHERE tenant_id::text = $1
  AND session_id::text <> $2
  AND instructor_user_id::text = $3
  AND tstzrange(starts_at, ends_at, '[)') && tstzrange($4::timestamptz, $5::timestamptz, '[)')
LIMIT 1
`, s.TenantID, s.SessionID, *s.InstructorUserID, s.StartsAt, s.EndsAt).Scan(&instructorConflict)
	if err == nil {
		return ErrInstructorBusy
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	return nil
}

func enforceCalendarRules(ctx context.Context, tx *sql.Tx, s *Session) error {
	var ok bool
	err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM academic_terms
  WHERE tenant_id::text = $1
    AND is_active = TRUE
    AND ($2::timestamptz)::date BETWEEN start_date AND end_date
)
`, s.TenantID, s.StartsAt).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOutsideTerm
	}

	err = tx.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM calendar_blackout_dates
  WHERE tenant_id::text = $1
    AND is_active = TRUE
    AND blackout_date = ($2::timestamptz)::date
    AND (room_id IS NULL OR room_id::text = $3)
)
`, s.TenantID, s.StartsAt, s.RoomID).Scan(&ok)
	if err != nil {
		return err
	}
	if ok {
		return ErrBlackoutDate
	}

	err = tx.QueryRowContext(ctx, `
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
`, s.TenantID, s.StartsAt, s.EndsAt, s.RoomID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOutsideSlot
	}

	return nil
}

func lockAndCountActiveBookings(ctx context.Context, tx *sql.Tx, tenantID, sessionID string) (int, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT booking_id
FROM bookings
WHERE tenant_id::text = $1
  AND session_id::text = $2
  AND (
    state IN ('confirmed', 'rescheduled', 'checked_in')
    OR (state = 'held' AND hold_expires_at > NOW())
  )
FOR UPDATE
`, tenantID, sessionID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var ignore string
		if err := rows.Scan(&ignore); err != nil {
			return 0, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func lockBooking(ctx context.Context, tx *sql.Tx, tenantID, bookingID, learnerUserID string, tenantWide bool) (*Booking, error) {
	b := &Booking{}
	query := `
SELECT booking_id::text, tenant_id::text, session_id::text, learner_user_id::text, state, hold_expires_at, reschedule_count, created_at, updated_at
FROM bookings
WHERE tenant_id::text = $1 AND booking_id::text = $2`
	args := []any{tenantID, bookingID}
	if !tenantWide {
		query += ` AND learner_user_id::text = $3`
		args = append(args, learnerUserID)
	}
	query += ` FOR UPDATE`
	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&b.BookingID,
		&b.TenantID,
		&b.SessionID,
		&b.LearnerUserID,
		&b.State,
		&b.HoldExpiresAt,
		&b.RescheduleCount,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func lockBookingByID(ctx context.Context, tx *sql.Tx, tenantID, bookingID string) (*Booking, error) {
	b := &Booking{}
	err := tx.QueryRowContext(ctx, `
SELECT booking_id::text, tenant_id::text, session_id::text, learner_user_id::text, state, hold_expires_at, reschedule_count, created_at, updated_at
FROM bookings
WHERE tenant_id::text = $1 AND booking_id::text = $2
FOR UPDATE
`, tenantID, bookingID).Scan(
		&b.BookingID,
		&b.TenantID,
		&b.SessionID,
		&b.LearnerUserID,
		&b.State,
		&b.HoldExpiresAt,
		&b.RescheduleCount,
		&b.CreatedAt,
		&b.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

func cleanupExpiredHolds(ctx context.Context, tx *sql.Tx, tenantID string) error {
	_, err := tx.ExecContext(ctx, `
WITH expired AS (
  UPDATE bookings
  SET state = 'canceled',
      hold_expires_at = NULL,
      updated_by_user_id = NULL,
      updated_at = NOW()
  WHERE tenant_id::text = $1
    AND state = 'held'
    AND hold_expires_at < NOW()
  RETURNING booking_id
)
INSERT INTO booking_state_transitions (tenant_id, booking_id, from_state, to_state, changed_by_user_id, reason)
SELECT $1::uuid, booking_id, 'held', 'canceled', NULL, 'hold_timeout'
FROM expired
`, tenantID)
	return err
}

func insertTransition(ctx context.Context, tx *sql.Tx, tenantID, bookingID string, from *State, to State, changedByUserID, reason string) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO booking_state_transitions (tenant_id, booking_id, from_state, to_state, changed_by_user_id, reason)
VALUES ($1::uuid, $2::uuid, $3, $4, NULLIF($5, '')::uuid, $6)
`, tenantID, bookingID, from, to, changedByUserID, reason)
	return err
}

func ptrState(s State) *State {
	return &s
}

func reasonOrDefault(reason, fallback string) string {
	if reason == "" {
		return fallback
	}
	return reason
}
