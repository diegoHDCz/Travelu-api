package user

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/diegoczajka/travelu-api/internal/platform/database/dbtest"
)

var userColumns = []string{
	"id", "name", "email", "phone", "password_hash",
	"first_name", "last_name", "role", "is_active", "created_at", "updated_at",
}

func TestMySQLRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	// Empty optional fields must be sent as NULL so the unique indexes allow
	// more than one user without them.
	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(sqlmock.AnyArg(), "Diego", "diego@example.com", nil, "hash",
			nil, nil, "customer", true, now, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), User{
		Name: "Diego", Email: "diego@example.com", PasswordHash: "hash",
		Role: "customer", IsActive: true, CreatedAt: now, UpdatedAt: now,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "diego@example.com", created.Email)
}

func TestMySQLRepository_Create_DuplicateKeys(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    error
	}{
		{"email", "Duplicate entry 'x' for key 'users.uq_users_email'", ErrEmailTaken},
		{"phone", "Duplicate entry 'x' for key 'users.uq_users_phone'", ErrPhoneTaken},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)

			mock.ExpectExec(`INSERT INTO users`).
				WillReturnError(&mysql.MySQLError{Number: mysqlErrDuplicateEntry, Message: tc.message})

			_, err := repo.Create(context.Background(), User{Name: "A", Email: "a@example.com"})
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_Create_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectExec(`INSERT INTO users`).WillReturnError(boom)

	_, err := repo.Create(context.Background(), User{Name: "A"})
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_Getters(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name  string
		where string
		value string
		call  func(*MySQLRepository, string) (User, error)
	}{
		{"by id", `WHERE id = \?`, "u-1", func(r *MySQLRepository, v string) (User, error) { return r.GetByID(context.Background(), v) }},
		{"by email", `WHERE email = \?`, "diego@example.com", func(r *MySQLRepository, v string) (User, error) { return r.GetByEmail(context.Background(), v) }},
		{"by phone", `WHERE phone = \?`, "+5511999998888", func(r *MySQLRepository, v string) (User, error) { return r.GetByPhone(context.Background(), v) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)

			mock.ExpectQuery(`FROM users\s+` + tc.where).
				WithArgs(tc.value).
				WillReturnRows(sqlmock.NewRows(userColumns).AddRow(
					"u-1", "Diego", "diego@example.com", "+5511999998888", "hash",
					"Diego", nil, "customer", true, now, now))

			got, err := tc.call(repo, tc.value)
			require.NoError(t, err)
			assert.Equal(t, User{
				ID: "u-1", Name: "Diego", Email: "diego@example.com", Phone: "+5511999998888",
				PasswordHash: "hash", FirstName: "Diego", Role: "customer",
				IsActive: true, CreatedAt: now, UpdatedAt: now,
			}, got)
		})
	}
}

func TestMySQLRepository_Getters_Errors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name  string
		dbErr error
		want  error
		call  func(*MySQLRepository) (User, error)
	}{
		{"by id not found", sql.ErrNoRows, ErrNotFound, func(r *MySQLRepository) (User, error) { return r.GetByID(context.Background(), "x") }},
		{"by id db error", boom, boom, func(r *MySQLRepository) (User, error) { return r.GetByID(context.Background(), "x") }},
		{"by email not found", sql.ErrNoRows, ErrNotFound, func(r *MySQLRepository) (User, error) { return r.GetByEmail(context.Background(), "x") }},
		{"by email db error", boom, boom, func(r *MySQLRepository) (User, error) { return r.GetByEmail(context.Background(), "x") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)

			mock.ExpectQuery(`FROM users`).WillReturnError(tc.dbErr)

			_, err := tc.call(repo)
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO users`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	_, err = repo.WithTx(tx).Create(context.Background(), User{Name: "A"})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}
