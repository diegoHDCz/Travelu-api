package notification

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
		_, _ = db.Exec("DELETE FROM notifications")
		_, _ = db.Exec("DELETE FROM users")
	}
	cleanup()
	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	return db
}

// seedUser creates a user row to satisfy notifications' user_id foreign key.
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

	created, err := repo.Create(ctx, Notification{
		UserID: userID, Title: "Booking confirmed", Body: "body", Type: TypeBookingConfirmed,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	})
	require.NoError(t, err)

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.False(t, fetched.IsRead)
}

func TestMySQLRepository_MarkRead(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	userID := seedUser(t, db)

	created, err := repo.Create(ctx, Notification{UserID: userID, Title: "t", Body: "b", Type: TypeTripReminder, CreatedAt: time.Now().UTC()})
	require.NoError(t, err)

	require.NoError(t, repo.MarkRead(ctx, created.ID))

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, fetched.IsRead)
}

func TestMySQLRepository_MarkRead_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)

	err := repo.MarkRead(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMySQLRepository_ListByUserID(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	userID := seedUser(t, db)

	_, err := repo.Create(ctx, Notification{UserID: userID, Title: "t", Body: "b", Type: TypeRateTrip, CreatedAt: time.Now().UTC()})
	require.NoError(t, err)

	notifications, err := repo.ListByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, notifications, 1)
}
