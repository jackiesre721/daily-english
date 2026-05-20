package repository

import (
	"fmt"
	"daily-english/internal/model"
)

type DailyRepo struct {
	db *DB
}

func NewDailyRepo(db *DB) *DailyRepo {
	return &DailyRepo{db: db}
}

func (r *DailyRepo) GetByDate(userID, date string) (*model.DailyRecord, error) {
	var dr model.DailyRecord
	var checkin int
	err := r.db.QueryRow(
		`SELECT id, user_id, date, new_articles_learned, new_words_learned, articles_reviewed, correct_count, total_count, accuracy_rate, time_spent_seconds, checkin_completed
		FROM daily_records WHERE user_id = ? AND date = ?`, userID, date).
		Scan(&dr.ID, &dr.UserID, &dr.Date, &dr.NewArticlesLearned, &dr.NewWordsLearned, &dr.ArticlesReviewed, &dr.CorrectCount, &dr.TotalCount, &dr.AccuracyRate, &dr.TimeSpentSeconds, &checkin)
	dr.CheckinCompleted = checkin == 1
	if err != nil {
		return nil, err
	}
	return &dr, nil
}

func (r *DailyRepo) Upsert(dr *model.DailyRecord) error {
	checkin := 0
	if dr.CheckinCompleted {
		checkin = 1
	}
	_, err := r.db.Exec(
		`INSERT INTO daily_records (id, user_id, date, new_articles_learned, new_words_learned, articles_reviewed, correct_count, total_count, accuracy_rate, time_spent_seconds, checkin_completed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, date) DO UPDATE SET
			new_articles_learned = excluded.new_articles_learned,
			new_words_learned = excluded.new_words_learned,
			articles_reviewed = excluded.articles_reviewed,
			correct_count = excluded.correct_count,
			total_count = excluded.total_count,
			accuracy_rate = excluded.accuracy_rate,
			time_spent_seconds = excluded.time_spent_seconds,
			checkin_completed = excluded.checkin_completed`,
		dr.ID, dr.UserID, dr.Date, dr.NewArticlesLearned, dr.NewWordsLearned, dr.ArticlesReviewed, dr.CorrectCount, dr.TotalCount, dr.AccuracyRate, dr.TimeSpentSeconds, checkin)
	return err
}

func (r *DailyRepo) GetRecentAccuracy(userID string, days int) ([]float64, error) {
	rows, err := r.db.Query(
		`SELECT accuracy_rate FROM daily_records WHERE user_id = ? AND accuracy_rate > 0 ORDER BY date DESC LIMIT ?`, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rates []float64
	for rows.Next() {
		var r float64
		if err := rows.Scan(&r); err != nil {
			return rates, err
		}
		rates = append(rates, r)
	}
	return rates, nil
}

func (r *DailyRepo) GetMonthRecords(userID string, year, month int) ([]model.DailyRecord, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, date, new_articles_learned, new_words_learned, articles_reviewed, correct_count, total_count, accuracy_rate, time_spent_seconds, checkin_completed
		FROM daily_records WHERE user_id = ? AND strftime('%Y', date)=? AND strftime('%m', date)=? ORDER BY date`,
		userID, fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", month))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.DailyRecord
	for rows.Next() {
		var dr model.DailyRecord
		var checkin int
		if err := rows.Scan(&dr.ID, &dr.UserID, &dr.Date, &dr.NewArticlesLearned, &dr.NewWordsLearned, &dr.ArticlesReviewed, &dr.CorrectCount, &dr.TotalCount, &dr.AccuracyRate, &dr.TimeSpentSeconds, &checkin); err != nil {
			return list, err
		}
		dr.CheckinCompleted = checkin == 1
		list = append(list, dr)
	}
	return list, nil
}

func (r *DailyRepo) ListAll(userID string) ([]model.DailyRecord, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, date, new_articles_learned, new_words_learned, articles_reviewed, correct_count, total_count, accuracy_rate, time_spent_seconds, checkin_completed
		FROM daily_records WHERE user_id = ? ORDER BY date`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.DailyRecord
	for rows.Next() {
		var dr model.DailyRecord
		var checkin int
		if err := rows.Scan(&dr.ID, &dr.UserID, &dr.Date, &dr.NewArticlesLearned, &dr.NewWordsLearned, &dr.ArticlesReviewed, &dr.CorrectCount, &dr.TotalCount, &dr.AccuracyRate, &dr.TimeSpentSeconds, &checkin); err != nil {
			return list, err
		}
		dr.CheckinCompleted = checkin == 1
		list = append(list, dr)
	}
	return list, nil
}

func (r *DailyRepo) DeleteAll(userID string) error {
	_, err := r.db.Exec(`DELETE FROM daily_records WHERE user_id = ?`, userID)
	return err
}