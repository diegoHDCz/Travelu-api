package booking

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// BookingHandlers is the set of HTTP handlers exposed by the booking domain.
// RegisterRoutes wires every booking endpoint onto mux behind requireAuth,
// so callers only need to know about this one method instead of each
// route/method/middleware combination.
type BookingHandlers interface {
	RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware)
	Create(w http.ResponseWriter, r *http.Request)
	GetByID(w http.ResponseWriter, r *http.Request)
	ListMine(w http.ResponseWriter, r *http.Request)
}
