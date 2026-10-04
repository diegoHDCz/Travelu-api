package review

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
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
	reviewID := uuid.NewString()
	r := Review{
		ID:         reviewID,
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

// Update overwrites the rating, title and comment of an existing review.
// Only the review's original author may update it; anyone else gets
// ErrNotFound, so this endpoint can't be used to probe which IDs exist or
// who owns them.
func (s *Service) Update(ctx context.Context, input UpdateInput) (Review, error) {
	existing, err := s.repo.GetByID(ctx, input.ID)
	if err != nil {
		return Review{}, err
	}
	if existing.CustomerID != input.CustomerID {
		return Review{}, ErrNotFound
	}

	existing.Rating = input.Rating
	existing.Title = strings.TrimSpace(input.Title)
	existing.Comment = strings.TrimSpace(input.Comment)
	existing.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, existing)
}
