package booking

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/diegoczajka/travelu-api/internal/platform/database/dbtest"
)

var bookingColumns = []string{
	"id", "customer_id", "listing_id", "trip_date_id", "check_in_date", "check_out_date",
	"number_of_guests", "total_price", "currency", "status", "payment_status", "payment_id",
	"special_requests", "created_at", "updated_at",
}

func TestMySQLRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	checkIn := now.AddDate(0, 1, 0)

	// Empty optional strings and the zero check-out date must go out as NULL.
	mock.ExpectExec(`INSERT INTO bookings`).
		WithArgs(sqlmock.AnyArg(), "cust-1", "list-1", nil, checkIn, nil,
			2, 300.5, "USD", "pending", "unpaid", nil,
			"late arrival", now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), Booking{
		CustomerID: "cust-1", ListingID: "list-1", CheckInDate: checkIn,
		NumberOfGuests: 2, TotalPrice: 300.5, Currency: "USD", Status: "pending",
		PaymentStatus: "unpaid", SpecialRequests: "late arrival", CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
}

func TestMySQLRepository_Create_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`INSERT INTO bookings`).WillReturnError(boom)

	_, err := repo.Create(context.Background(), Booking{CustomerID: "cust-1"})
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_GetByID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	checkIn := now.AddDate(0, 1, 0)
	checkOut := checkIn.AddDate(0, 0, 3)

	mock.ExpectQuery(`FROM bookings\s+WHERE id = \?`).
		WithArgs("b-1").
		WillReturnRows(sqlmock.NewRows(bookingColumns).AddRow(
			"b-1", "cust-1", "list-1", "td-1", checkIn, checkOut,
			2, 300.5, "USD", "confirmed", "paid", "pay-1",
			nil, now, now))

	got, err := repo.GetByID(context.Background(), "b-1")
	require.NoError(t, err)
	assert.Equal(t, Booking{
		ID: "b-1", CustomerID: "cust-1", ListingID: "list-1", TripDateID: "td-1",
		CheckInDate: checkIn, CheckOutDate: checkOut, NumberOfGuests: 2, TotalPrice: 300.5,
		Currency: "USD", Status: "confirmed", PaymentStatus: "paid", PaymentID: "pay-1",
		CreatedAt: now, UpdatedAt: now,
	}, got)
}

func TestMySQLRepository_GetByID_Errors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name  string
		dbErr error
		want  error
	}{
		{"not found", sql.ErrNoRows, ErrNotFound},
		{"db error", boom, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)

			mock.ExpectQuery(`FROM bookings`).WillReturnError(tc.dbErr)

			_, err := repo.GetByID(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_ListByCustomerID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM bookings\s+WHERE customer_id = \?\s+ORDER BY created_at DESC`).
		WithArgs("cust-1").
		WillReturnRows(sqlmock.NewRows(bookingColumns).
			AddRow("b-2", "cust-1", "list-1", nil, nil, nil, 1, 10.0, "USD", "pending", "unpaid", nil, nil, now, now).
			AddRow("b-1", "cust-1", "list-2", nil, nil, nil, 3, 20.0, "USD", "pending", "unpaid", nil, nil, now, now))

	got, err := repo.ListByCustomerID(context.Background(), "cust-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "b-2", got[0].ID)
	assert.True(t, got[0].CheckInDate.IsZero())
	assert.Equal(t, 3, got[1].NumberOfGuests)
}

func TestMySQLRepository_ListByCustomerID_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectQuery(`FROM bookings`).WillReturnError(boom)

	_, err := repo.ListByCustomerID(context.Background(), "cust-1")
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO bookings`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	_, err = repo.WithTx(tx).Create(context.Background(), Booking{CustomerID: "cust-1"})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}
