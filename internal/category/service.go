package category

import (
	"context"
	"strings"
	"time"
)

// Service implements the category business rules on top of a Repository.
type Service struct {
	repo Repository
}

// NewService builds a category Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create trims the name and lowercases the slug before persisting. Returns
// ErrNameTaken or ErrSlugTaken if the corresponding field is already
// registered.
func (s *Service) Create(ctx context.Context, input CreateInput) (Category, error) {
	now := time.Now().UTC()
	c := Category{
		Name:        strings.TrimSpace(input.Name),
		Slug:        strings.ToLower(strings.TrimSpace(input.Slug)),
		Description: strings.TrimSpace(input.Description),
		Icon:        strings.TrimSpace(input.Icon),
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, c)
}

// GetByID returns the category with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (Category, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySlug returns the category with the given slug, or ErrNotFound.
func (s *Service) GetBySlug(ctx context.Context, slug string) (Category, error) {
	return s.repo.GetBySlug(ctx, strings.ToLower(strings.TrimSpace(slug)))
}

// List returns every category.
func (s *Service) List(ctx context.Context) ([]Category, error) {
	return s.repo.List(ctx)
}
