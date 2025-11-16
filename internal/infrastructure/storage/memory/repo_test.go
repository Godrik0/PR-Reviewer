package memory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pr-reviewer/internal/domain"
)

func TestNewMemoryRepository(t *testing.T) {
	repo := NewMemoryRepository()
	assert.NotNil(t, repo)
	assert.NotNil(t, repo.teams)
	assert.NotNil(t, repo.users)
	assert.NotNil(t, repo.prs)
	assert.NotNil(t, repo.prReviewers)
}

func TestMemoryRepository_CreateTeam(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	team := &domain.Team{
		TeamName: "backend",
	}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	}

	err := repo.CreateTeam(ctx, team, members)
	require.NoError(t, err)

	// Проверяем существование команды
	exists, err := repo.TeamExists(ctx, "backend")
	require.NoError(t, err)
	assert.True(t, exists)

	// Проверяем существование пользователей
	user, err := repo.GetUser(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "Alice", user.Username)
	assert.Equal(t, "backend", user.TeamName)
}

func TestMemoryRepository_CreateTeam_AlreadyExists(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
	}

	// Создаем команду
	err := repo.CreateTeam(ctx, team, members)
	require.NoError(t, err)

	// Пробуем создать снова
	err = repo.CreateTeam(ctx, team, members)
	assert.Error(t, err)
	assert.Equal(t, domain.ErrTeamAlreadyExists, err)
}

func TestMemoryRepository_GetTeam(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
	}

	// Создаем команду
	err := repo.CreateTeam(ctx, team, members)
	require.NoError(t, err)

	// Получаем команду
	result, err := repo.GetTeam(ctx, "backend")
	require.NoError(t, err)
	assert.Equal(t, "backend", result.TeamName)
	assert.Len(t, result.Members, 2)
}

func TestMemoryRepository_GetTeam_NotFound(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	// Пытаемся получить несуществующую команду
	_, err := repo.GetTeam(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Equal(t, domain.ErrTeamNotFound, err)
}

func TestMemoryRepository_GetUser(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
	}

	err := repo.CreateTeam(ctx, team, members)
	require.NoError(t, err)

	// Получаем юзера
	user, err := repo.GetUser(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "u1", user.UserID)
	assert.Equal(t, "Alice", user.Username)
	assert.True(t, user.IsActive)
}

func TestMemoryRepository_SetUserActive(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
	}

	err := repo.CreateTeam(ctx, team, members)
	require.NoError(t, err)

	// Деактивируем юзера
	err = repo.SetUserActive(ctx, "u1", false)
	require.NoError(t, err)

	// Проверяем изменение
	user, err := repo.GetUser(ctx, "u1")
	require.NoError(t, err)
	assert.False(t, user.IsActive)
}

func TestMemoryRepository_CreatePR(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	pr := &domain.PullRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
	}
	reviewers := []string{"u2", "u3"}

	err := repo.CreatePR(ctx, pr, reviewers)
	require.NoError(t, err)

	// Проверяем изменения в PR
	result, reviewerList, err := repo.GetPRWithReviewers(ctx, "pr-1")
	require.NoError(t, err)
	assert.Equal(t, "pr-1", result.PullRequestID)
	assert.Equal(t, "Add feature", result.PullRequestName)
	assert.Equal(t, domain.PRStatusOpen, result.Status)
	assert.Len(t, reviewerList, 2)
}

func TestMemoryRepository_CreatePR_AlreadyExists(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	pr := &domain.PullRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
	}

	err := repo.CreatePR(ctx, pr, []string{})
	require.NoError(t, err)

	// Пытаемся создать снова
	err = repo.CreatePR(ctx, pr, []string{})
	assert.Error(t, err)
	assert.Equal(t, domain.ErrPRAlreadyExists, err)
}

func TestMemoryRepository_MergePR(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	pr := &domain.PullRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
	}

	err := repo.CreatePR(ctx, pr, []string{"u2"})
	require.NoError(t, err)

	err = repo.MergePR(ctx, "pr-1")
	require.NoError(t, err)

	result, err := repo.GetPR(ctx, "pr-1")
	require.NoError(t, err)
	assert.Equal(t, domain.PRStatusMerged, result.Status)
	assert.NotNil(t, result.MergedAt)
}

func TestMemoryRepository_GetUserReviews(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	pr1 := &domain.PullRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Feature A",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
	}

	pr2 := &domain.PullRequest{
		PullRequestID:   "pr-2",
		PullRequestName: "Feature B",
		AuthorID:        "u1",
		Status:          domain.PRStatusOpen,
	}

	err := repo.CreatePR(ctx, pr1, []string{"u2", "u3"})
	require.NoError(t, err)
	err = repo.CreatePR(ctx, pr2, []string{"u2"})
	require.NoError(t, err)

	prs, err := repo.GetUserReviews(ctx, "u2")
	require.NoError(t, err)
	assert.Len(t, prs, 2)

	prs, err = repo.GetUserReviews(ctx, "u3")
	require.NoError(t, err)
	assert.Len(t, prs, 1)
}

func TestMemoryRepository_GetActiveTeamMembers(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	team := &domain.Team{TeamName: "backend"}
	members := []domain.User{
		{UserID: "u1", Username: "Alice", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "Bob", TeamName: "backend", IsActive: false},
		{UserID: "u3", Username: "Charlie", TeamName: "backend", IsActive: true},
	}

	err := repo.CreateTeam(ctx, team, members)
	require.NoError(t, err)

	activeMembers, err := repo.GetActiveTeamMembers(ctx, "backend", "")
	require.NoError(t, err)
	assert.Len(t, activeMembers, 2)
	// Check that we have both active users
	userIDs := make(map[string]bool)
	for _, member := range activeMembers {
		userIDs[member.UserID] = true
		assert.True(t, member.IsActive)
	}
	assert.True(t, userIDs["u1"])
	assert.True(t, userIDs["u3"])

	activeMembers, err = repo.GetActiveTeamMembers(ctx, "backend", "u1")
	require.NoError(t, err)
	assert.Len(t, activeMembers, 1)
	assert.Equal(t, "u3", activeMembers[0].UserID)
	assert.True(t, activeMembers[0].IsActive)
}
