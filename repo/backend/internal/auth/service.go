package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"trainingops/backend/internal/config"
	"trainingops/backend/internal/security"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserLocked         = errors.New("user is temporarily locked")
	ErrInactiveUser       = errors.New("user is inactive")
	ErrInvalidSession     = errors.New("invalid session")
)

type Service struct {
	repo *Repository
	enc  *security.Encryptor
	cfg  *config.Config
}

func NewService(repo *Repository, enc *security.Encryptor, cfg *config.Config) *Service {
	return &Service{repo: repo, enc: enc, cfg: cfg}
}

func (s *Service) Login(ctx context.Context, tenantSlug, username, password, ip, userAgent string) (string, time.Time, error) {
	u, err := s.repo.GetUserByTenantUsername(ctx, tenantSlug, username)
	if err != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}
	if !u.IsActive {
		return "", time.Time{}, ErrInactiveUser
	}
	if u.LockoutUntil != nil && u.LockoutUntil.After(time.Now().UTC()) {
		return "", time.Time{}, ErrUserLocked
	}

	if err := security.ComparePassword(u.PasswordHash, password); err != nil {
		locked := (u.FailedAttempts + 1) >= 5
		lockUntil := time.Now().UTC().Add(15 * time.Minute)
		_ = s.repo.IncrementFailedLogin(ctx, u.TenantID, u.UserID, locked, lockUntil)
		return "", time.Time{}, ErrInvalidCredentials
	}

	if err := s.repo.ResetFailedLogin(ctx, u.TenantID, u.UserID); err != nil {
		return "", time.Time{}, err
	}

	token, tokenHash, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}

	ipEnc, err := s.enc.Encrypt([]byte(ip))
	if err != nil {
		return "", time.Time{}, err
	}
	uaEnc, err := s.enc.Encrypt([]byte(userAgent))
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().UTC().Add(s.cfg.SessionTTL)
	err = s.repo.CreateSession(ctx, Session{
		SessionID:    uuid.NewString(),
		TenantID:     u.TenantID,
		UserID:       u.UserID,
		TokenHash:    tokenHash,
		ExpiresAt:    expiresAt,
		ClientIPEnc:  ipEnc,
		UserAgentEnc: uaEnc,
	})
	if err != nil {
		return "", time.Time{}, err
	}

	return CookieValue(u.TenantID, token), expiresAt, nil
}

func (s *Service) Logout(ctx context.Context, tenantID, token string) error {
	if token == "" || tenantID == "" {
		return nil
	}
	hash := tokenHash(token)
	return s.repo.RevokeSessionByToken(ctx, tenantID, hash)
}

func (s *Service) ValidateAndRotate(ctx context.Context, tenantID, token string) (string, *Session, bool, error) {
	if token == "" || tenantID == "" {
		return "", nil, false, ErrInvalidSession
	}
	oldHash := tokenHash(token)
	newToken, newHash, err := newToken()
	if err != nil {
		return "", nil, false, err
	}

	rotateIfBefore := time.Now().UTC().Add(-s.cfg.SessionRotateEvery)
	session, err := s.repo.RotateSessionByToken(ctx, tenantID, oldHash, newHash, rotateIfBefore)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", nil, false, ErrInvalidSession
		}
		return "", nil, false, err
	}

	roles, err := s.repo.GetRolesByTenantUser(ctx, session.TenantID, session.UserID)
	if err != nil {
		return "", nil, false, err
	}
	session.Roles = roles

	rotated := string(session.TokenHash) == string(newHash)
	if !rotated {
		newToken = CookieValue(session.TenantID, token)
	} else {
		newToken = CookieValue(session.TenantID, newToken)
	}
	return newToken, session, rotated, nil
}

func newToken() (string, []byte, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	return token, sum[:], nil
}

func tokenHash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func CookieValue(tenantID, raw string) string {
	return fmt.Sprintf("v1.%s.%s", tenantID, raw)
}

func ParseCookieValue(cookieValue string) (string, string) {
	const prefix = "v1."
	if len(cookieValue) <= len(prefix) || cookieValue[:len(prefix)] != prefix {
		return "", ""
	}
	rest := cookieValue[len(prefix):]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '.' {
			if i == 0 || i+1 >= len(rest) {
				return "", ""
			}
			return rest[:i], rest[i+1:]
		}
	}
	return "", ""
}
