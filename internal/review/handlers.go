package review

import (
	"net/http"
	"time"

	"github.com/diegoczajka/travelu-api/internal/auth"
	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Handler exposes the review HTTP endpoints.
type Handler struct {
	service *Service
}

// var _ enforces at compile time that *Handler satisfies ReviewHandlers; a
// missing or mis-signatured method fails the build here instead of at the
// call site that constructs the interface.
var _ ReviewHandlers = (*Handler)(nil)

// NewHandler builds a review Handler.
func NewHandler(service *Service) ReviewHandlers {
	return &Handler{service: service}
}

// RegisterRoutes wires every review endpoint onto mux behind requireAuth, so
// callers only need to know about this one method instead of each
// route/method/middleware combination.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("POST /api/v1/reviews", requireAuth(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/reviews", requireAuth(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/reviews/{id}", requireAuth(http.HandlerFunc(h.GetByID)))
	mux.Handle("PUT /api/v1/reviews/{id}", requireAuth(http.HandlerFunc(h.Update)))
}

type reviewResponse struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	ListingID  string `json:"listing_id"`
	BookingID  string `json:"booking_id"`
	Rating     int    `json:"rating"`
	Title      string `json:"title,omitempty"`
	Comment    string `json:"comment,omitempty"`
	IsVerified bool   `json:"is_verified"`
	IsApproved bool   `json:"is_approved"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func newReviewResponse(r Review) reviewResponse {
	return reviewResponse{
		ID:         r.ID,
		CustomerID: r.CustomerID,
		ListingID:  r.ListingID,
		BookingID:  r.BookingID,
		Rating:     r.Rating,
		Title:      r.Title,
		Comment:    r.Comment,
		IsVerified: r.IsVerified,
		IsApproved: r.IsApproved,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
	}
}

type createRequest struct {
	BookingID string `json:"booking_id" validate:"required,uuid"`
	Rating    int    `json:"rating" validate:"required,min=1,max=5"`
	Title     string `json:"title" validate:"omitempty,max=255"`
	Comment   string `json:"comment" validate:"omitempty,max=1000"`
	ListingID string `json:"listing_id" validate:"required,uuid"`
}

// Create handles POST /api/v1/reviews.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	customerID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		httpx.WriteError(r.Context(), w, auth.ErrUnauthorized)
		return
	}

	review, err := h.service.Create(r.Context(), CreateInput{
		CustomerID: customerID,
		Rating:     req.Rating,
		Comment:    req.Comment,
		BookingID:  req.BookingID,
		Title:      req.Title,
		ListingID:  req.ListingID,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newReviewResponse(review))
}

type updateRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Title   string `json:"title" validate:"omitempty,max=255"`
	Comment string `json:"comment" validate:"omitempty,max=1000"`
}

// Update handles PUT /api/v1/reviews/{id}. A review that exists but belongs
// to someone else is reported as not found, not forbidden, so this endpoint
// can't be used to probe which IDs exist or who wrote them.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	reviewID := r.PathValue("id")
	if reviewID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_review_id", "review ID is required"))
		return
	}

	customerID, ok := auth.UserIDFrom(r.Context())
	if !ok {
		httpx.WriteError(r.Context(), w, auth.ErrUnauthorized)
		return
	}

	updated, err := h.service.Update(r.Context(), UpdateInput{
		ID:         reviewID,
		CustomerID: customerID,
		Rating:     req.Rating,
		Title:      req.Title,
		Comment:    req.Comment,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newReviewResponse(updated))
}

// GetByID handles GET /api/v1/reviews/{id}. Reviews are public content, so
// no ownership check applies here, unlike Update.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	reviewID := r.PathValue("id")
	if reviewID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_review_id", "review ID is required"))
		return
	}

	review, err := h.service.GetByID(r.Context(), reviewID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newReviewResponse(review))
}

// List handles GET /api/v1/reviews?listing_id={id}.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	listingID := r.URL.Query().Get("listing_id")
	if listingID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_listing_id", "listing_id query parameter is required"))
		return
	}

	reviews, err := h.service.ListByListingID(r.Context(), listingID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	response := make([]reviewResponse, 0, len(reviews))
	for _, rv := range reviews {
		response = append(response, newReviewResponse(rv))
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}
