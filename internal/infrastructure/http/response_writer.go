package http

import (
	"net/http"
	"pr-reviewer/internal/domain"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	errResp := domain.NewErrorResponse(domain.ErrUnauthorized)
	w.Write([]byte(`{"error":{"code":"UNAUTHORIZED","message":"unauthorized"}}`))
	_ = errResp
}
