package repository

import (
	"encoding/json"
	"fmt"
	"daily-english/internal/model"
	"strings"
	"time"
)

type VocabularyRepo struct {
	db          *DB
	articleRepo *ArticleRepo
}

func NewVocabularyRepo(db *DB, ar *ArticleRepo) *VocabularyRepo {
	return &VocabularyRepo{db: db, articleRepo: ar}
}

func (r *VocabularyRepo) FindByWord(word string) (*model.Vocabulary, error) {
	var v model.Vocabulary
	err := r.db.QueryRow(
		`SELECT id, word, phonetic, chinese, explanation, source, status, encounter_count, source_articles, created_at, updated_at FROM vocabulary WHERE LOWER(word) = LOWER(?)`, word,
	).Scan(&v.ID, &v.Word, &v.Phonetic, &v.Chinese, &v.Explanation, &v.Source, &v.Status, &v.EncounterCount, &v.SourceArticles, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *VocabularyRepo) Upsert(v *model.Vocabulary) error {
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO vocabulary (id, word, phonetic, chinese, explanation, source, status, encounter_count, source_articles, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		v.ID, strings.ToLower(v.Word), v.Phonetic, v.Chinese, v.Explanation, v.Source, v.Status, v.EncounterCount, v.SourceArticles, v.CreatedAt, v.UpdatedAt,
	)
	return err
}

func (r *VocabularyRepo) LinkArticle(articleID, wordText, vocabID string) error {
	_, err := r.db.Exec(
		`INSERT OR IGNORE INTO article_words (article_id, word_text, vocabulary_id) VALUES (?, ?, ?)`,
		articleID, wordText, vocabID,
	)
	return err
}

// AddToStudy adds or updates a word in the vocabulary (study list).
// If already exists, increments encounter_count and appends articleID.
func (r *VocabularyRepo) AddToStudy(word, articleID, source string) error {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return fmt.Errorf("empty word")
	}

	existing, err := r.FindByWord(word)
	if err == nil && existing != nil {
		// Update: increment encounter count and append article
		articles := []string{}
		json.Unmarshal([]byte(existing.SourceArticles), &articles)
		found := false
		for _, a := range articles {
			if a == articleID {
				found = true
				break
			}
		}
		if !found && articleID != "" {
			articles = append(articles, articleID)
		}
		articlesJSON, _ := json.Marshal(articles)
		_, err := r.db.Exec(
			"UPDATE vocabulary SET encounter_count = encounter_count + 1, source_articles = ?, updated_at = ? WHERE id = ?",
			string(articlesJSON), time.Now().Format("2006-01-02 15:04:05"), existing.ID,
		)
		return err
	}

	return nil // word not found — caller should Upsert first
}

// Remove deletes a word from the vocabulary (study list).
func (r *VocabularyRepo) Remove(word string) error {
	word = strings.ToLower(strings.TrimSpace(word))
	_, err := r.db.Exec("DELETE FROM vocabulary WHERE LOWER(word) = ?", word)
	return err
}

// GetStudySetForArticle returns all vocabulary entries linked to a given article.
func (r *VocabularyRepo) GetStudySetForArticle(articleID string) (map[string]*model.Vocabulary, error) {
	rows, err := r.db.Query(`
		SELECT v.id, v.word, v.phonetic, v.chinese, v.explanation, v.source, v.status, v.encounter_count, v.source_articles, v.created_at, v.updated_at
		FROM vocabulary v
		JOIN article_words aw ON aw.vocabulary_id = v.id
		WHERE aw.article_id = ?
	`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]*model.Vocabulary)
	for rows.Next() {
		var v model.Vocabulary
		if err := rows.Scan(&v.ID, &v.Word, &v.Phonetic, &v.Chinese, &v.Explanation, &v.Source, &v.Status, &v.EncounterCount, &v.SourceArticles, &v.CreatedAt, &v.UpdatedAt); err != nil {
			continue
		}
		result[v.Word] = &v
	}
	return result, nil
}

