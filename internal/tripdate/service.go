package tripdate

import (
	"context"
	"time"
)

// Service implements the trip date business rules on top of a Repository.
type Service struct {
	repo Repository
}

// NewService builds a tripdate Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create persists a new trip date. CurrentBookings always starts at zero.
func (s *Service) Create(ctx context.Context, input CreateInput) (TripDate, error) {
	now := time.Now().UTC()
	td := TripDate{
		ListingID:   input.ListingID,
		StartDate:   input.StartDate,
		EndDate:     input.EndDate,
		MaxCapacity: input.MaxCapacity,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, td)
}

// GetByID returns the trip date with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (TripDate, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByListingID returns every trip date for the given listing.
func (s *Service) ListByListingID(ctx context.Context, listingID string) ([]TripDate, error) {
	return s.repo.ListByListingID(ctx, listingID)
}
