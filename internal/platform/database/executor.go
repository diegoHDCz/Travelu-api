package database

import (
	"context"
	"database/sql"
)

// SQLExecutor is the subset of *sqlx.DB and *sqlx.Tx that repositories need.
// Both types satisfy it, so a repository built against this interface can
// run against the pool or inside a transaction started by WithTx.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}
