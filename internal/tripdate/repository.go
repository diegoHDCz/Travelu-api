package tripdate

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, td TripDate) (TripDate, error)
	GetByID(ctx context.Context, id string) (TripDate, error)
	ListByListingID(ctx context.Context, listingID string) ([]TripDate, error)
}
