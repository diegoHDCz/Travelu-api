package review

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

var reviewColumns = []string{
	"id", "customer_id", "listing_id", "booking_id", "rating", "title", "comment",
	"is_verified", "is_approved", "created_at", "updated_at",
}

func TestPostgresRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	// Empty booking id and title must go out as NULL.
	mock.ExpectExec(`INSERT INTO reviews`).
		WithArgs(sqlmock.AnyArg(), "cust-1", "list-1", nil, 5, nil, "Great",
			false, true, now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), Review{
		CustomerID: "cust-1", ListingID: "list-1", Rating: 5, Comment: "Great",
		IsApproved: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
}

func TestPostgresRepository_Create_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`INSERT INTO reviews`).WillReturnError(boom)

	_, err := repo.Create(context.Background(), Review{CustomerID: "cust-1"})
	require.ErrorIs(t, err, boom)
}

func TestPostgresRepository_GetByID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM reviews\s+WHERE id = \$1`).
		WithArgs("r-1").
		WillReturnRows(sqlmock.NewRows(reviewColumns).
			AddRow("r-1", "cust-1", "list-1", "b-1", 4, "Nice", "Good stay", true, true, now, now))

	got, err := repo.GetByID(context.Background(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, Review{
		ID: "r-1", CustomerID: "cust-1", ListingID: "list-1", BookingID: "b-1", Rating: 4,
		Title: "Nice", Comment: "Good stay", IsVerified: true, IsApproved: true, CreatedAt: now, UpdatedAt: now,
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

			mock.ExpectQuery(`FROM reviews`).WillReturnError(tc.dbErr)

			_, err := repo.GetByID(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPostgresRepository_ListByListingID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	// Only approved reviews are listed.
	mock.ExpectQuery(`FROM reviews\s+WHERE listing_id = \$1 AND is_approved = TRUE\s+ORDER BY created_at DESC`).
		WithArgs("list-1").
		WillReturnRows(sqlmock.NewRows(reviewColumns).
			AddRow("r-2", "cust-2", "list-1", nil, 3, nil, nil, false, true, now, now).
			AddRow("r-1", "cust-1", "list-1", nil, 5, "Top", nil, false, true, now, now))

	got, err := repo.ListByListingID(context.Background(), "list-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, 3, got[0].Rating)
	assert.Equal(t, "Top", got[1].Title)
}

func TestPostgresRepository_ListByListingID_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	boom := errors.New("boom")

	mock.ExpectQuery(`FROM reviews`).WillReturnError(boom)

	_, err := repo.ListByListingID(context.Background(), "list-1")
	require.ErrorIs(t, err, boom)
}

func TestPostgresRepository_Update(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)

	mock.ExpectExec(`UPDATE reviews\s+SET rating = \$1, title = \$2, comment = \$3, updated_at = \$4\s+WHERE id = \$5`).
		WithArgs(4, "Updated", nil, sqlmock.AnyArg(), "r-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	got, err := repo.Update(context.Background(), Review{ID: "r-1", Rating: 4, Title: "Updated"})
	require.NoError(t, err)
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestPostgresRepository_Update_Errors(t *testing.T) {
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
			repo := NewPostgresRepository(db)
			tc.result(mock.ExpectExec(`UPDATE reviews`))

			_, err := repo.Update(context.Background(), Review{ID: "r-1", Rating: 4})
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPostgresRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO reviews`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	_, err = repo.WithTx(tx).Create(context.Background(), Review{CustomerID: "cust-1"})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}
