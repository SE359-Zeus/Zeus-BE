CREATE TABLE IF NOT EXISTS production_orders (
    id TEXT PRIMARY KEY,
    product_model_code TEXT NOT NULL,
    target_quantity INTEGER NOT NULL CHECK (target_quantity > 0),
    status TEXT NOT NULL CHECK (status IN ('CLEAR_TO_BUILD', 'PARTIAL', 'SHORTAGE', 'PLANNED')),
    scheduled_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_production_orders_product_model_code
    ON production_orders(product_model_code);

CREATE TABLE IF NOT EXISTS bom_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_model_code TEXT NOT NULL,
    component_part_id TEXT NOT NULL,
    required_quantity_per_unit INTEGER NOT NULL CHECK (required_quantity_per_unit > 0),
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
    FOREIGN KEY (production_order_id) REFERENCES production_orders(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_shortage_logs_production_order_id
    ON shortage_logs(production_order_id);

CREATE INDEX IF NOT EXISTS idx_shortage_logs_part_id
    ON shortage_logs(part_id);