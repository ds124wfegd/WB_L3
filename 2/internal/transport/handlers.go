package transport

import (
	"net/http"

	"github.com/ds124wfegd/WB_L3/2/internal/entity"
	"github.com/ds124wfegd/WB_L3/2/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler interface {
	RegisterRoutes(router *gin.RouterGroup)
}

type URLHandler struct {
	urlService service.URLService
}

func NewURLHandler(urlService service.URLService) *URLHandler {
	return &URLHandler{
		urlService: urlService,
	}
}

func (h *URLHandler) ShortenURL(c *gin.Context) {
	var req entity.ShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	response, err := h.urlService.Shorten(req.URL, req.CustomShort)
	if err != nil {
		switch err {
		case service.ErrInvalidURL:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		case service.ErrShortURLExists:
			c.JSON(http.StatusConflict, gin.H{"error": "Custom short URL already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create URL"})
		}
		return
	}

	c.JSON(http.StatusCreated, response)
}

func (h *URLHandler) RedirectURL(c *gin.Context) {
	shortURL := c.Param("short_url")

	originalURL, err := h.urlService.Redirect(shortURL, c.GetHeader("User-Agent"), c.ClientIP())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "URL not found"})
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

func (h *URLHandler) GetURLs(c *gin.Context) {
	urls, err := h.urlService.GetAllURLs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get URLs"})
		return
	}

	c.JSON(http.StatusOK, urls)
}
