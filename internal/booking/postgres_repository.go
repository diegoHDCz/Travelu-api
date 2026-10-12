package booking

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

// bookingRow mirrors the bookings table layout; mapping to/from the domain
// Booking happens entirely in this file.
type bookingRow struct {
	ID              string         `db:"id"`
	CustomerID      string         `db:"customer_id"`
	ListingID       string         `db:"listing_id"`
	TripDateID      sql.NullString `db:"trip_date_id"`
	CheckInDate     sql.NullTime   `db:"check_in_date"`
	CheckOutDate    sql.NullTime   `db:"check_out_date"`
	NumberOfGuests  int            `db:"number_of_guests"`
	TotalPrice      float64        `db:"total_price"`
	Currency        string         `db:"currency"`
	Status          string         `db:"status"`
	PaymentStatus   string         `db:"payment_status"`
	PaymentID       sql.NullString `db:"payment_id"`
	SpecialRequests sql.NullString `db:"special_requests"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

func (r bookingRow) toDomain() Booking {
	b := Booking{
		ID:              r.ID,
		CustomerID:      r.CustomerID,
		ListingID:       r.ListingID,
		TripDateID:      r.TripDateID.String,
		NumberOfGuests:  r.NumberOfGuests,
		TotalPrice:      r.TotalPrice,
		Currency:        r.Currency,
		Status:          r.Status,
		PaymentStatus:   r.PaymentStatus,
		PaymentID:       r.PaymentID.String,
		SpecialRequests: r.SpecialRequests.String,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
	if r.CheckInDate.Valid {
		b.CheckInDate = r.CheckInDate.Time
	}
	if r.CheckOutDate.Valid {
		b.CheckOutDate = r.CheckOutDate.Time
	}
	return b
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

func (repo *MySQLRepository) Create(ctx context.Context, b Booking) (Booking, error) {
	const query = `
		INSERT INTO bookings (
			id, customer_id, listing_id, trip_date_id, check_in_date, check_out_date,
			number_of_guests, total_price, currency, status, payment_status, payment_id,
			special_requests, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	b.ID = uuid.NewString()

	_, err := repo.db.ExecContext(ctx, query,
		b.ID, b.CustomerID, b.ListingID, nullable(b.TripDateID), nullableTime(b.CheckInDate), nullableTime(b.CheckOutDate),
		b.NumberOfGuests, b.TotalPrice, b.Currency, b.Status, b.PaymentStatus, nullable(b.PaymentID),
		nullable(b.SpecialRequests), b.CreatedAt, b.UpdatedAt)
	if err != nil {
		return Booking{}, err
	}

	return b, nil
}

func (repo *MySQLRepository) GetByID(ctx context.Context, id string) (Booking, error) {
	const query = `
		SELECT id, customer_id, listing_id, trip_date_id, check_in_date, check_out_date,
			number_of_guests, total_price, currency, status, payment_status, payment_id,
			special_requests, created_at, updated_at
		FROM bookings
		WHERE id = ?`

	var row bookingRow
	if err := repo.db.GetContext(ctx, &row, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Booking{}, ErrNotFound
		}
		return Booking{}, err
	}

	return row.toDomain(), nil
}

func (repo *MySQLRepository) ListByCustomerID(ctx context.Context, customerID string) ([]Booking, error) {
	const query = `
		SELECT id, customer_id, listing_id, trip_date_id, check_in_date, check_out_date,
			number_of_guests, total_price, currency, status, payment_status, payment_id,
			special_requests, created_at, updated_at
		FROM bookings
		WHERE customer_id = ?
		ORDER BY created_at DESC`

	var rows []bookingRow
	if err := repo.db.SelectContext(ctx, &rows, query, customerID); err != nil {
		return nil, err
	}

	bookings := make([]Booking, len(rows))
	for i, row := range rows {
		bookings[i] = row.toDomain()
	}
	return bookings, nil
}
