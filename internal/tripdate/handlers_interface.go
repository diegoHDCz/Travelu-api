package tripdate

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// TripDateHandlers is the set of HTTP handlers exposed by the trip date
// domain. RegisterRoutes wires every trip date endpoint onto mux behind
// requireAuth, so callers only need to know about this one method instead
// of each route/method/middleware combination.
type TripDateHandlers interface {
	RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware)
	Create(w http.ResponseWriter, r *http.Request)
	GetByID(w http.ResponseWriter, r *http.Request)
	List(w http.ResponseWriter, r *http.Request)
}
