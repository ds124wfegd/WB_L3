package transport

import (
	"github.com/ds124wfegd/WB_L3/4/internal/service"
)

type ImageHandler struct {
	service service.ImageService
}

func NewImageHandler(service service.ImageService) *ImageHandler {
	return &ImageHandler{service: service}
}
