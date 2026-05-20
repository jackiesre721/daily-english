-- Phrases and article analyses

CREATE TABLE IF NOT EXISTS phrases (
    id              TEXT PRIMARY KEY,
    phrase          TEXT NOT NULL,
    chinese         TEXT NOT NULL DEFAULT '',
    explanation     TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL DEFAULT 'phrase',
    source          TEXT NOT NULL DEFAULT 'ai_analyze',
    source_articles TEXT NOT NULL DEFAULT '[]',
    source_context  TEXT NOT NULL DEFAULT '[]',
    created_at      TEXT NOT NULL DEFAULT (datetime('now','localtime')),
    updated_at      TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_phrases_phrase ON phrases(phrase);

CREATE TABLE IF NOT EXISTS article_analyses (
    id          TEXT PRIMARY KEY,
    article_id  TEXT NOT NULL,
    text        TEXT NOT NULL,
    analysis    TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_analyses_article ON article_analyses(article_id);
