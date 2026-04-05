package auth

import (
	"time"

	"trainingops/backend/internal/rbac"
)

type User struct {
	TenantID       string
	UserID         string
	Username       string
	PasswordHash   string
	FailedAttempts int
	LockoutUntil   *time.Time
	IsActive       bool
}

type Session struct {
	SessionID      string
	TenantID       string
	UserID         string
	Roles          []rbac.Role
	TokenHash      []byte
	ExpiresAt      time.Time
	RevokedAt      *time.Time
	LastRotatedAt  time.Time
	ClientIPEnc    []byte
	UserAgentEnc   []byte
	RotationNumber int
}
