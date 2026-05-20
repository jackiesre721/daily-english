package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"daily-english/internal/handler"
	"daily-english/internal/model"
	"daily-english/internal/repository"
	"daily-english/internal/router"

	"github.com/gin-gonic/gin"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"net/http/httputil"
	"net/url"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed migrations
var migrations embed.FS

//go:embed static
var staticFS embed.FS

type App struct {
	ctx     context.Context
	dataDir string
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAppDataDir() string {
	return a.dataDir
}

func (a *App) ChooseTextFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose a text file",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files", Pattern: "*.txt"},
			{DisplayName: "SRT Subtitles", Pattern: "*.srt"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
}

func getAppDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("get home dir: %v", err)
	}
	dir := filepath.Join(home, "Library", "Application Support", "DailyEnglish")
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("create app data dir: %v", err)
	}
	return dir
}

func main() {
	dataDir := getAppDataDir()
	dbPath := filepath.Join(dataDir, "dailyenglish.db")

	db := repository.InitDB(dbPath, migrations)
	defer db.Close()

	articleRepo := repository.NewArticleRepo(db)
	scenarioRepo := repository.NewScenarioRepo(db)
	userRepo := repository.NewUserRepo(db)
	learningRepo := repository.NewLearningRepo(db)
	dailyRepo := repository.NewDailyRepo(db)
	importSourceRepo := repository.NewImportSourceRepo(db)
	vocabRepo := repository.NewVocabularyRepo(db, articleRepo)
	phraseRepo := repository.NewPhraseRepo(db)
	analysisRepo := repository.NewAnalysisRepo(db)
	dictRepo := repository.OpenECDICT(dataDir)
	if dictRepo != nil {
		defer dictRepo.Close()
	}

	seedIfFirstRun(articleRepo, scenarioRepo, dataDir)

	if err := vocabRepo.BackfillFromArticles(); err != nil {
		fmt.Printf("vocabulary backfill: %v\n", err)
	}

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

	staticSub, _ := fs.Sub(staticFS, "static")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/" {
			path = "/home.html"
		}
		f, err := fs.ReadFile(staticSub, path[1:])
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, contentType(path), f)
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("find available port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	go func() {
		if err := r.Run(addr); err != nil {
			log.Fatalf("gin server error: %v", err)
		}
	}()

	fmt.Printf("gin server on %s\n", addr)

	app := &App{dataDir: dataDir}

	ginURL, _ := url.Parse("http://" + addr)
	proxy := httputil.NewSingleHostReverseProxy(ginURL)

	if err := wails.Run(&options.App{
		Title:     "Daily English",
		Width:     1200,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Handler: proxy,
		},
		OnStartup: app.startup,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHiddenInset(),
			About: &mac.AboutInfo{
				Title:   "Daily English",
				Message: "English learning companion\nv1.0.0",
			},
		},
	}); err != nil {
		log.Fatalf("wails error: %v", err)
	}
}

func contentType(path string) string {
	switch filepath.Ext(path) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".svg":
		return "image/svg+xml"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func seedIfFirstRun(articleRepo *repository.ArticleRepo, scenarioRepo *repository.ScenarioRepo, dataDir string) {
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
