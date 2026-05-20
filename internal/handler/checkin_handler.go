package handler

import (
	"fmt"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CheckinHandler struct {
	userRepo  *repository.UserRepo
	dailyRepo *repository.DailyRepo
}

func NewCheckinHandler(u *repository.UserRepo, d *repository.DailyRepo) *CheckinHandler {
	return &CheckinHandler{userRepo: u, dailyRepo: d}
}

func (h *CheckinHandler) Checkin(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Already checked in today
	if user.LastCheckinDate == today {
		c.JSON(http.StatusOK, gin.H{"status": "already", "streak_days": user.StreakDays})
		return
	}

	// Determine streak status
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	streakBroken := user.LastCheckinDate != yesterday && user.LastCheckinDate != ""
	useFreezeCard := streakBroken && user.FreezeCards > 0

	// Check if we'll award a freeze card (streak % 7 == 0 after increment)
	newStreak := user.StreakDays + 1
	if streakBroken && !useFreezeCard {
		newStreak = 1
	}
	awardFreezeCard := newStreak > 0 && newStreak%7 == 0

	// Atomic update to avoid race conditions
	user, err = h.userRepo.AtomicCheckin(today, yesterday, streakBroken, useFreezeCard, awardFreezeCard)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Upsert daily record
	dr, _ := h.dailyRepo.GetByDate("user_001", today)
	if dr == nil {
		dr = &model.DailyRecord{
			ID:     uuid.New().String()[:8],
			UserID: "user_001",
			Date:   today,
		}
	}
	dr.CheckinCompleted = true
	if err := h.dailyRepo.Upsert(dr); err != nil {
		fmt.Printf("checkin: upsert daily record: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"streak_days":  user.StreakDays,
		"max_streak":   user.MaxStreakDays,
		"freeze_cards": user.FreezeCards,
	})
}

func (h *CheckinHandler) GetCalendar(c *gin.Context) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if y := c.Query("year"); y != "" {
		fmt.Sscanf(y, "%d", &year)
	}
	if m := c.Query("month"); m != "" {
		fmt.Sscanf(m, "%d", &month)
	}

	records, err := h.dailyRepo.GetMonthRecords("user_001", year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type CalDay struct {
		Date    string `json:"date"`
		Checkin bool   `json:"checkin"`
		Learned int    `json:"learned"`
	}
	var days []CalDay
	for _, r := range records {
		days = append(days, CalDay{Date: r.Date, Checkin: r.CheckinCompleted, Learned: r.NewArticlesLearned})
	}

	c.JSON(http.StatusOK, gin.H{
		"year":         year,
		"month":        month,
		"streak_days":  user.StreakDays,
		"max_streak":   user.MaxStreakDays,
		"freeze_cards": user.FreezeCards,
		"days":         days,
	})
}
