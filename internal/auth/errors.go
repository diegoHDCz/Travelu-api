package auth

import (
	"errors"
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Sentinel domain errors, carrying the HTTP status and code httpx.WriteError
// uses to build the response.
var (
	ErrInvalidCredentials  = httpx.NewError(http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
	ErrInvalidRefreshToken = httpx.NewError(http.StatusUnauthorized, "invalid_refresh_token", "refresh token is invalid or expired")
	ErrUnauthorized        = httpx.NewError(http.StatusUnauthorized, "unauthorized", "missing or invalid authorization token")
	errTooManyRequests     = httpx.NewError(http.StatusTooManyRequests, "too_many_requests", "too many requests, try again later")
)

// errRefreshTokenNotFound is internal: it distinguishes a missing row from
// other repository failures and is never written to a client directly.
var errRefreshTokenNotFound = errors.New("refresh token not found")
