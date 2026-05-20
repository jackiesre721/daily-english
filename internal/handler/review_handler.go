package handler

import (
	"encoding/json"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ReviewHandler struct {
	learningRepo *repository.LearningRepo
	articleRepo  *repository.ArticleRepo
	userRepo     *repository.UserRepo
}

func NewReviewHandler(l *repository.LearningRepo, a *repository.ArticleRepo, u *repository.UserRepo) *ReviewHandler {
	return &ReviewHandler{learningRepo: l, articleRepo: a, userRepo: u}
}

func (h *ReviewHandler) GetQueue(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	// Read user review limit from settings
	limit := 20
	if user, err := h.userRepo.Get(); err == nil && user.SettingsJSON != "" {
		var settings struct {
			ReviewLimit int `json:"review_limit"`
		}
		if json.Unmarshal([]byte(user.SettingsJSON), &settings) == nil && settings.ReviewLimit > 0 {
			limit = settings.ReviewLimit
		}
	}

	queue, err := h.learningRepo.GetDueReviews("user_001", today, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if queue == nil {
		queue = make([]model.LearningProgress, 0)
	}
	// Attach article info (single query)
	type ReviewItem struct {
		model.LearningProgress
		TitleZh string `json:"title_zh"`
		TitleEn string `json:"title_en"`
	}
	ids := make([]string, len(queue))
	for i, lp := range queue {
		ids[i] = lp.ArticleID
	}
	titles, _ := h.articleRepo.TitlesByIDs(ids)

	var items []ReviewItem
	for _, lp := range queue {
		titleZh, titleEn := "", ""
		if t, ok := titles[lp.ArticleID]; ok {
			titleZh = t.TitleZh
			titleEn = t.TitleEn
		}
		items = append(items, ReviewItem{
			LearningProgress: lp,
			TitleZh:          titleZh,
			TitleEn:          titleEn,
		})
	}
	c.JSON(http.StatusOK, gin.H{"queue": items, "total": len(items)})
}

type ReviewAnswerRequest struct {
	ArticleID string `json:"article_id"`
	Quality   int    `json:"quality"` // 1-5
}

func (h *ReviewHandler) SubmitAnswer(c *gin.Context) {
	var req ReviewAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Quality < 1 || req.Quality > 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quality must be between 1 and 5"})
		return
	}

	lp, err := h.learningRepo.GetProgress("user_001", req.ArticleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "progress not found"})
		return
	}

	// SM-2 algorithm
	lp.ReviewCount++
	if req.Quality >= 3 {
		lp.CorrectCount++
	} else {
		lp.WrongCount++
	}

	switch {
	case req.Quality >= 5:
		lp.IntervalDays = int(float64(lp.IntervalDays) * lp.EaseFactor)
		if lp.IntervalDays < 1 {
			lp.IntervalDays = 1
		}
		lp.EaseFactor = lp.EaseFactor + 0.05
	case req.Quality == 4:
		lp.IntervalDays = int(float64(lp.IntervalDays) * lp.EaseFactor * 0.9)
		if lp.IntervalDays < 1 {
			lp.IntervalDays = 1
		}
	case req.Quality == 3:
		lp.IntervalDays = int(float64(lp.IntervalDays) * lp.EaseFactor * 0.7)
		if lp.IntervalDays < 1 {
			lp.IntervalDays = 1
		}
		lp.EaseFactor = lp.EaseFactor - 0.05
		if lp.EaseFactor < 1.3 {
			lp.EaseFactor = 1.3
		}
	default:
		// Quality 1-2: reset interval
		lp.IntervalDays = 1
		lp.EaseFactor = lp.EaseFactor - 0.1
		if lp.EaseFactor < 1.3 {
			lp.EaseFactor = 1.3
		}
	}

	lp.LastReviewDate = time.Now().Format("2006-01-02")
	lp.NextReviewDate = time.Now().AddDate(0, 0, lp.IntervalDays).Format("2006-01-02")

	switch {
	case lp.ReviewCount >= 7 && lp.IntervalDays >= 30 && req.Quality >= 4:
		lp.Status = "mastered"
	case lp.ReviewCount >= 3 && req.Quality >= 3:
		lp.Status = "familiar"
	default:
		lp.Status = "learning"
	}

	if err := h.learningRepo.UpdateProgress(lp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "ok",
		"next_review":    lp.NextReviewDate,
		"interval_days":  lp.IntervalDays,
		"ease_factor":    lp.EaseFactor,
	})
}
