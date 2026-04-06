package app

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"knowflow/internal/transport/http/handler"
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
	}

	playground := handler.NewPlaygroundHandler()
	router.GET("/playground", playground.Page)
	router.StaticFS("/playground/assets", playground.AssetsFS())

	if app != nil {
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
				api.GET("/kb/knowledge", app.KnowledgeHandler.List)
				api.GET("/kb/knowledge/:knowledge_id", app.KnowledgeHandler.Get)
				api.PUT("/kb/knowledge/:knowledge_id", app.KnowledgeHandler.Update)
				api.DELETE("/kb/knowledge/:knowledge_id", app.KnowledgeHandler.Delete)
				api.POST("/kb/knowledge/:knowledge_id/merge", app.KnowledgeHandler.Merge)
				api.POST("/kb/reindex", app.KnowledgeHandler.Reindex)
			}
		}
	}

	return router
}
