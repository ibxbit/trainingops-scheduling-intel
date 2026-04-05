BEGIN;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'app_role') THEN
    CREATE TYPE app_role AS ENUM (
      'administrator',
      'program_coordinator',
      'instructor',
      'learner'
    );
  END IF;
END$$;

CREATE TABLE IF NOT EXISTS user_roles (
  tenant_id    UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  user_id      UUID NOT NULL,
  role         app_role NOT NULL,
  assigned_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, user_id, role),
  FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, user_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_roles_tenant_user
  ON user_roles (tenant_id, user_id);

ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_user_roles ON user_roles;
CREATE POLICY tenant_isolation_user_roles ON user_roles
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

COMMIT;
