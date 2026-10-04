package listing

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, l Listing) (Listing, error)
	GetByID(ctx context.Context, id string) (Listing, error)
	ListActive(ctx context.Context) ([]Listing, error)
	Update(ctx context.Context, l Listing) (Listing, error)
}
