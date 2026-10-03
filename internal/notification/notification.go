// Package notification implements the notification domain: the entity, its
// persistence interface, business rules and the MySQL repository. It maps to
// the notifications table.
package notification

import "time"

// Type values a Notification may hold.
const (
	TypeBookingConfirmed = "BOOKING_CONFIRMED"
	TypeTripReminder     = "TRIP_REMINDER"
	TypeRateTrip         = "RATE_TRIP"
	TypeBookingCancelled = "BOOKING_CANCELLED"
)

// Notification is the domain entity. ReferenceID is optional (empty string
// means "not set") and points at whatever record (a booking, a trip date,
// ...) the notification refers to; its target table depends on Type.
type Notification struct {
	ID          string
	UserID      string
	Title       string
	Body        string
	Type        string
	ReferenceID string
	IsRead      bool
	CreatedAt   time.Time
}

// CreateInput is the data required to create a new notification.
type CreateInput struct {
	UserID      string `json:"user_id" validate:"required,uuid"`
	Title       string `json:"title" validate:"required,max=255"`
	Body        string `json:"body" validate:"required"`
	Type        string `json:"type" validate:"required,oneof=BOOKING_CONFIRMED TRIP_REMINDER RATE_TRIP BOOKING_CANCELLED"`
	ReferenceID string `json:"reference_id" validate:"omitempty,uuid"`
}
