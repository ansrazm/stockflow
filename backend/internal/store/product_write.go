package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrDuplicateSKU = errors.New("duplicate sku")
	ErrHasHistory   = errors.New("product has history")
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

const returningProduct = `RETURNING id, sku, name, price, stock, is_active, created_at,
	(SELECT max(m.created_at) FROM stock_movements m WHERE m.product_id = products.id)`

func scanProduct(row pgx.Row) (Product, error) {
	var p Product
	err := row.Scan(&p.ID, &p.SKU, &p.Name, &p.Price, &p.Stock, &p.IsActive, &p.CreatedAt, &p.LastMovementAt)
	return p, err
}

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func (s *Store) CreateProduct(ctx context.Context, sku, name string, price int) (Product, error) {
	p, err := scanProduct(s.pool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price) VALUES ($1, $2, $3) `+returningProduct,
		sku, name, price))
	if pgCode(err) == pgUniqueViolation {
		return Product{}, ErrDuplicateSKU
	}
	return p, err
}

// isActive nil means "leave unchanged".
func (s *Store) UpdateProduct(ctx context.Context, id int, name string, price int, isActive *bool) (Product, error) {
	p, err := scanProduct(s.pool.QueryRow(ctx,
		`UPDATE products SET name = $2, price = $3, is_active = COALESCE($4, is_active)
		 WHERE id = $1 `+returningProduct,
		id, name, price, isActive))
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

func (s *Store) DeleteProduct(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if pgCode(err) == pgForeignKeyViolation {
		return ErrHasHistory
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}