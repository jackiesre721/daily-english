package model

type ImportSource struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	FileName     string `json:"file_name"`
	ScenarioID   string `json:"scenario_id"`
	ArticleCount int    `json:"article_count"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message,omitempty"`
	ContentHash  string `json:"content_hash,omitempty"`
	CreatedAt    string `json:"created_at"`
}
