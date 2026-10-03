package user

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

func TestMySQLRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	created, err := repo.Create(ctx, User{
		Name:         "Diego",
		Email:        "diego@example.com",
		PasswordHash: "hash",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	byEmail, err := repo.GetByEmail(ctx, "diego@example.com")
	require.NoError(t, err)
	require.Equal(t, created.ID, byEmail.ID)

	byID, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "diego@example.com", byID.Email)
}

func TestMySQLRepository_Create_DuplicateEmail(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := repo.Create(ctx, User{Name: "A", Email: "dup@example.com", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	_, err = repo.Create(ctx, User{Name: "B", Email: "dup@example.com", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestMySQLRepository_Create_DuplicatePhone(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := repo.Create(ctx, User{Name: "A", Phone: "+5511999998888", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	_, err = repo.Create(ctx, User{Name: "B", Phone: "+5511999998888", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.ErrorIs(t, err, ErrPhoneTaken)
}

func TestMySQLRepository_Create_DuplicateUsername(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := repo.Create(ctx, User{Name: "A", Username: "dupuser", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	_, err = repo.Create(ctx, User{Name: "B", Username: "dupuser", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.ErrorIs(t, err, ErrUsernameTaken)
}

func TestMySQLRepository_Create_MultipleWithoutEmail(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := repo.Create(ctx, User{Name: "A", Phone: "+5511999998888", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	_, err = repo.Create(ctx, User{Name: "B", Username: "nouser", PasswordHash: "h", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err, "NULL email/phone/username must not collide across rows")
}

func TestMySQLRepository_GetByEmail_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)

	_, err := repo.GetByEmail(context.Background(), "missing@example.com")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestMySQLRepository_GetByPhoneAndUsername(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)

	created, err := repo.Create(ctx, User{Name: "Diego", Phone: "+5511999998888", Username: "diegoc", PasswordHash: "hash", CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	byPhone, err := repo.GetByPhone(ctx, "+5511999998888")
	require.NoError(t, err)
	require.Equal(t, created.ID, byPhone.ID)

	byUsername, err := repo.GetByUsername(ctx, "diegoc")
	require.NoError(t, err)
	require.Equal(t, created.ID, byUsername.ID)
}
