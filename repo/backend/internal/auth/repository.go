package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"trainingops/backend/internal/dbctx"
	"trainingops/backend/internal/rbac"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserByTenantUsername(ctx context.Context, tenantSlug, username string) (*User, error) {
	const q = `
SELECT t.tenant_id::text, u.user_id::text, u.username, u.password_hash, u.failed_attempts, u.lockout_until, u.is_active
FROM users u
JOIN tenants t ON t.tenant_id = u.tenant_id
WHERE t.tenant_slug = $1 AND u.username = $2
LIMIT 1`

	var u User
	err := dbctx.QueryRowContext(ctx, r.db, q, tenantSlug, username).Scan(
		&u.TenantID,
		&u.UserID,
		&u.Username,
		&u.PasswordHash,
		&u.FailedAttempts,
		&u.LockoutUntil,
		&u.IsActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repository) IncrementFailedLogin(ctx context.Context, tenantID, userID string, lockout bool, lockoutUntil time.Time) error {
	if lockout {
		_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE users
SET failed_attempts = failed_attempts + 1,
    lockout_until = $3,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND user_id::text = $2`, tenantID, userID, lockoutUntil)
		return err
	}

	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE users
SET failed_attempts = failed_attempts + 1,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND user_id::text = $2`, tenantID, userID)
	return err
}

func (r *Repository) ResetFailedLogin(ctx context.Context, tenantID, userID string) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE users
SET failed_attempts = 0,
    lockout_until = NULL,
    updated_at = NOW()
WHERE tenant_id::text = $1 AND user_id::text = $2`, tenantID, userID)
	return err
}

func (r *Repository) CreateSession(ctx context.Context, s Session) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
INSERT INTO auth_sessions (
  session_id, tenant_id, user_id, token_hash, expires_at, last_rotated_at, client_ip_enc, user_agent_enc, rotation_number
)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, NOW(), $6, $7, 0)
`, s.SessionID, s.TenantID, s.UserID, s.TokenHash, s.ExpiresAt, s.ClientIPEnc, s.UserAgentEnc)
	return err
}

func (r *Repository) GetRolesByTenantUser(ctx context.Context, tenantID, userID string) ([]rbac.Role, error) {
	rows, err := dbctx.QueryContext(ctx, r.db, `
SELECT role
FROM user_roles
WHERE tenant_id::text = $1 AND user_id::text = $2`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roles := make([]rbac.Role, 0, 2)
	for rows.Next() {
		var role rbac.Role
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return roles, nil
}

func (r *Repository) RotateSessionByToken(ctx context.Context, tenantID string, oldHash, newHash []byte, rotateIfBefore time.Time) (*Session, error) {
	tx, err := dbctx.BeginTx(ctx, r.db, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	const selectQ = `
SELECT session_id::text, tenant_id::text, user_id::text, token_hash, expires_at, revoked_at, last_rotated_at, client_ip_enc, user_agent_enc, rotation_number
FROM auth_sessions
WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
  AND tenant_id::text = $2
FOR UPDATE`

	var s Session
	err = tx.QueryRowContext(ctx, selectQ, oldHash, tenantID).Scan(
		&s.SessionID,
		&s.TenantID,
		&s.UserID,
		&s.TokenHash,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.LastRotatedAt,
		&s.ClientIPEnc,
		&s.UserAgentEnc,
		&s.RotationNumber,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if s.LastRotatedAt.Before(rotateIfBefore) {
		_, err = tx.ExecContext(ctx, `
UPDATE auth_sessions
SET token_hash = $2,
    last_rotated_at = NOW(),
    rotation_number = rotation_number + 1
WHERE session_id::text = $1 AND tenant_id::text = $3`, s.SessionID, newHash, tenantID)
		if err != nil {
			return nil, err
		}
		s.TokenHash = newHash
		s.RotationNumber++
		s.LastRotatedAt = time.Now().UTC()
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) RevokeSessionByToken(ctx context.Context, tenantID string, tokenHash []byte) error {
	_, err := dbctx.ExecContext(ctx, r.db, `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE token_hash = $1 AND revoked_at IS NULL AND tenant_id::text = $2`, tokenHash, tenantID)
	return err
}
