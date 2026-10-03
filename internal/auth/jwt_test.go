package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenManager_GenerateAndParse(t *testing.T) {
	tm := NewTokenManager(testJWTSecret, time.Minute)

	token, ttl, err := tm.Generate("42", "device-abc")
	require.NoError(t, err)
	assert.Equal(t, time.Minute, ttl)

	userID, deviceToken, err := tm.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, "42", userID)
	assert.Equal(t, "device-abc", deviceToken)
}

func TestTokenManager_GenerateAndParse_NoDeviceToken(t *testing.T) {
	tm := NewTokenManager(testJWTSecret, time.Minute)

	token, _, err := tm.Generate("42", "")
	require.NoError(t, err)

	userID, deviceToken, err := tm.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, "42", userID)
	assert.Empty(t, deviceToken)
}

func TestTokenManager_RejectsNoneAlgorithm(t *testing.T) {
	tm := NewTokenManager(testJWTSecret, time.Minute)

	claims := jwt.RegisteredClaims{
		Subject:   "1",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
	}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenString, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, _, err = tm.Parse(tokenString)
	assert.Error(t, err)
}

func TestTokenManager_RejectsExpiredToken(t *testing.T) {
	tm := NewTokenManager(testJWTSecret, -time.Minute)

	token, _, err := tm.Generate("1", "")
	require.NoError(t, err)

	_, _, err = tm.Parse(token)
	assert.Error(t, err)
}
