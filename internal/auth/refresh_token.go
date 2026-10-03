package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

// RefreshToken is the domain entity for an opaque refresh token. Only its
// SHA-256 hash is ever persisted; the plain value is returned to the client
// once, at issuance time, and never stored.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// RefreshTokenRepository is the persistence interface consumed by Service.
type RefreshTokenRepository interface {
	Create(ctx context.Context, rt RefreshToken) (RefreshToken, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
}

// generateOpaqueToken returns a new 32-byte random token (base64url encoded)
// along with the hex-encoded SHA-256 hash that should be persisted.
func generateOpaqueToken() (plain, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}

	plain = base64.RawURLEncoding.EncodeToString(buf)
	return plain, hashToken(plain), nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
