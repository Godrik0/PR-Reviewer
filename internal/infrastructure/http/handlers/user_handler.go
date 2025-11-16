package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/usecase"
)

type UserHandler struct {
	service *usecase.UserService
	logger  logger.Logger
}

func NewUserHandler(service *usecase.UserService, logger logger.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

// POST /users/setIsActive
func (h *UserHandler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req domain.SetIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "invalid request body"))
		return
	}

	h.logger.Debug("Set user active request received", "user_id", req.UserID, "is_active", req.IsActive)

	user, err := h.service.SetUserActive(r.Context(), req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusNotFound
			if appErr.Code == domain.ErrCodeNotFound {
				statusCode = http.StatusNotFound
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error setting user active", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// GET /users/getReview
func (h *UserHandler) GetReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "user_id is required"))
		return
	}

	h.logger.Debug("Get user reviews request received", "user_id", userID)

	reviews, err := h.service.GetUserReviews(r.Context(), userID)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusNotFound
			if appErr.Code == domain.ErrCodeNotFound {
				statusCode = http.StatusNotFound
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error getting user reviews", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	respondJSON(w, http.StatusOK, reviews)
}
