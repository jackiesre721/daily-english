package handler

import (
	"encoding/json"
	"daily-english/internal/ai"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type VocabularyHandler struct {
	vocabRepo    *repository.VocabularyRepo
	userRepo     *repository.UserRepo
	articleRepo  *repository.ArticleRepo
	phraseRepo   *repository.PhraseRepo
	analysisRepo *repository.AnalysisRepo
	dictRepo     *repository.DictionaryRepo
}

func NewVocabularyHandler(vr *repository.VocabularyRepo, ur *repository.UserRepo, ar *repository.ArticleRepo, pr *repository.PhraseRepo, anr *repository.AnalysisRepo, dr *repository.DictionaryRepo) *VocabularyHandler {
	return &VocabularyHandler{vocabRepo: vr, userRepo: ur, articleRepo: ar, phraseRepo: pr, analysisRepo: anr, dictRepo: dr}
}

func (h *VocabularyHandler) loadAIClient() (*ai.Client, error) {
	user, err := h.userRepo.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to load user settings")
	}
	var settings model.UserSettings
	if user.SettingsJSON != "" {
		json.Unmarshal([]byte(user.SettingsJSON), &settings)
	}
	if settings.AIConfig == nil || settings.AIConfig.BaseURL == "" || settings.AIConfig.AuthToken == "" {
		return nil, fmt.Errorf("AI not configured")
	}
	return ai.NewClient(*settings.AIConfig), nil
}

