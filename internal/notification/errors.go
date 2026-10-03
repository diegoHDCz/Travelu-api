package notification

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// ErrNotFound is returned when no notification matches the requested ID.
var ErrNotFound = httpx.NewError(http.StatusNotFound, "notification_not_found", "notification not found")
