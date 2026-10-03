package devicetoken

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Sentinel domain errors. Each carries the HTTP status and machine-readable
// code httpx.WriteError will use to build the response.
var (
	ErrNotFound   = httpx.NewError(http.StatusNotFound, "device_token_not_found", "device token not found")
	ErrTokenTaken = httpx.NewError(http.StatusConflict, "device_token_taken", "device token already registered")
)
