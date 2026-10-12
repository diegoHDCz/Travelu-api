package category

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

func (repo *PostgresRepository) Create(ctx context.Context, c Category) (Category, error) {
	const query = `
		INSERT INTO categories (id, name, slug, description, icon, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	c.ID = uuid.NewString()
	c.CreatedAt = time.Now().UTC()
	c.UpdatedAt = time.Now().UTC()
	c.IsActive = true

	_, err := repo.db.ExecContext(ctx, query,
		c.ID, c.Name, c.Slug, nullable(c.Description), nullable(c.Icon), c.IsActive, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return Category{}, duplicateKeyError(pgErr)
		}
		return Category{}, err
	}

	return c, nil
}

// duplicateKeyError maps a unique_violation error to the domain error for
// the unique constraint it violated.
func duplicateKeyError(err *pgconn.PgError) error {
	if err.ConstraintName == "uq_categories_slug" {
		return ErrSlugTaken
	}
	return ErrNameTaken
}

func (repo *PostgresRepository) GetByID(ctx context.Context, id string) (Category, error) {
	const query = `
		SELECT id, name, slug, description, icon, is_active, created_at, updated_at
		FROM categories
		WHERE id = $1`
	return repo.getByField(ctx, query, id)
}

func (repo *PostgresRepository) GetBySlug(ctx context.Context, slug string) (Category, error) {
	const query = `
		SELECT id, name, slug, description, icon, is_active, created_at, updated_at
		FROM categories
		WHERE slug = $1`
	return repo.getByField(ctx, query, slug)
}

func (repo *PostgresRepository) getByField(ctx context.Context, query, value string) (Category, error) {
	var row categoryRow
	if err := repo.db.GetContext(ctx, &row, query, value); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, ErrNotFound
		}
		return Category{}, err
	}

	return row.toDomain(), nil
}

func (repo *PostgresRepository) List(ctx context.Context) ([]Category, error) {
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

func (repo *PostgresRepository) Update(ctx context.Context, c Category) (Category, error) {
	const query = `
		UPDATE categories
		SET name = $1, slug = $2, description = $3, icon = $4, is_active = $5, updated_at = $6
		WHERE id = $7`

	c.UpdatedAt = time.Now().UTC()

	result, err := repo.db.ExecContext(ctx, query,
		c.Name, c.Slug, nullable(c.Description), nullable(c.Icon), c.IsActive, c.UpdatedAt, c.ID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return Category{}, duplicateKeyError(pgErr)
		}
		return Category{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Category{}, err
	}
	if rowsAffected == 0 {
		return Category{}, ErrNotFound
	}

	return c, nil
}
