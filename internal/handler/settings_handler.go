package handler

import (
	"encoding/json"
	"daily-english/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	userRepo *repository.UserRepo
}

func NewSettingsHandler(u *repository.UserRepo) *SettingsHandler {
	return &SettingsHandler{userRepo: u}
}

func (h *SettingsHandler) Get(c *gin.Context) {
	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Strip auth_token from settings_json before sending to frontend
	safeSettings := user.SettingsJSON
	if safeSettings != "" {
		var raw map[string]interface{}
		if json.Unmarshal([]byte(safeSettings), &raw) == nil {
			if ai, ok := raw["ai"].(map[string]interface{}); ok {
				delete(ai, "auth_token")
			}
			if sanitized, err := json.Marshal(raw); err == nil {
				safeSettings = string(sanitized)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"name":          user.Name,
		"settings_json": safeSettings,
	})
}

type SettingsUpdateRequest struct {
	Name         string `json:"name"`
	SettingsJSON string `json:"settings_json"`
}

func (h *SettingsHandler) Update(c *gin.Context) {
	var req SettingsUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.SettingsJSON != "" {
		if !json.Valid([]byte(req.SettingsJSON)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "settings_json must be valid JSON"})
			return
		}
		user.SettingsJSON = req.SettingsJSON
	}

	if err := h.userRepo.Update(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
