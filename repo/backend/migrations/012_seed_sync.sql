BEGIN;

INSERT INTO calendar_time_slot_rules (rule_id, tenant_id, room_id, weekday, slot_start, slot_end, is_active)
VALUES
  ('11111111-1111-1111-1111-111111114001', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '2 day 10 hour')::smallint, '10:00', '11:00', TRUE),
  ('11111111-1111-1111-1111-111111114002', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '2 day 12 hour')::smallint, '12:00', '13:00', TRUE),
  ('11111111-1111-1111-1111-111111114003', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '3 day 10 hour')::smallint, '10:00', '11:00', TRUE),
  ('11111111-1111-1111-1111-111111114004', '11111111-1111-1111-1111-111111111111', NULL, EXTRACT(DOW FROM date_trunc('day', NOW()) + INTERVAL '4 day 10 hour')::smallint, '10:00', '11:00', TRUE)
ON CONFLICT (rule_id) DO UPDATE SET
  weekday = EXCLUDED.weekday,
  slot_start = EXCLUDED.slot_start,
  slot_end = EXCLUDED.slot_end,
  is_active = TRUE,
  updated_at = NOW();

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

COMMIT;
