package listing

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/platform/database"
)

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

// listingRow mirrors the travel_listings table layout; mapping to/from the
// domain Listing happens entirely in this file.
type listingRow struct {
	ID            string         `db:"id"`
	VendorID      string         `db:"vendor_id"`
	Title         string         `db:"title"`
	Description   string         `db:"description"`
	Category      string         `db:"category"`
	Location      string         `db:"location"`
	City          sql.NullString `db:"city"`
	Country       sql.NullString `db:"country"`
	Price         float64        `db:"price"`
	Currency      string         `db:"currency"`
	Capacity      sql.NullInt64  `db:"capacity"`
	AvailableFrom sql.NullTime   `db:"available_from"`
	AvailableTo   sql.NullTime   `db:"available_to"`
	Images        sql.NullString `db:"images"`
	Amenities     sql.NullString `db:"amenities"`
	Rating        float64        `db:"rating"`
	ReviewCount   int            `db:"review_count"`
	IsActive      bool           `db:"is_active"`
	CreatedAt     time.Time      `db:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at"`
}

func (r listingRow) toDomain() Listing {
	l := Listing{
		ID:          r.ID,
		VendorID:    r.VendorID,
		Title:       r.Title,
		Description: r.Description,
		Category:    r.Category,
		Location:    r.Location,
		City:        r.City.String,
		Country:     r.Country.String,
		Price:       r.Price,
		Currency:    r.Currency,
		Images:      r.Images.String,
		Amenities:   r.Amenities.String,
		Rating:      r.Rating,
		ReviewCount: r.ReviewCount,
		IsActive:    r.IsActive,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.Capacity.Valid {
		capacity := int(r.Capacity.Int64)
		l.Capacity = &capacity
	}
	if r.AvailableFrom.Valid {
		l.AvailableFrom = r.AvailableFrom.Time
	}
	if r.AvailableTo.Valid {
		l.AvailableTo = r.AvailableTo.Time
	}
	return l
}

// nullable turns an empty string into a SQL NULL.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableTime turns a zero time.Time into a SQL NULL.
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// nullableCapacity turns a nil *int into a SQL NULL.
func nullableCapacity(capacity *int) any {
	if capacity == nil {
		return nil
	}
	return *capacity
}

func (repo *PostgresRepository) Create(ctx context.Context, l Listing) (Listing, error) {
	const query = `
		INSERT INTO travel_listings (
			id, vendor_id, title, description, category, location, city, country,
			price, currency, capacity, available_from, available_to, images, amenities,
			rating, review_count, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`

	l.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		l.ID, l.VendorID, l.Title, l.Description, l.Category, l.Location, nullable(l.City), nullable(l.Country),
		l.Price, l.Currency, nullableCapacity(l.Capacity), nullableTime(l.AvailableFrom), nullableTime(l.AvailableTo),
		nullable(l.Images), nullable(l.Amenities), l.Rating, l.ReviewCount, l.IsActive, l.CreatedAt, l.UpdatedAt)
	if err != nil {
		return Listing{}, err
	}

	return l, nil
}

func (repo *PostgresRepository) GetByID(ctx context.Context, id string) (Listing, error) {
	const query = `
		SELECT id, vendor_id, title, description, category, location, city, country,
			price, currency, capacity, available_from, available_to, images, amenities,
			rating, review_count, is_active, created_at, updated_at
		FROM travel_listings
		WHERE id = $1`

	var row listingRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Listing{}, ErrNotFound
		}
		return Listing{}, err
	}

	return row.toDomain(), nil
}

func (repo *PostgresRepository) ListActive(ctx context.Context) ([]Listing, error) {
	const query = `
		SELECT id, vendor_id, title, description, category, location, city, country,
			price, currency, capacity, available_from, available_to, images, amenities,
			rating, review_count, is_active, created_at, updated_at
		FROM travel_listings
		WHERE is_active = TRUE
		ORDER BY created_at DESC`

	var rows []listingRow
	if err := repo.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}

	listings := make([]Listing, len(rows))
	for i, row := range rows {
		listings[i] = row.toDomain()
	}
	return listings, nil
}

func (repo *PostgresRepository) Update(ctx context.Context, l Listing) (Listing, error) {
	const query = `
		UPDATE travel_listings
		SET title = $1, description = $2, category = $3, location = $4, city = $5, country = $6,
			price = $7, currency = $8, capacity = $9, available_from = $10, available_to = $11,
			images = $12, amenities = $13, is_active = $14, updated_at = $15
		WHERE id = $16`

	l.UpdatedAt = time.Now().UTC()

	result, err := repo.db.ExecContext(ctx, query,
		l.Title, l.Description, l.Category, l.Location, nullable(l.City), nullable(l.Country),
		l.Price, l.Currency, nullableCapacity(l.Capacity), nullableTime(l.AvailableFrom), nullableTime(l.AvailableTo),
		nullable(l.Images), nullable(l.Amenities), l.IsActive, l.UpdatedAt, l.ID)
	if err != nil {
		return Listing{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return Listing{}, err
	}
	if rowsAffected == 0 {
		return Listing{}, ErrNotFound
	}

	return l, nil
}
