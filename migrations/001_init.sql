CREATE TABLE IF NOT EXISTS users (
    id                  TEXT PRIMARY KEY DEFAULT 'user_001',
    name                TEXT NOT NULL DEFAULT '',
    streak_days         INTEGER NOT NULL DEFAULT 0,
    max_streak_days     INTEGER NOT NULL DEFAULT 0,
    freeze_cards        INTEGER NOT NULL DEFAULT 0,
    daily_new_target    INTEGER NOT NULL DEFAULT 3,
    total_words_learned INTEGER NOT NULL DEFAULT 0,
    last_checkin_date   TEXT,
    settings_json       TEXT NOT NULL DEFAULT '{}',
    created_at          TEXT NOT NULL DEFAULT (datetime('now', 'localtime')),
    updated_at          TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE IF NOT EXISTS scenarios (
    id              TEXT PRIMARY KEY,
    name_zh         TEXT NOT NULL,
    name_en         TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    icon            TEXT NOT NULL DEFAULT '',
    parent_id       TEXT,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    total_articles  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS articles (
    id              TEXT PRIMARY KEY,
    scenario_id     TEXT NOT NULL,
    title_zh        TEXT NOT NULL,
    title_en        TEXT NOT NULL,
    type            TEXT NOT NULL DEFAULT 'sentence',
    difficulty      INTEGER NOT NULL DEFAULT 1,
    source          TEXT NOT NULL DEFAULT '',
    original_text   TEXT NOT NULL,
    translation     TEXT NOT NULL,
    context_note    TEXT NOT NULL DEFAULT '',
    words_json      TEXT NOT NULL DEFAULT '[]',
    quizzes_json    TEXT NOT NULL DEFAULT '[]',
    takeaway        TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL DEFAULT (datetime('now', 'localtime'))
);

CREATE TABLE IF NOT EXISTS learning_progress (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL DEFAULT 'user_001',
    article_id      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'new',
    step_completed  INTEGER NOT NULL DEFAULT 0,
    quiz_results_json TEXT NOT NULL DEFAULT '[]',
    review_count    INTEGER NOT NULL DEFAULT 0,
    correct_count   INTEGER NOT NULL DEFAULT 0,
    wrong_count     INTEGER NOT NULL DEFAULT 0,
    ease_factor     REAL NOT NULL DEFAULT 2.5,
    interval_days   INTEGER NOT NULL DEFAULT 1,
    next_review_date TEXT,
    last_review_date TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now', 'localtime')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now', 'localtime')),
    UNIQUE(user_id, article_id)
);

CREATE TABLE IF NOT EXISTS word_mastery (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL DEFAULT 'user_001',
    word_id             TEXT NOT NULL,
    source_article_ids  TEXT NOT NULL DEFAULT '[]',
    encounter_count     INTEGER NOT NULL DEFAULT 0,
    correct_count       INTEGER NOT NULL DEFAULT 0,
    wrong_count         INTEGER NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'new',
    last_seen_date      TEXT,
    UNIQUE(user_id, word_id)
);

CREATE TABLE IF NOT EXISTS daily_records (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL DEFAULT 'user_001',
    date                TEXT NOT NULL,
    new_articles_learned INTEGER NOT NULL DEFAULT 0,
    new_words_learned   INTEGER NOT NULL DEFAULT 0,
    articles_reviewed   INTEGER NOT NULL DEFAULT 0,
    correct_count       INTEGER NOT NULL DEFAULT 0,
    total_count         INTEGER NOT NULL DEFAULT 0,
    accuracy_rate       REAL NOT NULL DEFAULT 0,
    time_spent_seconds  INTEGER NOT NULL DEFAULT 0,
    checkin_completed   INTEGER NOT NULL DEFAULT 0,
    UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS achievements (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL DEFAULT 'user_001',
    badge_id    TEXT NOT NULL,
    earned_at   TEXT NOT NULL DEFAULT (datetime('now', 'localtime')),
    UNIQUE(user_id, badge_id)
);

CREATE INDEX IF NOT EXISTS idx_articles_scenario ON articles(scenario_id);
CREATE INDEX IF NOT EXISTS idx_learning_user_status ON learning_progress(user_id, status);
CREATE INDEX IF NOT EXISTS idx_learning_next_review ON learning_progress(user_id, next_review_date);
CREATE INDEX IF NOT EXISTS idx_daily_date ON daily_records(user_id, date);

-- 初始化默认用户
INSERT OR IGNORE INTO users (id, name) VALUES ('user_001', '小明');
