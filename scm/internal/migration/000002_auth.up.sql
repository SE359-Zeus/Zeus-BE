CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    active INTEGER NOT NULL DEFAULT 1,
    expires_at DATETIME,
    last_used_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at DATETIME
);

INSERT INTO api_keys (id, name, key_prefix, key_hash, active)
VALUES (
    lower(hex(randomblob(16))),
    'Default SCM API Key',
    'scmkey01',
    '$2a$10$w6KpSUPBMyQfgIMjcmKi5uS6sJisStIBfIGOHtJeQn0dnwLXfJhW2',
    1
);
