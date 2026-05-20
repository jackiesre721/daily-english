-- Study word list: words the user is actively learning
CREATE TABLE IF NOT EXISTS study_words (
    id               TEXT PRIMARY KEY,
    word             TEXT NOT NULL UNIQUE,
    vocabulary_id    TEXT NOT NULL,
    source           TEXT NOT NULL DEFAULT 'ai_recommend',
    status           TEXT NOT NULL DEFAULT 'new',
    encounter_count  INTEGER NOT NULL DEFAULT 1,
    source_articles  TEXT NOT NULL DEFAULT '[]',
    created_at       TEXT NOT NULL DEFAULT (datetime('now', 'localtime')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE INDEX IF NOT EXISTS idx_study_words_status ON study_words(status);
CREATE INDEX IF NOT EXISTS idx_study_words_vocab ON study_words(vocabulary_id);
