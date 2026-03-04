-- Base schema
--
-- Migrations tracking table
CREATE TABLE IF NOT EXISTS migrations (
    migration_number INTEGER PRIMARY KEY,
    migration_name TEXT NOT NULL,
    executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Redirects table
CREATE TABLE IF NOT EXISTS redirects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_host TEXT NOT NULL DEFAULT '',
    source_path TEXT NOT NULL DEFAULT '/',
    destination TEXT NOT NULL,
    redirect_type INTEGER NOT NULL DEFAULT 301,
    is_regex INTEGER NOT NULL DEFAULT 0,
    hit_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_redirects_host ON redirects(source_host);

-- Record execution of this migration
INSERT OR IGNORE INTO migrations (migration_number, migration_name)
VALUES (001, '001-base');
