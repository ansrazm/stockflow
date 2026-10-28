package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func testStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://stockflow:stockflow@localhost:5433/stockflow"
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("database not reachable (run docker compose up): %v", err)
	}
	t.Cleanup(pool.Close)
	return New(pool), pool
}

// seedProduct inserts a throwaway product and removes it (and anything
// that references it) when the test ends.
func seedProduct(t *testing.T, pool *pgxpool.Pool, prefix string, price, stock int) (id int, sku string) {
	t.Helper()
	ctx := context.Background()
	sku = fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	err := pool.QueryRow(ctx,
		`INSERT INTO products (sku, name, price, stock) VALUES ($1, $1, $2, $3) RETURNING id`,
		sku, price, stock).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		pool.Exec(ctx, `DELETE FROM sales_order_items WHERE order_id IN
			(SELECT order_id FROM sales_order_items WHERE sku = $1)`, sku)
		pool.Exec(ctx, `DELETE FROM sales_orders WHERE id NOT IN
			(SELECT order_id FROM sales_order_items)
			AND customer LIKE 'TST-%'`)
		pool.Exec(ctx, `DELETE FROM stock_movements WHERE product_id = $1`, id)
		pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	})
	return id, sku
}

func count(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// (a) insufficient stock is rejected and NOTHING is written
func TestCreateOrder_InsufficientStockWritesNothing(t *testing.T) {
	s, pool := testStore(t)
	idA, skuA := seedProduct(t, pool, "TST-A", 10000, 5)
	_, skuB := seedProduct(t, pool, "TST-B", 2500, 0) // out of stock

	_, err := s.CreateOrder(context.Background(), "TST-fail", []OrderItemInput{
		{SKU: skuA, Qty: 1},
		{SKU: skuB, Qty: 1},
	})

	var ins *InsufficientStockError
	if !errors.As(err, &ins) {
		t.Fatalf("want InsufficientStockError, got %v", err)
	}
	if ins.SKU != skuB || ins.Available != 0 {
		t.Errorf("got sku=%s available=%d, want sku=%s available=0", ins.SKU, ins.Available, skuB)
	}
	if got := count(t, pool, `SELECT stock FROM products WHERE id = $1`, idA); got != 5 {
		t.Errorf("stock of first item changed: got %d, want 5 (rollback failed)", got)
	}
	if got := count(t, pool, `SELECT count(*) FROM sales_order_items WHERE sku = ANY($1)`, []string{skuA, skuB}); got != 0 {
		t.Errorf("order items were written: %d rows", got)
	}
	if got := count(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1`, idA); got != 0 {
		t.Errorf("movement was written: %d rows", got)
	}
}

// (b) happy path writes the order, updates stock and records movements
func TestCreateOrder_HappyPath(t *testing.T) {
	s, pool := testStore(t)
	idA, skuA := seedProduct(t, pool, "TST-A", 10000, 5)
	idB, skuB := seedProduct(t, pool, "TST-B", 2500, 3)

	o, err := s.CreateOrder(context.Background(), "TST-ok", []OrderItemInput{
		{SKU: skuA, Qty: 2},
		{SKU: skuB, Qty: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	if o.Status != "CREATED" {
		t.Errorf("status = %s, want CREATED", o.Status)
	}
	if want := 2*10000 + 1*2500; o.Total != want {
		t.Errorf("total = %d, want %d (server-computed)", o.Total, want)
	}
	if len(o.Items) != 2 || o.Items[0].UnitPrice != 10000 {
		t.Errorf("items wrong: %+v", o.Items)
	}
	if got := count(t, pool, `SELECT stock FROM products WHERE id = $1`, idA); got != 3 {
		t.Errorf("stock A = %d, want 3", got)
	}
	if got := count(t, pool, `SELECT stock FROM products WHERE id = $1`, idB); got != 2 {
		t.Errorf("stock B = %d, want 2", got)
	}
	if got := count(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1 AND type = 'out'`, idA); got != 1 {
		t.Errorf("movements for A = %d, want 1", got)
	}
}
