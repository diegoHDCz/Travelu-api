package devicetoken

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/platform/database"
)

// pgUniqueViolation is the Postgres SQLSTATE for a unique constraint
// violation (unique_violation).
const pgUniqueViolation = "23505"

// PostgresRepository implements Repository on top of a sqlx Postgres pool or, when
// built via WithTx, on top of an in-flight transaction.
type PostgresRepository struct {
	db database.SQLExecutor
}

// NewPostgresRepository builds a PostgresRepository backed by db.
func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// WithTx returns a PostgresRepository whose operations run against tx instead
// of the pool, so they participate in a transaction started by
// database.WithTx.
func (repo *PostgresRepository) WithTx(tx *sqlx.Tx) *PostgresRepository {
	return &PostgresRepository{db: tx}
}

// deviceTokenRow mirrors the device_tokens table layout; mapping to/from the
// domain DeviceToken happens entirely in this file.
type deviceTokenRow struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	Token     string    `db:"token"`
	Platform  string    `db:"platform"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func (r deviceTokenRow) toDomain() DeviceToken {
	return DeviceToken{
		ID:        r.ID,
		UserID:    r.UserID,
		Token:     r.Token,
		Platform:  r.Platform,
		IsActive:  r.IsActive,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func (repo *PostgresRepository) Create(ctx context.Context, dt DeviceToken) (DeviceToken, error) {
	const query = `
		INSERT INTO device_tokens (id, user_id, token, platform, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	dt.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		dt.ID, dt.UserID, dt.Token, dt.Platform, dt.IsActive, dt.CreatedAt, dt.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return DeviceToken{}, ErrTokenTaken
		}
		return DeviceToken{}, err
	}

	return dt, nil
}

func (repo *PostgresRepository) GetByID(ctx context.Context, id string) (DeviceToken, error) {
	const query = `
		SELECT id, user_id, token, platform, is_active, created_at, updated_at
		FROM device_tokens
		WHERE id = $1`

	var row deviceTokenRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DeviceToken{}, ErrNotFound
		}
		return DeviceToken{}, err
	}

	return row.toDomain(), nil
}

func (repo *PostgresRepository) ListByUserID(ctx context.Context, userID string) ([]DeviceToken, error) {
	const query = `
		SELECT id, user_id, token, platform, is_active, created_at, updated_at
		FROM device_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC`

	var rows []deviceTokenRow
	if err := repo.db.SelectContext(ctx, &rows, query, userID); err != nil {
		return nil, err
	}

	tokens := make([]DeviceToken, len(rows))
	for i, row := range rows {
		tokens[i] = row.toDomain()
	}
	return tokens, nil
}

func (repo *PostgresRepository) Deactivate(ctx context.Context, id string) error {
	const query = `UPDATE device_tokens SET is_active = FALSE WHERE id = $1`

	result, err := repo.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}
