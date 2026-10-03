package devicetoken

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/diegoczajka/travelu-api/internal/platform/migrations"
	"github.com/diegoczajka/travelu-api/internal/user"
	"github.com/google/uuid"
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
		_, _ = db.Exec("DELETE FROM device_tokens")
		_, _ = db.Exec("DELETE FROM users")
	}
	cleanup()
	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	return db
}

// seedUser creates a user row to satisfy device_tokens' user_id foreign key.
func seedUser(t *testing.T, db *sqlx.DB) string {
	t.Helper()

	u, err := user.NewMySQLRepository(db).Create(context.Background(), user.User{
		Name: "User", Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		Role: user.RoleCustomer, IsActive: true, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	return u.ID
}

func TestMySQLRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	userID := seedUser(t, db)

	now := time.Now().UTC().Truncate(time.Microsecond)
	created, err := repo.Create(ctx, DeviceToken{
		UserID: userID, Token: "abc123", Platform: PlatformAndroid, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "abc123", fetched.Token)
}

func TestMySQLRepository_Create_DuplicateToken(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	userID := seedUser(t, db)
	now := time.Now().UTC()

	_, err := repo.Create(ctx, DeviceToken{UserID: userID, Token: "dup-token", Platform: PlatformIOS, IsActive: true, CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	_, err = repo.Create(ctx, DeviceToken{UserID: userID, Token: "dup-token", Platform: PlatformIOS, IsActive: true, CreatedAt: now, UpdatedAt: now})
	require.ErrorIs(t, err, ErrTokenTaken)
}

func TestMySQLRepository_Deactivate(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	userID := seedUser(t, db)
	now := time.Now().UTC()

	created, err := repo.Create(ctx, DeviceToken{UserID: userID, Token: "tok", Platform: PlatformWeb, IsActive: true, CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	require.NoError(t, repo.Deactivate(ctx, created.ID))

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.False(t, fetched.IsActive)
}

func TestMySQLRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, ErrNotFound)
}
