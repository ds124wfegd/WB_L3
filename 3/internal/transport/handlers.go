package transport

import (
	"github.com/ds124wfegd/WB_L3/3/internal/service"
)

type CommentHandler struct {
	service *service.CommentService
}

func NewCommentHandler(service *service.CommentService) *CommentHandler {
	return &CommentHandler{
		service: service,
	}
}
