package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

// (c) 20 buyers race for 5 units: exactly 5 win, stock ends at 0, never negative.
func TestCreateOrder_ConcurrentNeverOversells(t *testing.T) {
	s, pool := testStore(t)
	const stock, buyers = 5, 20
	id, sku := seedProduct(t, pool, "TST-RACE", 1000, stock)

	start := make(chan struct{})
	results := make(chan error, buyers)
	var wg sync.WaitGroup
	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // release everyone at once
			_, err := s.CreateOrder(context.Background(), fmt.Sprintf("TST-race-%d", i),
				[]OrderItemInput{{SKU: sku, Qty: 1}})
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	ok, rejected := 0, 0
	for err := range results {
		var ins *InsufficientStockError
		switch {
		case err == nil:
			ok++
		case errors.As(err, &ins):
			rejected++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != stock || rejected != buyers-stock {
		t.Errorf("ok=%d rejected=%d, want ok=%d rejected=%d", ok, rejected, stock, buyers-stock)
	}
	if got := count(t, pool, `SELECT stock FROM products WHERE id = $1`, id); got != 0 {
		t.Errorf("final stock = %d, want 0", got)
	}
	if got := count(t, pool, `SELECT count(*) FROM sales_order_items WHERE sku = $1`, sku); got != stock {
		t.Errorf("order items = %d, want %d", got, stock)
	}
	if got := count(t, pool, `SELECT coalesce(sum(qty),0) FROM stock_movements WHERE product_id = $1 AND type = 'out'`, id); got != stock {
		t.Errorf("out movements = %d, want %d", got, stock)
	}
}

// Orders listing the same SKUs in opposite order must not deadlock.
func TestCreateOrder_OppositeSkuOrderNoDeadlock(t *testing.T) {
	s, pool := testStore(t)
	idA, skuA := seedProduct(t, pool, "TST-DLA", 1000, 50)
	idB, skuB := seedProduct(t, pool, "TST-DLB", 2000, 50)

	const orders = 10
	var wg sync.WaitGroup
	errs := make(chan error, orders)
	for i := 0; i < orders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			items := []OrderItemInput{{SKU: skuA, Qty: 1}, {SKU: skuB, Qty: 1}}
			if i%2 == 1 {
				items[0], items[1] = items[1], items[0]
			}
			_, err := s.CreateOrder(context.Background(), fmt.Sprintf("TST-dl-%d", i), items)
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("order failed (deadlock?): %v", err)
		}
	}
	for _, id := range []int{idA, idB} {
		if got := count(t, pool, `SELECT stock FROM products WHERE id = $1`, id); got != 50-orders {
			t.Errorf("stock = %d, want %d", got, 50-orders)
		}
	}
}

// Concurrent cancels: exactly one wins, stock is restored exactly once.
func TestCancelOrder_ConcurrentRestoresOnce(t *testing.T) {
	s, pool := testStore(t)
	id, sku := seedProduct(t, pool, "TST-CXL", 1000, 5)

	o, err := s.CreateOrder(context.Background(), "TST-cancel", []OrderItemInput{{SKU: sku, Qty: 2}})
	if err != nil {
		t.Fatal(err)
	}

	const attempts = 10
	var wg sync.WaitGroup
	errs := make(chan error, attempts)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CancelOrder(context.Background(), o.ID)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	won, already := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			won++
		case errors.Is(err, ErrAlreadyCancelled):
			already++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if won != 1 || already != attempts-1 {
		t.Errorf("won=%d already=%d, want 1 and %d", won, already, attempts-1)
	}
	if got := count(t, pool, `SELECT stock FROM products WHERE id = $1`, id); got != 5 {
		t.Errorf("stock after cancel = %d, want 5 (restored exactly once)", got)
	}
	if got := count(t, pool, `SELECT count(*) FROM stock_movements WHERE product_id = $1 AND type = 'in'`, id); got != 1 {
		t.Errorf("restore movements = %d, want 1", got)
	}
}
