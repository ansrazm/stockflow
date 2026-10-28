package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ansrazm/stockflow/backend/internal/store"
)

// Pointers tell "missing" apart from zero values (price 0 is valid, absent is not).
type productBody struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Price    *int   `json:"price"`
	IsActive *bool  `json:"is_active"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_JSON",
			"body must be valid JSON with the expected fields (price must be an integer)", nil)
		return false
	}
	return true
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil || id < 1 {
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "id must be a positive integer", nil)
		return 0, false
	}
	return id, true
}

func validationError(w http.ResponseWriter, fields map[string]string) {
	Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "validation failed",
		map[string]any{"fields": fields})
}

func internalError(w http.ResponseWriter, op string, err error) {
	log.Printf("%s: %v", op, err)
	Error(w, http.StatusInternalServerError, "INTERNAL", "internal error", nil)
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var b productBody
	if !decodeJSON(w, r, &b) {
		return
	}
	sku, name := strings.TrimSpace(b.SKU), strings.TrimSpace(b.Name)

	fields := map[string]string{}
	if sku == "" {
		fields["sku"] = "sku is required"
	}
	if name == "" {
		fields["name"] = "name is required"
	}
	if b.Price == nil {
		fields["price"] = "price is required"
	} else if *b.Price < 0 {
		fields["price"] = "price must be >= 0"
	}
	if len(fields) > 0 {
		validationError(w, fields)
		return
	}

	p, err := h.store.CreateProduct(r.Context(), sku, name, *b.Price)
	switch {
	case errors.Is(err, store.ErrDuplicateSKU):
		Error(w, http.StatusConflict, "SKU_ALREADY_EXISTS",
			"a product with this sku already exists", map[string]any{"sku": sku})
	case err != nil:
		internalError(w, "create product", err)
	default:
		JSON(w, http.StatusCreated, p)
	}
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var b productBody
	if !decodeJSON(w, r, &b) {
		return
	}
	name := strings.TrimSpace(b.Name)

	fields := map[string]string{}
	if name == "" {
		fields["name"] = "name is required"
	}
	if b.Price == nil {
		fields["price"] = "price is required"
	} else if *b.Price < 0 {
		fields["price"] = "price must be >= 0"
	}
	if len(fields) > 0 {
		validationError(w, fields)
		return
	}

	p, err := h.store.UpdateProduct(r.Context(), id, name, *b.Price, b.IsActive)
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", "product not found", nil)
	case err != nil:
		internalError(w, "update product", err)
	default:
		JSON(w, http.StatusOK, p)
	}
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	err := h.store.DeleteProduct(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", "product not found", nil)
	case errors.Is(err, store.ErrHasHistory):
		Error(w, http.StatusConflict, "PRODUCT_HAS_HISTORY",
			"product has movement history and cannot be deleted", nil)
	case err != nil:
		internalError(w, "delete product", err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}