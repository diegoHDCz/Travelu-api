package review

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/platform/database"
)

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

// reviewRow mirrors the reviews table layout; mapping to/from the domain
// Review happens entirely in this file.
type reviewRow struct {
	ID         string         `db:"id"`
	CustomerID string         `db:"customer_id"`
	ListingID  string         `db:"listing_id"`
	BookingID  sql.NullString `db:"booking_id"`
	Rating     int            `db:"rating"`
	Title      sql.NullString `db:"title"`
	Comment    sql.NullString `db:"comment"`
	IsVerified bool           `db:"is_verified"`
	IsApproved bool           `db:"is_approved"`
	CreatedAt  time.Time      `db:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at"`
}

func (r reviewRow) toDomain() Review {
	return Review{
		ID:         r.ID,
		CustomerID: r.CustomerID,
		ListingID:  r.ListingID,
		BookingID:  r.BookingID.String,
		Rating:     r.Rating,
		Title:      r.Title.String,
		Comment:    r.Comment.String,
		IsVerified: r.IsVerified,
		IsApproved: r.IsApproved,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
	}
}

// nullable turns an empty string into a SQL NULL.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (repo *MySQLRepository) Create(ctx context.Context, r Review) (Review, error) {
	const query = `
		INSERT INTO reviews (id, customer_id, listing_id, booking_id, rating, title, comment, is_verified, is_approved, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	r.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		r.ID, r.CustomerID, r.ListingID, nullable(r.BookingID), r.Rating, nullable(r.Title), nullable(r.Comment),
		r.IsVerified, r.IsApproved, r.CreatedAt, r.UpdatedAt)
	if err != nil {
		return Review{}, err
	}

	return r, nil
}

func (repo *MySQLRepository) GetByID(ctx context.Context, id string) (Review, error) {
	const query = `
		SELECT id, customer_id, listing_id, booking_id, rating, title, comment, is_verified, is_approved, created_at, updated_at
		FROM reviews
		WHERE id = ?`

	var row reviewRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Review{}, ErrNotFound
		}
		return Review{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRepository) ListByListingID(ctx context.Context, listingID string) ([]Review, error) {
	const query = `
		SELECT id, customer_id, listing_id, booking_id, rating, title, comment, is_verified, is_approved, created_at, updated_at
		FROM reviews
		WHERE listing_id = ? AND is_approved = TRUE
		ORDER BY created_at DESC`

	var rows []reviewRow
	if err := repo.db.SelectContext(ctx, &rows, query, listingID); err != nil {
		return nil, err
	}

	reviews := make([]Review, len(rows))
	for i, row := range rows {
		reviews[i] = row.toDomain()
	}
	return reviews, nil
}

func (repo *MySQLRepository) Update(ctx context.Context, r Review) (Review, error) {
	const query = `
		UPDATE reviews
		SET rating = ?, title = ?, comment = ?, updated_at = ?
		WHERE id = ?`

	r.UpdatedAt = time.Now().UTC()

	result, err := repo.db.ExecContext(ctx, query, r.Rating, nullable(r.Title), nullable(r.Comment), r.UpdatedAt, r.ID)
	if err != nil {
		return Review{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Review{}, err
	}
	if rowsAffected == 0 {
		return Review{}, ErrNotFound
	}

	return r, nil
}
