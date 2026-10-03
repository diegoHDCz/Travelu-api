package user

import "context"

// Repository is the persistence interface consumed by Service. It is
// defined here, in the package that consumes it, and implemented by
// MySQLRepository (and by fakes in tests).
type Repository interface {
	Create(ctx context.Context, u User) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	GetByPhone(ctx context.Context, phone string) (User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
}
