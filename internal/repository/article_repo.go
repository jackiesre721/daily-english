package repository

import (
	"encoding/json"
	"daily-english/internal/model"
	"strings"
)

type ArticleRepo struct {
	db *DB
}

func NewArticleRepo(db *DB) *ArticleRepo {
	return &ArticleRepo{db: db}
}

const articleCols = `id, scenario_id, title_zh, title_en, type, difficulty, source, original_text, translation, context_note, segments_json, words_json, quizzes_json, takeaway, tags, html_content, created_at`

func (r *ArticleRepo) ListByScenario(scenarioID string) ([]model.Article, error) {
	rows, err := r.db.Query(`SELECT `+articleCols+` FROM articles WHERE scenario_id = ? ORDER BY difficulty, id`, scenarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArticles(rows)
}

func (r *ArticleRepo) GetByID(id string) (*model.Article, error) {
	row := r.db.QueryRow(`SELECT `+articleCols+` FROM articles WHERE id = ?`, id)
	return scanArticle(row)
}

func (r *ArticleRepo) GetUnlearned(userID string, limit int) ([]model.Article, error) {
	rows, err := r.db.Query(`SELECT a.id, a.scenario_id, a.title_zh, a.title_en, a.type, a.difficulty, a.source, a.original_text, a.translation, a.context_note, a.segments_json, a.words_json, a.quizzes_json, a.takeaway, a.tags, a.html_content, a.created_at
		FROM articles a
		LEFT JOIN learning_progress lp ON a.id = lp.article_id AND lp.user_id = ?
		WHERE lp.id IS NULL
		ORDER BY a.difficulty, a.id
		LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArticles(rows)
}

func (r *ArticleRepo) CountByScenario(scenarioID string) (int, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM articles WHERE scenario_id = ?`, scenarioID).Scan(&count)
	return count, err
}

func (r *ArticleRepo) Import(articles []model.Article) (imported int, err error) {
	tx, err := r.db.BeginTx()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	for _, a := range articles {
		wordsJSON, _ := json.Marshal(a.Words)
		quizzesJSON, _ := json.Marshal(a.Quizzes)
		segmentsJSON, _ := json.Marshal(a.Segments)
		tagsJSON, _ := json.Marshal(a.Tags)
		_, e := tx.Exec(`INSERT OR REPLACE INTO articles (id, scenario_id, title_zh, title_en, type, difficulty, source, import_source_id, original_text, translation, context_note, segments_json, words_json, quizzes_json, takeaway, tags, html_content) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a.ID, a.ScenarioID, a.TitleZh, a.TitleEn, a.Type, a.Difficulty, a.Source, a.ImportSourceID, a.OriginalText, a.Translation, a.ContextNote, string(segmentsJSON), string(wordsJSON), string(quizzesJSON), a.Takeaway, string(tagsJSON), a.HTMLContent)
		if e != nil {
			return imported, e
		}
		imported++
	}
	return imported, tx.Commit()
}

func scanArticle(row interface{ Scan(...interface{}) error }) (*model.Article, error) {
	a := &model.Article{}
	var wordsJSON, quizzesJSON, segmentsJSON, tagsJSON string
	err := row.Scan(&a.ID, &a.ScenarioID, &a.TitleZh, &a.TitleEn, &a.Type, &a.Difficulty, &a.Source, &a.OriginalText, &a.Translation, &a.ContextNote, &segmentsJSON, &wordsJSON, &quizzesJSON, &a.Takeaway, &tagsJSON, &a.HTMLContent, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(segmentsJSON), &a.Segments)
	json.Unmarshal([]byte(wordsJSON), &a.Words)
	json.Unmarshal([]byte(quizzesJSON), &a.Quizzes)
	json.Unmarshal([]byte(tagsJSON), &a.Tags)
	return a, nil
}

func scanArticles(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]model.Article, error) {
	var articles []model.Article
	for rows.Next() {
		a, err := scanArticle(rows)
		if err != nil {
			return articles, err
		}
		articles = append(articles, *a)
	}
	return articles, nil
}

func (r *ArticleRepo) DeleteByID(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	// Clean up related data
	tx.Exec(`DELETE FROM article_words WHERE article_id = ?`, id)
	tx.Exec(`DELETE FROM article_analyses WHERE article_id = ?`, id)
	tx.Exec(`DELETE FROM learning_progress WHERE article_id = ?`, id)
	_, err = tx.Exec(`DELETE FROM articles WHERE id = ?`, id)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *ArticleRepo) UpdateSegments(id string, segments []model.Segment) error {
	segmentsJSON, _ := json.Marshal(segments)
	_, err := r.db.Exec(`UPDATE articles SET segments_json = ? WHERE id = ?`, string(segmentsJSON), id)
	return err
}

func (r *ArticleRepo) ListByImportSource(sourceID string) ([]model.Article, error) {
	rows, err := r.db.Query(`SELECT `+articleCols+` FROM articles WHERE import_source_id = ? ORDER BY created_at`, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArticles(rows)
}

func (r *ArticleRepo) ListByAllScenarios() ([]model.Article, error) {
	rows, err := r.db.Query(`SELECT ` + articleCols + ` FROM articles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanArticles(rows)
}

// ArticleSummary is a lightweight article representation for listing.
type ArticleSummary struct {
	ID         string   `json:"id"`
	ScenarioID string   `json:"scenario_id"`
	TitleZh    string   `json:"title_zh"`
	TitleEn    string   `json:"title_en"`
	Type       string   `json:"type"`
	Difficulty int      `json:"difficulty"`
	Source     string   `json:"source"`
	WordCount  int      `json:"word_count"`
	Tags       []string `json:"tags"`
	CreatedAt  string   `json:"created_at"`
}

func (r *ArticleRepo) ListAllSummary(page, limit int, search, source string) ([]ArticleSummary, int, error) {
	var whereClauses []string
	var args []interface{}

	if source != "" && source != "all" {
		whereClauses = append(whereClauses, "(source = ? OR scenario_id = ?)")
		args = append(args, source, source)
	}
	if search != "" {
		whereClauses = append(whereClauses, "(title_zh LIKE ? OR title_en LIKE ?)")
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	where := ""
	if len(whereClauses) > 0 {
		where = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Count
	var total int
	r.db.QueryRow("SELECT COUNT(*) FROM articles"+where, args...).Scan(&total)

	// Query
	offset := (page - 1) * limit
	query := "SELECT id, scenario_id, title_zh, title_en, type, difficulty, source, json_array_length(words_json), tags, created_at FROM articles" + where + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, limit, offset)
	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []ArticleSummary
	for rows.Next() {
		var s ArticleSummary
		var tagsJSON string
		if err := rows.Scan(&s.ID, &s.ScenarioID, &s.TitleZh, &s.TitleEn, &s.Type, &s.Difficulty, &s.Source, &s.WordCount, &tagsJSON, &s.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(tagsJSON), &s.Tags)
		result = append(result, s)
	}
	return result, total, nil
}

// ListAllSources returns distinct source values for filter chips.
func (r *ArticleRepo) ListAllSources() ([]struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}, error) {
	rows, err := r.db.Query(`SELECT COALESCE(NULLIF(source, ''), scenario_id) as src, COALESCE(NULLIF(source, ''), scenario_id) as name FROM articles GROUP BY src ORDER BY src`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	for rows.Next() {
		var item struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *ArticleRepo) UpdateTags(id string, tags []string) error {
	tagsJSON, _ := json.Marshal(tags)
	_, err := r.db.Exec("UPDATE articles SET tags = ? WHERE id = ?", string(tagsJSON), id)
	return err
}

// TitlesByIDs returns a map of articleID -> {title_zh, title_en} in a single query.
func (r *ArticleRepo) TitlesByIDs(ids []string) (map[string]struct {
	TitleZh string
	TitleEn string
}, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := "SELECT id, title_zh, title_en FROM articles WHERE id IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]struct {
		TitleZh string
		TitleEn string
	})
	for rows.Next() {
		var id, zh, en string
		if err := rows.Scan(&id, &zh, &en); err == nil {
			result[id] = struct {
				TitleZh string
				TitleEn string
			}{zh, en}
		}
	}
	return result, nil
}
