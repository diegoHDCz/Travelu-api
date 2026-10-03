package booking

import (
	"context"
	"strings"
	"time"
)

const defaultCurrency = "USD"
const defaultGuests = 1

// Service implements the booking business rules on top of a Repository.
type Service struct {
	repo Repository
}

// NewService builds a booking Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create persists a new booking. NumberOfGuests defaults to 1 and Currency
// defaults to USD when left unset; Status and PaymentStatus always start at
// PENDING.
func (s *Service) Create(ctx context.Context, input CreateInput) (Booking, error) {
	guests := input.NumberOfGuests
	if guests == 0 {
		guests = defaultGuests
	}

	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = defaultCurrency
	}

	now := time.Now().UTC()
	b := Booking{
		CustomerID:      input.CustomerID,
		ListingID:       input.ListingID,
		TripDateID:      input.TripDateID,
		CheckInDate:     input.CheckInDate,
		CheckOutDate:    input.CheckOutDate,
		NumberOfGuests:  guests,
		TotalPrice:      input.TotalPrice,
		Currency:        currency,
		Status:          StatusPending,
		PaymentStatus:   PaymentStatusPending,
		PaymentID:       input.PaymentID,
		SpecialRequests: input.SpecialRequests,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	return s.repo.Create(ctx, b)
}

// GetByID returns the booking with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (Booking, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByCustomerID returns every booking made by the given customer.
func (s *Service) ListByCustomerID(ctx context.Context, customerID string) ([]Booking, error) {
	return s.repo.ListByCustomerID(ctx, customerID)
}
