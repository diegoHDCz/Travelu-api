package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/platform/database"
)

// MySQLRefreshTokenRepository implements RefreshTokenRepository on top of a
// sqlx MySQL pool or, when built via WithTx, on top of an in-flight
// transaction.
type MySQLRefreshTokenRepository struct {
	db database.SQLExecutor
}

// NewMySQLRefreshTokenRepository builds a MySQLRefreshTokenRepository backed by db.
func NewMySQLRefreshTokenRepository(db *sqlx.DB) *MySQLRefreshTokenRepository {
	return &MySQLRefreshTokenRepository{db: db}
}

// WithTx returns a MySQLRefreshTokenRepository whose operations run against
// tx instead of the pool, so they participate in a transaction started by
// database.WithTx.
func (repo *MySQLRefreshTokenRepository) WithTx(tx *sqlx.Tx) *MySQLRefreshTokenRepository {
	return &MySQLRefreshTokenRepository{db: tx}
}

type refreshTokenRow struct {
	ID        string       `db:"id"`
	UserID    string       `db:"user_id"`
	TokenHash string       `db:"token_hash"`
	ExpiresAt time.Time    `db:"expires_at"`
	RevokedAt sql.NullTime `db:"revoked_at"`
	CreatedAt time.Time    `db:"created_at"`
}

func (r refreshTokenRow) toDomain() RefreshToken {
	rt := RefreshToken{
		ID:        r.ID,
		UserID:    r.UserID,
		TokenHash: r.TokenHash,
		ExpiresAt: r.ExpiresAt,
		CreatedAt: r.CreatedAt,
	}
	if r.RevokedAt.Valid {
		rt.RevokedAt = &r.RevokedAt.Time
	}
	return rt
}

func (repo *MySQLRefreshTokenRepository) Create(ctx context.Context, rt RefreshToken) (RefreshToken, error) {
	const query = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)`

	rt.ID = uuid.NewString()

	if _, err := repo.db.ExecContext(ctx, query, rt.ID, rt.UserID, rt.TokenHash, rt.ExpiresAt, rt.CreatedAt); err != nil {
		return RefreshToken{}, err
	}

	return rt, nil
}

func (repo *MySQLRefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error) {
	const query = `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = ?`

	var row refreshTokenRow
	if err := repo.db.GetContext(ctx, &row, query, tokenHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RefreshToken{}, errRefreshTokenNotFound
		}
		return RefreshToken{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	const query = `UPDATE refresh_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`
	_, err := repo.db.ExecContext(ctx, query, time.Now().UTC(), id)
	return err
}

func (repo *MySQLRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	const query = `UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`
	_, err := repo.db.ExecContext(ctx, query, time.Now().UTC(), userID)
	return err
}
