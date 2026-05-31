PRAGMA foreign_keys=off;

CREATE TABLE new_purchase_orders (
    id TEXT PRIMARY KEY,
    vendor_id TEXT NOT NULL REFERENCES suppliers(id),
    status TEXT NOT NULL DEFAULT 'Draft' REFERENCES purchase_order_states(name),
    total_value REAL NOT NULL DEFAULT 0.0,
    payment_terms TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME,
    notes TEXT
);

INSERT INTO new_purchase_orders (id, vendor_id, status, total_value, payment_terms, created_at, updated_at, deleted_at, notes)
SELECT id, vendor_id, status, total_value, payment_terms, created_at, updated_at, deleted_at, notes FROM purchase_orders;

DROP TABLE purchase_orders;

ALTER TABLE new_purchase_orders RENAME TO purchase_orders;

PRAGMA foreign_keys=on;
