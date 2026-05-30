CREATE TABLE IF NOT EXISTS inventory_ledger (
    id TEXT PRIMARY KEY,
    sku TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('IN', 'OUT', 'ADJ')),
    qty_change INTEGER NOT NULL,
    running_balance INTEGER NOT NULL,
    location TEXT NOT NULL DEFAULT 'WH-A',
    operator_id TEXT NOT NULL,
    reference TEXT NOT NULL,
    reference_type TEXT NOT NULL CHECK(reference_type IN ('goods_receipt', 'shipment', 'adjustment', 'initial')),
    reference_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inventory_ledger_sku ON inventory_ledger(sku);
CREATE INDEX IF NOT EXISTS idx_inventory_ledger_type ON inventory_ledger(type);
CREATE INDEX IF NOT EXISTS idx_inventory_ledger_created_at ON inventory_ledger(created_at);
