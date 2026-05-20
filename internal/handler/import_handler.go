package handler

import (
	"encoding/json"
	"fmt"
	"daily-english/internal/ai"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// optimizeJob tracks async AI optimization progress
type optimizeJob struct {
	ArticleID string
	Status    string // pending, running, completed, failed
	Step      string // format, words, quizzes, done
	Error     string
	mu        sync.Mutex
}

type ImportHandler struct {
	importSourceRepo *repository.ImportSourceRepo
	articleRepo      *repository.ArticleRepo
	userRepo         *repository.UserRepo
	scenarioRepo     *repository.ScenarioRepo
	vocabRepo        *repository.VocabularyRepo
	jobs             map[string]*optimizeJob
	jobsMu           sync.Mutex
}

func NewImportHandler(isr *repository.ImportSourceRepo, ar *repository.ArticleRepo, ur *repository.UserRepo, sr *repository.ScenarioRepo, vr *repository.VocabularyRepo) *ImportHandler {
	h := &ImportHandler{
		importSourceRepo: isr,
		articleRepo:      ar,
		userRepo:         ur,
		scenarioRepo:     sr,
		vocabRepo:        vr,
		jobs:             make(map[string]*optimizeJob),
	}
	// Periodically clean up completed/failed jobs
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			h.jobsMu.Lock()
			for id, job := range h.jobs {
				job.mu.Lock()
				done := job.Status == "completed" || job.Status == "failed"
				job.mu.Unlock()
				if done {
					delete(h.jobs, id)
				}
			}
			h.jobsMu.Unlock()
		}
	}()
	return h
}

// ImportTxt saves the uploaded text as a raw article (no AI processing).
func (h *ImportHandler) ImportTxt(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".txt") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .txt files are supported"})
		return
	}

	content, err := io.ReadAll(io.LimitReader(file, 512*1024))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
		return
	}
	text := strings.TrimSpace(string(content))
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is empty"})
		return
	}

	contentHash := repository.ContentHash(text)
	existing, _ := h.importSourceRepo.FindByContentHash(contentHash)
	if existing != nil && existing.Status == "completed" {
		c.JSON(http.StatusConflict, gin.H{"error": "该文件已导入过", "source_id": existing.ID})
		return
	}

	name := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename))
	sourceID := uuid.New().String()[:12]
	source := &model.ImportSource{
		ID:          sourceID,
		Name:        name,
		FileName:    header.Filename,
		ScenarioID:  "ai_imported",
		Status:      "completed",
		ContentHash: contentHash,
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
	}
	h.importSourceRepo.Create(source)

	articleID := "art_" + sourceID
	article := &model.Article{
		ID:             articleID,
		ScenarioID:     "ai_imported",
		TitleZh:        name,
		TitleEn:        name,
		Type:           "raw",
		Difficulty:     0,
		Source:         "user_upload: " + header.Filename,
		ImportSourceID: sourceID,
		OriginalText:   text,
	}

	if _, err := h.articleRepo.Import([]model.Article{*article}); err != nil {
		source.Status = "failed"
		source.ErrorMessage = "save article: " + err.Error()
		h.importSourceRepo.Update(source)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save article"})
		return
	}

	source.ArticleCount = 1
	h.importSourceRepo.Update(source)
	h.ensureScenario()

	c.JSON(http.StatusOK, gin.H{"status": "ok", "article": article, "source": source})
}

