package listing

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
		_, _ = db.Exec("DELETE FROM travel_listings")
		_, _ = db.Exec("DELETE FROM users")
	}
	cleanup()
	t.Cleanup(func() {
		cleanup()
		db.Close()
	})

	return db
}

// seedVendor creates a user row to satisfy travel_listings' vendor_id
// foreign key.
func seedVendor(t *testing.T, db *sqlx.DB) string {
	t.Helper()

	vendor, err := user.NewMySQLRepository(db).Create(context.Background(), user.User{
		Name:         "Vendor",
		Email:        uuid.NewString() + "@example.com",
		PasswordHash: "hash",
		Role:         user.RoleVendor,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})
	require.NoError(t, err)
	return vendor.ID
}

func TestMySQLRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	vendorID := seedVendor(t, db)

	now := time.Now().UTC().Truncate(time.Microsecond)
	capacity := 4
	created, err := repo.Create(ctx, Listing{
		VendorID:    vendorID,
		Title:       "Beach Hotel",
		Description: "A nice hotel",
		Category:    CategoryHotel,
		Location:    "Rio de Janeiro",
		Price:       199.90,
		Currency:    "USD",
		Capacity:    &capacity,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "Beach Hotel", fetched.Title)
	require.NotNil(t, fetched.Capacity)
	require.Equal(t, 4, *fetched.Capacity)
}

func TestMySQLRepository_ListActive(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	vendorID := seedVendor(t, db)
	now := time.Now().UTC()

	_, err := repo.Create(ctx, Listing{
		VendorID: vendorID, Title: "Active", Description: "d", Category: CategoryHotel,
		Location: "loc", Price: 10, Currency: "USD", IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	_, err = repo.Create(ctx, Listing{
		VendorID: vendorID, Title: "Inactive", Description: "d", Category: CategoryHotel,
		Location: "loc", Price: 10, Currency: "USD", IsActive: false, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	active, err := repo.ListActive(ctx)
	require.NoError(t, err)
	require.Len(t, active, 1)
	require.Equal(t, "Active", active[0].Title)
}

func TestMySQLRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, ErrNotFound)
}
