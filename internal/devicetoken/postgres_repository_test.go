package devicetoken

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/diegoczajka/travelu-api/internal/platform/database/dbtest"
)

var deviceTokenColumns = []string{"id", "user_id", "token", "platform", "is_active", "created_at", "updated_at"}

func TestPostgresRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectExec(`INSERT INTO device_tokens`).
		WithArgs(sqlmock.AnyArg(), "u-1", "abc123", PlatformAndroid, true, now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), DeviceToken{
		UserID: "u-1", Token: "abc123", Platform: PlatformAndroid, IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
}

func TestPostgresRepository_Create_Errors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name  string
		dbErr error
		want  error
	}{
		{"duplicate token", &pgconn.PgError{Code: pgUniqueViolation, ConstraintName: "uq_device_tokens_token"}, ErrTokenTaken},
		{"db error", boom, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewPostgresRepository(db)

			mock.ExpectExec(`INSERT INTO device_tokens`).WillReturnError(tc.dbErr)

			_, err := repo.Create(context.Background(), DeviceToken{UserID: "u-1", Token: "abc123"})
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPostgresRepository_GetByID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM device_tokens\s+WHERE id = \$1`).
		WithArgs("dt-1").
		WillReturnRows(sqlmock.NewRows(deviceTokenColumns).
			AddRow("dt-1", "u-1", "abc123", PlatformAndroid, true, now, now))

	got, err := repo.GetByID(context.Background(), "dt-1")
	require.NoError(t, err)
	assert.Equal(t, DeviceToken{
		ID: "dt-1", UserID: "u-1", Token: "abc123", Platform: PlatformAndroid,
		IsActive: true, CreatedAt: now, UpdatedAt: now,
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

			mock.ExpectQuery(`FROM device_tokens`).WillReturnError(tc.dbErr)

			_, err := repo.GetByID(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPostgresRepository_ListByUserID(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM device_tokens\s+WHERE user_id = \$1\s+ORDER BY created_at DESC`).
		WithArgs("u-1").
		WillReturnRows(sqlmock.NewRows(deviceTokenColumns).
			AddRow("dt-2", "u-1", "t2", PlatformAndroid, true, now, now).
			AddRow("dt-1", "u-1", "t1", PlatformAndroid, false, now, now))

	got, err := repo.ListByUserID(context.Background(), "u-1")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "t2", got[0].Token)
	assert.False(t, got[1].IsActive)
}

func TestPostgresRepository_ListByUserID_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)
	boom := errors.New("boom")

	mock.ExpectQuery(`FROM device_tokens`).WillReturnError(boom)

	_, err := repo.ListByUserID(context.Background(), "u-1")
	require.ErrorIs(t, err, boom)
}

func TestPostgresRepository_Deactivate(t *testing.T) {
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
			repo := NewPostgresRepository(db)

			tc.result(mock.ExpectExec(`UPDATE device_tokens SET is_active = FALSE WHERE id = \$1`).WithArgs("dt-1"))

			err := repo.Deactivate(context.Background(), "dt-1")
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPostgresRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewPostgresRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE device_tokens`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	require.NoError(t, repo.WithTx(tx).Deactivate(context.Background(), "dt-1"))
	require.NoError(t, tx.Commit())
}
