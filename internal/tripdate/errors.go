package tripdate

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// ErrNotFound is returned when no trip date matches the requested ID.
var ErrNotFound = httpx.NewError(http.StatusNotFound, "trip_date_not_found", "trip date not found")
