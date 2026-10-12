package notification

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

var notificationColumns = []string{"id", "user_id", "title", "body", "type", "reference_id", "is_read", "created_at"}

func TestMySQLRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	// An empty reference id must go out as NULL.
	mock.ExpectExec(`INSERT INTO notifications`).
		WithArgs(sqlmock.AnyArg(), "u-1", "Hi", "Body", TypeTripReminder, nil, false, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), Notification{
		UserID: "u-1", Title: "Hi", Body: "Body", Type: TypeTripReminder, CreatedAt: now,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
}

func TestMySQLRepository_Create_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`INSERT INTO notifications`).WillReturnError(boom)

	_, err := repo.Create(context.Background(), Notification{UserID: "u-1"})
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_GetByID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM notifications\s+WHERE id = \?`).
		WithArgs("n-1").
		WillReturnRows(sqlmock.NewRows(notificationColumns).
			AddRow("n-1", "u-1", "Confirmed", "Your booking", TypeBookingConfirmed, "b-1", true, now))

	got, err := repo.GetByID(context.Background(), "n-1")
	require.NoError(t, err)
	assert.Equal(t, Notification{
		ID: "n-1", UserID: "u-1", Title: "Confirmed", Body: "Your booking",
		Type: TypeBookingConfirmed, ReferenceID: "b-1", IsRead: true, CreatedAt: now,
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

			mock.ExpectQuery(`FROM notifications`).WillReturnError(tc.dbErr)

			_, err := repo.GetByID(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_ListByUserID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM notifications\s+WHERE user_id = \?\s+ORDER BY created_at DESC`).
		WithArgs("u-1").
		WillReturnRows(sqlmock.NewRows(notificationColumns).
			AddRow("n-2", "u-1", "B", "b", TypeRateTrip, nil, false, now).
			AddRow("n-1", "u-1", "A", "a", TypeTripReminder, nil, true, now))

	got, err := repo.ListByUserID(context.Background(), "u-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "n-2", got[0].ID)
	assert.Empty(t, got[0].ReferenceID)
	assert.True(t, got[1].IsRead)
}

func TestMySQLRepository_ListByUserID_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectQuery(`FROM notifications`).WillReturnError(boom)

	_, err := repo.ListByUserID(context.Background(), "u-1")
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_MarkRead(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name   string
		result func(*sqlmock.ExpectedExec)
		want   error
	}{
		{"ok", func(e *sqlmock.ExpectedExec) { e.WillReturnResult(sqlmock.NewResult(0, 1)) }, nil},
		{"not found", func(e *sqlmock.ExpectedExec) { e.WillReturnResult(sqlmock.NewResult(0, 0)) }, ErrNotFound},
		{"db error", func(e *sqlmock.ExpectedExec) { e.WillReturnError(boom) }, boom},
		{"rows affected error", func(e *sqlmock.ExpectedExec) { e.WillReturnResult(sqlmock.NewErrorResult(boom)) }, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)

			tc.result(mock.ExpectExec(`UPDATE notifications SET is_read = TRUE WHERE id = \?`).WithArgs("n-1"))

			err := repo.MarkRead(context.Background(), "n-1")
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE notifications`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	require.NoError(t, repo.WithTx(tx).MarkRead(context.Background(), "n-1"))
	require.NoError(t, tx.Commit())
}
