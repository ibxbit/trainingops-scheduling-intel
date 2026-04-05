BEGIN;

CREATE EXTENSION IF NOT EXISTS btree_gist;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'booking_state') THEN
    CREATE TYPE booking_state AS ENUM (
      'held',
      'confirmed',
      'rescheduled',
      'canceled',
      'checked_in'
    );
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS rooms (
  room_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  capacity     INTEGER NOT NULL CHECK (capacity > 0),
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, room_id)
);

CREATE TABLE IF NOT EXISTS sessions (
  session_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  title                TEXT NOT NULL,
  instructor_user_id   UUID,
  room_id              UUID NOT NULL,
  starts_at            TIMESTAMPTZ NOT NULL,
  ends_at              TIMESTAMPTZ NOT NULL,
  capacity             INTEGER NOT NULL CHECK (capacity > 0),
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (ends_at > starts_at),
  UNIQUE (tenant_id, session_id),
  FOREIGN KEY (tenant_id, instructor_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL,
  FOREIGN KEY (tenant_id, room_id) REFERENCES rooms(tenant_id, room_id) ON DELETE RESTRICT
);

ALTER TABLE sessions
  DROP CONSTRAINT IF EXISTS ex_sessions_room_overlap;

ALTER TABLE sessions
  ADD CONSTRAINT ex_sessions_room_overlap
  EXCLUDE USING GIST (
    tenant_id WITH =,
    room_id WITH =,
    tstzrange(starts_at, ends_at, '[)') WITH &&
  );

ALTER TABLE sessions
  DROP CONSTRAINT IF EXISTS ex_sessions_instructor_overlap;

ALTER TABLE sessions
  ADD CONSTRAINT ex_sessions_instructor_overlap
  EXCLUDE USING GIST (
    tenant_id WITH =,
    instructor_user_id WITH =,
    tstzrange(starts_at, ends_at, '[)') WITH &&
  )
  WHERE (instructor_user_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS idx_sessions_tenant_time
  ON sessions (tenant_id, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS bookings (
  booking_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  session_id           UUID NOT NULL,
  learner_user_id      UUID NOT NULL,
  state                booking_state NOT NULL,
  hold_expires_at      TIMESTAMPTZ,
  reschedule_count     INTEGER NOT NULL DEFAULT 0 CHECK (reschedule_count BETWEEN 0 AND 2),
  previous_booking_id  UUID,
  created_by_user_id   UUID NOT NULL,
  updated_by_user_id   UUID,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (
    (state = 'held' AND hold_expires_at IS NOT NULL)
    OR (state <> 'held' AND hold_expires_at IS NULL)
  ),
  UNIQUE (tenant_id, booking_id),
  FOREIGN KEY (tenant_id, session_id) REFERENCES sessions(tenant_id, session_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, learner_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, created_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE RESTRICT,
  FOREIGN KEY (tenant_id, updated_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL,
  FOREIGN KEY (tenant_id, previous_booking_id) REFERENCES bookings(tenant_id, booking_id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS ux_bookings_active_per_learner_session
  ON bookings (tenant_id, session_id, learner_user_id)
  WHERE state IN ('held', 'confirmed', 'rescheduled', 'checked_in');

CREATE INDEX IF NOT EXISTS idx_bookings_tenant_session_state
  ON bookings (tenant_id, session_id, state);

CREATE INDEX IF NOT EXISTS idx_bookings_tenant_learner_state
  ON bookings (tenant_id, learner_user_id, state);

CREATE TABLE IF NOT EXISTS booking_state_transitions (
  transition_id        BIGSERIAL PRIMARY KEY,
  tenant_id            UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  booking_id           UUID NOT NULL,
  from_state           booking_state,
  to_state             booking_state NOT NULL,
  changed_by_user_id   UUID,
  changed_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  reason               TEXT NOT NULL,
  FOREIGN KEY (tenant_id, booking_id) REFERENCES bookings(tenant_id, booking_id) ON DELETE CASCADE,
  FOREIGN KEY (tenant_id, changed_by_user_id) REFERENCES users(tenant_id, user_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_booking_transitions_tenant_booking_time
  ON booking_state_transitions (tenant_id, booking_id, changed_at DESC);

ALTER TABLE rooms ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE bookings ENABLE ROW LEVEL SECURITY;
ALTER TABLE booking_state_transitions ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_rooms ON rooms;
CREATE POLICY tenant_isolation_rooms ON rooms
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_sessions ON sessions;
CREATE POLICY tenant_isolation_sessions ON sessions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_bookings ON bookings;
CREATE POLICY tenant_isolation_bookings ON bookings
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_booking_transitions ON booking_state_transitions;
CREATE POLICY tenant_isolation_booking_transitions ON booking_state_transitions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
