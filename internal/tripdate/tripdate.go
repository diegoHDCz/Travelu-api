// Package tripdate implements the trip date domain: a bookable date range
// for a listing. It maps to the trip_dates table.
package tripdate

import "time"

// TripDate is the domain entity. MaxCapacity is optional: nil means it falls
// back to the listing's own capacity.
type TripDate struct {
	ID              string
	ListingID       string
	StartDate       time.Time
	EndDate         time.Time
	MaxCapacity     *int
	CurrentBookings int
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateInput is the data required to create a new trip date.
type CreateInput struct {
	ListingID   string    `json:"listing_id" validate:"required,uuid"`
	StartDate   time.Time `json:"start_date" validate:"required"`
	EndDate     time.Time `json:"end_date" validate:"required,gtfield=StartDate"`
	MaxCapacity *int      `json:"max_capacity" validate:"omitempty,gt=0"`
}
