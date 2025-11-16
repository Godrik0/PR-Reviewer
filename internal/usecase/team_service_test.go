package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/storage/memory"
)

func TestTeamService_CreateTeam(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockTx.On("WithinTransaction", mock.Anything, mock.Anything).Return(nil)

	service := NewTeamService(repo, mockTx, mockLogger)

	req := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	result, err := service.CreateTeam(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", result.TeamName)
	assert.Len(t, result.Members, 2)
}

func TestTeamService_CreateTeam_AlreadyExists(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockTx.On("WithinTransaction", mock.Anything, mock.Anything).Return(nil)

	service := NewTeamService(repo, mockTx, mockLogger)

	req := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}

	_, err := service.CreateTeam(context.Background(), req)
	require.NoError(t, err)

	_, err = service.CreateTeam(context.Background(), req)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrTeamAlreadyExists, err)
}

func TestTeamService_GetTeam(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockTx.On("WithinTransaction", mock.Anything, mock.Anything).Return(nil)

	service := NewTeamService(repo, mockTx, mockLogger)

	req := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}
	_, err := service.CreateTeam(context.Background(), req)
	require.NoError(t, err)

	result, err := service.GetTeam(context.Background(), "backend")
	require.NoError(t, err)
	assert.Equal(t, "backend", result.TeamName)
	assert.Len(t, result.Members, 1)
}

func TestTeamService_GetTeam_NotFound(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := NewTeamService(repo, mockTx, mockLogger)

	_, err := service.GetTeam(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, domain.ErrTeamNotFound, err)
}

func TestTeamService_DeactivateTeamUsers(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockTx.On("WithinTransaction", mock.Anything, mock.Anything).Return(nil)

	service := NewTeamService(repo, mockTx, mockLogger)

	req := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}
	_, err := service.CreateTeam(context.Background(), req)
	require.NoError(t, err)

	deactivateReq := domain.DeactivateTeamUsersRequest{
		TeamName: "backend",
		UserIDs:  []string{"u1", "u2"},
	}
	result, err := service.DeactivateTeamUsers(context.Background(), deactivateReq)
	require.NoError(t, err)
	assert.Len(t, result.DeactivatedUsers, 2)

	team, err := service.GetTeam(context.Background(), "backend")
	require.NoError(t, err)
	for _, member := range team.Members {
		assert.False(t, member.IsActive)
	}
}

func TestTeamService_DeactivateTeamUsers_WithPRReassignment(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)
	ctx := context.Background()

	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()
	mockTx.On("WithinTransaction", mock.Anything, mock.Anything).Return(nil)

	service := NewTeamService(repo, mockTx, mockLogger)

	teamReq := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
			{UserID: "u4", Username: "David", IsActive: true},
			{UserID: "u5", Username: "Eve", IsActive: true},
		},
	}
	_, err := service.CreateTeam(ctx, teamReq)
	require.NoError(t, err)

	now := time.Now()
	pr1 := &domain.PullRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Feature A",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
		CreatedAt:       &now,
	}
	err = repo.CreatePR(ctx, pr1, []string{"u2", "u3"})
	require.NoError(t, err)

	pr2 := &domain.PullRequest{
		PullRequestID:   "pr-002",
		PullRequestName: "Feature B",
		AuthorID:        "u2",
		Status:          domain.PRStatusOpen,
		CreatedAt:       &now,
	}
	err = repo.CreatePR(ctx, pr2, []string{"u3", "u4"})
	require.NoError(t, err)

	pr3 := &domain.PullRequest{
		PullRequestID:   "pr-003",
		PullRequestName: "Feature C",
		AuthorID:        "u5",
		Status:          domain.PRStatusOpen,
		CreatedAt:       &now,
	}
	err = repo.CreatePR(ctx, pr3, []string{"u2"})
	require.NoError(t, err)

	t.Run("deactivates users and reassigns their PRs", func(t *testing.T) {
		deactivateReq := domain.DeactivateTeamUsersRequest{
			TeamName: "backend",
			UserIDs:  []string{"u2", "u3"},
		}

		result, err := service.DeactivateTeamUsers(ctx, deactivateReq)
		require.NoError(t, err)
		assert.NotNil(t, result)

		assert.ElementsMatch(t, []string{"u2", "u3"}, result.DeactivatedUsers)

		assert.NotEmpty(t, result.ReassignedPRs)

		user2, err := repo.GetUser(ctx, "u2")
		require.NoError(t, err)
		assert.False(t, user2.IsActive, "u2 должен быть деактивирован")

		user3, err := repo.GetUser(ctx, "u3")
		require.NoError(t, err)
		assert.False(t, user3.IsActive, "u3 должен быть деактивирован")

		// Проверяем, что PR имеют новых ревьюверов
		_, AssignedReviewers, err := repo.GetPRWithReviewers(ctx, "pr-001")
		require.NoError(t, err)
		assert.NotContains(t, AssignedReviewers, "u2")
		assert.NotContains(t, AssignedReviewers, "u3")

		_, AssignedReviewers, err = repo.GetPRWithReviewers(ctx, "pr-002")
		require.NoError(t, err)
		assert.NotContains(t, AssignedReviewers, "u2")
		assert.NotContains(t, AssignedReviewers, "u3")

		_, AssignedReviewers, err = repo.GetPRWithReviewers(ctx, "pr-003")
		require.NoError(t, err)
		assert.NotContains(t, AssignedReviewers, "u2")
	})
}
