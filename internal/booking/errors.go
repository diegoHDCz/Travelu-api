package booking

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// ErrNotFound is returned when no booking matches the requested ID.
var ErrNotFound = httpx.NewError(http.StatusNotFound, "booking_not_found", "booking not found")
