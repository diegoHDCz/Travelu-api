package tripdate

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/diegoczajka/travelu-api/internal/listing"
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
		_, _ = db.Exec("DELETE FROM trip_dates")
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

// seedListing creates a user (vendor) and a listing to satisfy trip_dates'
// listing_id foreign key.
func seedListing(t *testing.T, db *sqlx.DB) string {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	vendor, err := user.NewMySQLRepository(db).Create(ctx, user.User{
		Name: "Vendor", Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		Role: user.RoleVendor, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	l, err := listing.NewMySQLRepository(db).Create(ctx, listing.Listing{
		VendorID: vendor.ID, Title: "Hotel", Description: "d", Category: listing.CategoryHotel,
		Location: "loc", Price: 10, Currency: "USD", IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	return l.ID
}

func TestMySQLRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	listingID := seedListing(t, db)

	now := time.Now().UTC().Truncate(time.Microsecond)
	maxCapacity := 10
	created, err := repo.Create(ctx, TripDate{
		ListingID: listingID, StartDate: now, EndDate: now.Add(48 * time.Hour),
		MaxCapacity: &maxCapacity, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.MaxCapacity)
	require.Equal(t, 10, *fetched.MaxCapacity)
}

func TestMySQLRepository_ListByListingID(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	listingID := seedListing(t, db)
	now := time.Now().UTC()

	_, err := repo.Create(ctx, TripDate{ListingID: listingID, StartDate: now, EndDate: now.Add(time.Hour), IsActive: true, CreatedAt: now, UpdatedAt: now})
	require.NoError(t, err)

	tripDates, err := repo.ListByListingID(ctx, listingID)
	require.NoError(t, err)
	require.Len(t, tripDates, 1)
}

func TestMySQLRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, ErrNotFound)
}
