// Package migrations embeds the goose SQL migrations and applies them to
// the MySQL database at application startup.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

// Run applies every pending migration embedded under sql/.
func Run(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectMySQL, db, sqlFiles)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
