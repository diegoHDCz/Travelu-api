package category

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

// categoryRow mirrors the categories table layout; mapping to/from the
// domain Category happens entirely in this file.
type categoryRow struct {
	ID          string         `db:"id"`
	Name        string         `db:"name"`
	Slug        string         `db:"slug"`
	Description sql.NullString `db:"description"`
	Icon        sql.NullString `db:"icon"`
	IsActive    bool           `db:"is_active"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

func (r categoryRow) toDomain() Category {
	return Category{
		ID:          r.ID,
		Name:        r.Name,
		Slug:        r.Slug,
		Description: r.Description.String,
		Icon:        r.Icon.String,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// nullable turns an empty string into a SQL NULL.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (repo *MySQLRepository) Create(ctx context.Context, c Category) (Category, error) {
	const query = `
		INSERT INTO categories (id, name, slug, description, icon, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	c.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		c.ID, c.Name, c.Slug, nullable(c.Description), nullable(c.Icon), c.IsActive, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
			return Category{}, duplicateKeyError(mysqlErr)
		}
		return Category{}, err
	}

	return c, nil
}

// duplicateKeyError maps a ER_DUP_ENTRY error to the domain error for the
// unique key it violated.
func duplicateKeyError(err *mysql.MySQLError) error {
	if strings.Contains(err.Message, "uq_categories_slug") {
		return ErrSlugTaken
	}
	return ErrNameTaken
}

func (repo *MySQLRepository) GetByID(ctx context.Context, id string) (Category, error) {
	const query = `
		SELECT id, name, slug, description, icon, is_active, created_at, updated_at
		FROM categories
		WHERE id = ?`
	return repo.getByField(ctx, query, id)
}

func (repo *MySQLRepository) GetBySlug(ctx context.Context, slug string) (Category, error) {
	const query = `
		SELECT id, name, slug, description, icon, is_active, created_at, updated_at
		FROM categories
		WHERE slug = ?`
	return repo.getByField(ctx, query, slug)
}

func (repo *MySQLRepository) getByField(ctx context.Context, query, value string) (Category, error) {
	var row categoryRow
	if err := repo.db.GetContext(ctx, &row, query, value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, ErrNotFound
		}
		return Category{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRepository) List(ctx context.Context) ([]Category, error) {
	const query = `
		SELECT id, name, slug, description, icon, is_active, created_at, updated_at
		FROM categories
		ORDER BY name`

	var rows []categoryRow
	if err := repo.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	categories := make([]Category, len(rows))
	for i, row := range rows {
		categories[i] = row.toDomain()
	}
	return categories, nil
}
