package tripdate

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

var tripDateColumns = []string{
	"id", "listing_id", "start_date", "end_date", "max_capacity", "current_bookings", "is_active", "created_at", "updated_at",
}

func TestPostgresRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		maxCapacity *int
		wantArg     any
	}{
		{"with capacity", func() *int { c := 10; return &c }(), 10},
		{"unlimited capacity", nil, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewPostgresRepository(db)
			now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			start := now.AddDate(0, 1, 0)
			end := start.AddDate(0, 0, 7)

			mock.ExpectExec(`INSERT INTO trip_dates`).
				WithArgs(sqlmock.AnyArg(), "list-1", start, end, tc.wantArg, 0, true, now, now).
				WillReturnResult(sqlmock.NewResult(0, 1))

			created, err := repo.Create(context.Background(), TripDate{
				ListingID: "list-1", StartDate: start, EndDate: end, MaxCapacity: tc.maxCapacity,
				IsActive: true, CreatedAt: now, UpdatedAt: now,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, created.ID)
		})
	}
}

func TestPostgresRepository_Create_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`INSERT INTO trip_dates`).WillReturnError(boom)

	_, err := repo.Create(context.Background(), TripDate{ListingID: "list-1"})
	require.ErrorIs(t, err, boom)
}

func TestPostgresRepository_GetByID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	start := now.AddDate(0, 1, 0)
	end := start.AddDate(0, 0, 7)

	mock.ExpectQuery(`FROM trip_dates\s+WHERE id = \$1`).
		WithArgs("td-1").
		WillReturnRows(sqlmock.NewRows(tripDateColumns).
			AddRow("td-1", "list-1", start, end, int64(10), 3, true, now, now))

	got, err := repo.GetByID(context.Background(), "td-1")
	require.NoError(t, err)
	maxCapacity := 10
	assert.Equal(t, TripDate{
		ID: "td-1", ListingID: "list-1", StartDate: start, EndDate: end, MaxCapacity: &maxCapacity,
		CurrentBookings: 3, IsActive: true, CreatedAt: now, UpdatedAt: now,
	}, got)
}

func TestPostgresRepository_GetByID_Errors(t *testing.T) {
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
			repo := NewPostgresRepository(db)

			mock.ExpectQuery(`FROM trip_dates`).WillReturnError(tc.dbErr)

			_, err := repo.GetByID(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPostgresRepository_ListByListingID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM trip_dates\s+WHERE listing_id = \$1\s+ORDER BY start_date`).
		WithArgs("list-1").
		WillReturnRows(sqlmock.NewRows(tripDateColumns).
			AddRow("td-1", "list-1", now, now.AddDate(0, 0, 3), nil, 0, true, now, now).
			AddRow("td-2", "list-1", now.AddDate(0, 1, 0), now.AddDate(0, 1, 3), int64(5), 5, false, now, now))

	got, err := repo.ListByListingID(context.Background(), "list-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Nil(t, got[0].MaxCapacity)
	require.NotNil(t, got[1].MaxCapacity)
	assert.Equal(t, 5, *got[1].MaxCapacity)
}

func TestPostgresRepository_ListByListingID_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	boom := errors.New("boom")

	mock.ExpectQuery(`FROM trip_dates`).WillReturnError(boom)

	_, err := repo.ListByListingID(context.Background(), "list-1")
	require.ErrorIs(t, err, boom)
}

func TestPostgresRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO trip_dates`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	_, err = repo.WithTx(tx).Create(context.Background(), TripDate{ListingID: "list-1"})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}
