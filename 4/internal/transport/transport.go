package transport

import (
	"github.com/gin-gonic/gin"
)

func InitRoutes(imgHandler *ImageHandler) *gin.Engine {
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

	router.POST("/upload", imgHandler.UploadImage)
	router.GET("/image/:id", imgHandler.GetImage)
	router.DELETE("/image/:id", imgHandler.DeleteImage)

	router.Static("/static", "/app/internal/web/templates")
	router.LoadHTMLGlob("/app/internal/web/templates/*.html")

	router.GET("/", func(c *gin.Context) {
		c.File("/app/internal/web/templates/index.html")
	})

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "image-processor-service",
		})
	})
	return router
}
