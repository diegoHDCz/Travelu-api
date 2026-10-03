package listing

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// ErrNotFound is returned when no listing matches the requested ID.
var ErrNotFound = httpx.NewError(http.StatusNotFound, "listing_not_found", "listing not found")
