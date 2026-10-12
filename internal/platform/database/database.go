// Package database opens and configures the Postgres connection pool used by
// every repository in the application.
package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Options configures the connection pool. DSN is a postgres:// URL; in
// production it must carry sslmode=require, and when it points at Supabase's
// transaction pooler (port 6543) also default_query_exec_mode=simple_protocol,
// since that pooler can't keep prepared statements across transactions.
type Options struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Open creates the sqlx pool and verifies connectivity with a single ping,
// so the process fails fast at startup if the database is unreachable.
func Open(ctx context.Context, opts Options) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", opts.DSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
