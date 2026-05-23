PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS sales_order_status_lut (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    is_terminal INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS clients (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    tier TEXT NOT NULL CHECK (tier IN ('B2B', 'B2C')),
    default_destination_address TEXT NOT NULL DEFAULT '',
    total_lifetime_orders INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sales_orders (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    client_name TEXT NOT NULL,
    destination_address TEXT NOT NULL DEFAULT '',
    required_date TEXT NOT NULL,
    status_id TEXT NOT NULL,
    total_value REAL NOT NULL DEFAULT 0,
    locked INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (client_id) REFERENCES clients(id) ON UPDATE CASCADE ON DELETE RESTRICT,
    FOREIGN KEY (status_id) REFERENCES sales_order_status_lut(id) ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_sales_orders_client_id ON sales_orders(client_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_status_id ON sales_orders(status_id);
CREATE INDEX IF NOT EXISTS idx_sales_orders_created_at ON sales_orders(created_at);

CREATE TABLE IF NOT EXISTS sales_order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    requested_qty INTEGER NOT NULL,
    allocated_qty INTEGER NOT NULL DEFAULT 0,
    unit_price REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (order_id) REFERENCES sales_orders(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sales_order_items_order_id ON sales_order_items(order_id);

CREATE TABLE IF NOT EXISTS inventory_reservations (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL UNIQUE,
    reserved_at TEXT NOT NULL,
    FOREIGN KEY (order_id) REFERENCES sales_orders(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS inventory_reservation_items (
    id TEXT PRIMARY KEY,
    reservation_id TEXT NOT NULL,
    sku TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    FOREIGN KEY (reservation_id) REFERENCES inventory_reservations(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_inventory_reservation_items_reservation_id ON inventory_reservation_items(reservation_id);

INSERT OR IGNORE INTO sales_order_status_lut (id, code, label, sort_order, is_terminal, created_at, updated_at) VALUES
    ('3d94ea1a-61e2-5f39-91c7-d2b0d3cf7f81', 'PENDING', 'Pending', 1, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('e56951b0-9fe9-54df-a44f-3c8c4a8f8d88', 'PROCESSING', 'Processing', 2, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('8d92bb22-1f70-5c43-9d73-6c1f389fa3cc', 'DELIVERING', 'Delivering', 3, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('f7e0e2fa-3a94-52b8-a1a5-7a87f07df8ab', 'COMPLETED', 'Completed', 4, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('7f2d7a8f-3b76-5b17-86b3-9f4d0b3a24d1', 'CANCELLED', 'Cancelled', 5, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);