package booking

import (
	"net/http"
	"time"

	"github.com/diegoczajka/travelu-api/internal/auth"
	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

// Handler exposes the booking HTTP endpoints.
type Handler struct {
	service *Service
}

// var _ enforces at compile time that *Handler satisfies BookingHandlers; a
// missing or mis-signatured method fails the build here instead of at the
// call site that constructs the interface.
var _ BookingHandlers = (*Handler)(nil)

// NewHandler builds a booking Handler.
func NewHandler(service *Service) BookingHandlers {
	return &Handler{service: service}
}

// RegisterRoutes wires every booking endpoint onto mux behind requireAuth,
// so callers only need to know about this one method instead of each
// route/method/middleware combination.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("POST /api/v1/bookings", requireAuth(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/bookings", requireAuth(http.HandlerFunc(h.ListMine)))
	mux.Handle("GET /api/v1/bookings/{id}", requireAuth(http.HandlerFunc(h.GetByID)))
}

type bookingResponse struct {
	ID              string    `json:"id"`
	CustomerID      string    `json:"customer_id"`
	ListingID       string    `json:"listing_id"`
	TripDateID      string    `json:"trip_date_id,omitempty"`
	CheckInDate     time.Time `json:"check_in_date,omitempty"`
	CheckOutDate    time.Time `json:"check_out_date,omitempty"`
	NumberOfGuests  int       `json:"number_of_guests"`
	TotalPrice      float64   `json:"total_price"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	PaymentStatus   string    `json:"payment_status"`
	PaymentID       string    `json:"payment_id,omitempty"`
	SpecialRequests string    `json:"special_requests,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func newBookingResponse(b Booking) bookingResponse {
	return bookingResponse{
		ID:              b.ID,
		CustomerID:      b.CustomerID,
		ListingID:       b.ListingID,
		TripDateID:      b.TripDateID,
		CheckInDate:     b.CheckInDate,
		CheckOutDate:    b.CheckOutDate,
		NumberOfGuests:  b.NumberOfGuests,
		TotalPrice:      b.TotalPrice,
		Currency:        b.Currency,
		Status:          b.Status,
		PaymentStatus:   b.PaymentStatus,
		PaymentID:       b.PaymentID,
		SpecialRequests: b.SpecialRequests,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	}
}

// createRequest mirrors CreateInput but without CustomerID: that comes from
// the authenticated caller, never from the request body.
type createRequest struct {
	ListingID       string    `json:"listing_id" validate:"required,uuid"`
	TripDateID      string    `json:"trip_date_id" validate:"omitempty,uuid"`
	CheckInDate     time.Time `json:"check_in_date" validate:"omitempty"`
	CheckOutDate    time.Time `json:"check_out_date" validate:"omitempty"`
	NumberOfGuests  int       `json:"number_of_guests" validate:"omitempty,gt=0"`
	TotalPrice      float64   `json:"total_price" validate:"required,gt=0"`
	Currency        string    `json:"currency" validate:"omitempty,len=3"`
	PaymentID       string    `json:"payment_id" validate:"omitempty,max=255"`
	SpecialRequests string    `json:"special_requests" validate:"omitempty"`
}

// Create handles POST /api/v1/bookings.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	customerID, _ := auth.UserIDFrom(r.Context())

	created, err := h.service.Create(r.Context(), CreateInput{
		CustomerID:      customerID,
		ListingID:       req.ListingID,
		TripDateID:      req.TripDateID,
		CheckInDate:     req.CheckInDate,
		CheckOutDate:    req.CheckOutDate,
		NumberOfGuests:  req.NumberOfGuests,
		TotalPrice:      req.TotalPrice,
		Currency:        req.Currency,
		PaymentID:       req.PaymentID,
		SpecialRequests: req.SpecialRequests,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newBookingResponse(created))
}

// GetByID handles GET /api/v1/bookings/{id}. A booking that exists but
// belongs to someone else is reported as not found, not forbidden, so this
// endpoint can't be used to probe which IDs exist.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	customerID, _ := auth.UserIDFrom(r.Context())

	b, err := h.service.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}
	if b.CustomerID != customerID {
		httpx.WriteError(r.Context(), w, ErrNotFound)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newBookingResponse(b))
}

// ListMine handles GET /api/v1/bookings.
func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	customerID, _ := auth.UserIDFrom(r.Context())

	bookings, err := h.service.ListByCustomerID(r.Context(), customerID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	responses := make([]bookingResponse, len(bookings))
	for i, b := range bookings {
		responses[i] = newBookingResponse(b)
	}

	httpx.WriteJSON(w, http.StatusOK, responses)
}
