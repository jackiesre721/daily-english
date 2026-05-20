package model

type Scenario struct {
	ID           string `json:"id"`
	NameZh       string `json:"name_zh"`
	NameEn       string `json:"name_en"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	ParentID     string `json:"parent_id,omitempty"`
	SortOrder    int    `json:"sort_order"`
	TotalArticles int   `json:"total_articles"`
	// computed
	ArticlesLearned int     `json:"articles_learned,omitempty"`
	ProgressPct     float64 `json:"progress_pct,omitempty"`
}
