package usecase

import (
	"context"
	"math/rand"
	"time"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/storage"
)

type PRService struct {
	repo   storage.Repository
	tx     domain.TransactionManager
	logger logger.Logger
	rand   *rand.Rand
}

func NewPRService(repo storage.Repository, tx domain.TransactionManager, logger logger.Logger) *PRService {
	return &PRService{
		repo:   repo,
		tx:     tx,
		logger: logger,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *PRService) CreatePR(ctx context.Context, req domain.CreatePRRequest) (*domain.PullRequestResponse, error) {
	var result *domain.PullRequestResponse

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {

		exists, err := s.repo.PRExists(ctx, req.PullRequestID)
		if err != nil {
			s.logger.Error("Failed to check PR existence", "error", err)
			return err
		}
		if exists {
			return domain.ErrPRAlreadyExists
		}

		author, err := s.repo.GetUser(ctx, req.AuthorID)
		if err != nil {
			s.logger.Error("Failed to get author", "error", err)
			return err
		}

		candidates, err := s.repo.GetActiveTeamMembers(ctx, author.TeamName, req.AuthorID)
		if err != nil {
			s.logger.Error("Failed to get team members", "error", err)
			return err
		}

		reviewers := s.selectReviewers(candidates, 2)
		reviewerIDs := make([]string, len(reviewers))
		for i, r := range reviewers {
			reviewerIDs[i] = r.UserID
		}

		now := time.Now()
		pr := &domain.PullRequest{
			PullRequestID:   req.PullRequestID,
			PullRequestName: req.PullRequestName,
			AuthorID:        req.AuthorID,
			Status:          domain.PRStatusOpen,
			CreatedAt:       &now,
		}

		if err := s.repo.CreatePR(ctx, pr, reviewerIDs); err != nil {
			s.logger.Error("Failed to create PR", "error", err)
			return err
		}

		result = &domain.PullRequestResponse{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            pr.Status,
			AssignedReviewers: reviewerIDs,
			CreatedAt:         pr.CreatedAt,
			MergedAt:          pr.MergedAt,
		}

		return nil
	})

	return result, err
}

func (s *PRService) MergePR(ctx context.Context, prID string) (*domain.PullRequestResponse, error) {
	var result *domain.PullRequestResponse

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {

		pr, reviewers, err := s.repo.GetPRWithReviewers(ctx, prID)
		if err != nil {
			return err
		}

		if pr.Status == domain.PRStatusMerged {
			result = &domain.PullRequestResponse{
				PullRequestID:     pr.PullRequestID,
				PullRequestName:   pr.PullRequestName,
				AuthorID:          pr.AuthorID,
				Status:            pr.Status,
				AssignedReviewers: reviewers,
				CreatedAt:         pr.CreatedAt,
				MergedAt:          pr.MergedAt,
			}
			return nil
		}

		if err := s.repo.MergePR(ctx, prID); err != nil {
			return err
		}

		pr, reviewers, err = s.repo.GetPRWithReviewers(ctx, prID)
		if err != nil {
			return err
		}

		result = &domain.PullRequestResponse{
			PullRequestID:     pr.PullRequestID,
			PullRequestName:   pr.PullRequestName,
			AuthorID:          pr.AuthorID,
			Status:            pr.Status,
			AssignedReviewers: reviewers,
			CreatedAt:         pr.CreatedAt,
			MergedAt:          pr.MergedAt,
		}

		return nil
	})

	return result, err
}

func (s *PRService) ReassignReviewer(ctx context.Context, req domain.ReassignRequest) (*domain.ReassignResponse, error) {
	var result *domain.ReassignResponse

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		pr, reviewers, err := s.repo.GetPRWithReviewers(ctx, req.PullRequestID)
		if err != nil {
			s.logger.Error("Failed to get PR with reviewers", "error", err)
			return err
		}

		if pr.Status == domain.PRStatusMerged {
			return domain.ErrPRMerged
		}

		isAssigned, err := s.repo.IsReviewerAssigned(ctx, req.PullRequestID, req.OldUserID)
		if err != nil {
			s.logger.Error("Failed to check reviewer assignment", "error", err)
			return err
		}
		if !isAssigned {
			return domain.ErrReviewerNotAssigned
		}

		oldReviewer, err := s.repo.GetUser(ctx, req.OldUserID)
		if err != nil {
			s.logger.Error("Failed to get old reviewer", "error", err)
			return err
		}

		candidates, err := s.repo.GetActiveTeamMembers(ctx, oldReviewer.TeamName, req.OldUserID)
		if err != nil {
			s.logger.Error("Failed to get team candidates", "error", err)
			return err
		}

		available := s.filterAvailableReviewers(candidates, pr, reviewers)

		if len(available) == 0 {
			return domain.ErrNoActiveCandidate
		}

		newReviewer := available[s.rand.Intn(len(available))]

		if err := s.repo.RemoveReviewer(ctx, req.PullRequestID, req.OldUserID); err != nil {
			s.logger.Error("Failed to remove old reviewer", "error", err)
			return err
		}

		if err := s.repo.AddReviewer(ctx, req.PullRequestID, newReviewer.UserID); err != nil {
			s.logger.Error("Failed to add new reviewer", "error", err)
			return err
		}

		updatedPR, revs, err := s.repo.GetPRWithReviewers(ctx, req.PullRequestID)
		if err != nil {
			s.logger.Error("Failed to get updated PR", "error", err)
			return err
		}

		result = &domain.ReassignResponse{
			PR: domain.PullRequestResponse{
				PullRequestID:     updatedPR.PullRequestID,
				PullRequestName:   updatedPR.PullRequestName,
				AuthorID:          updatedPR.AuthorID,
				Status:            updatedPR.Status,
				AssignedReviewers: revs,
				CreatedAt:         updatedPR.CreatedAt,
				MergedAt:          updatedPR.MergedAt,
			},
			ReplacedBy: newReviewer.UserID,
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Reassign reviewer failed", "error", err)
	}

	return result, err
}

func (s *PRService) selectReviewers(candidates []domain.User, n int) []domain.User {
	if len(candidates) == 0 {
		return []domain.User{}
	}

	if len(candidates) <= n {
		return candidates
	}

	perm := s.rand.Perm(len(candidates))

	selected := make([]domain.User, n)
	for i := 0; i < n; i++ {
		selected[i] = candidates[perm[i]]
	}

	return selected
}

func (s *PRService) filterAvailableReviewers(candidates []domain.User, pr *domain.PullRequest, reviewers []string) []domain.User {
	available := make([]domain.User, 0, len(candidates))
	for _, c := range candidates {
		if c.UserID == pr.AuthorID {
			continue
		}

		assigned := false
		for _, r := range reviewers {
			if r == c.UserID {
				assigned = true
				break
			}
		}
		if assigned {
			continue
		}

		available = append(available, c)
	}
	return available
}
