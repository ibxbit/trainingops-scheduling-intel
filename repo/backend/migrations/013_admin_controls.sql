BEGIN;

CREATE TABLE IF NOT EXISTS tenant_settings (
  tenant_id UUID PRIMARY KEY REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  allow_self_registration BOOLEAN NOT NULL DEFAULT FALSE,
  require_mfa BOOLEAN NOT NULL DEFAULT FALSE,
  max_active_bookings_per_learner INTEGER NOT NULL DEFAULT 3,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CHECK (max_active_bookings_per_learner BETWEEN 1 AND 20)
);

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_tenant_settings ON tenant_settings;
CREATE POLICY tenant_isolation_tenant_settings ON tenant_settings
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

CREATE TABLE IF NOT EXISTS role_permissions (
  tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
  role app_role NOT NULL,
  permission_key TEXT NOT NULL,
  allowed BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (tenant_id, role, permission_key)
);

ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_role_permissions ON role_permissions;
CREATE POLICY tenant_isolation_role_permissions ON role_permissions
  USING (tenant_id = current_setting('app.tenant_id', true)::uuid)
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::uuid);

INSERT INTO tenant_settings (tenant_id)
SELECT tenant_id FROM tenants
ON CONFLICT (tenant_id) DO NOTHING;

WITH permission_keys AS (
  SELECT unnest(ARRAY[
    'tenant.settings.view',
    'tenant.settings.manage',
    'rbac.matrix.view',
    'rbac.matrix.manage',
    'rbac.assignments.view',
    'rbac.assignments.manage'
  ]::text[]) AS permission_key
),
role_defaults AS (
  SELECT
    t.tenant_id,
    r.role::app_role AS role,
    p.permission_key,
    CASE
      WHEN r.role = 'administrator' THEN TRUE
      WHEN r.role = 'program_coordinator' AND p.permission_key IN ('tenant.settings.view', 'rbac.matrix.view', 'rbac.assignments.view') THEN TRUE
      ELSE FALSE
    END AS allowed
  FROM tenants t
  CROSS JOIN LATERAL (SELECT unnest(ARRAY['administrator', 'program_coordinator', 'instructor', 'learner']) AS role) r
  CROSS JOIN permission_keys p
)
INSERT INTO role_permissions (tenant_id, role, permission_key, allowed)
SELECT tenant_id, role, permission_key, allowed
FROM role_defaults
ON CONFLICT (tenant_id, role, permission_key) DO NOTHING;

COMMIT;
