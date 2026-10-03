package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/diegoczajka/travelu-api/internal/user"
)

// UserService is the subset of user.Service that auth needs. It is defined
// here, in the package that consumes it.
type UserService interface {
	Create(ctx context.Context, input user.CreateInput) (user.User, error)
	GetByLogin(ctx context.Context, login string) (user.User, error)
	GetByID(ctx context.Context, id string) (user.User, error)
}

// TokenPair is what Login and Refresh hand back to the client.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds
}

// Service implements registration, login, refresh-token rotation and logout.
type Service struct {
	users         UserService
	refreshTokens RefreshTokenRepository
	tokens        *TokenManager
	refreshTTL    time.Duration
}

// NewService builds an auth Service.
func NewService(users UserService, refreshTokens RefreshTokenRepository, tokens *TokenManager, refreshTTL time.Duration) *Service {
	return &Service{
		users:         users,
		refreshTokens: refreshTokens,
		tokens:        tokens,
		refreshTTL:    refreshTTL,
	}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, input user.CreateInput) (user.User, error) {
	return s.users.Create(ctx, input)
}

// Login verifies a login identifier (email, phone or username) plus
// password and issues a new token pair. It returns the same generic
// ErrInvalidCredentials whether the identifier does not exist or the
// password is wrong, so callers cannot enumerate registered accounts.
func (s *Service) Login(ctx context.Context, login, password string) (TokenPair, error) {
	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	return s.issueTokenPair(ctx, u.ID)
}

// Refresh rotates a refresh token: the presented token is revoked and a new
// pair is issued. Reuse of an already-revoked token is treated as a sign of
// theft and revokes every refresh token belonging to that user.
func (s *Service) Refresh(ctx context.Context, plainToken string) (TokenPair, error) {
	rt, err := s.refreshTokens.GetByTokenHash(ctx, hashToken(plainToken))
	if err != nil {
		if errors.Is(err, errRefreshTokenNotFound) {
			return TokenPair{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, err
	}

	if rt.RevokedAt != nil {
		if err := s.refreshTokens.RevokeAllForUser(ctx, rt.UserID); err != nil {
			return TokenPair{}, fmt.Errorf("revoke all refresh tokens after reuse: %w", err)
		}
		return TokenPair{}, ErrInvalidRefreshToken
	}

	if time.Now().UTC().After(rt.ExpiresAt) {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	if err := s.refreshTokens.Revoke(ctx, rt.ID); err != nil {
		return TokenPair{}, fmt.Errorf("revoke used refresh token: %w", err)
	}

	return s.issueTokenPair(ctx, rt.UserID)
}

// Logout revokes the given refresh token. It is idempotent: an unknown or
// already-revoked token is not an error.
func (s *Service) Logout(ctx context.Context, plainToken string) error {
	rt, err := s.refreshTokens.GetByTokenHash(ctx, hashToken(plainToken))
	if err != nil {
		if errors.Is(err, errRefreshTokenNotFound) {
			return nil
		}
		return err
	}

	if rt.RevokedAt != nil {
		return nil
	}

	return s.refreshTokens.Revoke(ctx, rt.ID)
}

func (s *Service) issueTokenPair(ctx context.Context, userID string) (TokenPair, error) {
	access, ttl, err := s.tokens.Generate(userID)
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate access token: %w", err)
	}

	plain, hash, err := generateOpaqueToken()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	if _, err := s.refreshTokens.Create(ctx, RefreshToken{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}); err != nil {
		return TokenPair{}, fmt.Errorf("store refresh token: %w", err)
	}

	return TokenPair{
		AccessToken:  access,
		RefreshToken: plain,
		ExpiresIn:    int64(ttl.Seconds()),
	}, nil
}
