package notification

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, n Notification) (Notification, error)
	GetByID(ctx context.Context, id string) (Notification, error)
	ListByUserID(ctx context.Context, userID string) ([]Notification, error)
	MarkRead(ctx context.Context, id string) error
}
