package handler

import (
	"fmt"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DataHandler struct {
	userRepo     *repository.UserRepo
	learningRepo *repository.LearningRepo
	dailyRepo    *repository.DailyRepo
	db           *repository.DB
}

func NewDataHandler(u *repository.UserRepo, l *repository.LearningRepo, d *repository.DailyRepo, db *repository.DB) *DataHandler {
	return &DataHandler{userRepo: u, learningRepo: l, dailyRepo: d, db: db}
}

type ExportData struct {
	ExportedAt      string                   `json:"exported_at"`
	User            *model.User              `json:"user"`
	LearningProgress []model.LearningProgress `json:"learning_progress"`
	DailyRecords    []model.DailyRecord      `json:"daily_records"`
}

func (h *DataHandler) Export(c *gin.Context) {
	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	lp, err := h.learningRepo.ListAll("user_001")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if lp == nil {
		lp = []model.LearningProgress{}
	}

	dr, err := h.dailyRepo.ListAll("user_001")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if dr == nil {
		dr = []model.DailyRecord{}
	}

	data := ExportData{
		ExportedAt:      time.Now().Format("2006-01-02 15:04:05"),
		User:            user,
		LearningProgress: lp,
		DailyRecords:    dr,
	}

	filename := fmt.Sprintf("daily-english-backup-%s.json", time.Now().Format("20060102"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.JSON(http.StatusOK, data)
}

type ImportData struct {
	User             *model.User              `json:"user"`
	LearningProgress []model.LearningProgress `json:"learning_progress"`
	DailyRecords     []model.DailyRecord      `json:"daily_records"`
}

func (h *DataHandler) Import(c *gin.Context) {
	var data ImportData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data format"})
		return
	}

	imported := 0

	tx, txErr := h.db.BeginTx()
	if txErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": txErr.Error()})
		return
	}
	committed := false
	defer func() {
		if !committed {
			tx.Rollback()
		}
	}()

	// Restore user settings (not streak/progress)
	if data.User != nil {
		user, uerr := h.userRepo.Get()
		if uerr == nil {
			if data.User.Name != "" {
				user.Name = data.User.Name
			}
			if data.User.SettingsJSON != "" {
				user.SettingsJSON = data.User.SettingsJSON
			}
			h.userRepo.Update(user)
			imported++
		}
	}

	// Restore learning progress
	for _, lp := range data.LearningProgress {
		existing, lerr := h.learningRepo.GetProgress(lp.UserID, lp.ArticleID)
		if lerr != nil || existing == nil {
			h.learningRepo.CreateProgress(&lp)
		} else {
			h.learningRepo.UpdateProgress(&lp)
		}
		imported++
	}

	// Restore daily records
	for _, dr := range data.DailyRecords {
		h.dailyRepo.Upsert(&dr)
		imported++
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	committed = true
	c.JSON(http.StatusOK, gin.H{"status": "ok", "imported": imported})
}

func (h *DataHandler) Reset(c *gin.Context) {
	// Delete learning progress
	if err := h.learningRepo.DeleteAll("user_001"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete daily records
	if err := h.dailyRepo.DeleteAll("user_001"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Reset user progress counters
	if err := h.userRepo.ResetProgress(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
