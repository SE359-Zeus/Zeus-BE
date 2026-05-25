PRAGMA foreign_keys = OFF;

ALTER TABLE shortage_logs RENAME TO shortage_logs_new;

CREATE TABLE shortage_logs (
    id TEXT PRIMARY KEY,
    production_order_id TEXT NOT NULL,
    part_id TEXT NOT NULL,
    shortage_qty INTEGER NOT NULL CHECK (shortage_qty > 0),
    FOREIGN KEY (production_order_id) REFERENCES production_orders(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    FOREIGN KEY (part_id) REFERENCES parts(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT
);

INSERT INTO shortage_logs (
    id,
    production_order_id,
    part_id,
    shortage_qty
)
SELECT
    id,
    production_order_id,
    part_id,
    shortage_qty
FROM shortage_logs_new;

DROP TABLE shortage_logs_new;
DROP TABLE IF EXISTS resolution_statuses;

CREATE INDEX IF NOT EXISTS idx_shortage_logs_production_order_id
    ON shortage_logs(production_order_id);

CREATE INDEX IF NOT EXISTS idx_shortage_logs_part_id
    ON shortage_logs(part_id);

PRAGMA foreign_keys = ON;