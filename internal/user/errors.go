package user

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Sentinel domain errors. Each carries the HTTP status and machine-readable
// code httpx.WriteError will use to build the response.
var (
	ErrNotFound      = httpx.NewError(http.StatusNotFound, "user_not_found", "user not found")
	ErrEmailTaken    = httpx.NewError(http.StatusConflict, "email_taken", "email already in use")
	ErrPhoneTaken    = httpx.NewError(http.StatusConflict, "phone_taken", "phone already in use")
	ErrUsernameTaken = httpx.NewError(http.StatusConflict, "username_taken", "username already in use")
	ErrInvalidPhone  = httpx.NewError(http.StatusUnprocessableEntity, "invalid_phone", "phone is not a valid number")
)
