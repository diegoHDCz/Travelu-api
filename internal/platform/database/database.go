// Package database opens and configures the MySQL connection pool used by
// every repository in the application.
package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	_ "github.com/go-sql-driver/mysql"
)

// Options configures the connection pool. DSN must already include
// parseTime=true&loc=UTC&charset=utf8mb4 (and tls=preferred in production).
type Options struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Open creates the sqlx pool and verifies connectivity with a single ping,
// so the process fails fast at startup if the database is unreachable.
func Open(ctx context.Context, opts Options) (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", opts.DSN)
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
