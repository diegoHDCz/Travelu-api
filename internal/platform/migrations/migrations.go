// Package migrations embeds the goose SQL migrations and applies them to
// the MySQL database at application startup.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

// Run applies every pending migration embedded under sql/.
func Run(ctx context.Context, db *sql.DB) error {
	// goose scans the root of the given fs.FS for migration files; sqlFiles
	// is rooted one level up, at sql/, so it must be re-rooted here or every
	// migration is silently invisible to the provider.
	migrationsFS, err := fs.Sub(sqlFiles, "sql")
	if err != nil {
		return fmt.Errorf("sub migrations fs: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectMySQL, db, migrationsFS)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
