package handler

import (
	"encoding/json"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LearningHandler struct {
	articleRepo  *repository.ArticleRepo
	learningRepo *repository.LearningRepo
	userRepo     *repository.UserRepo
}

func NewLearningHandler(a *repository.ArticleRepo, l *repository.LearningRepo, u *repository.UserRepo) *LearningHandler {
	return &LearningHandler{articleRepo: a, learningRepo: l, userRepo: u}
}

func (h *LearningHandler) GetTodayArticles(c *gin.Context) {
	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	articles, err := h.articleRepo.GetUnlearned("user_001", user.DailyNewTarget)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if articles == nil {
		articles = make([]model.Article, 0)
	}
	c.JSON(http.StatusOK, gin.H{"articles": articles, "target": user.DailyNewTarget})
}

func (h *LearningHandler) GetArticle(c *gin.Context) {
	id := c.Param("id")
	article, err := h.articleRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
		return
	}
	c.JSON(http.StatusOK, article)
}

type StepRequest struct {
	ArticleID   string `json:"article_id"`
	Step        int    `json:"step"`
	QuizAnswers []struct {
		QuizIndex int  `json:"quiz_index"`
		Correct   bool `json:"correct"`
	} `json:"quiz_answers,omitempty"`
}

func (h *LearningHandler) SubmitStep(c *gin.Context) {
	var req StepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// get or create progress
	lp, err := h.learningRepo.GetProgress("user_001", req.ArticleID)
	if err != nil {
		lp = &model.LearningProgress{
			ID:         uuid.New().String()[:8],
			UserID:     "user_001",
			ArticleID:  req.ArticleID,
			Status:     "learning",
			EaseFactor: 2.5,
		}
	}

	lp.StepCompleted = req.Step

	// record quiz results
	if len(req.QuizAnswers) > 0 {
		results, _ := json.Marshal(req.QuizAnswers)
		lp.QuizResultsJSON = string(results)
		for _, a := range req.QuizAnswers {
			if a.Correct {
				lp.CorrectCount++
			} else {
				lp.WrongCount++
			}
		}
	}

	// if step 4 completed, set next review date (tomorrow)
	if req.Step >= 4 {
		lp.Status = "learning"
		lp.IntervalDays = 1
		lp.NextReviewDate = tomorrow()
		lp.LastReviewDate = today()
	}

	if err != nil {
		h.learningRepo.CreateProgress(lp)
	} else {
		h.learningRepo.UpdateProgress(lp)
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "step_completed": req.Step})
}

func today() string {
	return time.Now().Format("2006-01-02")
}

func tomorrow() string {
	return time.Now().AddDate(0, 0, 1).Format("2006-01-02")
}
