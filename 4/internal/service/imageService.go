package service

import (
	"mime/multipart"

	"github.com/ds124wfegd/WB_L3/4/internal/entity"
)

func (s *imageService) ProcessImage(id string, file *multipart.FileHeader) (string, error) {
	// Сохраняем оригинальное изображение
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Создаем запись в репозитории
	image := &entity.Image{
		ID:     id,
		Status: "processing",
	}

	if err := s.repo.Save(image); err != nil {
		return "", err
	}

	// Сохраняем файл
	if err := s.repo.SaveFile(id, "original", src); err != nil {
		return "", err
	}

	// Отправляем в Kafka для обработки
	task := entity.ProcessingTask{
		ImageID: id,
		Operations: []entity.Operation{
			{Type: "resize", Width: 800, Height: 600},
			{Type: "thumbnail", Width: 150, Height: 150},
			{Type: "watermark", Text: "Processed"},
		},
	}

	if err := s.producer.SendMessage("image-processing", task); err != nil {
		return "", err
	}

	return id, nil
}

func (s *imageService) GetImage(id string) (*entity.Image, error) {
	return s.repo.FindByID(id)
}

func (s *imageService) DeleteImage(id string) error {
	return s.repo.Delete(id)
}
