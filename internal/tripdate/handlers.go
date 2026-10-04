package tripdate

import (
	"net/http"
	"time"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Handler exposes the trip date HTTP endpoints.
type Handler struct {
	service *Service
}

// var _ enforces at compile time that *Handler satisfies TripDateHandlers;
// a missing or mis-signatured method fails the build here instead of at
// the call site that constructs the interface.
var _ TripDateHandlers = (*Handler)(nil)

// NewHandler builds a trip date Handler.
func NewHandler(service *Service) TripDateHandlers {
	return &Handler{service: service}
}

// RegisterRoutes wires every trip date endpoint onto mux behind requireAuth,
// so callers only need to know about this one method instead of each
// route/method/middleware combination.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("POST /api/v1/trip-dates", requireAuth(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/trip-dates", requireAuth(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/trip-dates/{id}", requireAuth(http.HandlerFunc(h.GetByID)))
}

type tripDateResponse struct {
	ID              string `json:"id"`
	ListingID       string `json:"listing_id"`
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	MaxCapacity     *int   `json:"max_capacity,omitempty"`
	CurrentBookings int    `json:"current_bookings"`
	IsActive        bool   `json:"is_active"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func newTripDateResponse(td TripDate) tripDateResponse {
	return tripDateResponse{
		ID:              td.ID,
		ListingID:       td.ListingID,
		StartDate:       td.StartDate.Format(time.RFC3339),
		EndDate:         td.EndDate.Format(time.RFC3339),
		MaxCapacity:     td.MaxCapacity,
		CurrentBookings: td.CurrentBookings,
		IsActive:        td.IsActive,
		CreatedAt:       td.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       td.UpdatedAt.Format(time.RFC3339),
	}
}

type createRequest struct {
	ListingID   string    `json:"listing_id" validate:"required,uuid"`
	StartDate   time.Time `json:"start_date" validate:"required"`
	EndDate     time.Time `json:"end_date" validate:"required,gtfield=StartDate"`
	MaxCapacity *int      `json:"max_capacity" validate:"omitempty,gt=0"`
}

// Create handles POST /api/v1/trip-dates.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	created, err := h.service.Create(r.Context(), CreateInput{
		ListingID:   req.ListingID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		MaxCapacity: req.MaxCapacity,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newTripDateResponse(created))
}

// GetByID handles GET /api/v1/trip-dates/{id}.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	tripDateID := r.PathValue("id")
	if tripDateID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_trip_date_id", "trip date ID is required"))
		return
	}

	tripDate, err := h.service.GetByID(r.Context(), tripDateID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newTripDateResponse(tripDate))
}

// List handles GET /api/v1/trip-dates?listing_id={id}.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	listingID := r.URL.Query().Get("listing_id")
	if listingID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_listing_id", "listing_id query parameter is required"))
		return
	}

	tripDates, err := h.service.ListByListingID(r.Context(), listingID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	response := make([]tripDateResponse, 0, len(tripDates))
	for _, td := range tripDates {
		response = append(response, newTripDateResponse(td))
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}
