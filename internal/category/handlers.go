package category

import (
	"net/http"
	"time"

	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes wires every category endpoint onto mux behind requireAuth,
// so callers only need to know about this one method instead of each
// route/method/middleware combination.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth httpx.Middleware) {
	mux.Handle("POST /api/v1/categories", requireAuth(http.HandlerFunc(h.Create)))
	mux.Handle("GET /api/v1/categories", requireAuth(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/categories/slug/{slug}", requireAuth(http.HandlerFunc(h.GetBySlug)))
	mux.Handle("GET /api/v1/categories/{id}", requireAuth(http.HandlerFunc(h.GetByID)))
	mux.Handle("PUT /api/v1/categories/{id}", requireAuth(http.HandlerFunc(h.Update)))
}

type createCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"omitempty"`
	Slug        string `json:"slug" validate:"required,min=2,max=100"`
	Icon        string `json:"icon" validate:"omitempty,max=255"`
}

type updateCategoryRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	Description string `json:"description" validate:"omitempty"`
	Slug        string `json:"slug" validate:"required,min=2,max=100"`
	Icon        string `json:"icon" validate:"omitempty,max=255"`
	IsActive    bool   `json:"is_active"`
}

type categoryResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Slug        string `json:"slug"`
	Icon        string `json:"icon,omitempty"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func newCategoryResponse(c Category) categoryResponse {
	return categoryResponse{
		ID:          c.ID,
		Name:        c.Name,
		Description: c.Description,
		Slug:        c.Slug,
		Icon:        c.Icon,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   c.UpdatedAt.Format(time.RFC3339),
	}
}

// handles POST /api/v1/categories
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	created, err := h.service.Create(r.Context(), CreateInput{
		Name:        req.Name,
		Description: req.Description,
		Slug:        req.Slug,
		Icon:        req.Icon,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, created)
}

// GetByID handles GET /api/v1/categories/{id}. A category that exists but
// belongs to someone else is reported as not found, not forbidden, so this
// endpoint can't be used to probe which IDs exist.
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	categoryID := r.PathValue("id")

	if categoryID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_category_id", "category ID is required"))
		return
	}

	category, err := h.service.GetByID(r.Context(), categoryID)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCategoryResponse(category))
}

// List handles GET /api/v1/categories.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.List(r.Context())
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	var response []categoryResponse
	for _, c := range categories {
		response = append(response, newCategoryResponse(c))
	}

	httpx.WriteJSON(w, http.StatusOK, response)
}

// GetBySlug handles GET /api/v1/categories/slug/{slug}. A category that exists but
// belongs to someone else is reported as not found, not forbidden, so this
// endpoint can't be used to probe which slugs exist.
func (h *Handler) GetBySlug(w http.ResponseWriter, r *http.Request) {

	categorySlug := r.PathValue("slug")

	if categorySlug == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_category_slug", "category slug is required"))
		return
	}

	category, err := h.service.GetBySlug(r.Context(), categorySlug)
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCategoryResponse(category))
}

// update handles PUT /api/v1/categories/{id}.
func (g *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req updateCategoryRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	categoryID := r.PathValue("id")

	if categoryID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError(http.StatusBadRequest, "invalid_category_id", "category ID is required"))
		return
	}

	updated, err := g.service.Update(r.Context(), UpdateInput{
		ID:          categoryID,
		Name:        req.Name,
		Description: req.Description,
		Slug:        req.Slug,
		Icon:        req.Icon,
		IsActive:    req.IsActive,
	})
	if err != nil {
		httpx.WriteError(r.Context(), w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newCategoryResponse(updated))
}
