package category

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

var categoryColumns = []string{"id", "name", "slug", "description", "icon", "is_active", "created_at", "updated_at"}

func TestMySQLRepository_Create(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)

	mock.ExpectExec(`INSERT INTO categories`).
		WithArgs(sqlmock.AnyArg(), "Hotels", "hotels", nil, "bed", true, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	created, err := repo.Create(context.Background(), Category{Name: "Hotels", Slug: "hotels", Icon: "bed"})
	require.NoError(t, err)
	assert.NotEmpty(t, created.ID)
	assert.True(t, created.IsActive)
	assert.False(t, created.CreatedAt.IsZero())
}

func TestMySQLRepository_Create_Errors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name  string
		dbErr error
		want  error
	}{
		{"duplicate slug", &mysql.MySQLError{Number: mysqlErrDuplicateEntry, Message: "Duplicate entry 'x' for key 'categories.uq_categories_slug'"}, ErrSlugTaken},
		{"duplicate name", &mysql.MySQLError{Number: mysqlErrDuplicateEntry, Message: "Duplicate entry 'x' for key 'categories.uq_categories_name'"}, ErrNameTaken},
		{"db error", boom, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)

			mock.ExpectExec(`INSERT INTO categories`).WillReturnError(tc.dbErr)

			_, err := repo.Create(context.Background(), Category{Name: "Hotels", Slug: "hotels"})
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_Getters(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name  string
		where string
		value string
		call  func(*MySQLRepository, string) (Category, error)
	}{
		{"by id", `WHERE id = \?`, "c-1", func(r *MySQLRepository, v string) (Category, error) { return r.GetByID(context.Background(), v) }},
		{"by slug", `WHERE slug = \?`, "hotels", func(r *MySQLRepository, v string) (Category, error) { return r.GetBySlug(context.Background(), v) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)

			mock.ExpectQuery(`FROM categories\s+` + tc.where).
				WithArgs(tc.value).
				WillReturnRows(sqlmock.NewRows(categoryColumns).
					AddRow("c-1", "Hotels", "hotels", "Places to stay", nil, true, now, now))

			got, err := tc.call(repo, tc.value)
			require.NoError(t, err)
			assert.Equal(t, Category{
				ID: "c-1", Name: "Hotels", Slug: "hotels", Description: "Places to stay",
				IsActive: true, CreatedAt: now, UpdatedAt: now,
			}, got)
		})
	}
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

			mock.ExpectQuery(`FROM categories`).WillReturnError(tc.dbErr)

			_, err := repo.GetByID(context.Background(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_List(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	mock.ExpectQuery(`FROM categories\s+ORDER BY name`).
		WillReturnRows(sqlmock.NewRows(categoryColumns).
			AddRow("c-1", "Hotels", "hotels", nil, nil, true, now, now).
			AddRow("c-2", "Tours", "tours", nil, "map", false, now, now))

	got, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "hotels", got[0].Slug)
	assert.Equal(t, "map", got[1].Icon)
	assert.False(t, got[1].IsActive)
}

func TestMySQLRepository_List_DBError(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)
	boom := errors.New("boom")

	mock.ExpectQuery(`FROM categories`).WillReturnError(boom)

	_, err := repo.List(context.Background())
	require.ErrorIs(t, err, boom)
}

func TestMySQLRepository_Update(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)

	mock.ExpectExec(`UPDATE categories\s+SET name = \?, slug = \?, description = \?, icon = \?, is_active = \?, updated_at = \?\s+WHERE id = \?`).
		WithArgs("Hotels", "hotels", "desc", nil, false, sqlmock.AnyArg(), "c-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	got, err := repo.Update(context.Background(), Category{ID: "c-1", Name: "Hotels", Slug: "hotels", Description: "desc"})
	require.NoError(t, err)
	assert.Equal(t, "c-1", got.ID)
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestMySQLRepository_Update_Errors(t *testing.T) {
	boom := errors.New("boom")

	tests := []struct {
		name   string
		expect func(sqlmock.Sqlmock)
		want   error
	}{
		{"not found", func(m sqlmock.Sqlmock) {
			m.ExpectExec(`UPDATE categories`).WillReturnResult(sqlmock.NewResult(0, 0))
		}, ErrNotFound},
		{"duplicate slug", func(m sqlmock.Sqlmock) {
			m.ExpectExec(`UPDATE categories`).WillReturnError(&mysql.MySQLError{Number: mysqlErrDuplicateEntry, Message: "uq_categories_slug"})
		}, ErrSlugTaken},
		{"db error", func(m sqlmock.Sqlmock) {
			m.ExpectExec(`UPDATE categories`).WillReturnError(boom)
		}, boom},
		{"rows affected error", func(m sqlmock.Sqlmock) {
			m.ExpectExec(`UPDATE categories`).WillReturnResult(sqlmock.NewErrorResult(boom))
		}, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := dbtest.New(t)
			repo := NewMySQLRepository(db)
			tc.expect(mock)

			_, err := repo.Update(context.Background(), Category{ID: "c-1", Name: "Hotels", Slug: "hotels"})
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestMySQLRepository_WithTx(t *testing.T) {
	db, mock := dbtest.New(t)
	repo := NewMySQLRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO categories`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	tx, err := db.Beginx()
	require.NoError(t, err)
	_, err = repo.WithTx(tx).Create(context.Background(), Category{Name: "Hotels", Slug: "hotels"})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
}
