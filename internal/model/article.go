package model

type Word struct {
	WordID       string   `json:"word_id"`
	TextInArticle string  `json:"text_in_article"`
	StartPos     int      `json:"start_pos"`
	EndPos       int      `json:"end_pos"`
	English      string   `json:"english"`
	Phonetic     string   `json:"phonetic"`
	Chinese      string   `json:"chinese"`
	Explanation  string   `json:"explanation"`
	MemoryTip            string   `json:"memory_tip"`
	RelatedTerms         []string `json:"related_terms"`
	RecommendedForStudy  bool     `json:"recommended_for_study"`
}

type Quiz struct {
	Type          string   `json:"type"`           // fill_blank, understand, reorder
	Question      string   `json:"question"`
	Hint          string   `json:"hint,omitempty"`
	Options       []string `json:"options,omitempty"`
	Answer        interface{} `json:"answer"`         // int for fill_blank/understand, string for reorder
	AnswerStr     string   `json:"answer_text,omitempty"` // for reorder type
	Words         []string `json:"words,omitempty"`       // for reorder type
	TargetWordID  string   `json:"target_word_id"`
	TargetWordIDs []string `json:"target_word_ids,omitempty"`
}

type Article struct {
	ID            string `json:"id"`
	ScenarioID    string `json:"scenario_id"`
	TitleZh       string `json:"title_zh"`
	TitleEn       string `json:"title_en"`
	Type          string `json:"type"` // sentence, dialogue, passage, article
	Difficulty    int    `json:"difficulty"`
	Source          string `json:"source"`
	ImportSourceID  string `json:"import_source_id,omitempty"`
	OriginalText    string `json:"original_text"`
	Translation   string `json:"translation"`
	ContextNote   string `json:"context_note"`
	Segments      []Segment `json:"segments,omitempty"`
	Words         []Word `json:"words"`
	Quizzes       []Quiz `json:"quizzes"`
	Takeaway      string `json:"takeaway"`
	Tags          []string `json:"tags"`
	HTMLContent   string `json:"html_content"`
	CreatedAt     string `json:"created_at,omitempty"`
}

type Segment struct {
	Type        string `json:"type"`    // dialogue, narration, action
	Speaker     string `json:"speaker,omitempty"`
	Text        string `json:"text"`
	Translation string `json:"translation,omitempty"`
}

// Vocabulary is a word entry in the user's study list.
type Vocabulary struct {
	ID             string `json:"id"`
	Word           string `json:"word"`
	Phonetic       string `json:"phonetic"`
	Chinese        string `json:"chinese"`
	Explanation    string `json:"explanation"`
	Source         string `json:"source"`              // ai_import, user_add, ai_lookup
	Status         string `json:"status"`              // new, learning, mastered
	EncounterCount int    `json:"encounter_count"`
	SourceArticles string `json:"source_articles"`     // JSON array of article IDs
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

// Phrase stores a useful phrase/expression extracted from articles.
type Phrase struct {
	ID             string `json:"id"`
	Phrase         string `json:"phrase"`
	Chinese        string `json:"chinese"`
	Explanation    string `json:"explanation"`
	Category       string `json:"category"` // phrase, sentence, expression
	Source         string `json:"source"`
	SourceArticles string `json:"source_articles"` // JSON array of article IDs
	SourceContext  string `json:"source_context"`  // JSON [{article_id, sentence}]
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

// ArticleAnalysis stores an AI analysis result for a text selection.
type ArticleAnalysis struct {
	ID        string `json:"id"`
	ArticleID string `json:"article_id"`
	Text      string `json:"text"`
	Analysis  string `json:"analysis"`
	CreatedAt string `json:"created_at"`
}
