package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	root := os.Getenv("MIGRATIONS_DIR")
	if root == "" {
		root = "migrations"
	}

	db, err := openWithRetry(dsn, 30, 2*time.Second)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	if err := ensureTable(db); err != nil {
		log.Fatalf("ensure migrations table failed: %v", err)
	}

	files, err := listSQLFiles(root)
	if err != nil {
		log.Fatalf("list migrations failed: %v", err)
	}

	for _, file := range files {
		applied, err := isApplied(db, file)
		if err != nil {
			log.Fatalf("check migration failed (%s): %v", file, err)
		}
		if applied {
			continue
		}
		if err := apply(db, filepath.Join(root, file), file); err != nil {
			log.Fatalf("apply migration failed (%s): %v", file, err)
		}
		log.Printf("applied migration: %s", file)
	}
}

func openWithRetry(dsn string, attempts int, wait time.Duration) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	db := stdlib.OpenDB(*cfg)
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			return db, nil
		}
		time.Sleep(wait)
	}
	return nil, err
}

func ensureTable(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  filename TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`)
	return err
}

func listSQLFiles(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

func isApplied(db *sql.DB, filename string) (bool, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE filename = $1`, filename).Scan(&n)
	return n > 0, err
}

func apply(db *sql.DB, path, filename string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return errors.New("empty migration: " + filename)
	}
	if _, err := db.Exec(string(b)); err != nil {
		return fmt.Errorf("exec failed: %w", err)
	}
	_, err = db.Exec(`INSERT INTO schema_migrations (filename) VALUES ($1)`, filename)
	return err
}
