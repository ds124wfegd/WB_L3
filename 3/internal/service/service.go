package service

import (
	"github.com/ds124wfegd/WB_L3/3/internal/database"
)

type CommentService struct {
	repo *database.CommentRepository
}

func NewCommentService(repo *database.CommentRepository) *CommentService {
	return &CommentService{
		repo: repo,
	}
}
