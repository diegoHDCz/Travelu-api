package review

import (
	"net/http"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// ReviewHandlers is the set of HTTP handlers exposed by the review domain.
// RegisterRoutes wires every review endpoint onto mux behind requireAuth, so
// callers only need to know about this one method instead of each
// route/method/middleware combination.
type ReviewHandlers interface {
	RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	GetByID(w http.ResponseWriter, r *http.Request)
	List(w http.ResponseWriter, r *http.Request)
}
