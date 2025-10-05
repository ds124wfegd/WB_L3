package service

import (
	"errors"
	"time"

	"github.com/ds124wfegd/WB_L3/3/internal/entity"

	"github.com/google/uuid"
)

func (s *CommentService) CreateComment(req entity.CreateCommentRequest) (*entity.Comment, error) {
	if req.Author == "" || req.Text == "" {
		return nil, errors.New("author and text are required")
	}

	// Если указан parent_id, проверяем что родитель существует
	if req.ParentID != "" {
		if _, exists := s.repo.GetByID(req.ParentID); !exists {
			return nil, errors.New("parent comment not found")
		}
	}

	comment := entity.Comment{
		ID:        uuid.New().String(),
		ParentID:  req.ParentID,
		Author:    req.Author,
		Text:      req.Text,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.Create(comment); err != nil {
		return nil, err
	}

	return &comment, nil
}

func (s *CommentService) GetComments(parentID string, page, pageSize int, sortBy string) (*entity.CommentsResponse, error) {
	comments, total := s.repo.GetChildren(parentID, page, pageSize, sortBy)

	response := &entity.CommentsResponse{
		Comments: comments,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	return response, nil
}

func (s *CommentService) GetCommentTree(parentID string) ([]entity.Comment, error) {
	tree := s.repo.BuildTree(parentID, 0)
	return tree, nil
}

func (s *CommentService) DeleteComment(id string) error {
	if _, exists := s.repo.GetByID(id); !exists {
		return errors.New("comment not found")
	}

	if err := s.repo.Delete(id); err != nil {
		return err
	}

	return nil
}

func (s *CommentService) SearchComments(query string, page, pageSize int) (*entity.CommentsResponse, error) {
	if query == "" {
		return nil, errors.New("search query is required")
	}

	results, total := s.repo.Search(query, page, pageSize)

	response := &entity.CommentsResponse{
		Comments: results,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	return response, nil
}

func (s *CommentService) GetStats() (map[string]string, error) {
	return s.repo.GetStats()
}
