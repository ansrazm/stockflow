package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

type OrderItemInput struct {
	SKU string
	Qty int
}

type OrderItem struct {
	SKU       string `json:"sku"`
	Qty       int    `json:"qty"`
	UnitPrice int    `json:"unit_price"`
}

type Order struct {
	ID          int         `json:"id"`
	Customer    string      `json:"customer"`
	Status      string      `json:"status"`
	Total       int         `json:"total"`
	CreatedAt   time.Time   `json:"created_at"`
	CancelledAt *time.Time  `json:"cancelled_at"`
	Items       []OrderItem `json:"items"`
}

type UnknownSKUError struct{ SKU string }

func (e *UnknownSKUError) Error() string { return "unknown sku: " + e.SKU }

var ErrOrderTooLarge = errors.New("order total too large")

func (s *Store) CreateOrder(ctx context.Context, customer string, items []OrderItemInput) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx) // undoes every stock decrement unless we reach Commit

	// 1. every sku must exist (still before any write)
	skus := make([]string, len(items))
	for i, it := range items {
		skus[i] = it.SKU
	}
	rows, err := tx.Query(ctx, `SELECT sku FROM products WHERE sku = ANY($1)`, skus)
	if err != nil {
		return Order{}, err
	}
	known := map[string]bool{}
	for rows.Next() {
		var sku string
		if err := rows.Scan(&sku); err != nil {
			rows.Close()
			return Order{}, err
		}
		known[sku] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Order{}, err
	}
	for _, it := range items {
		if !known[it.SKU] {
			return Order{}, &UnknownSKUError{SKU: it.SKU}
		}
	}

	// 2. decrement stock, always in sku order (prevents deadlocks between orders)
	idx := make([]int, len(items))
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool { return items[idx[a]].SKU < items[idx[b]].SKU })

	type line struct{ productID, price int }
	lines := make([]line, len(items))
	var total int64
	for _, i := range idx {
		it := items[i]
		var l line
		err := tx.QueryRow(ctx,
			`UPDATE products SET stock = stock - $2
			 WHERE sku = $1 AND stock >= $2
			 RETURNING id, price`, it.SKU, it.Qty).Scan(&l.productID, &l.price)
		if errors.Is(err, pgx.ErrNoRows) {
			var available int
			if err := tx.QueryRow(ctx, `SELECT stock FROM products WHERE sku = $1`, it.SKU).Scan(&available); err != nil {
				return Order{}, err
			}
			return Order{}, &InsufficientStockError{SKU: it.SKU, Available: available}
		}
		if err != nil {
			return Order{}, err
		}
		lines[i] = l
		total += int64(l.price) * int64(it.Qty)
	}
	if total > math.MaxInt32 {
		return Order{}, ErrOrderTooLarge
	}

	// 3. the order itself, then its items and movements
	var o Order
	err = tx.QueryRow(ctx,
		`INSERT INTO sales_orders (customer, total) VALUES ($1, $2)
		 RETURNING id, customer, status, total, created_at, cancelled_at`,
		customer, int(total)).Scan(&o.ID, &o.Customer, &o.Status, &o.Total, &o.CreatedAt, &o.CancelledAt)
	if err != nil {
		return Order{}, err
	}

	o.Items = make([]OrderItem, 0, len(items))
	for i, it := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO sales_order_items (order_id, sku, qty, unit_price) VALUES ($1, $2, $3, $4)`,
			o.ID, it.SKU, it.Qty, lines[i].price); err != nil {
			return Order{}, err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_movements (product_id, type, qty, note) VALUES ($1, 'out', $2, $3)`,
			lines[i].productID, it.Qty, fmt.Sprintf("Order #%d", o.ID)); err != nil {
			return Order{}, err
		}
		o.Items = append(o.Items, OrderItem{SKU: it.SKU, Qty: it.Qty, UnitPrice: lines[i].price})
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return o, nil
}