package handler

import (
	"fmt"
	"daily-english/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	userRepo    *repository.UserRepo
	learningRepo *repository.LearningRepo
	dailyRepo   *repository.DailyRepo
}

func NewStatsHandler(u *repository.UserRepo, l *repository.LearningRepo, d *repository.DailyRepo) *StatsHandler {
	return &StatsHandler{userRepo: u, learningRepo: l, dailyRepo: d}
}

func (h *StatsHandler) Overview(c *gin.Context) {
	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	records, err := h.dailyRepo.GetRecentAccuracy("user_001", 30)
	if err != nil {
		fmt.Printf("stats: get recent accuracy: %v\n", err)
	}

	totalLearned, err := h.learningRepo.CountLearned("user_001", "")
	if err != nil {
		fmt.Printf("stats: count learned: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_words":     user.TotalWordsLearned,
		"total_articles":  totalLearned,
		"streak_days":     user.StreakDays,
		"max_streak":      user.MaxStreakDays,
		"freeze_cards":    user.FreezeCards,
		"recent_accuracy": records,
		"daily_target":    user.DailyNewTarget,
	})
}

func (h *StatsHandler) Trend(c *gin.Context) {
	now := time.Now()
	records, err := h.dailyRepo.GetMonthRecords("user_001", now.Year(), int(now.Month()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"records": records})
}
