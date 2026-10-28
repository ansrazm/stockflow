package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type InsufficientStockError struct {
	SKU       string
	Available int
}

func (e *InsufficientStockError) Error() string {
	return fmt.Sprintf("insufficient stock for %s: %d available", e.SKU, e.Available)
}

type Movement struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	Type      string    `json:"type"`
	Qty       int       `json:"qty"`
	Note      *string   `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) GetProduct(ctx context.Context, id int) (Product, error) {
	p, err := scanProduct(s.pool.QueryRow(ctx, `
		SELECT p.id, p.sku, p.name, p.price, p.stock, p.is_active, p.created_at,
		       (SELECT max(m.created_at) FROM stock_movements m WHERE m.product_id = p.id)
		FROM products p WHERE p.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

// AdjustStock changes stock and records the movement in ONE transaction.
func (s *Store) AdjustStock(ctx context.Context, id int, typ string, qty int, note string) (Product, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Product{}, err
	}
	defer tx.Rollback(ctx) // no-op after a successful Commit

	query := `UPDATE products SET stock = stock + $2 WHERE id = $1`
	if typ == "out" {
		// the guard lives inside the UPDATE: check and decrement are one atomic step
		query = `UPDATE products SET stock = stock - $2 WHERE id = $1 AND stock >= $2`
	}
	tag, err := tx.Exec(ctx, query, id, qty)
	if err != nil {
		return Product{}, err
	}
	if tag.RowsAffected() == 0 {
		var sku string
		var available int
		err := tx.QueryRow(ctx, `SELECT sku, stock FROM products WHERE id = $1`, id).Scan(&sku, &available)
		if errors.Is(err, pgx.ErrNoRows) {
			return Product{}, ErrNotFound
		}
		if err != nil {
			return Product{}, err
		}
		return Product{}, &InsufficientStockError{SKU: sku, Available: available}
	}

	var noteArg *string
	if n := strings.TrimSpace(note); n != "" {
		noteArg = &n
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO stock_movements (product_id, type, qty, note) VALUES ($1, $2, $3, $4)`,
		id, typ, qty, noteArg); err != nil {
		return Product{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Product{}, err
	}
	return s.GetProduct(ctx, id)
}

func (s *Store) ListMovements(ctx context.Context, productID int) ([]Movement, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM products WHERE id = $1)`, productID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, product_id, type, qty, note, created_at
		FROM stock_movements
		WHERE product_id = $1
		ORDER BY created_at DESC, id DESC`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Movement{}
	for rows.Next() {
		var m Movement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.Type, &m.Qty, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}