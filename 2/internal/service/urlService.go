package service

import (
	"math/rand"
	"net/url"
	"time"

	"github.com/ds124wfegd/WB_L3/2/internal/database/postgres"
	"github.com/ds124wfegd/WB_L3/2/internal/entity"
	"github.com/google/uuid"
)

type URLServiceImpl struct {
	urlRepo       postgres.URLRepositoryInterface
	analyticsRepo postgres.AnalyticsRepositoryInterface
	cacheRepo     postgres.CacheRepository
	config        *URLServiceConfig
}

type URLServiceConfig struct {
	ShortURLLength int
	BaseURL        string
	CacheTTL       time.Duration
}

func NewURLService(
	urlRepo postgres.URLRepositoryInterface,
	analyticsRepo postgres.AnalyticsRepositoryInterface,
	cacheRepo postgres.CacheRepository,
	config *URLServiceConfig,
) URLService {
	return &URLServiceImpl{
		urlRepo:       urlRepo,
		analyticsRepo: analyticsRepo,
		cacheRepo:     cacheRepo,
		config:        config,
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func (s *URLServiceImpl) generateShortURL() string {
	rand.Seed(time.Now().UnixNano())
	shortURL := make([]byte, s.config.ShortURLLength)
	for i := range shortURL {
		shortURL[i] = charset[rand.Intn(len(charset))]
	}
	return string(shortURL)
}

func (s *URLServiceImpl) Shorten(originalURL, customShort string) (*entity.ShortenResponse, error) {
	if _, err := url.ParseRequestURI(originalURL); err != nil {
		return nil, ErrInvalidURL
	}

	var shortURL string
	if customShort != "" {
		shortURL = customShort
		exists, err := s.urlRepo.Exists(shortURL)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrShortURLExists
		}
	} else {
		for {
			shortURL = s.generateShortURL()
			exists, err := s.urlRepo.Exists(shortURL)
			if err != nil {
				return nil, err
			}
			if !exists {
				break
			}
		}
	}

	url := &entity.URL{
		ID:          uuid.New().String(),
		OriginalURL: originalURL,
		ShortURL:    shortURL,
		CreatedAt:   time.Now(),
		Clicks:      0,
	}

	if err := s.urlRepo.Create(url); err != nil {
		return nil, err
	}

	s.cacheRepo.SetURL(shortURL, url)

	return &entity.ShortenResponse{
		ShortURL:     shortURL,
		OriginalURL:  originalURL,
		CreatedAt:    url.CreatedAt,
		ShortURLFull: s.config.BaseURL + "/s/" + shortURL,
	}, nil
}

func (s *URLServiceImpl) Redirect(shortURL, userAgent, ipAddress string) (string, error) {
	var originalURL string
	cachedURL, err := s.cacheRepo.GetURL(shortURL)
	if err == nil {
		originalURL = cachedURL.OriginalURL
	} else {
		url, err := s.urlRepo.GetByShortURL(shortURL)
		if err != nil {
			return "", ErrURLNotFound
		}
		originalURL = url.OriginalURL

		s.cacheRepo.SetURL(shortURL, url)
	}

	go s.recordClick(shortURL, userAgent, ipAddress)

	s.cacheRepo.IncrementPopularity(shortURL)

	return originalURL, nil
}

func (s *URLServiceImpl) recordClick(shortURL, userAgent, ipAddress string) {
	click := &entity.Click{
		ID:        uuid.New().String(),
		ShortURL:  shortURL,
		UserAgent: userAgent,
		IPAddress: ipAddress,
		Timestamp: time.Now(),
	}

	if err := s.analyticsRepo.RecordClick(click); err != nil {
		return
	}

	if err := s.urlRepo.IncrementClicks(shortURL); err != nil {
		return
	}
}

func (s *URLServiceImpl) GetAllURLs() ([]entity.URL, error) {
	return s.urlRepo.GetAll()
}
