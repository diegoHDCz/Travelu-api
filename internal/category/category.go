// Package category implements the category domain: the entity, its
// persistence interface, business rules and the MySQL repository.
package category

import "time"

// Category is the domain entity. Description and Icon are optional (empty
// string means "not set").
type Category struct {
	ID          string
	Name        string
	Slug        string
	Description string
	Icon        string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CreateInput is the data required to create a new category.
type CreateInput struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Slug        string `json:"slug" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"omitempty"`
	Icon        string `json:"icon" validate:"omitempty,max=255"`
}
