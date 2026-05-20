package model

type DailyRecord struct {
	ID                  string  `json:"id"`
	UserID              string  `json:"user_id"`
	Date                string  `json:"date"`
	NewArticlesLearned  int     `json:"new_articles_learned"`
	NewWordsLearned     int     `json:"new_words_learned"`
	ArticlesReviewed    int     `json:"articles_reviewed"`
	CorrectCount        int     `json:"correct_count"`
	TotalCount          int     `json:"total_count"`
	AccuracyRate        float64 `json:"accuracy_rate"`
	TimeSpentSeconds    int     `json:"time_spent_seconds"`
	CheckinCompleted    bool    `json:"checkin_completed"`
}
