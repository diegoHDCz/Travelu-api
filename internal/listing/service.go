package listing

import (
	"context"
	"strings"
	"time"
)

// defaultCurrency is used when CreateInput.Currency is left blank.
const defaultCurrency = "USD"

// Service implements the listing business rules on top of a Repository.
type Service struct {
	repo Repository
}

// NewService builds a listing Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create persists a new listing. Currency defaults to USD when input.Currency
// is blank; Rating and ReviewCount start at zero and are maintained by the
// review domain, not by this service.
func (s *Service) Create(ctx context.Context, input CreateInput) (Listing, error) {
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = defaultCurrency
	}

	now := time.Now().UTC()
	l := Listing{
		VendorID:      input.VendorID,
		Title:         strings.TrimSpace(input.Title),
		Description:   input.Description,
		Category:      input.Category,
		Location:      strings.TrimSpace(input.Location),
		City:          strings.TrimSpace(input.City),
		Country:       strings.TrimSpace(input.Country),
		Price:         input.Price,
		Currency:      currency,
		Capacity:      input.Capacity,
		AvailableFrom: input.AvailableFrom,
		AvailableTo:   input.AvailableTo,
		Images:        input.Images,
		Amenities:     input.Amenities,
		IsActive:      true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	return s.repo.Create(ctx, l)
}

// GetByID returns the listing with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (Listing, error) {
	return s.repo.GetByID(ctx, id)
}

// ListActive returns every active listing.
func (s *Service) ListActive(ctx context.Context) ([]Listing, error) {
	return s.repo.ListActive(ctx)
}

// Update overwrites the mutable fields of an existing listing. VendorID is
// not part of UpdateInput, so ownership is never changed by this call.
func (s *Service) Update(ctx context.Context, input UpdateInput) (Listing, error) {
	currency := strings.ToUpper(strings.TrimSpace(input.Currency))
	if currency == "" {
		currency = defaultCurrency
	}

	l := Listing{
		ID:            input.ID,
		Title:         strings.TrimSpace(input.Title),
		Description:   input.Description,
		Category:      input.Category,
		Location:      strings.TrimSpace(input.Location),
		City:          strings.TrimSpace(input.City),
		Country:       strings.TrimSpace(input.Country),
		Price:         input.Price,
		Currency:      currency,
		Capacity:      input.Capacity,
		AvailableFrom: input.AvailableFrom,
		AvailableTo:   input.AvailableTo,
		Images:        input.Images,
		Amenities:     input.Amenities,
		IsActive:      input.IsActive,
	}

	return s.repo.Update(ctx, l)
}
