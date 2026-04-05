package app

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"knowflow/internal/transport/http/middleware"
)

func NewRouter(app *App) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	if app != nil {
		router.Use(middleware.RequestContext())
		router.Use(middleware.Logging(app.Logger))

		if app.Metrics != nil {
			router.GET("/metrics", gin.WrapH(app.Metrics.Handler()))
		}

		api := router.Group("/api")
		{
			if app.DocumentHandler != nil {
				api.POST("/kb/documents", app.DocumentHandler.Upload)
			}
			if app.ChatHandler != nil {
				api.POST("/chat/query", app.ChatHandler.Query)
				api.POST("/chat/query/stream", app.ChatHandler.QueryStream)
				api.GET("/chat/sessions", app.ChatHandler.ListSessions)
				api.GET("/chat/sessions/:session_id/messages", app.ChatHandler.ListMessages)
			}
			if app.KnowledgeHandler != nil {
				api.POST("/kb/knowledge", app.KnowledgeHandler.Upsert)
				api.POST("/kb/reindex", app.KnowledgeHandler.Reindex)
			}
		}
	}

	return router
}
