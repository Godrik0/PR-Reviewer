package domain

import "fmt"

type ErrorCode string

const (
	ErrCodeTeamExists  ErrorCode = "TEAM_EXISTS"
	ErrCodePRExists    ErrorCode = "PR_EXISTS"
	ErrCodePRMerged    ErrorCode = "PR_MERGED"
	ErrCodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	ErrCodeNoCandidate ErrorCode = "NO_CANDIDATE"
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
	ErrCodeInternal    ErrorCode = "INTERNAL_ERROR"
	ErrCodeBadRequest  ErrorCode = "BAD_REQUEST"
	ErrCodeUnauth      ErrorCode = "UNAUTHORIZED"
)

type AppError struct {
	Code    ErrorCode
	Message string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewAppError(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

var (
	ErrTeamAlreadyExists   = NewAppError(ErrCodeTeamExists, "team_name already exists")
	ErrPRAlreadyExists     = NewAppError(ErrCodePRExists, "PR id already exists")
	ErrPRMerged            = NewAppError(ErrCodePRMerged, "cannot reassign on merged PR")
	ErrReviewerNotAssigned = NewAppError(ErrCodeNotAssigned, "reviewer is not assigned to this PR")
	ErrNoActiveCandidate   = NewAppError(ErrCodeNoCandidate, "no active replacement candidate in team")
	ErrTeamNotFound        = NewAppError(ErrCodeNotFound, "team not found")
	ErrUserNotFound        = NewAppError(ErrCodeNotFound, "user not found")
	ErrPRNotFound          = NewAppError(ErrCodeNotFound, "PR not found")
	ErrUnauthorized        = NewAppError(ErrCodeUnauth, "unauthorized")
	ErrInvalidToken        = NewAppError(ErrCodeUnauth, "invalid token")
)

func NewDatabaseError(operation string, err error) *AppError {
	return NewAppError(ErrCodeInternal, fmt.Sprintf("database %s failed: %v", operation, err))
}

type ErrorResponse struct {
	Error struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
	} `json:"error"`
}

func NewErrorResponse(err *AppError) ErrorResponse {
	var resp ErrorResponse
	resp.Error.Code = err.Code
	resp.Error.Message = err.Message
	return resp
}
