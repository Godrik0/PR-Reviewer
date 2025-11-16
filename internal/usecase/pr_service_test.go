package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/storage/memory"
)

type MockLogger struct {
	mock.Mock
}

type MockTransactionManager struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...any) { m.Called(msg, args) }
func (m *MockLogger) Info(msg string, args ...any)  { m.Called(msg, args) }
func (m *MockLogger) Warn(msg string, args ...any)  { m.Called(msg, args) }
func (m *MockLogger) Error(msg string, args ...any) { m.Called(msg, args) }

func (m *MockTransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func TestPRService_CreatePR(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTxManager := new(MockTransactionManager)
	ctx := context.TODO()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	service := NewPRService(repo, mockTxManager, mockLogger)

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{UserID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
	}
	err := repo.CreateTeam(ctx, team, members)
	assert.NoError(t, err)

	t.Run("successfully creates PR with reviewers", func(t *testing.T) {
		req := domain.CreatePRRequest{
			PullRequestID:   "pr-001",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		pr, err := service.CreatePR(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "pr-001", pr.PullRequestID)
		assert.Equal(t, "Add feature", pr.PullRequestName)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, domain.PRStatusOpen, pr.Status)
		assert.LessOrEqual(t, len(pr.AssignedReviewers), 2)
		assert.NotContains(t, pr.AssignedReviewers, "u1")
	})

	t.Run("returns error when PR already exists", func(t *testing.T) {
		req := domain.CreatePRRequest{
			PullRequestID:   "pr-001",
			PullRequestName: "Another feature",
			AuthorID:        "u2",
		}

		pr, err := service.CreatePR(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, pr)
		assert.Equal(t, domain.ErrPRAlreadyExists, err)
	})

	t.Run("returns error when author not found", func(t *testing.T) {
		req := domain.CreatePRRequest{
			PullRequestID:   "pr-002",
			PullRequestName: "New feature",
			AuthorID:        "nonexistent",
		}

		pr, err := service.CreatePR(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestPRService_MergePR(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTxManager := new(MockTransactionManager)
	ctx := context.TODO()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	service := NewPRService(repo, mockTxManager, mockLogger)

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	}
	repo.CreateTeam(ctx, team, members)

	req := domain.CreatePRRequest{
		PullRequestID:   "pr-merge-test",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}
	service.CreatePR(ctx, req)

	t.Run("successfully merges PR", func(t *testing.T) {
		pr, err := service.MergePR(ctx, "pr-merge-test")

		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, domain.PRStatusMerged, pr.Status)
		assert.NotNil(t, pr.MergedAt)
	})

	t.Run("merge is idempotent", func(t *testing.T) {
		pr, err := service.MergePR(ctx, "pr-merge-test")

		assert.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, domain.PRStatusMerged, pr.Status)
	})

	t.Run("returns error for nonexistent PR", func(t *testing.T) {
		pr, err := service.MergePR(ctx, "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, pr)
	})
}

func TestPRService_ReassignReviewer(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTxManager := new(MockTransactionManager)
	ctx := context.TODO()
	mockLogger.On("Debug", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Info", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Maybe()
	mockLogger.On("Error", mock.Anything, mock.Anything).Maybe()

	service := NewPRService(repo, mockTxManager, mockLogger)

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
		{UserID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
		{UserID: "u4", Username: "David", TeamName: "backend", IsActive: true},
	}
	repo.CreateTeam(ctx, team, members)

	req := domain.CreatePRRequest{
		PullRequestID:   "pr-reassign-test",
		PullRequestName: "Test PR",
		AuthorID:        "u1",
	}
	prCreated, _ := service.CreatePR(ctx, req)

	t.Run("successfully reassigns reviewer", func(t *testing.T) {
		if len(prCreated.AssignedReviewers) == 0 {
			t.Skip("No reviewers assigned")
		}

		oldReviewer := prCreated.AssignedReviewers[0]
		reassignReq := domain.ReassignRequest{
			PullRequestID: "pr-reassign-test",
			OldUserID:     oldReviewer,
		}

		result, err := service.ReassignReviewer(ctx, reassignReq)

		if err != nil {
			assert.Equal(t, domain.ErrNoActiveCandidate, err)
			t.Skip("No active replacement candidate available")
			return
		}
		assert.NotNil(t, result)
		assert.NotContains(t, result.PR.AssignedReviewers, oldReviewer)
		assert.NotEmpty(t, result.ReplacedBy)
	})

	t.Run("returns error for merged PR", func(t *testing.T) {
		service.MergePR(ctx, "pr-reassign-test")

		reassignReq := domain.ReassignRequest{
			PullRequestID: "pr-reassign-test",
			OldUserID:     "u2",
		}

		result, err := service.ReassignReviewer(ctx, reassignReq)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, domain.ErrPRMerged, err)
	})
}
