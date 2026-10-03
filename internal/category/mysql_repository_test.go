package category

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/diegoczajka/travelu-api/internal/platform/migrations"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
)

// openTestDB connects to TEST_DATABASE_DSN, applies migrations and cleans
// the tables this package touches. Skips the test if the variable is unset.
func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}

	db, err := sqlx.Open("mysql", dsn)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	require.NoError(t, migrations.Run(context.Background(), db.DB))

	cleanup := func() {
		_, _ = db.Exec("DELETE FROM categories")
	}
	cleanup()
	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	return db
}

func TestMySQLRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	created, err := repo.Create(ctx, Category{
		Name:      "Hotels",
		Slug:      "hotels",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	bySlug, err := repo.GetBySlug(ctx, "hotels")
	require.NoError(t, err)
	require.Equal(t, created.ID, bySlug.ID)

	byID, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "Hotels", byID.Name)
}

func TestMySQLRepository_Create_DuplicateSlug(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := repo.Create(ctx, Category{Name: "Hotels", Slug: "hotels", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	_, err = repo.Create(ctx, Category{Name: "Other", Slug: "hotels", CreatedAt: now, UpdatedAt: now})
	require.ErrorIs(t, err, ErrSlugTaken)
}

func TestMySQLRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, ErrNotFound)
}
