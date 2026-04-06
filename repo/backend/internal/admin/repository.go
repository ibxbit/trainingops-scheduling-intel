package admin

import (
	"context"
	"database/sql"
	"errors"

	"trainingops/backend/internal/dbctx"
	"trainingops/backend/internal/rbac"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidRole   = errors.New("invalid role")
	ErrInvalidPolicy = errors.New("invalid policy")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListTenantSettings(ctx context.Context, tenantID string) ([]TenantSettings, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT
  t.tenant_id::text,
  t.tenant_slug,
  t.name,
  COALESCE(s.allow_self_registration, FALSE),
  COALESCE(s.require_mfa, FALSE),
  COALESCE(s.max_active_bookings_per_learner, 3)
FROM tenants t
LEFT JOIN tenant_settings s ON s.tenant_id = t.tenant_id
WHERE t.tenant_id::text = $1
ORDER BY t.name ASC
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]TenantSettings, 0, 1)
	for rows.Next() {
		var item TenantSettings
		if err := rows.Scan(
			&item.TenantID,
			&item.TenantSlug,
			&item.TenantName,
			&item.AllowSelfRegistration,
			&item.RequireMFA,
			&item.MaxActiveBookingsPerLearner,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) CreateTenantSettings(ctx context.Context, settings TenantSettings) (*TenantSettings, error) {
	var out TenantSettings
	err := dbctx.QueryRowContext(ctx, r.db, `
WITH upsert_tenant AS (
  UPDATE tenants
  SET tenant_slug = $2,
      name = $3
  WHERE tenant_id::text = $1
  RETURNING tenant_id, tenant_slug, name
),
upsert_settings AS (
  INSERT INTO tenant_settings (tenant_id, allow_self_registration, require_mfa, max_active_bookings_per_learner, updated_at)
  VALUES ((SELECT tenant_id FROM upsert_tenant), $4, $5, $6, NOW())
  ON CONFLICT (tenant_id) DO UPDATE
  SET allow_self_registration = EXCLUDED.allow_self_registration,
      require_mfa = EXCLUDED.require_mfa,
      max_active_bookings_per_learner = EXCLUDED.max_active_bookings_per_learner,
      updated_at = NOW()
)
SELECT
  t.tenant_id::text,
  t.tenant_slug,
  t.name,
  s.allow_self_registration,
  s.require_mfa,
  s.max_active_bookings_per_learner
FROM tenants t
JOIN tenant_settings s ON s.tenant_id = t.tenant_id
WHERE t.tenant_id::text = $1
`,
		settings.TenantID,
		settings.TenantSlug,
		settings.TenantName,
		settings.AllowSelfRegistration,
		settings.RequireMFA,
		settings.MaxActiveBookingsPerLearner,
	).Scan(
		&out.TenantID,
		&out.TenantSlug,
		&out.TenantName,
		&out.AllowSelfRegistration,
		&out.RequireMFA,
		&out.MaxActiveBookingsPerLearner,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *Repository) UpdateTenantSettings(ctx context.Context, settings TenantSettings) (*TenantSettings, error) {
	return r.CreateTenantSettings(ctx, settings)
}

func (r *Repository) RolePermissionMatrix(ctx context.Context, tenantID string) ([]RolePermission, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT role::text, permission_key, allowed
FROM role_permissions
WHERE tenant_id::text = $1
ORDER BY role, permission_key
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RolePermission, 0, 16)
	for rows.Next() {
		var item RolePermission
		if err := rows.Scan(&item.Role, &item.Permission, &item.Allowed); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) UpdateRolePermissionMatrix(ctx context.Context, tenantID string, assignments []RolePermission) error {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, assignment := range assignments {
		if !rbac.IsKnownRole(assignment.Role) {
			return ErrInvalidRole
		}
		if !isValidPermissionKey(assignment.Permission) {
			return ErrInvalidPolicy
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO role_permissions (tenant_id, role, permission_key, allowed, updated_at)
VALUES ($1::uuid, $2, $3, $4, NOW())
ON CONFLICT (tenant_id, role, permission_key) DO UPDATE
SET allowed = EXCLUDED.allowed,
    updated_at = NOW()
`, tenantID, assignment.Role, assignment.Permission, assignment.Allowed); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) ListUserRoleAssignments(ctx context.Context, tenantID string) ([]UserRoleAssignment, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT
  u.user_id::text,
  u.username,
  COALESCE(array_remove(array_agg(ur.role ORDER BY ur.role), NULL), ARRAY[]::app_role[])
FROM users u
LEFT JOIN user_roles ur ON ur.tenant_id = u.tenant_id AND ur.user_id = u.user_id
WHERE u.tenant_id::text = $1
GROUP BY u.user_id, u.username
ORDER BY u.username ASC
`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]UserRoleAssignment, 0, 8)
	for rows.Next() {
		var item UserRoleAssignment
		var roles []string
		if err := rows.Scan(&item.UserID, &item.Username, &roles); err != nil {
			return nil, err
		}
		item.Roles = make([]rbac.Role, 0, len(roles))
		for _, role := range roles {
			item.Roles = append(item.Roles, rbac.Role(role))
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Repository) AssignUserRole(ctx context.Context, tenantID, userID string, role rbac.Role) error {
	if !rbac.IsKnownRole(role) {
		return ErrInvalidRole
	}
	res, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO user_roles (tenant_id, user_id, role)
VALUES ($1::uuid, $2::uuid, $3)
ON CONFLICT (tenant_id, user_id, role) DO NOTHING
`, tenantID, userID, role)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		var exists int
		err := dbctx.QueryRowContext(ctx, r.db, `
SELECT COUNT(*)
FROM users
WHERE tenant_id::text = $1 AND user_id::text = $2
`, tenantID, userID).Scan(&exists)
		if err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func (r *Repository) RevokeUserRole(ctx context.Context, tenantID, userID string, role rbac.Role) error {
	if !rbac.IsKnownRole(role) {
		return ErrInvalidRole
	}
	res, err := dbctx.ExecContext(ctx, r.db, `
DELETE FROM user_roles
WHERE tenant_id::text = $1 AND user_id::text = $2 AND role = $3
`, tenantID, userID, role)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		var exists int
		err := dbctx.QueryRowContext(ctx, r.db, `
SELECT COUNT(*)
FROM users
WHERE tenant_id::text = $1 AND user_id::text = $2
`, tenantID, userID).Scan(&exists)
		if err != nil {
			return err
		}
		if exists == 0 {
			return ErrNotFound
		}
	}
	return nil
}

func isValidPermissionKey(permission string) bool {
	for _, allowed := range []string{
		"tenant.settings.view",
		"tenant.settings.manage",
		"rbac.matrix.view",
		"rbac.matrix.manage",
		"rbac.assignments.view",
		"rbac.assignments.manage",
	} {
		if permission == allowed {
			return true
		}
	}
	return false
}
