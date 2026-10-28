# StockFlow — Inventory & Sales (mini ERP module)

Go + PostgreSQL backend, React + TypeScript (Vite) frontend. A warehouse supervisor can manage products, adjust stock, and record sales orders that consume stock.

```
stockflow/
├── backend/             # Go service (cmd/api, internal/httpapi, internal/store, migrations/)
├── frontend/            # React + TS (Vite, strict mode)
├── docker-compose.yml   # postgres 18 with migrations + seed applied on first start
└── README.md
```

## What is and is not in this submission

| Area | Status |
|---|---|
| All must-have API endpoints (products, stock, movements, orders, cancel) | Done |
| Atomic order creation, 409 with `sku` + `available`, concurrency-safe | Done |
| Frontend: products page, adjust modal, create order, order detail + cancel, summary cards | Done |
| Backend tests (insufficient stock writes nothing, happy path, `-race` concurrency tests) | Done |
| Frontend unit tests (Vitest), `useDebounce` hook | Done |
| `docker compose` runs **db only** (backend and frontend run on the host) | Partial: the 2-command route below |
| Idempotency-Key on `POST /api/orders` | Not done |
| `GET /api/reports/daily-sales` + summary card | Not done |
| Skeleton loading rows | Not done (plain loading text) |

## Run it (3 terminals, about 5 minutes)

Prerequisites: Docker, Go, Node.js 20+.

```bash
# 1. Database (postgres on host port 5433; migrations + seed run on first start)
docker compose up -d

# 2. Backend (listens on :8080; default DSN points at the compose database)
cd backend
go run ./cmd/api

# 3. Frontend (http://localhost:5173)
cd frontend
npm install
npm run dev
```

Override the database with `DATABASE_URL` (default: `postgres://stockflow:stockflow@localhost:5433/stockflow`).

Reset to a clean seeded database: `docker compose down -v && docker compose up -d`. The SQL in `backend/migrations/` only runs when the volume is empty, which is why `-v` is needed.

Dev proxy: Vite proxies `/api` to `http://localhost:8080` (see `frontend/vite.config.ts`). I chose the proxy over CORS middleware so the backend has no CORS surface at all and the browser only ever talks to one origin. As a side effect, when the backend is down the UI shows "Bad Gateway", which is the Vite proxy's response, not the Go service's.

### Tests

```bash
# Backend. Needs the compose database running; tests skip themselves if the DB is unreachable.
cd backend
go test ./... -race

# Frontend
cd frontend
npm test
```

## API examples (curl)

Errors always use one envelope: `{"error":{"code":"...","message":"...", ...extra fields}}`.

```bash
# List / search (case-insensitive on sku or name); status filter: all|active|inactive|low
curl 'localhost:8080/api/products?search=kbd&status=active&page=1&limit=20'

# Create product (starts with stock 0; use the stock endpoint below to add stock)
curl -X POST localhost:8080/api/products -H 'Content-Type: application/json' \
  -d '{"sku":"CBL-009","name":"USB-C Cable 1m","price":45000}'

# Adjust stock (id 3 as an example). Stock out beyond available returns 409:
# {"error":{"code":"INSUFFICIENT_STOCK","available":18,...}}
curl -X POST localhost:8080/api/products/3/stock -H 'Content-Type: application/json' \
  -d '{"type":"out","qty":20,"note":"cycle count"}'

# Movement history, newest first
curl localhost:8080/api/products/3/movements

# Create an order. unit_price and total come from the server.
curl -X POST localhost:8080/api/orders -H 'Content-Type: application/json' \
  -d '{"customer":"john","items":[{"sku":"KBD-001","qty":2}]}'
# -> 201 with items, status CREATED and the computed total
# -> 409 {"error":{"code":"INSUFFICIENT_STOCK","sku":"RAM-001","available":0,...}}, nothing written
# -> 400 validation errors per field (empty customer, qty <= 0, duplicate sku, unknown sku, empty items)

# Order detail and cancel (second cancel returns 409)
curl localhost:8080/api/orders/21
curl -X POST localhost:8080/api/orders/21/cancel
```

## Design decisions

**Where the stock guard lives.** Never SELECT-then-UPDATE. Both paths put the guard in the UPDATE itself:

```sql
UPDATE products SET stock = stock - $2 WHERE sku = $1 AND stock >= $2 RETURNING id, price
```

If it matches zero rows, the request is rejected. Under concurrency Postgres serializes writers on the row, so the second writer re-evaluates `stock >= qty` against the committed value. The `CHECK (stock >= 0)` constraint stays as a second line of defence.

**One transaction per operation.** Order creation decrements every item, inserts the order, its items, and one `out` movement per item inside one transaction. Any failure rolls back everything, including stock already decremented for earlier lines, so "insufficient stock for one item" leaves nothing changed. The same applies to stock adjustment (stock change + movement row).

**Deadlock avoidance.** Items are decremented in SKU order, and cancel restores in SKU order too, so two concurrent orders with opposite item order cannot lock each other in a cycle.

