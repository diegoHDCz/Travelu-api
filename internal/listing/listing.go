// Package listing implements the travel listing domain: the entity, its
// persistence interface, business rules and the MySQL repository. It maps to
// the travel_listings table (named "travel" in the originating Kotlin spec;
// renamed here for consistency with every other table matching its domain).
package listing

import "time"

// Category values a Listing may hold.
const (
	CategoryHotel    = "HOTEL"
	CategoryFlight   = "FLIGHT"
	CategoryActivity = "ACTIVITY"
	CategoryPackage  = "PACKAGE"
)

// Listing is the domain entity. City, Country, Capacity, AvailableFrom,
// AvailableTo, Images and Amenities are each individually optional: a zero
// value (empty string, nil, or the zero time.Time) means "not set".
type Listing struct {
	ID            string
	VendorID      string
	Title         string
	Description   string
	Category      string
	Location      string
	City          string
	Country       string
	Price         float64
	Currency      string
	Capacity      *int
	AvailableFrom time.Time
	AvailableTo   time.Time
	Images        string
	Amenities     string
	Rating        float64
	ReviewCount   int
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// CreateInput is the data required to create a new listing.
type CreateInput struct {
	VendorID      string    `json:"vendor_id" validate:"required,uuid"`
	Title         string    `json:"title" validate:"required,min=2,max=255"`
	Description   string    `json:"description" validate:"required"`
	Category      string    `json:"category" validate:"required,oneof=HOTEL FLIGHT ACTIVITY PACKAGE"`
	Location      string    `json:"location" validate:"required,max=255"`
	City          string    `json:"city" validate:"omitempty,max=100"`
	Country       string    `json:"country" validate:"omitempty,max=100"`
	Price         float64   `json:"price" validate:"required,gt=0"`
	Currency      string    `json:"currency" validate:"omitempty,len=3"`
	Capacity      *int      `json:"capacity" validate:"omitempty,gt=0"`
	AvailableFrom time.Time `json:"available_from" validate:"omitempty"`
	AvailableTo   time.Time `json:"available_to" validate:"omitempty"`
	Images        string    `json:"images" validate:"omitempty"`
	Amenities     string    `json:"amenities" validate:"omitempty"`
}

// UpdateInput is the data required to update an existing listing. VendorID
// is intentionally absent: ownership of a listing cannot be transferred
// through this endpoint.
type UpdateInput struct {
	ID            string    `json:"id" validate:"required,uuid"`
	Title         string    `json:"title" validate:"required,min=2,max=255"`
	Description   string    `json:"description" validate:"required"`
	Category      string    `json:"category" validate:"required,oneof=HOTEL FLIGHT ACTIVITY PACKAGE"`
	Location      string    `json:"location" validate:"required,max=255"`
	City          string    `json:"city" validate:"omitempty,max=100"`
	Country       string    `json:"country" validate:"omitempty,max=100"`
	Price         float64   `json:"price" validate:"required,gt=0"`
	Currency      string    `json:"currency" validate:"omitempty,len=3"`
	Capacity      *int      `json:"capacity" validate:"omitempty,gt=0"`
	AvailableFrom time.Time `json:"available_from" validate:"omitempty"`
	AvailableTo   time.Time `json:"available_to" validate:"omitempty"`
	Images        string    `json:"images" validate:"omitempty"`
	Amenities     string    `json:"amenities" validate:"omitempty"`
	IsActive      bool      `json:"is_active"`
}
