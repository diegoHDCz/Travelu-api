package category

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, c Category) (Category, error)
	GetByID(ctx context.Context, id string) (Category, error)
	GetBySlug(ctx context.Context, slug string) (Category, error)
	List(ctx context.Context) ([]Category, error)
}
