package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/storage/memory"
	"pr-reviewer/internal/usecase"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...any) { m.Called(msg, args) }
func (m *MockLogger) Info(msg string, args ...any)  { m.Called(msg, args) }
func (m *MockLogger) Warn(msg string, args ...any)  { m.Called(msg, args) }
func (m *MockLogger) Error(msg string, args ...any) { m.Called(msg, args) }

type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestTeamHandler_CreateTeam(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := usecase.NewTeamService(repo, mockTx, mockLogger)
	handler := NewTeamHandler(service, mockLogger)

	reqBody := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.CreateTeam(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response struct {
		Team domain.TeamResponse `json:"team"`
	}
	err := json.NewDecoder(w.Body).Decode(&response)

	assert.NoError(t, err)
	assert.Equal(t, "backend", response.Team.TeamName)
	assert.Len(t, response.Team.Members, 2)
}

func TestTeamHandler_CreateTeam_AlreadyExists(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := usecase.NewTeamService(repo, mockTx, mockLogger)
	handler := NewTeamHandler(service, mockLogger)

	reqBody := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}

	// Создаем команду
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.CreateTeam(w, req)

	// Пробуем создать снова
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.CreateTeam(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTeamHandler_GetTeam(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := usecase.NewTeamService(repo, mockTx, mockLogger)
	handler := NewTeamHandler(service, mockLogger)

	// Создаем команду
	reqBody := domain.CreateTeamRequest{
		TeamName: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.CreateTeam(w, req)

	// Получаем команду
	req = httptest.NewRequest(http.MethodGet, "/team/get?team_name=backend", nil)
	w = httptest.NewRecorder()
	handler.GetTeam(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response domain.TeamResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "backend", response.TeamName)
}

func TestTeamHandler_GetTeam_NotFound(t *testing.T) {
	repo := memory.NewMemoryRepository()
	mockLogger := new(MockLogger)
	mockTx := new(MockTransactionManager)

	mockLogger.On("Debug", mock.Anything, mock.Anything).Return()
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	service := usecase.NewTeamService(repo, mockTx, mockLogger)
	handler := NewTeamHandler(service, mockLogger)

	// Пытаемся получить несуществующую команду
	req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=nonexistent", nil)
	w := httptest.NewRecorder()
	handler.GetTeam(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRespondJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "success"}

	respondJSON(w, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response["message"])
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	appErr := domain.NewAppError(domain.ErrCodeNotFound, "resource not found")

	respondError(w, http.StatusNotFound, appErr)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response domain.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, domain.ErrCodeNotFound, response.Error.Code)
}
