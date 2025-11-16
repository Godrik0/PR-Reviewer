package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/usecase"
)

type TeamHandler struct {
	service *usecase.TeamService
	logger  logger.Logger
}

func NewTeamHandler(service *usecase.TeamService, logger logger.Logger) *TeamHandler {
	return &TeamHandler{
		service: service,
		logger:  logger,
	}
}

// POST /team/add
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "invalid request body"))
		return
	}

	h.logger.Debug("Create team request received", "name", req.TeamName)

	team, err := h.service.CreateTeam(r.Context(), req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusBadRequest
			if appErr.Code == domain.ErrCodeTeamExists {
				statusCode = http.StatusBadRequest
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error creating team", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"team": team,
	})
}

// GET /team/get
func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "team_name is required"))
		return
	}

	h.logger.Debug("Get team request received", "team_name", teamName)

	team, err := h.service.GetTeam(r.Context(), teamName)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusNotFound
			if appErr.Code == domain.ErrCodeNotFound {
				statusCode = http.StatusNotFound
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error getting team", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	respondJSON(w, http.StatusOK, team)
}

// POST /team/deactivateUsers
func (h *TeamHandler) DeactivateTeamUsers(w http.ResponseWriter, r *http.Request) {
	var req domain.DeactivateTeamUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "invalid request body"))
		return
	}

	h.logger.Debug("Deactivate team users request received", "team_name", req.TeamName, "user_ids", req.UserIDs)

	result, err := h.service.DeactivateTeamUsers(r.Context(), req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusBadRequest
			if appErr.Code == domain.ErrCodeNotFound {
				statusCode = http.StatusNotFound
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error deactivating team users", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	respondJSON(w, http.StatusOK, result)
}