// lookupWordDictOrAI finds a word's definition: DB cache → ECDICT dictionary → AI.
// Returns the vocabulary entry and the source ("cached", "ecdict", or "ai_lookup").
func (h *VocabularyHandler) lookupWordDictOrAI(word string) (*model.Vocabulary, string, error) {
	// 1. Check vocabulary DB cache
	existing, err := h.vocabRepo.FindByWord(word)
	if err == nil && existing != nil {
		return existing, "cached", nil
	}

	// 2. Try ECDICT dictionary
	if h.dictRepo != nil {
		entry, _ := h.dictRepo.LookupByForm(word)
		if entry != nil {
			now := time.Now().Format("2006-01-02 15:04:05")
			v := &model.Vocabulary{
				ID:             "v_" + word,
				Word:           word,
				Phonetic:       entry.Phonetic,
				Chinese:        entry.Translation,
				Explanation:    entry.Definition,
				Source:         "ecdict",
				Status:         "new",
				EncounterCount: 1,
				SourceArticles: "[]",
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			return v, "ecdict", nil
		}
	}

	// 3. Fallback to AI
	client, err := h.loadAIClient()
	if err != nil {
		return nil, "", err
	}

	result, err := client.LookupWord(word)
	if err != nil {
		return nil, "", err
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	v := &model.Vocabulary{
		ID:             "v_" + word,
		Word:           result.Word,
		Phonetic:       result.Phonetic,
		Chinese:        result.Chinese,
		Explanation:    result.Explanation,
		Source:         "ai_lookup",
		Status:         "new",
		EncounterCount: 1,
		SourceArticles: "[]",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	return v, "ai_lookup", nil
}

// LookupWord returns vocabulary entry from DB (no AI call).
func (h *VocabularyHandler) LookupWord(c *gin.Context) {
	word := c.Query("word")
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "word parameter required"})
		return
	}
	v, err := h.vocabRepo.FindByWord(word)
	if err != nil {
		// Try dictionary even if not in vocabulary
		if h.dictRepo != nil {
			entry, _ := h.dictRepo.LookupByForm(word)
			if entry != nil {
				c.JSON(http.StatusOK, gin.H{"found": true, "source": "ecdict", "word": entry.Word, "phonetic": entry.Phonetic, "chinese": entry.Translation, "definition": entry.Definition})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"found": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"found": true, "source": "cached", "vocabulary": v})
}

// AILookupAndAdd looks up a word, saves to vocabulary, links to article.
func (h *VocabularyHandler) AILookupAndAdd(c *gin.Context) {
	var req struct {
		Word      string `json:"word"`
		ArticleID string `json:"article_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	word := strings.ToLower(strings.TrimSpace(req.Word))
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "word is required"})
		return
	}

	v, source, err := h.lookupWordDictOrAI(word)
	if err != nil {
		if source == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		}
		return
	}

	if source != "cached" {
		if err := h.vocabRepo.Upsert(v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save vocabulary"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"vocabulary": v, "source": source, "in_study_list": false})
}

// TranslateSentence translates an English sentence to Chinese via AI.
func (h *VocabularyHandler) TranslateSentence(c *gin.Context) {
	var req struct {
		Sentence  string `json:"sentence"`
		ArticleID string `json:"article_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	sentence := strings.TrimSpace(req.Sentence)
	if sentence == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sentence is required"})
		return
	}

	client, err := h.loadAIClient()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := client.TranslateSentence(sentence)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "translation failed: " + err.Error()})
		return
	}

	if req.ArticleID != "" && h.articleRepo != nil {
		article, aerr := h.articleRepo.GetByID(req.ArticleID)
		if aerr == nil && article != nil {
			found := false
			for i := range article.Segments {
				if strings.TrimSpace(article.Segments[i].Text) == sentence {
					article.Segments[i].Translation = result.Translation
					found = true
					break
				}
			}
			if !found {
				article.Segments = append(article.Segments, model.Segment{
					Type:        "narration",
					Text:        sentence,
					Translation: result.Translation,
				})
			}
			if err := h.articleRepo.UpdateSegments(req.ArticleID, article.Segments); err != nil {
				fmt.Printf("warning: failed to save segments for article %s: %v\n", req.ArticleID, err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"translation": result.Translation})
}

// AnalyzeText analyzes selected text using AI and saves results.
func (h *VocabularyHandler) AnalyzeText(c *gin.Context) {
	var req struct {
		Text      string `json:"text"`
		ArticleID string `json:"article_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text is required"})
		return
	}
	wordCount := len(strings.Fields(text))
	if wordCount > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "selected text exceeds 500 words"})
		return
	}

	client, err := h.loadAIClient()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := client.AnalyzeText(text)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "analysis failed: " + err.Error()})
		return
	}

	articleID := req.ArticleID

	// Save analysis record
	if h.analysisRepo != nil && articleID != "" {
		analysis := &model.ArticleAnalysis{
			ID:        fmt.Sprintf("anl_%d", time.Now().UnixNano()),
			ArticleID: articleID,
			Text:      text,
			Analysis:  result.Summary,
			CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		}
		h.analysisRepo.Create(analysis)
	}

	// Save words to vocabulary
	if h.vocabRepo != nil && articleID != "" && len(result.Words) > 0 {
		for _, w := range result.Words {
			word := strings.ToLower(strings.TrimSpace(w.Word))
			if word == "" {
				continue
			}
			v := &model.Vocabulary{
				ID:          fmt.Sprintf("vw_%s", strings.ReplaceAll(word, " ", "_")),
				Word:        word,
				Phonetic:    w.Phonetic,
				Chinese:     w.Chinese,
				Explanation: w.Explanation,
				Source:      "ai_analyze",
				Status:      "new",
			}
			h.vocabRepo.Upsert(v)
			h.vocabRepo.LinkArticle(articleID, word, v.ID)
		}
	}

	// Save phrases
	if h.phraseRepo != nil && articleID != "" && len(result.Phrases) > 0 {
		for _, p := range result.Phrases {
			phrase := strings.TrimSpace(p.Phrase)
			if phrase == "" {
				continue
			}
			existing, _ := h.phraseRepo.FindByPhrase(phrase)
			if existing != nil {
				// Append source context
				h.phraseRepo.AddSourceContext(phrase, articleID, text)
			} else {
				contextJSON, _ := json.Marshal([]map[string]string{{"article_id": articleID, "sentence": text}})
				articlesJSON, _ := json.Marshal([]string{articleID})
				newPhrase := &model.Phrase{
					ID:             fmt.Sprintf("ph_%d", time.Now().UnixNano()),
					Phrase:         phrase,
					Chinese:        p.Chinese,
					Explanation:    p.Explanation,
					Category:       p.Category,
					Source:         "ai_analyze",
					SourceArticles: string(articlesJSON),
					SourceContext:  string(contextJSON),
					CreatedAt:      time.Now().Format("2006-01-02 15:04:05"),
					UpdatedAt:      time.Now().Format("2006-01-02 15:04:05"),
				}
				h.phraseRepo.Upsert(newPhrase)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": result.Summary,
		"translation": result.Translation,
		"words":    result.Words,
		"phrases":  result.Phrases,
	})
}

// ListPhrases returns all saved phrases.
func (h *VocabularyHandler) ListPhrases(c *gin.Context) {
	if h.phraseRepo == nil {
		c.JSON(http.StatusOK, gin.H{"phrases": []model.Phrase{}})
		return
	}
	phrases, err := h.phraseRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if phrases == nil {
		phrases = []model.Phrase{}
	}
	c.JSON(http.StatusOK, gin.H{"phrases": phrases})
}

// ListAnalyses returns all analysis records for an article.
func (h *VocabularyHandler) ListAnalyses(c *gin.Context) {
	articleID := c.Param("id")
	if h.analysisRepo == nil {
		c.JSON(http.StatusOK, gin.H{"analyses": []model.ArticleAnalysis{}})
		return
	}
	analyses, err := h.analysisRepo.ListByArticle(articleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if analyses == nil {
		analyses = []model.ArticleAnalysis{}
	}
	c.JSON(http.StatusOK, gin.H{"analyses": analyses})
}

// GetArticleVocabulary returns all vocabulary entries linked to an article.
func (h *VocabularyHandler) GetArticleVocabulary(c *gin.Context) {
	articleID := c.Param("articleId")
	words, err := h.vocabRepo.GetArticleVocabulary(articleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if words == nil {
		words = []repository.ArticleVocabulary{}
	}
	c.JSON(http.StatusOK, gin.H{"words": words})
}

// AddToStudyList ensures a word exists in vocabulary and links to article.
func (h *VocabularyHandler) AddToStudyList(c *gin.Context) {
	var req struct {
		Word      string `json:"word"`
		ArticleID string `json:"article_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	word := strings.ToLower(strings.TrimSpace(req.Word))
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "word is required"})
		return
	}

	v, source, err := h.lookupWordDictOrAI(word)
	if err != nil {
		if source == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "word not found and AI not configured"})
		} else {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		}
		return
	}

	if source != "cached" {
		if err := h.vocabRepo.Upsert(v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save vocabulary"})
			return
		}
	}

	if req.ArticleID != "" {
		h.vocabRepo.LinkArticle(req.ArticleID, word, v.ID)
		h.vocabRepo.AddToStudy(word, req.ArticleID, "user_add")
	}

	// Increment total_words_learned
	if user, err := h.userRepo.Get(); err == nil {
		user.TotalWordsLearned++
		h.userRepo.Update(user)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "vocabulary": v, "source": source})
}

