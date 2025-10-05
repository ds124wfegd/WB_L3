package transport

import (
	"net/http"

	"github.com/ds124wfegd/WB_L3/2/internal/service"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService service.AnalyticsService
}

func NewAnalyticsHandler(analyticsService service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

func (h *AnalyticsHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/analytics/:short_url", h.GetAnalytics)
}

func (h *AnalyticsHandler) GetAnalytics(c *gin.Context) {
	shortURL := c.Param("short_url")

	analytics, err := h.analyticsService.GetAnalytics(shortURL)
	if err != nil {
		switch err {
		case service.ErrURLNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics"})
		}
		return
	}

	c.JSON(http.StatusOK, analytics)
}
