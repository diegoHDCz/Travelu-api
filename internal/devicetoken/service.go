package devicetoken

import (
	"context"
	"time"
)

// Service implements the device token business rules on top of a
// Repository.
type Service struct {
	repo Repository
}

// NewService builds a devicetoken Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create registers a new, active device token. Returns ErrTokenTaken if the
// token is already registered.
func (s *Service) Create(ctx context.Context, input CreateInput) (DeviceToken, error) {
	now := time.Now().UTC()
	dt := DeviceToken{
		UserID:    input.UserID,
		Token:     input.Token,
		Platform:  input.Platform,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	return s.repo.Create(ctx, dt)
}

// GetByID returns the device token with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (DeviceToken, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByUserID returns every device token registered by the given user.
func (s *Service) ListByUserID(ctx context.Context, userID string) ([]DeviceToken, error) {
	return s.repo.ListByUserID(ctx, userID)
}

// Deactivate marks the device token with the given ID as inactive, e.g. on
// logout or uninstall.
func (s *Service) Deactivate(ctx context.Context, id string) error {
	return s.repo.Deactivate(ctx, id)
}
