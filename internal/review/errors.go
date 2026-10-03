package review

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// ErrNotFound is returned when no review matches the requested ID.
var ErrNotFound = httpx.NewError(http.StatusNotFound, "review_not_found", "review not found")
