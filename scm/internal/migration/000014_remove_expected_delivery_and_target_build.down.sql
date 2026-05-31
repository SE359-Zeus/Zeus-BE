ALTER TABLE purchase_orders ADD COLUMN target_build TEXT;
ALTER TABLE purchase_orders ADD COLUMN expected_delivery DATETIME NOT NULL DEFAULT '2026-01-01 00:00:00';
