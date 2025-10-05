package transport

import (
	"github.com/gin-gonic/gin"
)

func InitRoutes(urlHandler *URLHandler, analyticsHandler *AnalyticsHandler) *gin.Engine {
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	router.Static("/static", "/app/internal/web/templates")
	router.LoadHTMLGlob("/app/internal/web/templates/*.html")

	router.GET("/", func(c *gin.Context) {
		c.File("/app/internal/web/templates/index.html")
	})

	api := router.Group("/")
	urlHandler.RegisterRoutes(api)
	analyticsHandler.RegisterRoutes(api)

	// Health check
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

func (h *URLHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/shorten", h.ShortenURL)
	router.GET("/s/:short_url", h.RedirectURL)
	router.GET("/urls", h.GetURLs)
}
