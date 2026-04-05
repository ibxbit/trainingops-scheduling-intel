package dbctx

import (
	"context"
	"database/sql"
)

type connKey struct{}

func WithConn(ctx context.Context, conn *sql.Conn) context.Context {
	if conn == nil {
		return ctx
	}
	return context.WithValue(ctx, connKey{}, conn)
}

func ConnFromContext(ctx context.Context) *sql.Conn {
	conn, _ := ctx.Value(connKey{}).(*sql.Conn)
	return conn
}

func QueryRowContext(ctx context.Context, db *sql.DB, query string, args ...any) *sql.Row {
	if conn := ConnFromContext(ctx); conn != nil {
		return conn.QueryRowContext(ctx, query, args...)
	}
	return db.QueryRowContext(ctx, query, args...)
}

func QueryContext(ctx context.Context, db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	if conn := ConnFromContext(ctx); conn != nil {
		return conn.QueryContext(ctx, query, args...)
	}
	return db.QueryContext(ctx, query, args...)
}

func ExecContext(ctx context.Context, db *sql.DB, query string, args ...any) (sql.Result, error) {
	if conn := ConnFromContext(ctx); conn != nil {
		return conn.ExecContext(ctx, query, args...)
	}
	return db.ExecContext(ctx, query, args...)
}

func BeginTx(ctx context.Context, db *sql.DB, opts *sql.TxOptions) (*sql.Tx, error) {
	if conn := ConnFromContext(ctx); conn != nil {
		return conn.BeginTx(ctx, opts)
	}
	return db.BeginTx(ctx, opts)
}
