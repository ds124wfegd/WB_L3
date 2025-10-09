package service

import (
	"mime/multipart"

	"github.com/ds124wfegd/WB_L3/4/internal/database"
	"github.com/ds124wfegd/WB_L3/4/internal/entity"
	"github.com/ds124wfegd/WB_L3/4/internal/pkg/kafka"
	"github.com/ds124wfegd/WB_L3/4/internal/pkg/processor"
)

type ImageService interface {
	ProcessImage(id string, file *multipart.FileHeader) (string, error)
	GetImage(id string) (*entity.Image, error)
	DeleteImage(id string) error
}

type imageService struct {
	repo      database.ImageRepository
	producer  kafka.Producer
	processor processor.ImageProcessor
}

func NewImageService(repo database.ImageRepository, producer kafka.Producer, processor processor.ImageProcessor) ImageService {
	return &imageService{
		repo:      repo,
		producer:  producer,
		processor: processor,
	}
}
