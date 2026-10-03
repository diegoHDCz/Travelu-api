// Package user implements the user domain: the entity, its persistence
// interface, business rules and the MySQL repository.
package user

import "time"

// User is the domain entity. It carries no database tags: the repository is
// responsible for mapping to and from storage. Email, Phone and Username are
// each individually optional (empty string means "not set"), but at least
// one of the three is always present — enforced by CreateInput's validation.
type User struct {
	ID           string
	Name         string
	Email        string
	Phone        string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateInput is the data required to register a new user. Email, Phone and
// Username are each optional, but required_without_all enforces that at
// least one of them is present — this covers users who sign up without an
// email and log in with their phone or username instead.
type CreateInput struct {
	Name     string `json:"name" validate:"required,min=2,max=255"`
	Email    string `json:"email" validate:"omitempty,email,max=255,required_without_all=Phone Username"`
	Phone    string `json:"phone" validate:"omitempty,max=20,required_without_all=Email Username"`
	Username string `json:"username" validate:"omitempty,min=3,max=30,alphanum,required_without_all=Email Phone"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}
