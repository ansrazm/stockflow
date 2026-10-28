-- Seed data. Assumes an empty database (order ids 1 and 2 are referenced below).
-- Invariant kept: products.stock = sum(in) - sum(out) from stock_movements.

-- 14 products with initial stock, each backed by an 'in' movement.
WITH seed(sku, name, price, initial, active) AS (VALUES
    ('KBD-001', 'Mechanical Keyboard 87-key',  650000,  25, TRUE),
    ('MSE-001', 'Wireless Mouse',              185000,  40, TRUE),
    ('MON-001', 'Monitor 24" IPS',            1850000,   8, TRUE),
    ('HDP-001', 'USB Headset',                  320000,  15, TRUE),
    ('WEB-001', 'Webcam 1080p',                 450000,   6, TRUE),
    ('USB-001', 'USB-C Hub 7-in-1',             275000,  50, TRUE),
    ('CBL-001', 'HDMI Cable 2m',                 65000, 120, TRUE),
    ('LAP-001', 'Aluminium Laptop Stand',       210000,  18, TRUE),
    ('SSD-001', 'NVMe SSD 512GB',               780000,  12, TRUE),
    ('RAM-001', 'DDR4 RAM 16GB',                520000,   0, TRUE),
    ('PSU-001', 'Power Supply 550W',            640000,   4, TRUE),
    ('CHR-001', 'Ergonomic Office Chair',      2400000,   3, TRUE),
    ('DSK-001', 'Standing Desk (discontinued)',3200000,   5, FALSE),
    ('PAD-001', 'Mousepad XL',                   95000,  75, TRUE)
),
ins AS (
    INSERT INTO products (sku, name, price, stock, is_active)
    SELECT sku, name, price, initial, active FROM seed
    RETURNING id, stock
)
INSERT INTO stock_movements (product_id, type, qty, note, created_at)
SELECT id, 'in', stock, 'Initial stock', now() - interval '5 days'
FROM ins WHERE stock > 0;

-- Two sample orders: #1 CREATED, #2 CANCELLED.
INSERT INTO sales_orders (customer, status, total, created_at, cancelled_at) VALUES
    ('Budi Santoso', 'CREATED',   0, now() - interval '2 days', NULL),
    ('Sari Dewi',    'CANCELLED', 0, now() - interval '1 day',  now() - interval '20 hours');

INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
SELECT v.order_id, v.sku, v.qty, p.price
FROM (VALUES (1, 'KBD-001', 2), (1, 'MSE-001', 3), (2, 'MON-001', 1)) AS v(order_id, sku, qty)
JOIN products p ON p.sku = v.sku;

UPDATE sales_orders o
SET total = (SELECT sum(qty * unit_price) FROM sales_order_items WHERE order_id = o.id);

-- Every order item produced an 'out' movement...
INSERT INTO stock_movements (product_id, type, qty, note, created_at)
SELECT p.id, 'out', i.qty, 'order #' || i.order_id, o.created_at
FROM sales_order_items i
JOIN products p ON p.sku = i.sku
JOIN sales_orders o ON o.id = i.order_id;

-- ...and the cancelled order was restored with an 'in' movement.
INSERT INTO stock_movements (product_id, type, qty, note, created_at)
SELECT p.id, 'in', i.qty, 'cancel order #' || i.order_id, o.cancelled_at
FROM sales_order_items i
JOIN products p ON p.sku = i.sku
JOIN sales_orders o ON o.id = i.order_id
WHERE o.status = 'CANCELLED';

-- Net stock effect: only the CREATED order reduces stock.
UPDATE products p
SET stock = p.stock - i.qty
FROM sales_order_items i
JOIN sales_orders o ON o.id = i.order_id
WHERE i.sku = p.sku AND o.status = 'CREATED';
