package review

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, r Review) (Review, error)
	GetByID(ctx context.Context, id string) (Review, error)
	ListByListingID(ctx context.Context, listingID string) ([]Review, error)
}
