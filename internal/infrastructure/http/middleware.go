package http

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"pr-reviewer/internal/infrastructure/auth"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/metrics"
)

func AuthMiddleware(auth auth.Authenticator, logger logger.Logger, requireAdmin bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn("Missing authorization header")
				respondUnauthorized(w)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			var valid bool
			if requireAdmin {
				valid = auth.ValidateAdminToken(token)
			} else {
				valid = auth.ValidateUserToken(token)
			}

			if !valid {
				logger.Warn("Invalid token", slog.Bool("requireAdmin", requireAdmin))
				respondUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func LoggingMiddleware(logger logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			logger.Debug("HTTP request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Duration("duration", duration),
			)
		})
	}
}

func MetricsMiddleware(metrics metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			if metrics != nil {
				duration := time.Since(start).Seconds()
				metrics.IncHTTPRequests(r.Method, r.URL.Path, rw.statusCode)
				metrics.ObserveHTTPDuration(r.Method, r.URL.Path, duration)
			}
		})
	}
}
