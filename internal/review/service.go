package review

import (
	"context"
	"strings"
	"time"
)

// Service implements the review business rules on top of a Repository.
type Service struct {
	repo Repository
}

// NewService builds a review Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create persists a new review. IsVerified starts false (the booking domain
// is responsible for marking a review as a verified purchase) and IsApproved
// starts true, matching the originating schema's defaults.
func (s *Service) Create(ctx context.Context, input CreateInput) (Review, error) {
	now := time.Now().UTC()
	r := Review{
		CustomerID: input.CustomerID,
		ListingID:  input.ListingID,
		BookingID:  input.BookingID,
		Rating:     input.Rating,
		Title:      strings.TrimSpace(input.Title),
		Comment:    strings.TrimSpace(input.Comment),
		IsVerified: false,
		IsApproved: true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	return s.repo.Create(ctx, r)
}

// GetByID returns the review with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (Review, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByListingID returns every review for the given listing.
func (s *Service) ListByListingID(ctx context.Context, listingID string) ([]Review, error) {
	return s.repo.ListByListingID(ctx, listingID)
}
