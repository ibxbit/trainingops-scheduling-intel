BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS tenants (
  tenant_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_slug  TEXT NOT NULL UNIQUE,
  name         TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
  user_id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  username          CITEXT NOT NULL,
  password_hash     TEXT NOT NULL,
  failed_attempts   INTEGER NOT NULL DEFAULT 0,
  lockout_until     TIMESTAMPTZ,
  is_active         BOOLEAN NOT NULL DEFAULT TRUE,
  pii_encrypted     BYTEA,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, username),
  UNIQUE (tenant_id, user_id)
);

CREATE TABLE IF NOT EXISTS auth_sessions (
  session_id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  user_id           UUID NOT NULL,
  token_hash        BYTEA NOT NULL UNIQUE,
  expires_at        TIMESTAMPTZ NOT NULL,
  last_rotated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  rotation_number   INTEGER NOT NULL DEFAULT 0,
  client_ip_enc     BYTEA,
  user_agent_enc    BYTEA,
  revoked_at        TIMESTAMPTZ,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, user_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_tenant_user
  ON auth_sessions (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS idx_auth_sessions_active
  ON auth_sessions (tenant_id, expires_at)
  WHERE revoked_at IS NULL;

ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth_sessions ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_users ON users;
CREATE POLICY tenant_isolation_users ON users
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

DROP POLICY IF EXISTS tenant_isolation_sessions ON auth_sessions;
CREATE POLICY tenant_isolation_sessions ON auth_sessions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
