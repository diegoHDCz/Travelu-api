package booking

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
		_, _ = db.Exec("DELETE FROM bookings")
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

// seedCustomerAndListing creates a customer and a vendor listing to satisfy
// bookings' customer_id and listing_id foreign keys.
func seedCustomerAndListing(t *testing.T, db *sqlx.DB) (customerID, listingID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	userRepo := user.NewMySQLRepository(db)

	customer, err := userRepo.Create(ctx, user.User{
		Name: "Customer", Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		Role: user.RoleCustomer, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	vendor, err := userRepo.Create(ctx, user.User{
		Name: "Vendor", Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		Role: user.RoleVendor, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	l, err := listing.NewMySQLRepository(db).Create(ctx, listing.Listing{
		VendorID: vendor.ID, Title: "Hotel", Description: "d", Category: listing.CategoryHotel,
		Location: "loc", Price: 10, Currency: "USD", IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	return customer.ID, l.ID
}

func TestMySQLRepository_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	customerID, listingID := seedCustomerAndListing(t, db)

	now := time.Now().UTC().Truncate(time.Microsecond)
	created, err := repo.Create(ctx, Booking{
		CustomerID: customerID, ListingID: listingID, NumberOfGuests: 2, TotalPrice: 199.90,
		Currency: "USD", Status: StatusPending, PaymentStatus: PaymentStatusPending,
		CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)

	fetched, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, 2, fetched.NumberOfGuests)
}

func TestMySQLRepository_ListByCustomerID(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)
	ctx := context.Background()
	customerID, listingID := seedCustomerAndListing(t, db)
	now := time.Now().UTC()

	_, err := repo.Create(ctx, Booking{
		CustomerID: customerID, ListingID: listingID, NumberOfGuests: 1, TotalPrice: 50,
		Currency: "USD", Status: StatusPending, PaymentStatus: PaymentStatusPending, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)

	bookings, err := repo.ListByCustomerID(ctx, customerID)
	require.NoError(t, err)
	require.Len(t, bookings, 1)
}

func TestMySQLRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewMySQLRepository(db)

	_, err := repo.GetByID(context.Background(), "00000000-0000-0000-0000-000000000000")
	require.ErrorIs(t, err, ErrNotFound)
}
