package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ansrazm/stockflow/backend/internal/store"
)

type orderItemBody struct {
	SKU string `json:"sku"`
	Qty *int   `json:"qty"`
}

type orderBody struct {
	Customer string          `json:"customer"`
	Items    []orderItemBody `json:"items"`
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var b orderBody
	if !decodeJSON(w, r, &b) {
		return
	}

	fields := map[string]string{}
	customer := strings.TrimSpace(b.Customer)
	if customer == "" {
		fields["customer"] = "customer is required"
	}
	if len(b.Items) == 0 {
		fields["items"] = "at least one item is required"
	}

	items := make([]store.OrderItemInput, 0, len(b.Items))
	seen := map[string]bool{}
	for i, it := range b.Items {
		sku := strings.TrimSpace(it.SKU)
		switch {
		case sku == "":
			fields[fmt.Sprintf("items[%d].sku", i)] = "sku is required"
		case seen[sku]:
			fields[fmt.Sprintf("items[%d].sku", i)] = "duplicate sku in order: " + sku
		default:
			seen[sku] = true
		}

		qty := 0
		if it.Qty == nil {
			fields[fmt.Sprintf("items[%d].qty", i)] = "qty is required"
		} else if *it.Qty < 1 || *it.Qty > maxQty {
			fields[fmt.Sprintf("items[%d].qty", i)] = "qty must be an integer between 1 and 1000000"
		} else {
			qty = *it.Qty
		}
		items = append(items, store.OrderItemInput{SKU: sku, Qty: qty})
	}
	if len(fields) > 0 {
		validationError(w, fields)
		return
	}

	o, err := h.store.CreateOrder(r.Context(), customer, items)
	var ins *store.InsufficientStockError
	var unk *store.UnknownSKUError
	switch {
	case errors.As(err, &unk):
		Error(w, http.StatusBadRequest, "UNKNOWN_SKU",
			"unknown sku: "+unk.SKU, map[string]any{"sku": unk.SKU})
	case errors.As(err, &ins):
		Error(w, http.StatusConflict, "INSUFFICIENT_STOCK",
			fmt.Sprintf("insufficient stock for %s: only %d available", ins.SKU, ins.Available),
			map[string]any{"sku": ins.SKU, "available": ins.Available})
	case errors.Is(err, store.ErrOrderTooLarge):
		Error(w, http.StatusBadRequest, "VALIDATION_ERROR", "order total is too large", nil)
	case err != nil:
		internalError(w, "create order", err)
	default:
		JSON(w, http.StatusCreated, o)
	}
}