-- Vocabulary: standalone word dictionary, reusable across articles
CREATE TABLE IF NOT EXISTS vocabulary (
    id          TEXT PRIMARY KEY,
    word        TEXT NOT NULL UNIQUE,
    phonetic    TEXT NOT NULL DEFAULT '',
    chinese     TEXT NOT NULL DEFAULT '',
    explanation TEXT NOT NULL DEFAULT '',
    source      TEXT NOT NULL DEFAULT 'ai_import',
    created_at  TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

-- Article-Words junction: links articles to vocabulary entries
CREATE TABLE IF NOT EXISTS article_words (
    article_id    TEXT NOT NULL,
    word_text     TEXT NOT NULL,
    vocabulary_id TEXT,
    PRIMARY KEY (article_id, word_text)
);

CREATE INDEX IF NOT EXISTS idx_vocabulary_word ON vocabulary(word);
CREATE INDEX IF NOT EXISTS idx_article_words_article ON article_words(article_id);
