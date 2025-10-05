package service

import (
	"github.com/ds124wfegd/WB_L3/2/internal/entity"
)

type URLService interface {
	Shorten(url, customShort string) (*entity.ShortenResponse, error)
	Redirect(shortURL, userAgent, ipAddress string) (string, error)
	GetAllURLs() ([]entity.URL, error)
}

type AnalyticsService interface {
	GetAnalytics(shortURL string) (*entity.Analytics, error)
}

var (
	ErrInvalidURL     = &ServiceError{"invalid URL"}
	ErrShortURLExists = &ServiceError{"short URL already exists"}
	ErrURLNotFound    = &ServiceError{"URL not found"}
)

type ServiceError struct {
	message string
}

func (e *ServiceError) Error() string {
	return e.message
}
