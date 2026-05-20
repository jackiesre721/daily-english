package model

type User struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	StreakDays       int    `json:"streak_days"`
	MaxStreakDays    int    `json:"max_streak_days"`
	FreezeCards      int    `json:"freeze_cards"`
	DailyNewTarget   int    `json:"daily_new_target"`
	TotalWordsLearned int   `json:"total_words_learned"`
	LastCheckinDate  string `json:"last_checkin_date"`
	SettingsJSON     string `json:"settings_json"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type UserSettings struct {
	AIConfig *AIConfig `json:"ai,omitempty"`
}

type AIConfig struct {
	BaseURL   string `json:"base_url"`
	AuthToken string `json:"auth_token"`
	Model     string `json:"model"`
}
