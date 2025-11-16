package usecase

import (
	"context"
	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/storage"
)

type UserService struct {
	repo   storage.Repository
	tx     domain.TransactionManager
	logger logger.Logger
}

func NewUserService(repo storage.Repository, tx domain.TransactionManager, logger logger.Logger) *UserService {
	return &UserService{
		repo:   repo,
		tx:     tx,
		logger: logger,
	}
}

func (s *UserService) SetUserActive(ctx context.Context, req domain.SetIsActiveRequest) (*domain.User, error) {
	var result *domain.User

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		_, err := s.repo.GetUser(ctx, req.UserID)
		if err != nil {
			s.logger.Error("Failed to get user", "error", err)
			return err
		}

		if err := s.repo.SetUserActive(ctx, req.UserID, req.IsActive); err != nil {
			s.logger.Error("Failed to set user active", "error", err)
			return err
		}

		user, err := s.repo.GetUser(ctx, req.UserID)
		if err != nil {
			s.logger.Error("Failed to get updated user", "error", err)
			return err
		}

		result = user

		return nil
	})

	return result, err
}

func (s *UserService) GetUserReviews(ctx context.Context, userID string) (*domain.UserReviewsResponse, error) {
	_, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user", "error", err)
		return nil, err
	}

	prs, err := s.repo.GetUserReviews(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user reviews", "error", err)
		return nil, err
	}

	shortPRs := make([]domain.PullRequestShort, len(prs))
	for i, pr := range prs {
		shortPRs[i] = domain.PullRequestShort{
			PullRequestID:   pr.PullRequestID,
			PullRequestName: pr.PullRequestName,
			AuthorID:        pr.AuthorID,
			Status:          pr.Status,
		}
	}

	return &domain.UserReviewsResponse{
		UserID:       userID,
		PullRequests: shortPRs,
	}, nil
}
