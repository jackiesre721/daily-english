package handler

import (
	"encoding/json"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ArticleHandler struct {
	repo *repository.ArticleRepo
}

func NewArticleHandler(repo *repository.ArticleRepo) *ArticleHandler {
	return &ArticleHandler{repo: repo}
}

func (h *ArticleHandler) ListByScenario(c *gin.Context) {
	scenarioID := c.Param("scenarioId")
	articles, err := h.repo.ListByScenario(scenarioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if articles == nil {
		articles = make([]model.Article, 0)
	}
	c.JSON(http.StatusOK, gin.H{"articles": articles})
}

func (h *ArticleHandler) Get(c *gin.Context) {
	id := c.Param("id")
	article, err := h.repo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "article not found"})
		return
	}
	c.JSON(http.StatusOK, article)
}

func (h *ArticleHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.DeleteByID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *ArticleHandler) Import(c *gin.Context) {
	body, err := c.GetRawData()
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	var articles []model.Article
	if err := json.Unmarshal(body, &articles); err != nil {
		var wrapper struct {
			Articles []model.Article `json:"articles"`
		}
		if err2 := json.Unmarshal(body, &wrapper); err2 != nil || len(wrapper.Articles) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON format"})
			return
		}
		articles = wrapper.Articles
	}

	imported, err := h.repo.Import(articles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"imported": imported})
}

// ListAll returns a paginated summary of articles for the library view.
func (h *ArticleHandler) ListAll(c *gin.Context) {
	page := getIntParam(c, "page", 0)
	limit := getIntParam(c, "limit", 20)
	search := c.Query("search")
	source := c.Query("source")

	if page < 1 {
		// Backward compatible: return all when no page param
		if page == 0 {
			articles, _, err := h.repo.ListAllSummary(1, 99999, search, source)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if articles == nil {
				articles = make([]repository.ArticleSummary, 0)
			}
			c.JSON(http.StatusOK, gin.H{"articles": articles})
			return
		}
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	articles, total, err := h.repo.ListAllSummary(page, limit, search, source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if articles == nil {
		articles = make([]repository.ArticleSummary, 0)
	}
	pages := total / limit
	if total%limit > 0 {
		pages++
	}
	c.JSON(http.StatusOK, gin.H{
		"articles": articles,
		"total":    total,
		"page":     page,
		"limit":    limit,
		"pages":    pages,
	})
}

// ListSources returns distinct source values for filter chips.
func (h *ArticleHandler) ListSources(c *gin.Context) {
	sources, err := h.repo.ListAllSources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if sources == nil {
		sources = []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{}
	}
	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

// UpdateTags updates the tags of an article.
func (h *ArticleHandler) UpdateTags(c *gin.Context) {
	id := c.Param("id")
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.repo.UpdateTags(id, body.Tags); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
