package listing

import (
	"net/http"
	"time"

	"github.com/diegoczajka/travelu-api/internal/auth"
	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Handler exposes the listing HTTP endpoints.
type Handler struct {
	service *Service
}

// var _ enforces at compile time that *Handler satisfies ListingHandlers;
// a missing or mis-signatured method fails the build here instead of at
// the call site that constructs the interface.
var _ ListingHandlers = (*Handler)(nil)

// NewHandler builds a listing Handler.
func NewHandler(service *Service) ListingHandlers {
	return &Handler{service: service}
}

// RegisterRoutes wires every listing endpoint onto mux behind requireAuth,
// so callers only need to know about this one method instead of each
// route/method/middleware combination.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("POST /api/v1/listings", requireAuth(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/listings", requireAuth(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/listings/{id}", requireAuth(http.HandlerFunc(h.GetByID)))
	mux.Handle("PUT /api/v1/listings/{id}", requireAuth(http.HandlerFunc(h.Update)))
}

type listingResponse struct {
	ID            string  `json:"id"`
	VendorID      string  `json:"vendor_id"`
	Title         string  `json:"title"`
	Description   string  `json:"description,omitempty"`
	Category      string  `json:"category"`
	Location      string  `json:"location"`
	City          string  `json:"city,omitempty"`
	Country       string  `json:"country,omitempty"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	Capacity      *int    `json:"capacity,omitempty"`
	AvailableFrom string  `json:"available_from,omitempty"`
	AvailableTo   string  `json:"available_to,omitempty"`
	Images        string  `json:"images,omitempty"`
	Amenities     string  `json:"amenities,omitempty"`
	Rating        float64 `json:"rating"`
	ReviewCount   int     `json:"review_count"`
	IsActive      bool    `json:"is_active"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// formatTime renders t as RFC3339, or "" when t is the zero value (not set).
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

func newListingResponse(l Listing) listingResponse {
	return listingResponse{
		ID:            l.ID,
		VendorID:      l.VendorID,
		Title:         l.Title,
		Description:   l.Description,
		Category:      l.Category,
		Location:      l.Location,
		City:          l.City,
		Country:       l.Country,
		Price:         l.Price,
		Currency:      l.Currency,
		Capacity:      l.Capacity,
		AvailableFrom: formatTime(l.AvailableFrom),
		AvailableTo:   formatTime(l.AvailableTo),
		Images:        l.Images,
		Amenities:     l.Amenities,
		Rating:        l.Rating,
		ReviewCount:   l.ReviewCount,
		IsActive:      l.IsActive,
		CreatedAt:     formatTime(l.CreatedAt),
		UpdatedAt:     formatTime(l.UpdatedAt),
	}
}

// createListingRequest mirrors CreateInput but without VendorID: that comes
// from the authenticated caller, never from the request body.
type createListingRequest struct {
	Title         string    `json:"title" validate:"required,min=2,max=255"`
	Description   string    `json:"description" validate:"required"`
	Category      string    `json:"category" validate:"required,oneof=HOTEL FLIGHT ACTIVITY PACKAGE"`
	Location      string    `json:"location" validate:"required,max=255"`
	City          string    `json:"city" validate:"omitempty,max=100"`
	Country       string    `json:"country" validate:"omitempty,max=100"`
	Price         float64   `json:"price" validate:"required,gt=0"`
	Currency      string    `json:"currency" validate:"omitempty,len=3"`
	Capacity      *int      `json:"capacity" validate:"omitempty,gt=0"`
	AvailableFrom time.Time `json:"available_from" validate:"omitempty"`
	AvailableTo   time.Time `json:"available_to" validate:"omitempty"`
	Images        string    `json:"images" validate:"omitempty"`
	Amenities     string    `json:"amenities" validate:"omitempty"`
}

type updateListingRequest struct {
	Title         string    `json:"title" validate:"required,min=2,max=255"`
	Description   string    `json:"description" validate:"required"`
	Category      string    `json:"category" validate:"required,oneof=HOTEL FLIGHT ACTIVITY PACKAGE"`
	Location      string    `json:"location" validate:"required,max=255"`
	City          string    `json:"city" validate:"omitempty,max=100"`
	Country       string    `json:"country" validate:"omitempty,max=100"`
	Price         float64   `json:"price" validate:"required,gt=0"`
	Currency      string    `json:"currency" validate:"omitempty,len=3"`
	Capacity      *int      `json:"capacity" validate:"omitempty,gt=0"`
	AvailableFrom time.Time `json:"available_from" validate:"omitempty"`
	AvailableTo   time.Time `json:"available_to" validate:"omitempty"`
	Images        string    `json:"images" validate:"omitempty"`
	Amenities     string    `json:"amenities" validate:"omitempty"`
	IsActive      bool      `json:"is_active"`
}

// Create handles POST /api/v1/listings.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createListingRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	vendorID, _ := auth.UserIDFrom(r.Context())

	created, err := h.service.Create(r.Context(), CreateInput{
		VendorID:      vendorID,
		Title:         req.Title,
		Description:   req.Description,
		Category:      req.Category,
		Location:      req.Location,
		City:          req.City,
		Country:       req.Country,
		Price:         req.Price,
		Currency:      req.Currency,
		Capacity:      req.Capacity,
		AvailableFrom: req.AvailableFrom,
		AvailableTo:   req.AvailableTo,
		Images:        req.Images,
		Amenities:     req.Amenities,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newListingResponse(created))
}

// Update handles PUT /api/v1/listings/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateListingRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	listingID := r.PathValue("id")
	if listingID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_listing_id", "listing ID is required"))
		return
	}

	updated, err := h.service.Update(r.Context(), UpdateInput{
		ID:            listingID,
		Title:         req.Title,
		Description:   req.Description,
		Category:      req.Category,
		Location:      req.Location,
		City:          req.City,
		Country:       req.Country,
		Price:         req.Price,
		Currency:      req.Currency,
		Capacity:      req.Capacity,
		AvailableFrom: req.AvailableFrom,
		AvailableTo:   req.AvailableTo,
		Images:        req.Images,
		Amenities:     req.Amenities,
		IsActive:      req.IsActive,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newListingResponse(updated))
}

// GetByID handles GET /api/v1/listings/{id}.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	listingID := r.PathValue("id")
	if listingID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_listing_id", "listing ID is required"))
		return
	}

	listing, err := h.service.GetByID(r.Context(), listingID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newListingResponse(listing))
}

// List handles GET /api/v1/listings.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	listings, err := h.service.ListActive(r.Context())
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	response := make([]listingResponse, 0, len(listings))
	for _, l := range listings {
		response = append(response, newListingResponse(l))
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}
