package database

import "github.com/ds124wfegd/WB_L3/3/internal/entity"

type Repository interface {
	Create(comment entity.Comment) error
	GetByID(id string) (*entity.Comment, bool)
	GetChildren(parentID string, page, pageSize int, sortBy string) ([]entity.Comment, int)
	Delete(id string) error
	Search(query string, page, pageSize int) ([]entity.Comment, int)
	BuildTree(parentID string, depth int) []entity.Comment
	GetAllComments() ([]entity.Comment, error)
}
