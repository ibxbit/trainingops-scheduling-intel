BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

INSERT INTO tenants (tenant_id, tenant_slug, name)
VALUES
  ('11111111-1111-1111-1111-111111111111', 'acme-training', 'Acme Training'),
  ('22222222-2222-2222-2222-222222222222', 'beta-training', 'Beta Training')
ON CONFLICT (tenant_slug) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO users (user_id, tenant_id, username, password_hash, failed_attempts, is_active)
VALUES
  ('11111111-1111-1111-1111-111111111101', '11111111-1111-1111-1111-111111111111', 'admin', crypt('AdminPass1234', gen_salt('bf')), 0, TRUE),
  ('11111111-1111-1111-1111-111111111102', '11111111-1111-1111-1111-111111111111', 'coordinator', crypt('CoordPass1234', gen_salt('bf')), 0, TRUE),
  ('11111111-1111-1111-1111-111111111103', '11111111-1111-1111-1111-111111111111', 'instructor', crypt('InstrPass1234', gen_salt('bf')), 0, TRUE),
  ('11111111-1111-1111-1111-111111111104', '11111111-1111-1111-1111-111111111111', 'learner1', crypt('LearnerPass12', gen_salt('bf')), 0, TRUE),
  ('11111111-1111-1111-1111-111111111105', '11111111-1111-1111-1111-111111111111', 'learner2', crypt('LearnerPass12', gen_salt('bf')), 0, TRUE),
  ('22222222-2222-2222-2222-222222222201', '22222222-2222-2222-2222-222222222222', 'learnerx', crypt('LearnerPass12', gen_salt('bf')), 0, TRUE)
ON CONFLICT (tenant_id, username) DO UPDATE
SET password_hash = EXCLUDED.password_hash,
    failed_attempts = 0,
    lockout_until = NULL,
    is_active = TRUE,
    updated_at = NOW();

INSERT INTO user_roles (tenant_id, user_id, role)
VALUES
  ('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111101', 'administrator'),
  ('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111102', 'program_coordinator'),
  ('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111103', 'instructor'),
  ('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111104', 'learner'),
  ('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111105', 'learner'),
  ('22222222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222201', 'learner')
ON CONFLICT DO NOTHING;

INSERT INTO rooms (room_id, tenant_id, name, capacity)
VALUES
  ('11111111-1111-1111-1111-111111112001', '11111111-1111-1111-1111-111111111111', 'Room A', 2),
  ('11111111-1111-1111-1111-111111112002', '11111111-1111-1111-1111-111111111111', 'Room B', 1)
ON CONFLICT (room_id) DO UPDATE SET
  name = EXCLUDED.name,
  capacity = EXCLUDED.capacity;

INSERT INTO academic_terms (term_id, tenant_id, name, start_date, end_date, is_active)
VALUES
  ('11111111-1111-1111-1111-111111113001', '11111111-1111-1111-1111-111111111111', 'Always On', CURRENT_DATE - INTERVAL '365 days', CURRENT_DATE + INTERVAL '365 days', TRUE)
ON CONFLICT (term_id) DO UPDATE SET
  name = EXCLUDED.name,
  start_date = EXCLUDED.start_date,
  end_date = EXCLUDED.end_date,
  is_active = TRUE;

INSERT INTO calendar_time_slot_rules (rule_id, tenant_id, room_id, weekday, slot_start, slot_end, is_active)
VALUES
  ('11111111-1111-1111-1111-111111114001', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '2 day 10 hour')::smallint, '10:00', '11:00', TRUE),
  ('11111111-1111-1111-1111-111111114002', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '2 day 12 hour')::smallint, '12:00', '13:00', TRUE),
  ('11111111-1111-1111-1111-111111114003', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '3 day 10 hour')::smallint, '10:00', '11:00', TRUE),
  ('11111111-1111-1111-1111-111111114004', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '4 day 10 hour')::smallint, '10:00', '11:00', TRUE)
