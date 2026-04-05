package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type Config struct {
	HTTPAddr             string
	StorageRoot          string
	SessionCookieName    string
	SessionTTL           time.Duration
	SessionSecureCookie  bool
	SessionRotateEvery   time.Duration
	EncryptionKey        []byte
	AllowedUploadFormats map[string]struct{}
	DBConfig             *pgx.ConnConfig
}

func Load() (*Config, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	dbCfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	encKey := []byte(os.Getenv("ENCRYPTION_KEY"))
	if len(encKey) != 32 {
		return nil, errors.New("ENCRYPTION_KEY must be exactly 32 bytes")
	}

	secureCookie := true
	if raw := os.Getenv("SESSION_SECURE_COOKIE"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, errors.New("SESSION_SECURE_COOKIE must be true/false")
		}
		secureCookie = v
	}

	allowed := map[string]struct{}{
		"pdf":  {},
		"txt":  {},
		"md":   {},
		"docx": {},
	}

	return &Config{
		HTTPAddr:             envOr("HTTP_ADDR", ":8080"),
		StorageRoot:          envOr("STORAGE_ROOT", "data"),
		SessionCookieName:    envOr("SESSION_COOKIE_NAME", "trainingops_session"),
		SessionTTL:           24 * time.Hour,
		SessionRotateEvery:   5 * time.Minute,
		SessionSecureCookie:  secureCookie,
		EncryptionKey:        encKey,
		AllowedUploadFormats: allowed,
		DBConfig:             dbCfg,
	}, nil

}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
