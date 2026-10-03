package booking

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, b Booking) (Booking, error)
	GetByID(ctx context.Context, id string) (Booking, error)
	ListByCustomerID(ctx context.Context, customerID string) ([]Booking, error)
}
