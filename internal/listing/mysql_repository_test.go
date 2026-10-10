package listing

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

var listingColumns = []string{
	"id", "vendor_id", "title", "description", "category", "location", "city", "country",
	"price", "currency", "capacity", "available_from", "available_to", "images", "amenities",
	"rating", "review_count", "is_active", "created_at", "updated_at",
}

func TestMySQLRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	capacity := 4

	// Empty optional strings and zero dates must go out as NULL.
	mock.ExpectExec(`INSERT INTO travel_listings`).
		WithArgs(sqlmock.AnyArg(), "v-1", "Beach Hotel", "A nice hotel", CategoryHotel, "Rio", "Rio de Janeiro", nil,
			199.9, "USD", 4, nil, nil, nil, nil, 0.0, 0, true, now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), Listing{
		VendorID: "v-1", Title: "Beach Hotel", Description: "A nice hotel", Category: CategoryHotel,
		Location: "Rio", City: "Rio de Janeiro", Price: 199.9, Currency: "USD", Capacity: &capacity,
		IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
}

func TestMySQLRepository_Create_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`INSERT INTO travel_listings`).WillReturnError(boom)

	_, err := repo.Create(context.Background(), Listing{VendorID: "v-1"})
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_GetByID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	from := now.AddDate(0, 1, 0)
	to := from.AddDate(0, 1, 0)

	mock.ExpectQuery(`FROM travel_listings\s+WHERE id = \?`).
		WithArgs("l-1").
		WillReturnRows(sqlmock.NewRows(listingColumns).AddRow(
			"l-1", "v-1", "Beach Hotel", "A nice hotel", CategoryHotel, "Rio", "Rio de Janeiro", "BR",
			199.9, "USD", int64(4), from, to, `["a.jpg"]`, `["wifi"]`,
			4.5, 10, true, now, now))

	got, err := repo.GetByID(context.Background(), "l-1")
	require.NoError(t, err)
	capacity := 4
	assert.Equal(t, Listing{
		ID: "l-1", VendorID: "v-1", Title: "Beach Hotel", Description: "A nice hotel", Category: CategoryHotel,
		Location: "Rio", City: "Rio de Janeiro", Country: "BR", Price: 199.9, Currency: "USD",
		Capacity: &capacity, AvailableFrom: from, AvailableTo: to, Images: `["a.jpg"]`, Amenities: `["wifi"]`,
		Rating: 4.5, ReviewCount: 10, IsActive: true, CreatedAt: now, UpdatedAt: now,
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

			mock.ExpectQuery(`FROM travel_listings`).WillReturnError(tc.dbErr)

			_, err := repo.GetByID(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_ListActive(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM travel_listings\s+WHERE is_active = TRUE\s+ORDER BY created_at DESC`).
		WillReturnRows(sqlmock.NewRows(listingColumns).
			AddRow("l-1", "v-1", "A", "d", CategoryHotel, "loc", nil, nil, 10.0, "USD", nil, nil, nil, nil, nil, 0.0, 0, true, now, now))

	got, err := repo.ListActive(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Nil(t, got[0].Capacity)
	assert.True(t, got[0].AvailableFrom.IsZero())
}

func TestMySQLRepository_ListActive_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectQuery(`FROM travel_listings`).WillReturnError(boom)

	_, err := repo.ListActive(context.Background())
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_Update(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectExec(`UPDATE travel_listings\s+SET title = \?`).
		WithArgs("New", "d", CategoryHotel, "loc", nil, "BR",
			50.0, "USD", nil, from, nil,
			nil, "pool", false, sqlmock.AnyArg(), "l-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	got, err := repo.Update(context.Background(), Listing{
		ID: "l-1", Title: "New", Description: "d", Category: CategoryHotel, Location: "loc",
		Country: "BR", Price: 50, Currency: "USD", AvailableFrom: from, Amenities: "pool",
	})
	require.NoError(t, err)
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestMySQLRepository_Update_Errors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name   string
		result func(*sqlmock.ExpectedExec)
		want   error
	}{
		{"not found", func(e *sqlmock.ExpectedExec) { e.WillReturnResult(sqlmock.NewResult(0, 0)) }, ErrNotFound},
		{"db error", func(e *sqlmock.ExpectedExec) { e.WillReturnError(boom) }, boom},
		{"rows affected error", func(e *sqlmock.ExpectedExec) { e.WillReturnResult(sqlmock.NewErrorResult(boom)) }, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)
			tc.result(mock.ExpectExec(`UPDATE travel_listings`))

			_, err := repo.Update(context.Background(), Listing{ID: "l-1"})
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO travel_listings`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	_, err = repo.WithTx(tx).Create(context.Background(), Listing{VendorID: "v-1"})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}
