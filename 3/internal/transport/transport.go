package transport

import (
	"github.com/ds124wfegd/WB_L3/3/internal/service"

	"github.com/gin-gonic/gin"
)

func InitRoutes(service *service.CommentService) *gin.Engine {
	handler := NewCommentHandler(service)
	router := gin.Default()

	api := router.Group("/comments")
	{
		api.POST("", handler.CreateComment)
		api.GET("", handler.GetComments)
		api.GET("/tree", handler.GetCommentTree)
		api.DELETE("/:id", handler.DeleteComment)
		api.GET("/search", handler.SearchComments)
		api.GET("/stats", handler.GetStats)
	}

	router.Static("/static", "/app/internal/web/templates")
	router.LoadHTMLGlob("/app/internal/web/templates/*.html")

	router.GET("/", func(c *gin.Context) {
		c.File("/app/internal/web/templates/index.html")
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "ok",
			"service":  "url-shortener",
			"database": "connected",
			"redis":    "connected",
		})
	})
	return router
}
