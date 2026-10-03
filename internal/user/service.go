package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service implements the user business rules on top of a Repository.
type Service struct {
	repo Repository
}

// NewService builds a user Service backed by repo.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create hashes the given password and persists a new user. Email and
// Username are trimmed and lowercased; Phone is sanitized to digits-only
// (keeping a leading '+') and checked against ValidPhone before anything is
// written, since the raw input may still carry formatting such as spaces,
// parentheses or dashes. Returns ErrInvalidPhone if Phone is set but not a
// valid E.164 number, or ErrEmailTaken/ErrPhoneTaken/ErrUsernameTaken if the
// corresponding field is already registered.
func (s *Service) Create(ctx context.Context, input CreateInput) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	phone := ""
	if input.Phone != "" {
		phone = SanitizePhone(input.Phone)
		if !ValidPhone(phone) {
			return User{}, ErrInvalidPhone
		}
	}

	now := time.Now().UTC()
	u := User{
		Name:         strings.TrimSpace(input.Name),
		Email:        strings.ToLower(strings.TrimSpace(input.Email)),
		Phone:        phone,
		Username:     strings.ToLower(strings.TrimSpace(input.Username)),
		PasswordHash: string(hash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.repo.Create(ctx, u)
	if err != nil {
		return User{}, err
	}

	return created, nil
}

// GetByEmail returns the user with the given email, or ErrNotFound.
func (s *Service) GetByEmail(ctx context.Context, email string) (User, error) {
	return s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
}

// GetByLogin resolves a login identifier that may be an email, a phone
// number or a username, and returns the matching user (or ErrNotFound). The
// identifier's shape decides how it is looked up: containing '@' means
// email; otherwise, sanitizing it into a valid E.164 number means phone;
// anything else is treated as a username.
func (s *Service) GetByLogin(ctx context.Context, login string) (User, error) {
	login = strings.TrimSpace(login)

	switch {
	case strings.Contains(login, "@"):
		return s.repo.GetByEmail(ctx, strings.ToLower(login))
	case ValidPhone(SanitizePhone(login)):
		return s.repo.GetByPhone(ctx, SanitizePhone(login))
	default:
		return s.repo.GetByUsername(ctx, strings.ToLower(login))
	}
}

// GetByID returns the user with the given ID, or ErrNotFound.
func (s *Service) GetByID(ctx context.Context, id string) (User, error) {
	return s.repo.GetByID(ctx, id)
}
