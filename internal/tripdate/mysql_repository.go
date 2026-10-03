package tripdate

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

// tripDateRow mirrors the trip_dates table layout; mapping to/from the
// domain TripDate happens entirely in this file.
type tripDateRow struct {
	ID              string        `db:"id"`
	ListingID       string        `db:"listing_id"`
	StartDate       time.Time     `db:"start_date"`
	EndDate         time.Time     `db:"end_date"`
	MaxCapacity     sql.NullInt64 `db:"max_capacity"`
	CurrentBookings int           `db:"current_bookings"`
	IsActive        bool          `db:"is_active"`
	CreatedAt       time.Time     `db:"created_at"`
	UpdatedAt       time.Time     `db:"updated_at"`
}

func (r tripDateRow) toDomain() TripDate {
	td := TripDate{
		ID:              r.ID,
		ListingID:       r.ListingID,
		StartDate:       r.StartDate,
		EndDate:         r.EndDate,
		CurrentBookings: r.CurrentBookings,
		IsActive:        r.IsActive,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
	if r.MaxCapacity.Valid {
		maxCapacity := int(r.MaxCapacity.Int64)
		td.MaxCapacity = &maxCapacity
	}
	return td
}

// nullableCapacity turns a nil *int into a SQL NULL.
func nullableCapacity(capacity *int) any {
	if capacity == nil {
		return nil
	}
	return *capacity
}

func (repo *MySQLRepository) Create(ctx context.Context, td TripDate) (TripDate, error) {
	const query = `
		INSERT INTO trip_dates (id, listing_id, start_date, end_date, max_capacity, current_bookings, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	td.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		td.ID, td.ListingID, td.StartDate, td.EndDate, nullableCapacity(td.MaxCapacity), td.CurrentBookings, td.IsActive, td.CreatedAt, td.UpdatedAt)
	if err != nil {
		return TripDate{}, err
	}

	return td, nil
}

func (repo *MySQLRepository) GetByID(ctx context.Context, id string) (TripDate, error) {
	const query = `
		SELECT id, listing_id, start_date, end_date, max_capacity, current_bookings, is_active, created_at, updated_at
		FROM trip_dates
		WHERE id = ?`

	var row tripDateRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TripDate{}, ErrNotFound
		}
		return TripDate{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRepository) ListByListingID(ctx context.Context, listingID string) ([]TripDate, error) {
	const query = `
		SELECT id, listing_id, start_date, end_date, max_capacity, current_bookings, is_active, created_at, updated_at
		FROM trip_dates
		WHERE listing_id = ?
		ORDER BY start_date`

	var rows []tripDateRow
	if err := repo.db.SelectContext(ctx, &rows, query, listingID); err != nil {
		return nil, err
	}

	tripDates := make([]TripDate, len(rows))
	for i, row := range rows {
		tripDates[i] = row.toDomain()
	}
	return tripDates, nil
}
