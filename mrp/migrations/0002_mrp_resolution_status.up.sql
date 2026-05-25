PRAGMA foreign_keys = OFF;

BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS resolution_statuses (
    id INTEGER PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT OR IGNORE INTO resolution_statuses (id, code) VALUES
    (1, 'planned'),
    (2, 'partial'),
    (3, 'shortage'),
    (4, 'ready_to_build');

ALTER TABLE shortage_logs RENAME TO shortage_logs_old;

CREATE TABLE shortage_logs (
    id TEXT PRIMARY KEY,
    production_order_id TEXT NOT NULL,
    part_id TEXT NOT NULL,
    shortage_qty INTEGER NOT NULL CHECK (shortage_qty > 0),
    resolution_status_id INTEGER NOT NULL DEFAULT 1,
    resolution_status TEXT NOT NULL DEFAULT 'planned',
    FOREIGN KEY (production_order_id) REFERENCES production_orders(id)
        ON UPDATE CASCADE
        ON DELETE CASCADE,
    FOREIGN KEY (part_id) REFERENCES parts(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT,
    FOREIGN KEY (resolution_status_id) REFERENCES resolution_statuses(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT
);

INSERT INTO shortage_logs (
    id,
    production_order_id,
    part_id,
    shortage_qty,
    resolution_status_id,
    resolution_status
)
SELECT
    id,
    production_order_id,
    part_id,
    shortage_qty,
    CASE lower(COALESCE(resolution_status, 'planned'))
        WHEN 'partial' THEN 2
        WHEN 'shortage' THEN 3
        WHEN 'ready_to_build' THEN 4
        ELSE 1
    END,
    CASE lower(COALESCE(resolution_status, 'planned'))
        WHEN 'partial' THEN 'partial'
        WHEN 'shortage' THEN 'shortage'
        WHEN 'ready_to_build' THEN 'ready_to_build'
        ELSE 'planned'
    END
FROM shortage_logs_old;

DROP TABLE shortage_logs_old;

CREATE INDEX IF NOT EXISTS idx_shortage_logs_resolution_status_id
    ON shortage_logs(resolution_status_id);

CREATE INDEX IF NOT EXISTS idx_shortage_logs_production_order_id
    ON shortage_logs(production_order_id);

CREATE INDEX IF NOT EXISTS idx_shortage_logs_part_id
    ON shortage_logs(part_id);

COMMIT;

PRAGMA foreign_keys = ON;