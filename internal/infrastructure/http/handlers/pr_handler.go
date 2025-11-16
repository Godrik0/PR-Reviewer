package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/usecase"
)

type PRHandler struct {
	service *usecase.PRService
	logger  logger.Logger
}

func NewPRHandler(service *usecase.PRService, logger logger.Logger) *PRHandler {
	return &PRHandler{
		service: service,
		logger:  logger,
	}
}

// POST /pullRequest/create
func (h *PRHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req domain.CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "invalid request body"))
		return
	}

	h.logger.Debug("Create PR request received", "name", req.PullRequestName, "author_id", req.AuthorID)

	pr, err := h.service.CreatePR(r.Context(), req)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusBadRequest
			switch appErr.Code {
			case domain.ErrCodePRExists:
				statusCode = http.StatusConflict
			case domain.ErrCodeNotFound:
				statusCode = http.StatusNotFound
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error creating PR", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"pr": pr,
	})
}

// POST /pullRequest/merge
func (h *PRHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req domain.MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "invalid request body"))
		return
	}

	h.logger.Debug("Merge PR request received", "pr_id", req.PullRequestID)

	pr, err := h.service.MergePR(r.Context(), req.PullRequestID)
	if err != nil {
		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusNotFound
			if appErr.Code == domain.ErrCodeNotFound {
				statusCode = http.StatusNotFound
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error merging PR", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	h.logger.Debug("PR merged successfully", "pr_id", req.PullRequestID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr": pr,
	})
}

// POST /pullRequest/reassign
func (h *PRHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req domain.ReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body", slog.Any("error", err))
		respondError(w, http.StatusBadRequest, domain.NewAppError(domain.ErrCodeBadRequest, "invalid request body"))
		return
	}

	h.logger.Debug("Reassign reviewer request received",
		"pr_id", req.PullRequestID,
		"old_user_id", req.OldUserID)

	response, err := h.service.ReassignReviewer(r.Context(), req)
	if err != nil {
		h.logger.Warn("Reassign reviewer failed",
			"pr_id", req.PullRequestID,
			"old_user_id", req.OldUserID,
			"error", err)

		if appErr, ok := err.(*domain.AppError); ok {
			statusCode := http.StatusConflict
			switch appErr.Code {
			case domain.ErrCodeNotFound:
				statusCode = http.StatusNotFound
			case domain.ErrCodePRMerged, domain.ErrCodeNotAssigned, domain.ErrCodeNoCandidate:
				statusCode = http.StatusConflict
			}
			respondError(w, statusCode, appErr)
			return
		}
		h.logger.Error("Internal error reassigning reviewer", slog.Any("error", err))
		respondError(w, http.StatusInternalServerError, domain.NewAppError(domain.ErrCodeInternal, "internal server error"))
		return
	}

	h.logger.Debug("Reviewer reassigned successfully",
		"pr_id", req.PullRequestID,
		"old_user_id", req.OldUserID,
		"new_reviewer", response.ReplacedBy)

	respondJSON(w, http.StatusOK, response)
}
