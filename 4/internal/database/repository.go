package database

import (
	"io"

	"github.com/ds124wfegd/WB_L3/4/internal/entity"
	"github.com/ds124wfegd/WB_L3/4/internal/pkg/storage"
)

type ImageRepository interface {
	Save(image *entity.Image) error
	FindByID(id string) (*entity.Image, error)
	Delete(id string) error
	SaveFile(id string, format string, file io.Reader) error
	GetFilePath(id string, format string) string
}

type fileImageRepository struct {
	storage storage.FileStorage
}
