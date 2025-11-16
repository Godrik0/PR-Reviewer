package usecase

import (
	"context"
	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/storage"
)

type TeamService struct {
	repo   storage.Repository
	tx     domain.TransactionManager
	logger logger.Logger
}

func NewTeamService(repo storage.Repository, tx domain.TransactionManager, logger logger.Logger) *TeamService {
	return &TeamService{
		repo:   repo,
		tx:     tx,
		logger: logger,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, req domain.CreateTeamRequest) (*domain.TeamResponse, error) {
	var result *domain.TeamResponse

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		exists, err := s.repo.TeamExists(ctx, req.TeamName)
		if err != nil {
			s.logger.Error("Failed to check team existence", "error", err)
			return err
		}
		if exists {
			return domain.ErrTeamAlreadyExists
		}

		team := &domain.Team{
			TeamName: req.TeamName,
		}

		members := make([]domain.User, len(req.Members))
		for i, m := range req.Members {
			members[i] = domain.User{
				UserID:   m.UserID,
				Username: m.Username,
				TeamName: req.TeamName,
				IsActive: m.IsActive,
			}
		}

		if err := s.repo.CreateTeam(ctx, team, members); err != nil {
			s.logger.Error("Failed to create team", "error", err)
			return err
		}

		responseMembers := make([]domain.TeamMember, len(req.Members))
		for i, m := range req.Members {
			responseMembers[i] = domain.TeamMember{
				UserID:   m.UserID,
				Username: m.Username,
				IsActive: m.IsActive,
			}
		}

		result = &domain.TeamResponse{
			TeamName: req.TeamName,
			Members:  responseMembers,
		}

		return nil
	})

	return result, err
}

func (s *TeamService) GetTeam(ctx context.Context, teamName string) (*domain.TeamResponse, error) {
	team, err := s.repo.GetTeam(ctx, teamName)
	if err != nil {
		s.logger.Error("Failed to get team", "error", err)
		return nil, err
	}

	members := make([]domain.TeamMember, len(team.Members))
	for i, m := range team.Members {
		members[i] = domain.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}

	return &domain.TeamResponse{
		TeamName: team.TeamName,
		Members:  members,
	}, nil
}

func (s *TeamService) DeactivateTeamUsers(ctx context.Context, req domain.DeactivateTeamUsersRequest) (*domain.DeactivateTeamUsersResponse, error) {
	var result *domain.DeactivateTeamUsersResponse

	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		validUserIDs, err := s.getValidTeamUserIDsForDeactivation(ctx, req)
		if err != nil {
			return err
		}

		prs, reviewersMap, err := s.repo.GetOpenPRsWithReviewers(ctx, validUserIDs)
		if err != nil {
			return err
		}

		reassignments, summaries := s.planReviewerReassignments(ctx, prs, reviewersMap, validUserIDs)

		if err := s.applyDeactivationChanges(ctx, validUserIDs, reassignments); err != nil {
			return err
		}

		result = &domain.DeactivateTeamUsersResponse{
			DeactivatedUsers: validUserIDs,
			ReassignedPRs:    summaries,
		}

		return nil
	})

	return result, err
}

func (s *TeamService) getValidTeamUserIDsForDeactivation(ctx context.Context, req domain.DeactivateTeamUsersRequest) ([]string, error) {
	_, err := s.repo.GetTeam(ctx, req.TeamName)
	if err != nil {
		return nil, err
	}

	validUserIDs := s.filterValidTeamUsers(ctx, req)
	if len(validUserIDs) == 0 {
		return nil, domain.NewAppError(domain.ErrCodeBadRequest, "no valid users to deactivate")
	}

	return validUserIDs, nil
}

func (s *TeamService) planReviewerReassignments(ctx context.Context, prs []domain.PullRequest, reviewersMap map[string][]string, deactivatingUserIDs []string) ([]domain.PRReassignment, []domain.PRReassignmentSummary) {
	// Планируем переназначения ревьюверов для всех PR
	deactivatingSet := s.createUserIDSet(deactivatingUserIDs)
	reassignments := make([]domain.PRReassignment, 0)
	summaries := make([]domain.PRReassignmentSummary, 0)

	// Проходим по всем PR и планируем переназначения
	for _, pr := range prs {
		prReassignments, summary := s.processPRReassignments(ctx, pr, reviewersMap[pr.PullRequestID], deactivatingSet)
		reassignments = append(reassignments, prReassignments...)

		if len(summary.OldReviewers) > 0 {
			summaries = append(summaries, summary)
		}
	}

	return reassignments, summaries
}

