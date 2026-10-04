package listing

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

type ListingHandlers interface {
	// RegisterRoutes wires every listing endpoint onto mux behind requireAuth,
	// so callers only need to know about this one method instead of each
	// route/method/middleware combination.
	RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	GetByID(w http.ResponseWriter, r *http.Request)
	List(w http.ResponseWriter, r *http.Request)
}