ON CONFLICT (rule_id) DO UPDATE SET
  is_active = TRUE,
  slot_start = EXCLUDED.slot_start,
  slot_end = EXCLUDED.slot_end;

INSERT INTO sessions (session_id, tenant_id, title, instructor_user_id, room_id, starts_at, ends_at, capacity)
VALUES
  ('11111111-1111-1111-1111-111111115001', '11111111-1111-1111-1111-111111111111', 'Session Available', '11111111-1111-1111-1111-111111111103', '11111111-1111-1111-1111-111111112001', date_trunc('day', NOW()) + INTERVAL '2 day 10 hour', date_trunc('day', NOW()) + INTERVAL '2 day 11 hour', 2),
  ('11111111-1111-1111-1111-111111115002', '11111111-1111-1111-1111-111111111111', 'Session Full', '11111111-1111-1111-1111-111111111103', '11111111-1111-1111-1111-111111112002', date_trunc('day', NOW()) + INTERVAL '2 day 12 hour', date_trunc('day', NOW()) + INTERVAL '2 day 13 hour', 1),
  ('11111111-1111-1111-1111-111111115003', '11111111-1111-1111-1111-111111111111', 'Session Reschedule A', '11111111-1111-1111-1111-111111111103', '11111111-1111-1111-1111-111111112001', date_trunc('day', NOW()) + INTERVAL '3 day 10 hour', date_trunc('day', NOW()) + INTERVAL '3 day 11 hour', 2),
  ('11111111-1111-1111-1111-111111115004', '11111111-1111-1111-1111-111111111111', 'Session Reschedule B', '11111111-1111-1111-1111-111111111103', '11111111-1111-1111-1111-111111112001', date_trunc('day', NOW()) + INTERVAL '4 day 10 hour', date_trunc('day', NOW()) + INTERVAL '4 day 11 hour', 2),
  ('11111111-1111-1111-1111-111111115005', '11111111-1111-1111-1111-111111111111', 'Session Near Cutoff', '11111111-1111-1111-1111-111111111103', '11111111-1111-1111-1111-111111112001', NOW() + INTERVAL '2 hours', NOW() + INTERVAL '3 hours', 2)
ON CONFLICT (session_id) DO UPDATE SET
  starts_at = EXCLUDED.starts_at,
  ends_at = EXCLUDED.ends_at,
  capacity = EXCLUDED.capacity,
  room_id = EXCLUDED.room_id,
  instructor_user_id = EXCLUDED.instructor_user_id,
  title = EXCLUDED.title;

INSERT INTO bookings (booking_id, tenant_id, session_id, learner_user_id, state, reschedule_count, created_by_user_id, updated_by_user_id, hold_expires_at)
VALUES
  ('11111111-1111-1111-1111-111111116001', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111115002', '11111111-1111-1111-1111-111111111105', 'confirmed', 0, '11111111-1111-1111-1111-111111111105', '11111111-1111-1111-1111-111111111105', NULL),
  ('11111111-1111-1111-1111-111111116002', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111115005', '11111111-1111-1111-1111-111111111104', 'confirmed', 0, '11111111-1111-1111-1111-111111111104', '11111111-1111-1111-1111-111111111104', NULL)
ON CONFLICT (booking_id) DO UPDATE SET
  session_id = EXCLUDED.session_id,
  state = EXCLUDED.state,
  hold_expires_at = NULL,
  updated_by_user_id = EXCLUDED.updated_by_user_id,
  updated_at = NOW();

INSERT INTO approval_requests (approval_request_id, tenant_id, request_type, reference_id, status, submitted_by_user_id)
VALUES
  ('11111111-1111-1111-1111-111111117001', '11111111-1111-1111-1111-111111111111', 'booking', '11111111-1111-1111-1111-111111116001', 'pending', '11111111-1111-1111-1111-111111111104')
ON CONFLICT (approval_request_id) DO UPDATE SET
  status = 'pending',
  submitted_by_user_id = EXCLUDED.submitted_by_user_id,
  reviewed_by_user_id = NULL,
  reviewed_at = NULL;

COMMIT;
