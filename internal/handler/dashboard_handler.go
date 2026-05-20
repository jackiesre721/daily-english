package handler

import (
	"fmt"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	userRepo     *repository.UserRepo
	articleRepo  *repository.ArticleRepo
	learningRepo *repository.LearningRepo
	scenarioRepo *repository.ScenarioRepo
}

func NewDashboardHandler(u *repository.UserRepo, a *repository.ArticleRepo, l *repository.LearningRepo, s *repository.ScenarioRepo) *DashboardHandler {
	return &DashboardHandler{userRepo: u, articleRepo: a, learningRepo: l, scenarioRepo: s}
}

func (h *DashboardHandler) Get(c *gin.Context) {
	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	today := time.Now().Format("2006-01-02")

	// count today's learned
	todayLearned, err := h.learningRepo.CountTodayLearned("user_001", today)
	if err != nil {
		fmt.Printf("dashboard: count today learned: %v\n", err)
	}

	// count due reviews
	dueReviews, err := h.learningRepo.CountDueReviews("user_001", today)
	if err != nil {
		fmt.Printf("dashboard: count due reviews: %v\n", err)
	}

	// get scenarios with progress (single query)
	scenarios, err := h.scenarioRepo.ListTopLevel()
	if err != nil {
		fmt.Printf("dashboard: list scenarios: %v\n", err)
	}
	learnedMap, err := h.learningRepo.CountLearnedByScenarios("user_001")
	if err != nil {
		fmt.Printf("dashboard: count learned by scenarios: %v\n", err)
	}

	type ScenarioWithProgress struct {
		model.Scenario
		ArticlesLearned int     `json:"articles_learned"`
		ProgressPct     float64 `json:"progress_pct"`
	}
	var scenariosProgress []ScenarioWithProgress
	for _, s := range scenarios {
		learned := learnedMap[s.ID]
		pct := 0.0
		if s.TotalArticles > 0 {
			pct = float64(learned) / float64(s.TotalArticles) * 100
		}
		scenariosProgress = append(scenariosProgress, ScenarioWithProgress{
			Scenario:        s,
			ArticlesLearned: learned,
			ProgressPct:     pct,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"user":   user,
		"today":  gin.H{"date": today, "new_articles_done": todayLearned, "review_due": dueReviews},
		"tasks": gin.H{
			"learn":  gin.H{"done": todayLearned, "completed": todayLearned > 0},
			"review": gin.H{"done": 0, "total": dueReviews, "completed": dueReviews == 0},
		},
		"scenarios_recent": scenariosProgress,
	})
}
