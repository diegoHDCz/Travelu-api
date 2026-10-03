package devicetoken

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, dt DeviceToken) (DeviceToken, error)
	GetByID(ctx context.Context, id string) (DeviceToken, error)
	ListByUserID(ctx context.Context, userID string) ([]DeviceToken, error)
	Deactivate(ctx context.Context, id string) error
}
