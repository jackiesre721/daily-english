CREATE TABLE IF NOT EXISTS import_sources (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    file_name       TEXT NOT NULL DEFAULT '',
    scenario_id     TEXT NOT NULL DEFAULT 'ai_imported',
    article_count   INTEGER NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'pending',
    error_message   TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

-- Add import_source_id column to articles table if not exists
-- SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we handle this in Go code
