package usecase

import (
	"context"
	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/storage"
)

type MetricsService struct {
	repo   storage.Repository
	tx     domain.TransactionManager
	logger logger.Logger
}

func NewMetricsService(repo storage.Repository, tx domain.TransactionManager, logger logger.Logger) *MetricsService {
	return &MetricsService{
		repo:   repo,
		tx:     tx,
		logger: logger,
	}
}

func (s *MetricsService) GetAssignmentStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := s.repo.GetAssignmentStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get assignment stats", "error", err)
		return nil, err
	}

	return map[string]interface{}{
		"reviewer_assignments": stats,
	}, nil
}
