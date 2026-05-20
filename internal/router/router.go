package router

import (
	"daily-english/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(
	api *gin.RouterGroup,
	ah *handler.ArticleHandler,
	sh *handler.ScenarioHandler,
	dh *handler.DashboardHandler,
	lh *handler.LearningHandler,
	rh *handler.ReviewHandler,
	ch *handler.CheckinHandler,
	sth *handler.StatsHandler,
	seh *handler.SettingsHandler,
	dah *handler.DataHandler,
	ih *handler.ImportHandler,
	vh *handler.VocabularyHandler,
) {
	// dashboard
	api.GET("/dashboard", dh.Get)

	// articles
	api.GET("/articles", ah.ListAll)
	api.GET("/articles/sources", ah.ListSources)
	api.GET("/articles/scenario/:scenarioId", ah.ListByScenario)
	api.GET("/articles/:id", ah.Get)
	api.PUT("/articles/:id/tags", ah.UpdateTags)
	api.DELETE("/articles/:id", ah.Delete)
	api.POST("/articles/import", ah.Import)

	// scenarios
	api.GET("/scenarios", sh.List)
	api.GET("/scenarios/:id", sh.Get)

	// learning
	api.GET("/learn/today", lh.GetTodayArticles)
	api.GET("/learn/article/:id", lh.GetArticle)
	api.POST("/learn/step", lh.SubmitStep)

	// review
	api.GET("/review/queue", rh.GetQueue)
	api.POST("/review/answer", rh.SubmitAnswer)

	// checkin
	api.POST("/checkin", ch.Checkin)
	api.GET("/checkin/calendar", ch.GetCalendar)

	// stats
	api.GET("/stats/overview", sth.Overview)
	api.GET("/stats/trend", sth.Trend)

	// settings
	api.GET("/settings", seh.Get)
	api.PUT("/settings", seh.Update)

	// data management
	api.GET("/data/export", dah.Export)
	api.POST("/data/import", dah.Import)
	api.POST("/data/reset", dah.Reset)

	// AI article import
	api.POST("/articles/import-txt", ih.ImportTxt)
	api.POST("/articles/import-url", ih.ImportURL)
	api.POST("/articles/optimize/:id", ih.OptimizeArticle)
	api.GET("/articles/optimize-status/:id", ih.OptimizeStatus)
	api.GET("/import-sources", ih.ListSources)
	api.GET("/import-sources/:id/articles", ih.GetSourceArticles)
	api.DELETE("/import-sources/:id", ih.DeleteSource)

	// vocabulary
	api.GET("/vocabulary/lookup", vh.LookupWord)
	api.POST("/vocabulary/ai-lookup", vh.AILookupAndAdd)
	api.GET("/vocabulary/article-words/:articleId", vh.GetArticleVocabulary)
	api.POST("/articles/translate-sentence", vh.TranslateSentence)
	api.POST("/articles/analyze", vh.AnalyzeText)
	api.GET("/phrases", vh.ListPhrases)
	api.GET("/articles/:id/analyses", vh.ListAnalyses)

	// study list (生词库)
	api.GET("/vocabulary/list", vh.ListAllVocab)
	api.POST("/vocabulary/study/add", vh.AddToStudyList)
	api.POST("/vocabulary/study/remove", vh.RemoveFromStudyList)
	api.GET("/vocabulary/study/status/:articleId", vh.GetArticleStudyStatus)
}
