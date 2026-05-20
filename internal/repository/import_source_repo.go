package repository

import (
	"crypto/sha256"
	"fmt"
	"daily-english/internal/model"
)

type ImportSourceRepo struct {
	db *DB
}

func NewImportSourceRepo(db *DB) *ImportSourceRepo {
	return &ImportSourceRepo{db: db}
}

func (r *ImportSourceRepo) Create(s *model.ImportSource) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO import_sources (id, name, file_name, scenario_id, article_count, status, error_message, content_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.Name, s.FileName, s.ScenarioID, s.ArticleCount, s.Status, s.ErrorMessage, s.ContentHash, s.CreatedAt)
	return err
}

func (r *ImportSourceRepo) GetByID(id string) (*model.ImportSource, error) {
	var s model.ImportSource
	err := r.db.QueryRow(`SELECT id, name, file_name, scenario_id, article_count, status, error_message, content_hash, created_at FROM import_sources WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.FileName, &s.ScenarioID, &s.ArticleCount, &s.Status, &s.ErrorMessage, &s.ContentHash, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ImportSourceRepo) ListAll() ([]model.ImportSource, error) {
	rows, err := r.db.Query(`SELECT id, name, file_name, scenario_id, article_count, status, error_message, content_hash, created_at FROM import_sources ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.ImportSource
	for rows.Next() {
		var s model.ImportSource
		if err := rows.Scan(&s.ID, &s.Name, &s.FileName, &s.ScenarioID, &s.ArticleCount, &s.Status, &s.ErrorMessage, &s.ContentHash, &s.CreatedAt); err != nil {
			return list, err
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *ImportSourceRepo) Update(s *model.ImportSource) error {
	_, err := r.db.Exec(`UPDATE import_sources SET article_count=?, status=?, error_message=? WHERE id=?`,
		s.ArticleCount, s.Status, s.ErrorMessage, s.ID)
	return err
}

func (r *ImportSourceRepo) FindByContentHash(hash string) (*model.ImportSource, error) {
	var s model.ImportSource
	err := r.db.QueryRow(`SELECT id, name, file_name, scenario_id, article_count, status, error_message, content_hash, created_at FROM import_sources WHERE content_hash = ?`, hash).
		Scan(&s.ID, &s.Name, &s.FileName, &s.ScenarioID, &s.ArticleCount, &s.Status, &s.ErrorMessage, &s.ContentHash, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func ContentHash(text string) string {
	h := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", h[:16])
}

func (r *ImportSourceRepo) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM import_sources WHERE id = ?`, id)
	return err
}

func (r *ImportSourceRepo) ListAllPaginated(page, limit int) ([]model.ImportSource, int, error) {
	var total int
	r.db.QueryRow("SELECT COUNT(*) FROM import_sources").Scan(&total)

	offset := (page - 1) * limit
	rows, err := r.db.Query(`SELECT id, name, file_name, scenario_id, article_count, status, error_message, content_hash, created_at FROM import_sources ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []model.ImportSource
	for rows.Next() {
		var s model.ImportSource
		if err := rows.Scan(&s.ID, &s.Name, &s.FileName, &s.ScenarioID, &s.ArticleCount, &s.Status, &s.ErrorMessage, &s.ContentHash, &s.CreatedAt); err != nil {
			return list, total, err
		}
		list = append(list, s)
	}
	return list, total, nil
}
