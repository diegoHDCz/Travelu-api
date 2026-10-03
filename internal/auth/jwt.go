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
// claims, plus a custom device_token claim identifying the device the token
// was issued to.
type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenManager builds a TokenManager. secret must be at least 32 bytes;
// config.Load already enforces that for JWT_SECRET.
func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

// claims is the JWT payload: the standard registered claims plus the device
// token the client presented at login/refresh time, if any.
type claims struct {
	DeviceToken string `json:"device_token,omitempty"`
	jwt.RegisteredClaims
}

// Generate issues a signed access token for userID and returns its TTL.
// deviceToken is the push-notification token of the device the client is
// authenticating from; pass "" if the client did not send one.
func (m *TokenManager) Generate(userID, deviceToken string) (token string, ttl time.Duration, err error) {
	jti, err := randomID()
	if err != nil {
		return "", 0, fmt.Errorf("generate jti: %w", err)
	}

	now := time.Now().UTC()
	c := claims{
		DeviceToken: deviceToken,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}

	return signed, m.ttl, nil
}

// Parse validates tokenString and returns the user ID and device token
// carried in its claims. The signing algorithm is checked explicitly: only
// HS256 is ever accepted, regardless of what the token header claims.
func (m *TokenManager) Parse(tokenString string) (userID, deviceToken string, err error) {
	c := &claims{}

	_, err = jwt.ParseWithClaims(tokenString, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return "", "", fmt.Errorf("parse token: %w", err)
	}

	if c.Subject == "" {
		return "", "", errors.New("token missing subject claim")
	}

	return c.Subject, c.DeviceToken, nil
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