**Cancel is idempotent-safe.** The status flip `UPDATE ... WHERE id = $1 AND status = 'CREATED'` is the guard. Only one concurrent cancel can win; the others get 409. Stock is restored (with `in` movements noted "Cancel order #N") only by the winner.

**Money.** Integer IDR everywhere, no floats. `unit_price` is copied from `products` at order time; the client never sends prices. The total is computed in `int64` and rejected with 400 if it would overflow the `INTEGER` column.

**Validation split.** The API validates everything again (the frontend checks are for UX only). Duplicate SKUs in one order are a 400, not merged, so the caller sees the mistake.

**`last_movement_at`.** Derived with a subquery on `stock_movements` (`max(created_at)`), using an index on `(product_id, created_at DESC)`, rather than denormalized onto the product row. One less thing to keep consistent.

**Frontend.** One API client module owns every `fetch` and error-envelope parsing; components never build URLs. Domain types live in one module, `strict` TypeScript, no `any`. Search is debounced with a `useDebounce` hook. The products page has explicit loading, empty, and error-with-retry states. The backend's 409 message (with the available quantity) is rendered inside the modal / order form, not only logged to the console.

## Test evidence

Automated:

- `internal/store/orders_test.go`: insufficient stock rejects the order and writes nothing; happy path writes order, items, movements, and decrements stock.
- `internal/store/orders_concurrency_test.go` (run with `-race`): 20 concurrent buyers against stock of 5 give exactly 5 successes, 15 rejections, final stock 0; opposite SKU order does not deadlock; concurrent cancels restore stock exactly once.
- `frontend/src/lib/*.test.ts` (Vitest): validation, order form (including duplicate SKU), summary, formatting.

Manual UI run-through (screenshots kept locally; seed data was adjusted during testing, so stock values in screenshots differ from a fresh seed):

| Check | Result |
|---|---|
| Products table, badges, last movement, summary cards | OK |
| Search `kbd` and a no-match query (empty state) | OK |
| Backend stopped: error message + Retry; table recovers after restart | OK |
| Adjust: Save disabled when empty; qty 0 shows "Quantity must be greater than 0" | OK |
| Adjust: Stock out 20 on MON-001 (stock 18) shows "Insufficient stock: only 18 available." in the modal; stock unchanged | OK |
| Order with RAM-001 (stock 0) shows "Insufficient stock for RAM-001: only 0 available." on that line | OK |
| Order with empty customer / empty SKU shows per-field messages | OK |
| Order #21 (KBD-001 x 2): total Rp 1.300.000, stock 25 to 23, summary Rp 130.725.000 | OK |
| Cancel #21: confirmation, stock restored, status CANCELLED, summary back to Rp 132.025.000 | OK |

Not manually screenshotted: duplicate SKU in the order form (covered by the Vitest case and server-side 400). I have not verified the summary against a direct `psql` `SUM(price*stock)` query.

## Known limitations / what I would improve next

1. **Summary cards and filters are computed client-side** from the first 100 products (the API caps `limit` at 100). Fine for the seed data, wrong beyond 100 products. Next: a server-side `/api/products/summary` and real pagination in the UI (the API already supports `page`/`limit`).
2. **Inactive products** appear in the order SKU dropdown (for example the discontinued DSK-001), and `POST /api/orders` does not reject inactive products. I did not get to deciding the business rule; I would reject them with a clear 400 and hide them in the dropdown.
3. **No Idempotency-Key** on order creation: a retried request after a network timeout can create two orders.
4. **No daily sales report** and no end-to-end `docker compose` for backend and frontend (no Dockerfiles).
5. **Loading state is text**, not skeleton rows.
6. The 409 handling for adjust-stock and orders is tested manually in the UI, but there is no automated frontend test of the error rendering (Vitest covers pure logic only).

## Honest hours

Wall clock: started at 07:00 when I received the test, last commit at 19:41, then manual UI testing and this README, roughly **13 hours elapsed**. Focused engineering time was about **6 to 6.5 hours**. The rest was not coding. Breakdown (approximate, reconstructed from git timestamps and memory):

| Activity | Hours |
|---|---|
| Reading the brief, breakfast, planning | ~2 |
| Toolchain setup from scratch (Go, Postgres, VS Code, Vite scaffold) and schema design | ~2.5 |
| Backend: API, order/stock transactions, cancel, store tests | ~2.5 |
| Frontend: API client, products page, adjust modal, order flow | ~2.5 |
| Tests (Vitest, Go concurrency) and manual UI testing | ~1.5 |
| Environment repair (OS update for Docker, Docker/Postgres issues) and breaks | ~2 |
| Interruption: unrelated client work | ~2 |

These overlap slightly with the frontend work (I answered a client between frontend commits), so the rows are approximate and do not sum exactly. Gaps in the commit history match the non-coding rows: no commits between about 13:16 and 15:24 (environment repair and break) and between about 16:29 and 19:04 (client interruption, then the order flow).

Git notes: the two Vitest commits share a message (`45a548d` and `791f92b`); `summary.test.ts` arrived in the second one. Commits `abf4823` and `7d3cf26` show the Postgres 18 volume-mount fix (the image expects the volume at `/var/lib/postgresql`).
