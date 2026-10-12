// Package dbtest provides a sqlmock-backed *sqlx.DB for repository unit
// tests, so they exercise the SQL mapping without a real Postgres server.
package dbtest

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// New returns a *sqlx.DB wired to a sqlmock and the mock itself. Queries are
// matched as regular expressions against the SQL text. When the test ends it
// fails if any registered expectation was not met.
func New(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}

	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		_ = db.Close()
	})

	return sqlx.NewDb(db, "pgx"), mock
}
