package transport

import (
	"net/http"
	"path/filepath"

	"github.com/ds124wfegd/WB_L3/4/internal/entity"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *ImageHandler) UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	// Проверка типа файла
	ext := filepath.Ext(file.Filename)
	if !isValidImageType(ext) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image type. Supported: jpg, jpeg, png, gif"})
		return
	}

	// Генерация ID
	id := uuid.New().String()

	// Сохранение и обработка
	imageID, err := h.service.ProcessImage(id, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, entity.UploadResponse{
		ID:     imageID,
		Status: "processing",
	})
}

func (h *ImageHandler) GetImage(c *gin.Context) {
	id := c.Param("id")

	image, err := h.service.GetImage(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	response := entity.ImageResponse{
		ID:     image.ID,
		Status: image.Status,
	}

	if image.Status == "completed" {
		response.Formats = image.Formats
	}

	c.JSON(http.StatusOK, response)
}

func (h *ImageHandler) DeleteImage(c *gin.Context) {
	id := c.Param("id")

	err := h.service.DeleteImage(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Image deleted successfully"})
}

func isValidImageType(ext string) bool {
	validTypes := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
	}
	return validTypes[ext]
}
