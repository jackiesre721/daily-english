package repository

import (
	"daily-english/internal/model"
	"encoding/json"
)

type PhraseRepo struct {
	db *DB
}

func NewPhraseRepo(db *DB) *PhraseRepo {
	return &PhraseRepo{db: db}
}

func (r *PhraseRepo) Upsert(p *model.Phrase) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO phrases (id, phrase, chinese, explanation, category, source, source_articles, source_context, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Phrase, p.Chinese, p.Explanation, p.Category, p.Source, p.SourceArticles, p.SourceContext, p.CreatedAt, p.UpdatedAt)
	return err
}

func (r *PhraseRepo) FindByPhrase(phrase string) (*model.Phrase, error) {
	var p model.Phrase
	err := r.db.QueryRow(`SELECT id, phrase, chinese, explanation, category, source, source_articles, source_context, created_at, updated_at FROM phrases WHERE phrase = ?`, phrase).
		Scan(&p.ID, &p.Phrase, &p.Chinese, &p.Explanation, &p.Category, &p.Source, &p.SourceArticles, &p.SourceContext, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *PhraseRepo) AddSourceContext(phrase, articleID, sentence string) error {
	p, err := r.FindByPhrase(phrase)
	if err != nil {
		return err
	}

	// Append to source_articles
	var articles []string
	json.Unmarshal([]byte(p.SourceArticles), &articles)
	found := false
	for _, a := range articles {
		if a == articleID {
			found = true
			break
		}
	}
	if !found {
		articles = append(articles, articleID)
	}

	// Append to source_context
	var contexts []map[string]string
	json.Unmarshal([]byte(p.SourceContext), &contexts)
	contextFound := false
	for _, c := range contexts {
		if c["article_id"] == articleID && c["sentence"] == sentence {
			contextFound = true
			break
		}
	}
	if !contextFound {
		contexts = append(contexts, map[string]string{"article_id": articleID, "sentence": sentence})
	}

	articlesJSON, _ := json.Marshal(articles)
	contextsJSON, _ := json.Marshal(contexts)
	_, err = r.db.Exec(`UPDATE phrases SET source_articles = ?, source_context = ?, updated_at = datetime('now','localtime') WHERE phrase = ?`,
		string(articlesJSON), string(contextsJSON), phrase)
	return err
}

func (r *PhraseRepo) ListAll() ([]model.Phrase, error) {
	rows, err := r.db.Query(`SELECT id, phrase, chinese, explanation, category, source, source_articles, source_context, created_at, updated_at FROM phrases ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Phrase
	for rows.Next() {
		var p model.Phrase
		if err := rows.Scan(&p.ID, &p.Phrase, &p.Chinese, &p.Explanation, &p.Category, &p.Source, &p.SourceArticles, &p.SourceContext, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		list = append(list, p)
	}
	return list, nil
}
