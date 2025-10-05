package service

import (
	"github.com/ds124wfegd/WB_L3/2/internal/database/postgres"
	"github.com/ds124wfegd/WB_L3/2/internal/entity"
)

type AnalyticsServiceImpl struct {
	analyticsRepo postgres.AnalyticsRepositoryInterface
	urlRepo       postgres.URLRepositoryInterface
}

func NewAnalyticsService(
	analyticsRepo postgres.AnalyticsRepositoryInterface,
	urlRepo postgres.URLRepositoryInterface,
) AnalyticsService {
	return &AnalyticsServiceImpl{
		analyticsRepo: analyticsRepo,
		urlRepo:       urlRepo,
	}
}

func (s *AnalyticsServiceImpl) GetAnalytics(shortURL string) (*entity.Analytics, error) {
	exists, err := s.urlRepo.Exists(shortURL)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrURLNotFound
	}

	return s.analyticsRepo.GetAnalytics(shortURL)
}