// RemoveFromStudyList removes a word from vocabulary entirely.
func (h *VocabularyHandler) RemoveFromStudyList(c *gin.Context) {
	var req struct {
		Word string `json:"word"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	word := strings.ToLower(strings.TrimSpace(req.Word))
	if word == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "word is required"})
		return
	}
	if err := h.vocabRepo.Remove(word); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetArticleStudyStatus returns which words from an article are in the vocabulary.
func (h *VocabularyHandler) GetArticleStudyStatus(c *gin.Context) {
	articleID := c.Param("articleId")
	studyMap, err := h.vocabRepo.GetStudySetForArticle(articleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type wordStatus struct {
		InStudyList bool   `json:"in_study_list"`
		Status      string `json:"status"`
		Source      string `json:"source"`
	}
	result := make(map[string]wordStatus)
	for w, v := range studyMap {
		result[w] = wordStatus{InStudyList: true, Status: v.Status, Source: v.Source}
	}
	c.JSON(http.StatusOK, gin.H{"study_words": result})
}

// ListAllVocab returns all vocabulary entries.
func (h *VocabularyHandler) ListAllVocab(c *gin.Context) {
	words, err := h.vocabRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if words == nil {
		words = []model.Vocabulary{}
	}
	c.JSON(http.StatusOK, gin.H{"words": words})
}
