// Package booking implements the booking domain: the entity, its persistence
// interface, business rules and the MySQL repository. It maps to the
// bookings table.
package booking

import "time"

// Status values a Booking may hold.
const (
	StatusPending   = "PENDING"
	StatusConfirmed = "CONFIRMED"
	StatusCancelled = "CANCELLED"
	StatusCompleted = "COMPLETED"
)

// PaymentStatus values a Booking may hold.
const (
	PaymentStatusPending  = "PENDING"
	PaymentStatusPaid     = "PAID"
	PaymentStatusRefunded = "REFUNDED"
)

// Booking is the domain entity. TripDateID, CheckInDate, CheckOutDate,
// PaymentID and SpecialRequests are each individually optional: a zero value
// (empty string or the zero time.Time) means "not set". CheckInDate and
// CheckOutDate only apply to flexible-date bookings that don't reference a
// TripDate (kept for backward compatibility with the originating schema).
type Booking struct {
	ID              string
	CustomerID      string
	ListingID       string
	TripDateID      string
	CheckInDate     time.Time
	CheckOutDate    time.Time
	NumberOfGuests  int
	TotalPrice      float64
	Currency        string
	Status          string
	PaymentStatus   string
	PaymentID       string
	SpecialRequests string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateInput is the data required to create a new booking.
type CreateInput struct {
	CustomerID      string    `json:"customer_id" validate:"required,uuid"`
	ListingID       string    `json:"listing_id" validate:"required,uuid"`
	TripDateID      string    `json:"trip_date_id" validate:"omitempty,uuid"`
	CheckInDate     time.Time `json:"check_in_date" validate:"omitempty"`
	CheckOutDate    time.Time `json:"check_out_date" validate:"omitempty"`
	NumberOfGuests  int       `json:"number_of_guests" validate:"omitempty,gt=0"`
	TotalPrice      float64   `json:"total_price" validate:"required,gt=0"`
	Currency        string    `json:"currency" validate:"omitempty,len=3"`
	PaymentID       string    `json:"payment_id" validate:"omitempty,max=255"`
	SpecialRequests string    `json:"special_requests" validate:"omitempty"`
}
