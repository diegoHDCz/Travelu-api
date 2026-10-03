package user

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/platform/database"
)

// mysqlErrDuplicateEntry is the MySQL error number for a unique key
// violation (ER_DUP_ENTRY).
const mysqlErrDuplicateEntry = 1062

// MySQLRepository implements Repository on top of a sqlx MySQL pool or, when
// built via WithTx, on top of an in-flight transaction.
type MySQLRepository struct {
	db database.SQLExecutor
}

// NewMySQLRepository builds a MySQLRepository backed by db.
func NewMySQLRepository(db *sqlx.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

// WithTx returns a MySQLRepository whose operations run against tx instead
// of the pool, so they participate in a transaction started by
// database.WithTx.
func (repo *MySQLRepository) WithTx(tx *sqlx.Tx) *MySQLRepository {
	return &MySQLRepository{db: tx}
}

// userRow mirrors the users table layout; mapping to/from the domain User
// happens entirely in this file. Email, Phone and Username are nullable in
// the database since each is individually optional.
type userRow struct {
	ID           string         `db:"id"`
	Name         string         `db:"name"`
	Email        sql.NullString `db:"email"`
	Phone        sql.NullString `db:"phone"`
	Username     sql.NullString `db:"username"`
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
		Username:     r.Username.String,
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
// stored as NULL rather than "" — required for the unique indexes on email,
// phone and username to allow more than one user without that field.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (repo *MySQLRepository) Create(ctx context.Context, u User) (User, error) {
	const query = `
		INSERT INTO users (id, name, email, phone, username, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	u.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		u.ID, u.Name, nullable(u.Email), nullable(u.Phone), nullable(u.Username), u.PasswordHash,
		nullable(u.FirstName), nullable(u.LastName), u.Role, u.IsActive, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
			return User{}, duplicateKeyError(mysqlErr)
		}
		return User{}, err
	}

	return u, nil
}

// duplicateKeyError maps a ER_DUP_ENTRY error to the domain error for the
// unique key it violated. MySQL 8 reports the key as "table.index_name" in
// the error message (e.g. "Duplicate entry 'x' for key 'users.uq_users_email'").
func duplicateKeyError(err *mysql.MySQLError) error {
	switch {
	case strings.Contains(err.Message, "uq_users_phone"):
		return ErrPhoneTaken
	case strings.Contains(err.Message, "uq_users_username"):
		return ErrUsernameTaken
	default:
		return ErrEmailTaken
	}
}

func (repo *MySQLRepository) GetByEmail(ctx context.Context, email string) (User, error) {
	const query = `
		SELECT id, name, email, phone, username, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE email = ?`
	return repo.getByField(ctx, query, email)
}

func (repo *MySQLRepository) GetByPhone(ctx context.Context, phone string) (User, error) {
	const query = `
		SELECT id, name, email, phone, username, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE phone = ?`
	return repo.getByField(ctx, query, phone)
}

func (repo *MySQLRepository) GetByUsername(ctx context.Context, username string) (User, error) {
	const query = `
		SELECT id, name, email, phone, username, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE username = ?`
	return repo.getByField(ctx, query, username)
}

func (repo *MySQLRepository) getByField(ctx context.Context, query, value string) (User, error) {
	var row userRow
	if err := repo.db.GetContext(ctx, &row, query, value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRepository) GetByID(ctx context.Context, id string) (User, error) {
	const query = `
		SELECT id, name, email, phone, username, password_hash, first_name, last_name, role, is_active, created_at, updated_at
		FROM users
		WHERE id = ?`

	var row userRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNotFound
		}
		return User{}, err
	}

	return row.toDomain(), nil
}
