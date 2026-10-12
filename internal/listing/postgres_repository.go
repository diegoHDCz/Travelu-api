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

func (repo *MySQLRepository) Create(ctx context.Context, l Listing) (Listing, error) {
	const query = `
		INSERT INTO travel_listings (
			id, vendor_id, title, description, category, location, city, country,
			price, currency, capacity, available_from, available_to, images, amenities,
			rating, review_count, is_active, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

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

func (repo *MySQLRepository) GetByID(ctx context.Context, id string) (Listing, error) {
	const query = `
		SELECT id, vendor_id, title, description, category, location, city, country,
			price, currency, capacity, available_from, available_to, images, amenities,
			rating, review_count, is_active, created_at, updated_at
		FROM travel_listings
		WHERE id = ?`

	var row listingRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Listing{}, ErrNotFound
		}
		return Listing{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRepository) ListActive(ctx context.Context) ([]Listing, error) {
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

func (repo *MySQLRepository) Update(ctx context.Context, l Listing) (Listing, error) {
	const query = `
		UPDATE travel_listings
		SET title = ?, description = ?, category = ?, location = ?, city = ?, country = ?,
			price = ?, currency = ?, capacity = ?, available_from = ?, available_to = ?,
			images = ?, amenities = ?, is_active = ?, updated_at = ?
		WHERE id = ?`

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
