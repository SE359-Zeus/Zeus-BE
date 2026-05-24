CREATE TABLE IF NOT EXISTS product_models (
    model_code TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS parts (
    id TEXT PRIMARY KEY,
    sku TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS production_orders (
    id TEXT PRIMARY KEY,
    product_model_code TEXT NOT NULL,
    target_quantity INTEGER NOT NULL CHECK (target_quantity > 0),
    status TEXT NOT NULL CHECK (status IN ('CLEAR_TO_BUILD', 'PARTIAL', 'SHORTAGE')),
    scheduled_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_model_code) REFERENCES product_models(model_code)
        ON UPDATE CASCADE
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_production_orders_product_model_code
    ON production_orders(product_model_code);

CREATE TABLE IF NOT EXISTS bom_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_model_code TEXT NOT NULL,
    component_part_id TEXT NOT NULL,
    required_quantity_per_unit INTEGER NOT NULL CHECK (required_quantity_per_unit > 0),
    FOREIGN KEY (parent_model_code) REFERENCES product_models(model_code)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    FOREIGN KEY (component_part_id) REFERENCES parts(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    UNIQUE (parent_model_code, component_part_id)
);

CREATE INDEX IF NOT EXISTS idx_bom_entries_parent_model_code
    ON bom_entries(parent_model_code);

CREATE INDEX IF NOT EXISTS idx_bom_entries_component_part_id
    ON bom_entries(component_part_id);

CREATE TABLE IF NOT EXISTS shortage_logs (
    id TEXT PRIMARY KEY,
    production_order_id TEXT NOT NULL,
    part_id TEXT NOT NULL,
    shortage_qty INTEGER NOT NULL CHECK (shortage_qty > 0),
    resolution_status TEXT NOT NULL DEFAULT 'EMITTED',
    FOREIGN KEY (production_order_id) REFERENCES production_orders(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    FOREIGN KEY (part_id) REFERENCES parts(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_shortage_logs_production_order_id
    ON shortage_logs(production_order_id);

CREATE INDEX IF NOT EXISTS idx_shortage_logs_part_id
    ON shortage_logs(part_id);