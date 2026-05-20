package repository

import "daily-english/internal/model"

type ScenarioRepo struct {
	db *DB
}

func NewScenarioRepo(db *DB) *ScenarioRepo {
	return &ScenarioRepo{db: db}
}

func (r *ScenarioRepo) ListTopLevel() ([]model.Scenario, error) {
	rows, err := r.db.Query(`SELECT id, name_zh, name_en, description, icon, parent_id, sort_order, total_articles FROM scenarios WHERE parent_id IS NULL OR parent_id = '' ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Scenario
	for rows.Next() {
		var s model.Scenario
		if err := rows.Scan(&s.ID, &s.NameZh, &s.NameEn, &s.Description, &s.Icon, &s.ParentID, &s.SortOrder, &s.TotalArticles); err != nil {
			return list, err
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *ScenarioRepo) GetByID(id string) (*model.Scenario, error) {
	var s model.Scenario
	err := r.db.QueryRow(`SELECT id, name_zh, name_en, description, icon, parent_id, sort_order, total_articles FROM scenarios WHERE id = ?`, id).
		Scan(&s.ID, &s.NameZh, &s.NameEn, &s.Description, &s.Icon, &s.ParentID, &s.SortOrder, &s.TotalArticles)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ScenarioRepo) ListChildren(parentID string) ([]model.Scenario, error) {
	rows, err := r.db.Query(`SELECT id, name_zh, name_en, description, icon, parent_id, sort_order, total_articles FROM scenarios WHERE parent_id = ? ORDER BY sort_order`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Scenario
	for rows.Next() {
		var s model.Scenario
		if err := rows.Scan(&s.ID, &s.NameZh, &s.NameEn, &s.Description, &s.Icon, &s.ParentID, &s.SortOrder, &s.TotalArticles); err != nil {
			return list, err
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *ScenarioRepo) Upsert(s model.Scenario) error {
	_, err := r.db.Exec(`INSERT OR REPLACE INTO scenarios (id, name_zh, name_en, description, icon, parent_id, sort_order, total_articles) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.NameZh, s.NameEn, s.Description, s.Icon, s.ParentID, s.SortOrder, s.TotalArticles)
	return err
}
