package auth

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

var refreshTokenColumns = []string{"id", "user_id", "token_hash", "expires_at", "revoked_at", "created_at"}

func TestMySQLRefreshTokenRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRefreshTokenRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectExec(`INSERT INTO refresh_tokens`).
		WithArgs(sqlmock.AnyArg(), "u-1", "abc123", now.Add(time.Hour), now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), RefreshToken{
		UserID: "u-1", TokenHash: "abc123", ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "abc123", created.TokenHash)
}

func TestMySQLRefreshTokenRepository_Create_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRefreshTokenRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`INSERT INTO refresh_tokens`).WillReturnError(boom)

	_, err := repo.Create(context.Background(), RefreshToken{UserID: "u-1"})
	require.ErrorIs(t, err, boom)
}

func TestMySQLRefreshTokenRepository_GetByTokenHash(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	revoked := now.Add(time.Minute)

	tests := []struct {
		name      string
		revokedAt any
		want      *time.Time
	}{
		{"active", nil, nil},
		{"revoked", revoked, &revoked},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRefreshTokenRepository(db)

			mock.ExpectQuery(`FROM refresh_tokens\s+WHERE token_hash = \?`).
				WithArgs("abc123").
				WillReturnRows(sqlmock.NewRows(refreshTokenColumns).
					AddRow("rt-1", "u-1", "abc123", now.Add(time.Hour), tc.revokedAt, now))

			got, err := repo.GetByTokenHash(context.Background(), "abc123")
			require.NoError(t, err)
			assert.Equal(t, RefreshToken{
				ID: "rt-1", UserID: "u-1", TokenHash: "abc123",
				ExpiresAt: now.Add(time.Hour), RevokedAt: tc.want, CreatedAt: now,
			}, got)
		})
	}
}

func TestMySQLRefreshTokenRepository_GetByTokenHash_Errors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name  string
		dbErr error
		want  error
	}{
		{"not found", sql.ErrNoRows, errRefreshTokenNotFound},
		{"db error", boom, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRefreshTokenRepository(db)

			mock.ExpectQuery(`FROM refresh_tokens`).WillReturnError(tc.dbErr)

			_, err := repo.GetByTokenHash(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRefreshTokenRepository_Revoke(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRefreshTokenRepository(db)

	mock.ExpectExec(`UPDATE refresh_tokens SET revoked_at = \? WHERE id = \? AND revoked_at IS NULL`).
		WithArgs(sqlmock.AnyArg(), "rt-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, repo.Revoke(context.Background(), "rt-1"))
}

func TestMySQLRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRefreshTokenRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`UPDATE refresh_tokens SET revoked_at = \? WHERE user_id = \? AND revoked_at IS NULL`).
		WithArgs(sqlmock.AnyArg(), "u-1").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(`UPDATE refresh_tokens`).WillReturnError(boom)

	require.NoError(t, repo.RevokeAllForUser(context.Background(), "u-1"))
	require.ErrorIs(t, repo.RevokeAllForUser(context.Background(), "u-1"), boom)
}

func TestMySQLRefreshTokenRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRefreshTokenRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE refresh_tokens`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	require.NoError(t, repo.WithTx(tx).Revoke(context.Background(), "rt-1"))
	require.NoError(t, tx.Commit())
}
