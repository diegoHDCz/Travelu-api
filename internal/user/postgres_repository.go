package user

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

// userRow mirrors the users table layout; mapping to/from the domain User
// happens entirely in this file. Email and Phone are nullable in the
// database since each is individually optional.
type userRow struct {
	ID           string         `db:"id"`
	Name         string         `db:"name"`
	Email        sql.NullString `db:"email"`
	Phone        sql.NullString `db:"phone"`
	PasswordHash string         `db:"password_hash"`
	FirstName    sql.NullString `db:"first_name"`
	LastName     sql.NullString `db:"last_name"`
	Role         string         `db:"role"`
	IsActive     bool           `db:"is_active"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

func (r userRow) toDomain() User {
	return User{
		ID:           r.ID,
		Name:         r.Name,
		Email:        r.Email.String,
		Phone:        r.Phone.String,
		PasswordHash: r.PasswordHash,
		FirstName:    r.FirstName.String,
		LastName:     r.LastName.String,
		Role:         r.Role,
		IsActive:     r.IsActive,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// nullable turns an empty string into a SQL NULL, so optional fields are
// stored as NULL rather than "" — required for the unique indexes on email
// and phone to allow more than one user without that field.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (repo *PostgresRepository) Create(ctx context.Context, u User) (User, error) {
	const query = `
		INSERT INTO users (id, name, email, phone, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	u.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		u.ID, u.Name, nullable(u.Email), nullable(u.Phone), u.PasswordHash,
		nullable(u.FirstName), nullable(u.LastName), u.Role, u.IsActive, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return User{}, duplicateKeyError(pgErr)
		}
		return User{}, err
	}

	return u, nil
}

// duplicateKeyError maps a unique_violation error to the domain error for
// the unique constraint it violated, using the constraint name Postgres
// reports (e.g. "uq_users_email").
func duplicateKeyError(err *pgconn.PgError) error {
	switch err.ConstraintName {
	case "uq_users_phone":
		return ErrPhoneTaken
	default:
		return ErrEmailTaken
	}
}

func (repo *PostgresRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id, name, email, phone, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE email = $1`
	return repo.getByField(ctx, query, email)
}

func (repo *PostgresRepository) GetByPhone(ctx context.Context, phone string) (User, error) {
	const query = `
		SELECT id, name, email, phone, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE phone = $1`
	return repo.getByField(ctx, query, phone)
}

func (repo *PostgresRepository) getByField(ctx context.Context, query, value string) (User, error) {
	var row userRow
	if err := repo.db.GetContext(ctx, &row, query, value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return row.toDomain(), nil
}

func (repo *PostgresRepository) GetByID(ctx context.Context, id string) (User, error) {
	const query = `
		SELECT id, name, email, phone, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE id = $1`

	var row userRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return row.toDomain(), nil
}
