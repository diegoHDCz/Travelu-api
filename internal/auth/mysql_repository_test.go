package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"

	"github.com/diegoczajka/travelu-api/internal/platform/migrations"
	"github.com/diegoczajka/travelu-api/internal/user"
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
		_, _ = db.Exec("DELETE FROM refresh_tokens")
		_, _ = db.Exec("DELETE FROM users")
	}
	cleanup()
	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	return db
}

func createTestUser(t *testing.T, db *sqlx.DB) user.User {
	t.Helper()

	repo := user.NewMySQLRepository(db)
	now := time.Now().UTC()
	u, err := repo.Create(context.Background(), user.User{
		Name:         "Diego",
		Email:        "diego@example.com",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	require.NoError(t, err)
	return u
}

func TestMySQLRefreshTokenRepository_CreateGetRevoke(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)
	repo := NewMySQLRefreshTokenRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	created, err := repo.Create(ctx, RefreshToken{
		UserID:    u.ID,
		TokenHash: "abc123",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	fetched, err := repo.GetByTokenHash(ctx, "abc123")
	require.NoError(t, err)
	require.Nil(t, fetched.RevokedAt)

	require.NoError(t, repo.Revoke(ctx, created.ID))

	fetched, err = repo.GetByTokenHash(ctx, "abc123")
	require.NoError(t, err)
	require.NotNil(t, fetched.RevokedAt)
}

func TestMySQLRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	db := openTestDB(t)
	u := createTestUser(t, db)
	repo := NewMySQLRefreshTokenRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	for _, hash := range []string{"h1", "h2"} {
		_, err := repo.Create(ctx, RefreshToken{UserID: u.ID, TokenHash: hash, ExpiresAt: now.Add(time.Hour), CreatedAt: now})
		require.NoError(t, err)
	}

	require.NoError(t, repo.RevokeAllForUser(ctx, u.ID))

	for _, hash := range []string{"h1", "h2"} {
		rt, err := repo.GetByTokenHash(ctx, hash)
		require.NoError(t, err)
		require.NotNil(t, rt.RevokedAt)
	}
}

func TestMySQLRefreshTokenRepository_GetByTokenHash_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRefreshTokenRepository(db)

	_, err := repo.GetByTokenHash(context.Background(), "missing")
	require.ErrorIs(t, err, errRefreshTokenNotFound)
}
