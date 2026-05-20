package model

type LearningProgress struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	ArticleID       string  `json:"article_id"`
	Status          string  `json:"status"` // new, learning, familiar, mastered
	StepCompleted   int     `json:"step_completed"`
	QuizResultsJSON string  `json:"quiz_results_json"`
	ReviewCount     int     `json:"review_count"`
	CorrectCount    int     `json:"correct_count"`
	WrongCount      int     `json:"wrong_count"`
	EaseFactor      float64 `json:"ease_factor"`
	IntervalDays    int     `json:"interval_days"`
	NextReviewDate  string  `json:"next_review_date,omitempty"`
	LastReviewDate  string  `json:"last_review_date,omitempty"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}