// BatchUpsertForArticle processes AI-generated words for an article:
// for each word, reuse existing vocabulary entry or create new one, then link to article.
// Returns a map of lowercase word -> vocabulary_id.
func (r *VocabularyRepo) BatchUpsertForArticle(articleID string, words []model.Word) (map[string]string, error) {
	vocabIDs := make(map[string]string)
	for _, w := range words {
		word := strings.ToLower(strings.TrimSpace(w.English))
		if word == "" {
			continue
		}

		existing, err := r.FindByWord(word)
		var vocabID string
		if err == nil && existing != nil {
			vocabID = existing.ID
		} else {
			vocabID = "v_" + word
			now := time.Now().Format("2006-01-02 15:04:05")
			v := &model.Vocabulary{
				ID:             vocabID,
				Word:           word,
				Phonetic:       w.Phonetic,
				Chinese:        w.Chinese,
				Explanation:    w.Explanation,
				Source:         "ai_import",
				Status:         "new",
				EncounterCount: 1,
				SourceArticles: "[]",
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := r.Upsert(v); err != nil {
				return vocabIDs, fmt.Errorf("upsert vocabulary %q: %v", word, err)
			}
		}
		vocabIDs[word] = vocabID

		wordText := strings.TrimSpace(w.TextInArticle)
		if wordText == "" {
			wordText = w.English
		}
		if err := r.LinkArticle(articleID, wordText, vocabID); err != nil {
			return vocabIDs, fmt.Errorf("link article word %q: %v", wordText, err)
		}
	}
	return vocabIDs, nil
}

// BatchAddForArticle adds AI-recommended words to study for a given article.
func (r *VocabularyRepo) BatchAddForArticle(articleID string, words []model.Word, vocabIDs map[string]string) error {
	for _, w := range words {
		word := strings.ToLower(strings.TrimSpace(w.English))
		if word == "" {
			continue
		}
		// Ensure vocabulary entry exists
		if _, err := r.FindByWord(word); err != nil {
			// Should have been created by BatchUpsertForArticle
			continue
		}
		if err := r.AddToStudy(word, articleID, "ai_recommend"); err != nil {
			fmt.Printf("[Vocabulary] warning: failed to add %q: %v\n", word, err)
		}
	}
	return nil
}

type ArticleVocabulary struct {
	WordText  string            `json:"word_text"`
	Vocabulary *model.Vocabulary `json:"vocabulary"`
}

func (r *VocabularyRepo) GetArticleVocabulary(articleID string) ([]ArticleVocabulary, error) {
	rows, err := r.db.Query(
		`SELECT aw.word_text, v.id, v.word, v.phonetic, v.chinese, v.explanation, v.source, v.status, v.encounter_count, v.source_articles, v.created_at, v.updated_at
		FROM article_words aw
		JOIN vocabulary v ON v.id = aw.vocabulary_id
		WHERE aw.article_id = ?`, articleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ArticleVocabulary
	for rows.Next() {
		var av ArticleVocabulary
		var v model.Vocabulary
		if err := rows.Scan(&av.WordText, &v.ID, &v.Word, &v.Phonetic, &v.Chinese, &v.Explanation, &v.Source, &v.Status, &v.EncounterCount, &v.SourceArticles, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return result, err
		}
		av.Vocabulary = &v
		result = append(result, av)
	}
	return result, nil
}

// ListAll returns all vocabulary entries ordered by most recently updated.
func (r *VocabularyRepo) ListAll() ([]model.Vocabulary, error) {
	rows, err := r.db.Query(`SELECT id, word, phonetic, chinese, explanation, source, status, encounter_count, source_articles, created_at, updated_at FROM vocabulary ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.Vocabulary
	for rows.Next() {
		var v model.Vocabulary
		if err := rows.Scan(&v.ID, &v.Word, &v.Phonetic, &v.Chinese, &v.Explanation, &v.Source, &v.Status, &v.EncounterCount, &v.SourceArticles, &v.CreatedAt, &v.UpdatedAt); err != nil {
			continue
		}
		result = append(result, v)
	}
	return result, nil
}

// BackfillFromArticles migrates existing words_json data into vocabulary + article_words tables.
func (r *VocabularyRepo) BackfillFromArticles() error {
	articles, err := r.articleRepo.ListByAllScenarios()
	if err != nil {
		return err
	}
	for _, a := range articles {
		if len(a.Words) == 0 {
			continue
		}
		var count int
		r.db.QueryRow(`SELECT COUNT(*) FROM article_words WHERE article_id = ?`, a.ID).Scan(&count)
		if count > 0 {
			continue
		}
		if _, err := r.BatchUpsertForArticle(a.ID, a.Words); err != nil {
			fmt.Printf("backfill article %s: %v\n", a.ID, err)
		}
	}
	return nil
}
