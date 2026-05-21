CREATE TABLE IF NOT EXISTS endpoint_roles (
    method         TEXT NOT NULL,
    path           TEXT NOT NULL,
    required_level TEXT NOT NULL,
    PRIMARY KEY (method, path)
);
