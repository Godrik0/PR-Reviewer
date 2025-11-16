package memory

import (
	"context"
	"sync"
	"time"

	"pr-reviewer/internal/domain"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	teams       map[string]*domain.Team
	users       map[string]*domain.User
	prs         map[string]*domain.PullRequest
	prReviewers map[string][]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		teams:       make(map[string]*domain.Team),
		users:       make(map[string]*domain.User),
		prs:         make(map[string]*domain.PullRequest),
		prReviewers: make(map[string][]string),
	}
}

func (r *MemoryRepository) CreateTeam(ctx context.Context, team *domain.Team, members []domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.teams[team.TeamName]; exists {
		return domain.ErrTeamAlreadyExists
	}

	r.teams[team.TeamName] = &domain.Team{
		TeamName: team.TeamName,
	}

	for i := range members {
		members[i].TeamName = team.TeamName
		r.users[members[i].UserID] = &members[i]
	}

	return nil
}

func (r *MemoryRepository) GetTeam(ctx context.Context, teamName string) (*domain.Team, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	team, exists := r.teams[teamName]
	if !exists {
		return nil, domain.ErrTeamNotFound
	}

	var members []domain.User
	for _, user := range r.users {
		if user.TeamName == teamName {
			members = append(members, *user)
		}
	}

	return &domain.Team{
		TeamName: team.TeamName,
		Members:  members,
	}, nil
}

func (r *MemoryRepository) TeamExists(ctx context.Context, teamName string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.teams[teamName]
	return exists, nil
}

func (r *MemoryRepository) CreateOrUpdateUser(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.UserID] = user
	return nil
}

func (r *MemoryRepository) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[userID]
	if !exists {
		return nil, domain.ErrUserNotFound
	}

	userCopy := *user
	return &userCopy, nil
}

func (r *MemoryRepository) GetUsersByTeam(ctx context.Context, teamName string) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var users []domain.User
	for _, user := range r.users {
		if user.TeamName == teamName {
			users = append(users, *user)
		}
	}

	return users, nil
}

func (r *MemoryRepository) SetUserActive(ctx context.Context, userID string, isActive bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, exists := r.users[userID]
	if !exists {
		return domain.ErrUserNotFound
	}

	user.IsActive = isActive
	return nil
}

func (r *MemoryRepository) GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var members []domain.User
	for _, user := range r.users {
		if user.TeamName == teamName && user.IsActive && user.UserID != excludeUserID {
			members = append(members, *user)
		}
	}

	return members, nil
}

func (r *MemoryRepository) CreatePR(ctx context.Context, pr *domain.PullRequest, reviewers []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.prs[pr.PullRequestID]; exists {
		return domain.ErrPRAlreadyExists
	}

	r.prs[pr.PullRequestID] = pr
	r.prReviewers[pr.PullRequestID] = reviewers

	return nil
}

func (r *MemoryRepository) GetPR(ctx context.Context, prID string) (*domain.PullRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pr, exists := r.prs[prID]
	if !exists {
		return nil, domain.ErrPRNotFound
	}

	prCopy := *pr
	return &prCopy, nil
}

func (r *MemoryRepository) GetPRWithReviewers(ctx context.Context, prID string) (*domain.PullRequest, []string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pr, exists := r.prs[prID]
	if !exists {
		return nil, nil, domain.ErrPRNotFound
	}

	prCopy := *pr
	reviewers := make([]string, len(r.prReviewers[prID]))
	copy(reviewers, r.prReviewers[prID])

	return &prCopy, reviewers, nil
}

func (r *MemoryRepository) PRExists(ctx context.Context, prID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.prs[prID]
	return exists, nil
}

func (r *MemoryRepository) MergePR(ctx context.Context, prID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pr, exists := r.prs[prID]
	if !exists {
		return domain.ErrPRNotFound
	}

	now := time.Now()
	pr.Status = domain.PRStatusMerged
	pr.MergedAt = &now

	return nil
}

func (r *MemoryRepository) GetPRReviewers(ctx context.Context, prID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reviewers, exists := r.prReviewers[prID]
	if !exists {
		return []string{}, nil
	}

	result := make([]string, len(reviewers))
	copy(result, reviewers)
	return result, nil
}

func (r *MemoryRepository) AddReviewer(ctx context.Context, prID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.prReviewers[prID] = append(r.prReviewers[prID], userID)
	return nil
}

func (r *MemoryRepository) RemoveReviewer(ctx context.Context, prID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	reviewers := r.prReviewers[prID]
	for i, id := range reviewers {
		if id == userID {
			r.prReviewers[prID] = append(reviewers[:i], reviewers[i+1:]...)
			return nil
		}
	}

	return nil
}

func (r *MemoryRepository) GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var prs []domain.PullRequest
	for prID, reviewers := range r.prReviewers {
		for _, reviewerID := range reviewers {
			if reviewerID == userID {
				if pr, exists := r.prs[prID]; exists {
					prs = append(prs, *pr)
				}
				break
			}
		}
	}

	return prs, nil
}

func (r *MemoryRepository) IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reviewers, exists := r.prReviewers[prID]
	if !exists {
		return false, nil
	}

	for _, id := range reviewers {
		if id == userID {
			return true, nil
		}
	}

	return false, nil
}

func (r *MemoryRepository) DeactivateUsers(ctx context.Context, userIDs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, userID := range userIDs {
		if user, exists := r.users[userID]; exists {
			user.IsActive = false
			r.users[userID] = user
		}
	}

	return nil
}

func (r *MemoryRepository) GetOpenPRsWithReviewers(ctx context.Context, reviewerIDs []string) ([]domain.PullRequest, map[string][]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reviewerSet := make(map[string]bool)
	for _, id := range reviewerIDs {
		reviewerSet[id] = true
	}

	affectedPRs := make([]domain.PullRequest, 0)
	reviewersMap := make(map[string][]string)

	for prID, pr := range r.prs {
		if pr.Status != domain.PRStatusOpen {
			continue
		}

		reviewers := r.prReviewers[prID]
		hasAffectedReviewer := false

		for _, revID := range reviewers {
			if reviewerSet[revID] {
				hasAffectedReviewer = true
				break
			}
		}

		if hasAffectedReviewer {
			affectedPRs = append(affectedPRs, *pr)
			reviewersMap[prID] = reviewers
		}
	}

	return affectedPRs, reviewersMap, nil
}

func (r *MemoryRepository) BulkReassignReviewers(ctx context.Context, reassignments []domain.PRReassignment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, reassign := range reassignments {
		reviewers := r.prReviewers[reassign.PullRequestID]

		newReviewers := make([]string, 0, len(reviewers))
		for _, revID := range reviewers {
			if revID != reassign.OldReviewerID {
				newReviewers = append(newReviewers, revID)
			}
		}

		if reassign.NewReviewerID != "" {
			newReviewers = append(newReviewers, reassign.NewReviewerID)
		}

		r.prReviewers[reassign.PullRequestID] = newReviewers
	}

	return nil
}

func (r *MemoryRepository) GetAssignmentStats(ctx context.Context) (map[string]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]int)
	for _, reviewers := range r.prReviewers {
		for _, reviewerID := range reviewers {
			stats[reviewerID]++
		}
	}

	return stats, nil
}
