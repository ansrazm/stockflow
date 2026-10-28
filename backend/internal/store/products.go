package store

import (
	"context"
	"fmt"
	"time"
)

const LowStockThreshold = 10

type Product struct {
	ID             int        `json:"id"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Price          int        `json:"price"`
	Stock          int        `json:"stock"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	LastMovementAt *time.Time `json:"last_movement_at"`
}

type ProductFilter struct {
	Search string
	Status string // all | active | inactive | low
	Page   int
	Limit  int
}

func (s *Store) ListProducts(ctx context.Context, f ProductFilter) ([]Product, int, error) {
	where := "WHERE 1=1"
	args := []any{}

	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		where += fmt.Sprintf(" AND (p.sku ILIKE $%d OR p.name ILIKE $%d)", len(args), len(args))
	}
	switch f.Status {
	case "active":
		where += " AND p.is_active"
	case "inactive":
		where += " AND NOT p.is_active"
	case "low":
		where += fmt.Sprintf(" AND p.stock <= %d", LowStockThreshold)
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM products p "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, (f.Page-1)*f.Limit)
	query := fmt.Sprintf(`
		SELECT p.id, p.sku, p.name, p.price, p.stock, p.is_active, p.created_at,
		       (SELECT max(m.created_at) FROM stock_movements m WHERE m.product_id = p.id)
		FROM products p
		%s
		ORDER BY p.id
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := []Product{} // empty slice, not nil, so JSON is [] and not null
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.LastMovementAt); err != nil {
			return nil, 0, err
		}
		items = append(items, p)
	}
	return items, total, rows.Err()
}