package postgres

import (
	"github.com/ds124wfegd/WB_L3/2/internal/entity"
)

type URLRepositoryInterface interface {
	Create(url *entity.URL) error
	GetByShortURL(shortURL string) (*entity.URL, error)
	Exists(shortURL string) (bool, error)
	GetAll() ([]entity.URL, error)
	IncrementClicks(shortURL string) error
}

type AnalyticsRepositoryInterface interface {
	RecordClick(click *entity.Click) error
	GetAnalytics(shortURL string) (*entity.Analytics, error)
}

type CacheRepository interface {
	SetURL(shortURL string, url *entity.URL) error
	GetURL(shortURL string) (*entity.URL, error)
	DeleteURL(shortURL string) error
	IncrementPopularity(shortURL string) error
	GetPopularURLs(count int) ([]string, error)
}
