package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pr-reviewer/internal/config"
	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/auth"
	httpInfra "pr-reviewer/internal/infrastructure/http"
	"pr-reviewer/internal/infrastructure/http/handlers"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/metrics"
	"pr-reviewer/internal/infrastructure/storage/memory"
	"pr-reviewer/internal/usecase"
)

type NoOpTransactionManager struct{}

func (n *NoOpTransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func setupTestServer(t *testing.T) *httpInfra.Server {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			ReadTimeout:  10,
			WriteTimeout: 10,
		},
		Auth: config.AuthConfig{
			Type:       "static",
			AdminToken: "test-admin-token",
			UserToken:  "test-user-token",
		},
		LogLevel: "error",
	}

	appLogger := logger.NewSlogLogger("error")
	var metricsCollector metrics.Metrics = nil
	repo := memory.NewMemoryRepository()
	authenticator := auth.NewStaticTokenAuth(cfg.Auth.AdminToken, cfg.Auth.UserToken)
	txManager := &NoOpTransactionManager{}

	teamService := usecase.NewTeamService(repo, txManager, appLogger)
	userService := usecase.NewUserService(repo, txManager, appLogger)
	prService := usecase.NewPRService(repo, txManager, appLogger)
	metricsService := usecase.NewMetricsService(repo, txManager, appLogger)

	teamHandler := handlers.NewTeamHandler(teamService, appLogger)
	userHandler := handlers.NewUserHandler(userService, appLogger)
	prHandler := handlers.NewPRHandler(prService, appLogger)

	return httpInfra.NewServer(
		cfg,
		teamHandler,
		userHandler,
		prHandler,
		metricsService,
		authenticator,
		metricsCollector,
		appLogger,
	)
}

func TestIntegration_CreateTeamAndPR(t *testing.T) {
	server := setupTestServer(t)

	teamReq := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
		},
	}

	body, _ := json.Marshal(teamReq)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	prReq := domain.CreatePRRequest{
		PullRequestID:   "pr-001",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
	}

	body, _ = json.Marshal(prReq)
	req = httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "test-admin-token")
	w = httptest.NewRecorder()

	server.Router().ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	var response struct {
		PR domain.PullRequestResponse `json:"pr"`
	}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "pr-001", response.PR.PullRequestID)
	assert.Equal(t, domain.PRStatusOpen, response.PR.Status)
	assert.LessOrEqual(t, len(response.PR.AssignedReviewers), 2)
	assert.NotContains(t, response.PR.AssignedReviewers, "u1")
}

func TestIntegration_FullWorkflow(t *testing.T) {
	server := setupTestServer(t)

	teamReq := domain.CreateTeamRequest{
		TeamName: "payments",
		Members: []domain.TeamMember{
			{UserID: "p1", Username: "PayAlice", IsActive: true},
			{UserID: "p2", Username: "PayBob", IsActive: true},
			{UserID: "p3", Username: "PayCharlie", IsActive: true},
			{UserID: "p4", Username: "PayDavid", IsActive: true},
		},
	}

	body, _ := json.Marshal(teamReq)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	prReq := domain.CreatePRRequest{
		PullRequestID:   "pr-workflow",
		PullRequestName: "Payment gateway integration",
		AuthorID:        "p1",
	}

	body, _ = json.Marshal(prReq)
	req = httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "test-admin-token")
	w = httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var createResp struct {
		PR domain.PullRequestResponse `json:"pr"`
	}
	json.NewDecoder(w.Body).Decode(&createResp)

	if len(createResp.PR.AssignedReviewers) > 0 {
		reassignReq := domain.ReassignRequest{
			PullRequestID: "pr-workflow",
			OldUserID:     createResp.PR.AssignedReviewers[0],
		}

		body, _ = json.Marshal(reassignReq)
		req = httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-admin-token")
		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	mergeReq := domain.MergePRRequest{
		PullRequestID: "pr-workflow",
	}

	body, _ = json.Marshal(mergeReq)
	req = httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "test-admin-token")
	w = httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var mergeResp struct {
		PR domain.PullRequestResponse `json:"pr"`
	}
	json.NewDecoder(w.Body).Decode(&mergeResp)
	assert.Equal(t, domain.PRStatusMerged, mergeResp.PR.Status)
	assert.NotNil(t, mergeResp.PR.MergedAt)
}

func TestIntegration_Authentication(t *testing.T) {
	server := setupTestServer(t)

	teamReq := domain.CreateTeamRequest{
		TeamName: "auth-test",
		Members: []domain.TeamMember{
			{UserID: "a1", Username: "AuthUser", IsActive: true},
		},
	}
	body, _ := json.Marshal(teamReq)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	t.Run("admin endpoint requires admin token", func(t *testing.T) {
		setActiveReq := domain.SetIsActiveRequest{
			UserID:   "a1",
			IsActive: false,
		}

		body, _ := json.Marshal(setActiveReq)

		// Без токена
		req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// С токеном пользователя (должен провалиться)
		req = httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-user-token")
		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		// С токеном администратора
		req = httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "test-admin-token")
		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("user endpoint accepts both tokens", func(t *testing.T) {
		// С токеном пользователя
		req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=auth-test", nil)
		req.Header.Set("Authorization", "test-user-token")
		w := httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// С токеном администратора
		req = httptest.NewRequest(http.MethodGet, "/team/get?team_name=auth-test", nil)
		req.Header.Set("Authorization", "test-admin-token")
		w = httptest.NewRecorder()
		server.Router().ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
