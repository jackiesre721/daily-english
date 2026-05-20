package repository

import (
	"daily-english/internal/model"
)

type AnalysisRepo struct {
	db *DB
}

func NewAnalysisRepo(db *DB) *AnalysisRepo {
	return &AnalysisRepo{db: db}
}

func (r *AnalysisRepo) Create(a *model.ArticleAnalysis) error {
	_, err := r.db.Exec(`INSERT INTO article_analyses (id, article_id, text, analysis, created_at) VALUES (?, ?, ?, ?, ?)`,
		a.ID, a.ArticleID, a.Text, a.Analysis, a.CreatedAt)
	return err
}

func (r *AnalysisRepo) ListByArticle(articleID string) ([]model.ArticleAnalysis, error) {
	rows, err := r.db.Query(`SELECT id, article_id, text, analysis, created_at FROM article_analyses WHERE article_id = ? ORDER BY created_at DESC`, articleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.ArticleAnalysis
	for rows.Next() {
		var a model.ArticleAnalysis
		if err := rows.Scan(&a.ID, &a.ArticleID, &a.Text, &a.Analysis, &a.CreatedAt); err != nil {
			continue
		}
		list = append(list, a)
	}
	return list, nil
}
