package database

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/ds124wfegd/WB_L3/4/internal/entity"
	"github.com/ds124wfegd/WB_L3/4/internal/pkg/storage"
)

func NewImageRepository(storage storage.FileStorage) ImageRepository {
	return &fileImageRepository{storage: storage}
}

func (r *fileImageRepository) Save(image *entity.Image) error {
	imagePath := r.getImageMetadataPath(image.ID)

	data, err := json.Marshal(image)
	if err != nil {
		return err
	}

	return r.storage.Save(imagePath, bytes.NewReader(data))
}

func (r *fileImageRepository) FindByID(id string) (*entity.Image, error) {
	imagePath := r.getImageMetadataPath(id)

	reader, err := r.storage.Get(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer reader.Close()

	var image entity.Image
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&image); err != nil {
		return nil, err
	}

	return &image, nil
}

func (r *fileImageRepository) Delete(id string) error {
	metadataPath := r.getImageMetadataPath(id)
	if err := r.storage.Delete(metadataPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	processedDir := filepath.Join("processed", id)
	if err := r.storage.Delete(processedDir); err != nil && !os.IsNotExist(err) {
		return err
	}

	originalPath := filepath.Join("original", id)
	if err := r.storage.Delete(originalPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (r *fileImageRepository) SaveFile(id string, format string, file io.Reader) error {
	var filePath string
	if format == "original" {
		filePath = filepath.Join("original", id)
	} else {
		filePath = filepath.Join("processed", id, format)
	}

	return r.storage.Save(filePath, file)
}

func (r *fileImageRepository) GetFilePath(id string, format string) string {
	if format == "original" {
		return filepath.Join("original", id)
	}
	return filepath.Join("processed", id, format)
}

func (r *fileImageRepository) getImageMetadataPath(id string) string {
	return filepath.Join("metadata", id+".json")
}
