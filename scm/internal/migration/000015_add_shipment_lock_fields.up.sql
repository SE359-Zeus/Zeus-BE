ALTER TABLE shipments ADD COLUMN locked_by TEXT;
ALTER TABLE shipments ADD COLUMN lock_expires_at DATETIME;
ALTER TABLE shipments ADD COLUMN operator_id TEXT;
ALTER TABLE shipments ADD COLUMN operator_name TEXT;
