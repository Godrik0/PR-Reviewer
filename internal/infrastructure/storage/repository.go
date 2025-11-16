package storage

import (
	"context"
	"pr-reviewer/internal/domain"
)

type Repository interface {
	// Team
	CreateTeam(ctx context.Context, team *domain.Team, members []domain.User) error
	GetTeam(ctx context.Context, teamName string) (*domain.Team, error)
	TeamExists(ctx context.Context, teamName string) (bool, error)

	// User
	CreateOrUpdateUser(ctx context.Context, user *domain.User) error
	GetUser(ctx context.Context, userID string) (*domain.User, error)
	GetUsersByTeam(ctx context.Context, teamName string) ([]domain.User, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) error
	GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]domain.User, error)

	// PR
	CreatePR(ctx context.Context, pr *domain.PullRequest, reviewers []string) error
	GetPR(ctx context.Context, prID string) (*domain.PullRequest, error)
	GetPRWithReviewers(ctx context.Context, prID string) (*domain.PullRequest, []string, error)
	PRExists(ctx context.Context, prID string) (bool, error)
	MergePR(ctx context.Context, prID string) error

	// PR Reviewer
	GetPRReviewers(ctx context.Context, prID string) ([]string, error)
	AddReviewer(ctx context.Context, prID, userID string) error
	RemoveReviewer(ctx context.Context, prID, userID string) error
	GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequest, error)
	IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error)

	// Mass deactivate
	DeactivateUsers(ctx context.Context, userIDs []string) error
	GetOpenPRsWithReviewers(ctx context.Context, reviewerIDs []string) ([]domain.PullRequest, map[string][]string, error)
	BulkReassignReviewers(ctx context.Context, reassignments []domain.PRReassignment) error

	// Statistics
	GetAssignmentStats(ctx context.Context) (map[string]int, error)
}
