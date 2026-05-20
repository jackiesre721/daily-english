package handler

import (
	"fmt"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ScenarioHandler struct {
	repo        *repository.ScenarioRepo
	learningRepo *repository.LearningRepo
}

func NewScenarioHandler(repo *repository.ScenarioRepo, lr *repository.LearningRepo) *ScenarioHandler {
	return &ScenarioHandler{repo: repo, learningRepo: lr}
}

func (h *ScenarioHandler) List(c *gin.Context) {
	scenarios, err := h.repo.ListTopLevel()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if scenarios == nil {
		c.JSON(http.StatusOK, make([]model.Scenario, 0))
		return
	}

	type ScenarioWithProgress struct {
		model.Scenario
		ArticlesLearned int     `json:"articles_learned"`
		ProgressPct     float64 `json:"progress_pct"`
	}

	learnedMap, _ := h.learningRepo.CountLearnedByScenarios("user_001")

	var result []ScenarioWithProgress
	for _, s := range scenarios {
		learned := learnedMap[s.ID]
		pct := 0.0
		if s.TotalArticles > 0 {
			pct = float64(learned) / float64(s.TotalArticles) * 100
		}
		result = append(result, ScenarioWithProgress{
			Scenario:        s,
			ArticlesLearned: learned,
			ProgressPct:     pct,
		})
	}
	c.JSON(http.StatusOK, result)
}

func (h *ScenarioHandler) Get(c *gin.Context) {
	id := c.Param("id")
	scenario, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scenario not found"})
		return
	}
	children, err := h.repo.ListChildren(id)
	if err != nil {
		fmt.Printf("scenario get: list children: %v\n", err)
	}
	if children == nil {
		children = make([]model.Scenario, 0)
	}
	learned, err := h.learningRepo.CountLearned("user_001", id)
	if err != nil {
		fmt.Printf("scenario get: count learned: %v\n", err)
	}
	type ScenarioWithProgress struct {
		model.Scenario
		ArticlesLearned int `json:"articles_learned"`
	}
	c.JSON(http.StatusOK, gin.H{
		"scenario": ScenarioWithProgress{Scenario: *scenario, ArticlesLearned: learned},
		"children": children,
	})
}
