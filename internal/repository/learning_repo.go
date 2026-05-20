package repository

import (
	"daily-english/internal/model"
)

type LearningRepo struct {
	db *DB
}

func NewLearningRepo(db *DB) *LearningRepo {
	return &LearningRepo{db: db}
}

func (r *LearningRepo) GetProgress(userID, articleID string) (*model.LearningProgress, error) {
	var lp model.LearningProgress
	err := r.db.QueryRow(
		`SELECT id, user_id, article_id, status, step_completed, quiz_results_json, review_count, correct_count, wrong_count, ease_factor, interval_days, next_review_date, last_review_date, created_at, updated_at
		FROM learning_progress WHERE user_id = ? AND article_id = ?`, userID, articleID).
		Scan(&lp.ID, &lp.UserID, &lp.ArticleID, &lp.Status, &lp.StepCompleted, &lp.QuizResultsJSON, &lp.ReviewCount, &lp.CorrectCount, &lp.WrongCount, &lp.EaseFactor, &lp.IntervalDays, &lp.NextReviewDate, &lp.LastReviewDate, &lp.CreatedAt, &lp.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &lp, nil
}

func (r *LearningRepo) CreateProgress(lp *model.LearningProgress) error {
	_, err := r.db.Exec(
		`INSERT OR REPLACE INTO learning_progress (id, user_id, article_id, status, step_completed, quiz_results_json, review_count, correct_count, wrong_count, ease_factor, interval_days, next_review_date, last_review_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now', 'localtime'), datetime('now', 'localtime'))`,
		lp.ID, lp.UserID, lp.ArticleID, lp.Status, lp.StepCompleted, lp.QuizResultsJSON, lp.ReviewCount, lp.CorrectCount, lp.WrongCount, lp.EaseFactor, lp.IntervalDays, lp.NextReviewDate, lp.LastReviewDate)
	return err
}

func (r *LearningRepo) UpdateProgress(lp *model.LearningProgress) error {
	_, err := r.db.Exec(
		`UPDATE learning_progress SET status=?, step_completed=?, quiz_results_json=?, review_count=?, correct_count=?, wrong_count=?, ease_factor=?, interval_days=?, next_review_date=?, last_review_date=?, updated_at=datetime('now', 'localtime')
		WHERE id=?`,
		lp.Status, lp.StepCompleted, lp.QuizResultsJSON, lp.ReviewCount, lp.CorrectCount, lp.WrongCount, lp.EaseFactor, lp.IntervalDays, lp.NextReviewDate, lp.LastReviewDate, lp.ID)
	return err
}

func (r *LearningRepo) CountLearned(userID, scenarioID string) (int, error) {
	var count int
	var err error
	if scenarioID == "" {
		err = r.db.QueryRow(
			`SELECT COUNT(*) FROM learning_progress WHERE user_id = ? AND status != 'new'`, userID).Scan(&count)
	} else {
		err = r.db.QueryRow(
			`SELECT COUNT(*) FROM learning_progress lp
			JOIN articles a ON a.id = lp.article_id
			WHERE lp.user_id = ? AND a.scenario_id = ? AND lp.status != 'new'`, userID, scenarioID).Scan(&count)
	}
	return count, err
}

// CountLearnedByScenarios returns a map of scenarioID -> learned count in a single query.
func (r *LearningRepo) CountLearnedByScenarios(userID string) (map[string]int, error) {
	rows, err := r.db.Query(
		`SELECT a.scenario_id, COUNT(*) FROM learning_progress lp
		JOIN articles a ON a.id = lp.article_id
		WHERE lp.user_id = ? AND lp.status != 'new'
		GROUP BY a.scenario_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]int)
	for rows.Next() {
		var scenarioID string
		var count int
		if err := rows.Scan(&scenarioID, &count); err == nil {
			result[scenarioID] = count
		}
	}
	return result, nil
}

func (r *LearningRepo) CountTodayLearned(userID, date string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM learning_progress WHERE user_id = ? AND date(updated_at, 'localtime') = ? AND step_completed >= 4`, userID, date).Scan(&count)
	return count, err
}

func (r *LearningRepo) GetDueReviews(userID, date string, limit int) ([]model.LearningProgress, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, article_id, status, step_completed, quiz_results_json, review_count, correct_count, wrong_count, ease_factor, interval_days, next_review_date, last_review_date, created_at, updated_at
		FROM learning_progress
		WHERE user_id = ? AND status != 'new' AND next_review_date <= ? AND step_completed >= 4
		ORDER BY next_review_date ASC LIMIT ?`, userID, date, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.LearningProgress
	for rows.Next() {
		var lp model.LearningProgress
		if err := rows.Scan(&lp.ID, &lp.UserID, &lp.ArticleID, &lp.Status, &lp.StepCompleted, &lp.QuizResultsJSON, &lp.ReviewCount, &lp.CorrectCount, &lp.WrongCount, &lp.EaseFactor, &lp.IntervalDays, &lp.NextReviewDate, &lp.LastReviewDate, &lp.CreatedAt, &lp.UpdatedAt); err != nil {
			return list, err
		}
		list = append(list, lp)
	}
	return list, nil
}

func (r *LearningRepo) CountDueReviews(userID, date string) (int, error) {
	var count int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM learning_progress
		WHERE user_id = ? AND status != 'new' AND next_review_date <= ? AND step_completed >= 4`, userID, date).Scan(&count)
	return count, err
}

func (r *LearningRepo) ListAll(userID string) ([]model.LearningProgress, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, article_id, status, step_completed, quiz_results_json, review_count, correct_count, wrong_count, ease_factor, interval_days, next_review_date, last_review_date, created_at, updated_at
		FROM learning_progress WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.LearningProgress
	for rows.Next() {
		var lp model.LearningProgress
		if err := rows.Scan(&lp.ID, &lp.UserID, &lp.ArticleID, &lp.Status, &lp.StepCompleted, &lp.QuizResultsJSON, &lp.ReviewCount, &lp.CorrectCount, &lp.WrongCount, &lp.EaseFactor, &lp.IntervalDays, &lp.NextReviewDate, &lp.LastReviewDate, &lp.CreatedAt, &lp.UpdatedAt); err != nil {
			return list, err
		}
		list = append(list, lp)
	}
	return list, nil
}

func (r *LearningRepo) DeleteAll(userID string) error {
	_, err := r.db.Exec(`DELETE FROM learning_progress WHERE user_id = ?`, userID)
	return err
}
