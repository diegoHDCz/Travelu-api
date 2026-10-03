package auth

import (
	"net/http"
	"time"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
	"github.com/diegoczajka/travelu-api/internal/user"
)

// Handler exposes the auth HTTP endpoints, including GET /api/v1/users/me:
// that route lives here (not in package user) because it needs the
// authenticated user ID set by RequireAuth, and user must not depend on
// auth (auth already depends on user for UserService).
type Handler struct {
	service *Service
}

// NewHandler builds an auth Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type userResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Phone     string    `json:"phone,omitempty"`
	Username  string    `json:"username,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func newUserResponse(u user.User) userResponse {
	return userResponse{ID: u.ID, Name: u.Name, Email: u.Email, Phone: u.Phone, Username: u.Username, CreatedAt: u.CreatedAt}
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input user.CreateInput
	if err := httpx.DecodeAndValidate(r, &input); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	created, err := h.service.Register(r.Context(), input)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, newUserResponse(created))
}

// loginRequest.Login accepts an email, a phone number or a username — the
// service decides which based on its shape. It is intentionally not
// validated beyond "required": a format-specific check here (e.g. "email")
// would reject the phone/username cases.
// loginRequest.DeviceToken is the push-notification token of the device the
// client is authenticating from. It is optional; when present, it is
// embedded in the access token's device_token claim.
type loginRequest struct {
	Login       string `json:"login" validate:"required"`
	Password    string `json:"password" validate:"required"`
	DeviceToken string `json:"device_token" validate:"omitempty,max=512"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func newTokenResponse(pair TokenPair) tokenResponse {
	return tokenResponse{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	pair, err := h.service.Login(r.Context(), req.Login, req.Password, req.DeviceToken)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newTokenResponse(pair))
}

// refreshRequest.DeviceToken behaves like loginRequest.DeviceToken: it is
// re-embedded in the rotated access token's device_token claim.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	DeviceToken  string `json:"device_token" validate:"omitempty,max=512"`
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	pair, err := h.service.Refresh(r.Context(), req.RefreshToken, req.DeviceToken)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newTokenResponse(pair))
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	if err := h.service.Logout(r.Context(), req.RefreshToken); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusNoContent, nil)
}

// Me handles GET /api/v1/users/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFrom(r.Context())
	if !ok {
		httpx.WriteError(r.Context(), w, ErrUnauthorized)
		return
	}

	u, err := h.service.users.GetByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newUserResponse(u))
}
