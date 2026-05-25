PRAGMA foreign_keys=off;

-- Recreate parts table with FKs
CREATE TABLE new_parts (
    id TEXT PRIMARY KEY,
    part_catalog_id TEXT NOT NULL REFERENCES part_catalogs(id),
    product_id TEXT REFERENCES products(id),
    serial_number TEXT NOT NULL,
    part_condition_id INTEGER NOT NULL REFERENCES part_conditions(id),
    manufactured_date DATETIME NOT NULL,
    installation_date DATETIME,
    removal_date DATETIME,
    scrapped_date DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
INSERT INTO new_parts SELECT * FROM parts;
DROP TABLE parts;
ALTER TABLE new_parts RENAME TO parts;

-- Recreate products
CREATE TABLE new_products (
    id TEXT PRIMARY KEY,
    product_model_code TEXT NOT NULL REFERENCES product_models(model_code),
    customer_id TEXT NOT NULL,
    product_name TEXT NOT NULL,
    serial_number TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
INSERT INTO new_products SELECT * FROM products;
DROP TABLE products;
ALTER TABLE new_products RENAME TO products;

-- Recreate purchase_orders
CREATE TABLE new_purchase_orders (
    id TEXT PRIMARY KEY,
    vendor_id TEXT NOT NULL REFERENCES suppliers(id),
    target_build TEXT,
    status TEXT NOT NULL DEFAULT 'Draft',
    total_value REAL NOT NULL DEFAULT 0.0,
    payment_terms TEXT,
    expected_delivery DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
INSERT INTO new_purchase_orders SELECT * FROM purchase_orders;
DROP TABLE purchase_orders;
ALTER TABLE new_purchase_orders RENAME TO purchase_orders;

-- Recreate goods_receipts
CREATE TABLE new_goods_receipts (
    id TEXT PRIMARY KEY,
    po_ref TEXT NOT NULL REFERENCES purchase_orders(id),
    vendor_id TEXT NOT NULL REFERENCES suppliers(id),
    status TEXT NOT NULL DEFAULT 'Pending',
    arrival_date DATETIME NOT NULL,
    operator_id TEXT,
    locked_by TEXT,
    lock_expires_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
INSERT INTO new_goods_receipts SELECT * FROM goods_receipts;
DROP TABLE goods_receipts;
ALTER TABLE new_goods_receipts RENAME TO goods_receipts;

-- Recreate part_catalogs
CREATE TABLE new_part_catalogs (
    id TEXT PRIMARY KEY,
    part_number TEXT NOT NULL,
    part_types_id INTEGER NOT NULL REFERENCES part_types(id),
    mfg_number TEXT NOT NULL,
    description TEXT,
    part_mfg_status INTEGER NOT NULL REFERENCES part_mfg_statuses(id),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);
INSERT INTO new_part_catalogs SELECT * FROM part_catalogs;
DROP TABLE part_catalogs;
ALTER TABLE new_part_catalogs RENAME TO part_catalogs;

-- Recreate parts_by_model with foreign keys
CREATE TABLE new_parts_by_model (
    part_catalog_id TEXT NOT NULL REFERENCES part_catalogs(id),
    product_model_code TEXT NOT NULL REFERENCES product_models(model_code),
    quantity INTEGER NOT NULL,
    PRIMARY KEY (part_catalog_id, product_model_code)
);
INSERT INTO new_parts_by_model SELECT * FROM parts_by_model;
DROP TABLE parts_by_model;
ALTER TABLE new_parts_by_model RENAME TO parts_by_model;

PRAGMA foreign_keys=on;