func (s *TeamService) processPRReassignments(ctx context.Context, pr domain.PullRequest, currentReviewers []string, deactivatingSet map[string]bool) ([]domain.PRReassignment, domain.PRReassignmentSummary) {
	// Получаем автора PR для фильтрации кандидатов
	author, err := s.repo.GetUser(ctx, pr.AuthorID)
	if err != nil {
		s.logger.Error("Failed to get PR author", "pr_id", pr.PullRequestID, "error", err)
		return nil, domain.PRReassignmentSummary{}
	}

	reassignments := make([]domain.PRReassignment, 0)
	oldReviewers := make([]string, 0)
	newReviewers := make([]string, 0)

	// Отслеживаем уже назначенных ревьюверов
	assignedReviewers := make(map[string]bool)

	// Сначала добавляем всех остающихся ревьюверов
	for _, reviewerID := range currentReviewers {
		if !deactivatingSet[reviewerID] {
			assignedReviewers[reviewerID] = true
			newReviewers = append(newReviewers, reviewerID)
		}
	}

	// Обрабатываем деактивируемых ревьюверов
	for _, reviewerID := range currentReviewers {
		if !deactivatingSet[reviewerID] {
			continue
		}

		oldReviewers = append(oldReviewers, reviewerID)
		replacement := s.findReviewerReplacement(ctx, reviewerID, author, assignedReviewers, deactivatingSet)

		if replacement != "" {
			newReviewers = append(newReviewers, replacement)
			assignedReviewers[replacement] = true
		}

		reassignments = append(reassignments, domain.PRReassignment{
			PullRequestID: pr.PullRequestID,
			OldReviewerID: reviewerID,
			NewReviewerID: replacement,
		})
	}

	return reassignments, domain.PRReassignmentSummary{
		PullRequestID: pr.PullRequestID,
		OldReviewers:  oldReviewers,
		NewReviewers:  newReviewers,
	}
}

func (s *TeamService) findReviewerReplacement(ctx context.Context, reviewerID string, author *domain.User, assignedReviewers map[string]bool, deactivatingSet map[string]bool) string {
	// Получаем команду ревьювера
	reviewer, err := s.repo.GetUser(ctx, reviewerID)
	if err != nil {
		s.logger.Error("Failed to get reviewer", "reviewer_id", reviewerID, "error", err)
		return ""
	}

	// Получаем активных кандидатов из команды
	candidates, err := s.repo.GetActiveTeamMembers(ctx, reviewer.TeamName, reviewerID)
	if err != nil {
		s.logger.Error("Failed to get candidates", "error", err)
		return ""
	}

	// Ищем подходящего кандидата
	for _, candidate := range candidates {
		if s.isValidReplacementCandidate(candidate, author.UserID, assignedReviewers, deactivatingSet) {
			return candidate.UserID
		}
	}

	return ""
}

func (s *TeamService) isValidReplacementCandidate(candidate domain.User, authorID string, assignedReviewers map[string]bool, deactivatingSet map[string]bool) bool {
	// Не назначаем автора PR
	if candidate.UserID == authorID {
		return false
	}

	// Не назначаем уже назначенных ревьюверов
	if assignedReviewers[candidate.UserID] {
		return false
	}

	// Не назначаем деактивируемых пользователей
	if deactivatingSet[candidate.UserID] {
		return false
	}

	return true
}

func (s *TeamService) applyDeactivationChanges(ctx context.Context, validUserIDs []string, reassignments []domain.PRReassignment) error {
	// Выполняем массовое переназначение
	if len(reassignments) > 0 {
		if err := s.repo.BulkReassignReviewers(ctx, reassignments); err != nil {
			s.logger.Error("Failed to bulk reassign reviewers", "error", err)
			return err
		}
	}

	// Деактивируем пользователей
	if err := s.repo.DeactivateUsers(ctx, validUserIDs); err != nil {
		return err
	}

	return nil
}

func (s *TeamService) createUserIDSet(userIDs []string) map[string]bool {
	set := make(map[string]bool, len(userIDs))
	for _, userID := range userIDs {
		set[userID] = true
	}
	return set
}

func (s *TeamService) filterValidTeamUsers(ctx context.Context, req domain.DeactivateTeamUsersRequest) []string {
	validUserIDs := make([]string, 0, len(req.UserIDs))
	for _, userID := range req.UserIDs {
		user, err := s.repo.GetUser(ctx, userID)
		if err != nil {
			continue
		}
		if user.TeamName != req.TeamName {
			continue
		}
		validUserIDs = append(validUserIDs, userID)
	}

	return validUserIDs
}
