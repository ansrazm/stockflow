package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansrazm/stockflow/backend/internal/store"
)

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	o, err := h.store.GetOrder(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", "order not found", nil)
	case err != nil:
		internalError(w, "get order", err)
	default:
		JSON(w, http.StatusOK, o)
	}
}

func (h *Handler) cancelOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	o, err := h.store.CancelOrder(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", "order not found", nil)
	case errors.Is(err, store.ErrAlreadyCancelled):
		Error(w, http.StatusConflict, "ALREADY_CANCELLED", "order is already cancelled", nil)
	case err != nil:
		internalError(w, "cancel order", err)
	default:
		JSON(w, http.StatusOK, o)
	}
}
