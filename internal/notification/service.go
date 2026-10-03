package notification

import (
	"context"
	"time"
)

// Service implements the notification business rules on top of a
// Repository.
type Service struct {
	repo Repository
}

// NewService builds a notification Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create persists a new, unread notification.
func (s *Service) Create(ctx context.Context, input CreateInput) (Notification, error) {
	n := Notification{
		UserID:      input.UserID,
		Title:       input.Title,
		Body:        input.Body,
		Type:        input.Type,
		ReferenceID: input.ReferenceID,
		IsRead:      false,
		CreatedAt:   time.Now().UTC(),
	}

	return s.repo.Create(ctx, n)
}

// GetByID returns the notification with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (Notification, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByUserID returns every notification for the given user.
func (s *Service) ListByUserID(ctx context.Context, userID string) ([]Notification, error) {
	return s.repo.ListByUserID(ctx, userID)
}

// MarkRead marks the notification with the given ID as read.
func (s *Service) MarkRead(ctx context.Context, id string) error {
	return s.repo.MarkRead(ctx, id)
}
