package storage

import (
	"io"
	"os"
	"path/filepath"
)

type FileStorage interface {
	Save(path string, data io.Reader) error
	Get(path string) (io.ReadCloser, error)
	Delete(path string) error
	Exists(path string) bool
}

type fileStorage struct {
	basePath string
}

func NewFileStorage(basePath string) FileStorage {
	return &fileStorage{basePath: basePath}
}

func (s *fileStorage) Save(path string, data io.Reader) error {
	fullPath := filepath.Join(s.basePath, path)
	
	// Создаем директорию если нужно
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, data)
	return err
}

func (s *fileStorage) Get(path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, path)
	return os.Open(fullPath)
}

func (s *fileStorage) Delete(path string) error {
	fullPath := filepath.Join(s.basePath, path)
	return os.Remove(fullPath)
}

func (s *fileStorage) Exists(path string) bool {
	fullPath := filepath.Join(s.basePath, path)
	_, err := os.Stat(fullPath)
	return !os.IsNotExist(err)
}