// OptimizeArticle starts async AI optimization, returns immediately.
func (h *ImportHandler) OptimizeArticle(c *gin.Context) {
	articleID := c.Param("id")

	// Check if already running
	h.jobsMu.Lock()
	if job, exists := h.jobs[articleID]; exists && job.Status == "running" {
		h.jobsMu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "optimization already in progress", "step": job.Step})
		return
	}

	article, err := h.articleRepo.GetByID(articleID)
	if err != nil {
		h.jobsMu.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
		return
	}

	user, err := h.userRepo.Get()
	if err != nil {
		h.jobsMu.Unlock()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user settings"})
		return
	}
	var settings model.UserSettings
	if user.SettingsJSON != "" {
		json.Unmarshal([]byte(user.SettingsJSON), &settings)
	}
	if settings.AIConfig == nil || settings.AIConfig.BaseURL == "" || settings.AIConfig.AuthToken == "" {
		h.jobsMu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI not configured"})
		return
	}

	job := &optimizeJob{ArticleID: articleID, Status: "running", Step: "format"}
	h.jobs[articleID] = job
	h.jobsMu.Unlock()

	// Run optimization in background goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				job.mu.Lock()
				job.Status = "failed"
				job.Error = fmt.Sprintf("panic: %v", r)
				job.mu.Unlock()
				fmt.Printf("[Optimize] PANIC for %s: %v\n", articleID, r)
			}
		}()

		client := ai.NewClient(*settings.AIConfig)
		rawText := article.OriginalText

		result, err := client.OptimizeArticle(rawText, func(step string) {
			job.mu.Lock()
			job.Step = step
			job.mu.Unlock()
			fmt.Printf("[Optimize] %s article=%s step=%s\n", time.Now().Format("15:04:05"), articleID, step)
		})

		if err != nil {
			job.mu.Lock()
			job.Status = "failed"
			job.Error = err.Error()
			job.mu.Unlock()
			fmt.Printf("[Optimize] FAILED for %s: %v\n", articleID, err)
			return
		}

		// Preserve IDs
		result.ID = article.ID
		result.ImportSourceID = article.ImportSourceID
		result.ScenarioID = article.ScenarioID
		if result.Source == "" {
			result.Source = article.Source
		}

		// Save
		if _, err := h.articleRepo.Import([]model.Article{*result}); err != nil {
			job.mu.Lock()
			job.Status = "failed"
			job.Error = "save failed: " + err.Error()
			job.mu.Unlock()
			return
		}

		// Sync vocabulary
		if len(result.Words) > 0 {
			vocabIDs, err := h.vocabRepo.BatchUpsertForArticle(result.ID, result.Words)
			if err != nil {
				fmt.Printf("vocab sync warning: %v\n", err)
			}
			// Auto-add AI-recommended words to study list
			if len(vocabIDs) > 0 {
				if err := h.vocabRepo.BatchAddForArticle(result.ID, result.Words, vocabIDs); err != nil {
					fmt.Printf("study word sync warning: %v\n", err)
				}
			}
		}

		job.mu.Lock()
		job.Status = "completed"
		job.Step = "done"
		job.mu.Unlock()
		fmt.Printf("[Optimize] DONE for %s\n", articleID)
	}()

	c.JSON(http.StatusOK, gin.H{"status": "started", "article_id": articleID})
}

// OptimizeStatus returns the current optimization progress.
func (h *ImportHandler) OptimizeStatus(c *gin.Context) {
	articleID := c.Param("id")

	h.jobsMu.Lock()
	job, exists := h.jobs[articleID]
	h.jobsMu.Unlock()

	if !exists {
		// Check if article is already optimized (type != raw)
		article, err := h.articleRepo.GetByID(articleID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
			return
		}
		if article.Type != "raw" && len(article.Words) > 0 {
			c.JSON(http.StatusOK, gin.H{"status": "completed", "step": "done"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "idle"})
		return
	}

	job.mu.Lock()
	defer job.mu.Unlock()
	resp := gin.H{"status": job.Status, "step": job.Step}
	if job.Error != "" {
		resp["error"] = job.Error
	}
	c.JSON(http.StatusOK, resp)
}

func (h *ImportHandler) ListSources(c *gin.Context) {
	pageStr := c.Query("page")
	if pageStr != "" {
		page := getIntParam(c, "page", 1)
		limit := getIntParam(c, "limit", 20)
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 100 {
			limit = 20
		}
		sources, total, err := h.importSourceRepo.ListAllPaginated(page, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if sources == nil {
			sources = []model.ImportSource{}
		}
		pages := total / limit
		if total%limit > 0 {
			pages++
		}
		c.JSON(http.StatusOK, gin.H{
			"sources": sources,
			"total":   total,
			"page":    page,
			"limit":   limit,
			"pages":   pages,
		})
		return
	}
	sources, err := h.importSourceRepo.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sources == nil {
		sources = []model.ImportSource{}
	}
	c.JSON(http.StatusOK, sources)
}

func (h *ImportHandler) GetSourceArticles(c *gin.Context) {
	id := c.Param("id")
	articles, err := h.articleRepo.ListByImportSource(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if articles == nil {
		articles = []model.Article{}
	}
	c.JSON(http.StatusOK, gin.H{"articles": articles})
}

func (h *ImportHandler) DeleteSource(c *gin.Context) {
	id := c.Param("id")
	_, err := h.importSourceRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "import source not found"})
		return
	}
	h.importSourceRepo.Delete(id)
	h.ensureScenario()
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ImportHandler) ensureScenario() {
	total, _ := h.articleRepo.CountByScenario("ai_imported")
	h.scenarioRepo.Upsert(model.Scenario{
		ID: "ai_imported", NameZh: "AI导入", NameEn: "AI Imported",
		Description: "通过AI解析导入的文章", Icon: "🤖", SortOrder: 99, TotalArticles: total,
	})
}
