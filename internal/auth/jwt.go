package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenManager issues and verifies HS256 access tokens with sub/exp/iat/jti
// claims.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager builds a TokenManager. secret must be at least 32 bytes;
// config.Load already enforces that for JWT_SECRET.
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

// Generate issues a signed access token for userID and returns its TTL.
func (m *TokenManager) Generate(userID string) (token string, ttl time.Duration, err error) {
	jti, err := randomID()
	if err != nil {
		return "", 0, fmt.Errorf("generate jti: %w", err)
	}

	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		IssuedAt:  jwt.NewNumericDate(now),
		ID:        jti,
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}

	return signed, m.ttl, nil
}

// Parse validates tokenString and returns the user ID carried in its
// subject claim. The signing algorithm is checked explicitly: only HS256 is
// ever accepted, regardless of what the token header claims.
func (m *TokenManager) Parse(tokenString string) (string, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return "", fmt.Errorf("parse token: %w", err)
	}

	if claims.Subject == "" {
		return "", errors.New("token missing subject claim")
	}

	return claims.Subject, nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
