package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ansrazm/stockflow/backend/internal/store"
)

type Handler struct {
	store *store.Store
}

func NewHandler(s *store.Store) *Handler {
	return &Handler{store: s}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/products", h.listProducts)
	r.Post("/products", h.createProduct)
	r.Put("/products/{id}", h.updateProduct)
	r.Delete("/products/{id}", h.deleteProduct)
	r.Post("/products/{id}/stock", h.adjustStock)
	r.Get("/products/{id}/movements", h.listMovements)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	page, ok := intParam(q.Get("page"), 1)
	if !ok || page < 1 {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "page must be a positive integer", nil)
		return
	}
	limit, ok := intParam(q.Get("limit"), 20)
	if !ok || limit < 1 || limit > 100 {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "limit must be between 1 and 100", nil)
		return
	}
	status := q.Get("status")
	switch status {
	case "", "all":
		status = "all"
	case "active", "inactive", "low":
	default:
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "status must be all, active, inactive or low", nil)
		return
	}

	items, total, err := h.store.ListProducts(r.Context(), store.ProductFilter{
		Search: q.Get("search"),
		Status: status,
		Page:   page,
		Limit:  limit,
	})
	if err != nil {
		Error(w, http.StatusInternalServerError, "INTERNAL", "internal error", nil)
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"data": items, "page": page, "limit": limit, "total": total,
	})
}

// intParam returns def when s is empty, and ok=false when s is not an integer.
func intParam(s string, def int) (int, bool) {
	if s == "" {
		return def, true
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}