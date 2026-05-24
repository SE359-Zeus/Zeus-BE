CREATE TABLE IF NOT EXISTS part_types (
    id INTEGER PRIMARY KEY,
    part_type_name TEXT NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE IF NOT EXISTS parts_by_model (
    part_catalog_id TEXT NOT NULL,
    product_model_code TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    PRIMARY KEY (part_catalog_id, product_model_code)
);
