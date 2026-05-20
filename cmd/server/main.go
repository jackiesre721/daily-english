package main

import (
	"encoding/json"
	"fmt"
	"daily-english/internal/config"
	"daily-english/internal/handler"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"daily-english/internal/router"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db := repository.InitDB(cfg.DBPath)
	defer db.Close()

	// repos
	articleRepo := repository.NewArticleRepo(db)
	scenarioRepo := repository.NewScenarioRepo(db)
	userRepo := repository.NewUserRepo(db)
	learningRepo := repository.NewLearningRepo(db)
	dailyRepo := repository.NewDailyRepo(db)
	importSourceRepo := repository.NewImportSourceRepo(db)
	vocabRepo := repository.NewVocabularyRepo(db, articleRepo)
	phraseRepo := repository.NewPhraseRepo(db)
	analysisRepo := repository.NewAnalysisRepo(db)
	dictRepo := repository.OpenECDICT(cfg.Data)
	if dictRepo != nil {
		defer dictRepo.Close()
	}

	// seed only on first run (when scenario table is empty)
	seedIfFirstRun(articleRepo, scenarioRepo, cfg.Data)

	// backfill vocabulary from existing articles
	if err := vocabRepo.BackfillFromArticles(); err != nil {
		fmt.Printf("vocabulary backfill: %v\n", err)
	}

	// handlers
	articleHandler := handler.NewArticleHandler(articleRepo)
	scenarioHandler := handler.NewScenarioHandler(scenarioRepo, learningRepo)
	dashboardHandler := handler.NewDashboardHandler(userRepo, articleRepo, learningRepo, scenarioRepo)
	learningHandler := handler.NewLearningHandler(articleRepo, learningRepo, userRepo)
	reviewHandler := handler.NewReviewHandler(learningRepo, articleRepo, userRepo)
	checkinHandler := handler.NewCheckinHandler(userRepo, dailyRepo)
	statsHandler := handler.NewStatsHandler(userRepo, learningRepo, dailyRepo)
	settingsHandler := handler.NewSettingsHandler(userRepo)
	dataHandler := handler.NewDataHandler(userRepo, learningRepo, dailyRepo, db)
	importHandler := handler.NewImportHandler(importSourceRepo, articleRepo, userRepo, scenarioRepo, vocabRepo)
	vocabHandler := handler.NewVocabularyHandler(vocabRepo, userRepo, articleRepo, phraseRepo, analysisRepo, dictRepo)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	router.RegisterAPIRoutes(api, articleHandler, scenarioHandler, dashboardHandler, learningHandler, reviewHandler, checkinHandler, statsHandler, settingsHandler, dataHandler, importHandler, vocabHandler)
	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.Static("/css", cfg.Static+"/css")
	r.Static("/js", cfg.Static+"/js")
	r.StaticFile("/", cfg.Static+"/home.html")
	for _, f := range []string{"home", "import", "settings", "article", "theme-preview"} {
		r.StaticFile("/"+f+".html", cfg.Static+"/"+f+".html")
	}

	fmt.Printf("server running at http://localhost:%s\n", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		fmt.Printf("server error: %v\n", err)
	}
}

func seedIfFirstRun(articleRepo *repository.ArticleRepo, scenarioRepo *repository.ScenarioRepo, dataDir string) {
	// Only seed if the scenarios table is empty — means never seeded before
	scenarios, _ := scenarioRepo.ListTopLevel()
	if len(scenarios) > 0 {
		return
	}
	data, err := os.ReadFile(dataDir + "/articles_peppa.json")
	if err != nil {
		fmt.Println("no seed data found, skipping")
		return
	}
	var articles []model.Article
	if err := json.Unmarshal(data, &articles); err != nil {
		fmt.Println("parse seed data error:", err)
		return
	}
	imported, err := articleRepo.Import(articles)
	if err != nil {
		fmt.Printf("seed: import articles: %v\n", err)
		return
	}
	scenarioRepo.Upsert(model.Scenario{
		ID: "daily_social", NameZh: "日常社交", NameEn: "Daily Social",
		Description: "日常生活中的英语表达", Icon: "💬", SortOrder: 1, TotalArticles: imported,
	})
	fmt.Printf("seeded %d articles\n", imported)
}
