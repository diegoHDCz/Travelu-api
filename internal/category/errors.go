package category

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Sentinel domain errors. Each carries the HTTP status and machine-readable
// code httpx.WriteError will use to build the response.
var (
	ErrNotFound  = httpx.NewError(http.StatusNotFound, "category_not_found", "category not found")
	ErrNameTaken = httpx.NewError(http.StatusConflict, "category_name_taken", "category name already in use")
	ErrSlugTaken = httpx.NewError(http.StatusConflict, "category_slug_taken", "category slug already in use")
)
