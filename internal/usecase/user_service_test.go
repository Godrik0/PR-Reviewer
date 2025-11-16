package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/storage/memory"
)

func TestUserService_SetUserActive(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockTx.On("WithinTransaction", mock.Anything, mock.Anything).Return(nil)

	service := NewUserService(repo, mockTx, mockLogger)

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
	}
	err := repo.CreateTeam(context.Background(), team, members)
	require.NoError(t, err)

	req := domain.SetIsActiveRequest{
		UserID:   "u1",
		IsActive: false,
	}

	result, err := service.SetUserActive(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "u1", result.UserID)
	assert.False(t, result.IsActive)
}

func TestUserService_SetUserActive_NotFound(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	mockTx.On("WithinTransaction", mock.Anything, mock.Anything).Return(nil)

	service := NewUserService(repo, mockTx, mockLogger)

	req := domain.SetIsActiveRequest{
		UserID:   "nonexistent",
		IsActive: false,
	}

	_, err := service.SetUserActive(context.Background(), req)
	assert.Error(t, err)
}

func TestUserService_GetUserReviews(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := NewUserService(repo, mockTx, mockLogger)

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	}
	err := repo.CreateTeam(context.Background(), team, members)
	require.NoError(t, err)

	pr := &domain.PullRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
	}
	err = repo.CreatePR(context.Background(), pr, []string{"u2"})
	require.NoError(t, err)

	result, err := service.GetUserReviews(context.Background(), "u2")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "u2", result.UserID)
	assert.Len(t, result.PullRequests, 1)
	assert.Equal(t, "pr-1", result.PullRequests[0].PullRequestID)
}

func TestUserService_GetUserReviews_NoPRs(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := NewUserService(repo, mockTx, mockLogger)

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	}
	err := repo.CreateTeam(context.Background(), team, members)
	require.NoError(t, err)

	result, err := service.GetUserReviews(context.Background(), "u2")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "u2", result.UserID)
	assert.Empty(t, result.PullRequests)
}
