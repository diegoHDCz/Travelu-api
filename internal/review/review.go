// Package review implements the review domain: the entity, its persistence
// interface, business rules and the MySQL repository. It maps to the
// reviews table.
package review

import "time"

// Review is the domain entity. BookingID, Title and Comment are each
// individually optional (empty string means "not set").
type Review struct {
	ID         string
	CustomerID string
	ListingID  string
	BookingID  string
	Rating     int
	Title      string
	Comment    string
	IsVerified bool
	IsApproved bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateInput is the data required to create a new review.
type CreateInput struct {
	CustomerID string `json:"customer_id" validate:"required,uuid"`
	ListingID  string `json:"listing_id" validate:"required,uuid"`
	BookingID  string `json:"booking_id" validate:"omitempty,uuid"`
	Rating     int    `json:"rating" validate:"required,min=1,max=5"`
	Title      string `json:"title" validate:"omitempty,max=255"`
	Comment    string `json:"comment" validate:"omitempty"`
}
