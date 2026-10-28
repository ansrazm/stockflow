package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var ErrAlreadyCancelled = errors.New("order already cancelled")

// satisfied by both *pgxpool.Pool and pgx.Tx
type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func getOrder(ctx context.Context, q queryer, id int) (Order, error) {
	var o Order
	err := q.QueryRow(ctx,
		`SELECT id, customer, status, total, created_at, cancelled_at
		 FROM sales_orders WHERE id = $1`, id).
		Scan(&o.ID, &o.Customer, &o.Status, &o.Total, &o.CreatedAt, &o.CancelledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}

	rows, err := q.Query(ctx,
		`SELECT sku, qty, unit_price FROM sales_order_items WHERE order_id = $1 ORDER BY id`, id)
	if err != nil {
		return Order{}, err
	}
	defer rows.Close()

	o.Items = []OrderItem{}
	for rows.Next() {
		var it OrderItem
		if err := rows.Scan(&it.SKU, &it.Qty, &it.UnitPrice); err != nil {
			return Order{}, err
		}
		o.Items = append(o.Items, it)
	}
	return o, rows.Err()
}

func (s *Store) GetOrder(ctx context.Context, id int) (Order, error) {
	return getOrder(ctx, s.pool, id)
}

func (s *Store) CancelOrder(ctx context.Context, id int) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	// the guard: only one request can flip CREATED -> CANCELLED
	tag, err := tx.Exec(ctx,
		`UPDATE sales_orders SET status = 'CANCELLED', cancelled_at = now()
		 WHERE id = $1 AND status = 'CREATED'`, id)
	if err != nil {
		return Order{}, err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM sales_orders WHERE id = $1)`, id).Scan(&exists); err != nil {
			return Order{}, err
		}
		if !exists {
			return Order{}, ErrNotFound
		}
		return Order{}, ErrAlreadyCancelled
	}

	// read items in sku order (same lock order as CreateOrder)
	rows, err := tx.Query(ctx,
		`SELECT sku, qty FROM sales_order_items WHERE order_id = $1 ORDER BY sku`, id)
	if err != nil {
		return Order{}, err
	}
	type line struct {
		sku string
		qty int
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.sku, &l.qty); err != nil {
			rows.Close()
			return Order{}, err
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Order{}, err
	}

	for _, l := range lines {
		var productID int
		if err := tx.QueryRow(ctx,
			`UPDATE products SET stock = stock + $2 WHERE sku = $1 RETURNING id`,
			l.sku, l.qty).Scan(&productID); err != nil {
			return Order{}, err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_movements (product_id, type, qty, note) VALUES ($1, 'in', $2, $3)`,
			productID, l.qty, fmt.Sprintf("Cancel order #%d", id)); err != nil {
			return Order{}, err
		}
	}

	o, err := getOrder(ctx, tx, id)
	if err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return o, nil
}
