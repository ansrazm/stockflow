package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ansrazm/stockflow/backend/internal/store"
)

const maxQty = 1_000_000

type stockBody struct {
	Type string `json:"type"`
	Qty  *int   `json:"qty"`
	Note string `json:"note"`
}

func (h *Handler) adjustStock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var b stockBody
	if !decodeJSON(w, r, &b) {
		return
	}

	fields := map[string]string{}
	if b.Type != "in" && b.Type != "out" {
		fields["type"] = `type must be "in" or "out"`
	}
	if b.Qty == nil {
		fields["qty"] = "qty is required"
	} else if *b.Qty < 1 || *b.Qty > maxQty {
		fields["qty"] = "qty must be an integer between 1 and 1000000"
	}
	if len(fields) > 0 {
		validationError(w, fields)
		return
	}

	p, err := h.store.AdjustStock(r.Context(), id, b.Type, *b.Qty, b.Note)
	var ins *store.InsufficientStockError
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", "product not found", nil)
	case errors.As(err, &ins):
		Error(w, http.StatusConflict, "INSUFFICIENT_STOCK",
			fmt.Sprintf("only %d available, but you tried to remove %d", ins.Available, *b.Qty),
			map[string]any{"sku": ins.SKU, "available": ins.Available})
	case err != nil:
		internalError(w, "adjust stock", err)
	default:
		JSON(w, http.StatusOK, p)
	}
}

func (h *Handler) listMovements(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	items, err := h.store.ListMovements(r.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		Error(w, http.StatusNotFound, "NOT_FOUND", "product not found", nil)
	case err != nil:
		internalError(w, "list movements", err)
	default:
		JSON(w, http.StatusOK, map[string]any{"data": items})
	}
}