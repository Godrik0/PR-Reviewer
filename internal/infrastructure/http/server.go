package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"pr-reviewer/internal/config"
	"pr-reviewer/internal/infrastructure/auth"
	"pr-reviewer/internal/infrastructure/http/handlers"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/metrics"
	"pr-reviewer/internal/infrastructure/storage"
	"pr-reviewer/internal/usecase"
)

type Repository = storage.Repository

type Server struct {
	cfg            *config.Config
	router         *chi.Mux
	server         *http.Server
	teamHandler    *handlers.TeamHandler
	userHandler    *handlers.UserHandler
	prHandler      *handlers.PRHandler
	metricsService *usecase.MetricsService
	auth           auth.Authenticator
	metrics        metrics.Metrics
	logger         logger.Logger
}

func NewServer(
	cfg *config.Config,
	teamHandler *handlers.TeamHandler,
	userHandler *handlers.UserHandler,
	prHandler *handlers.PRHandler,
	metricsService *usecase.MetricsService,
	auth auth.Authenticator,
	metrics metrics.Metrics,
	logger logger.Logger,
) *Server {
	s := &Server{
		cfg:            cfg,
		teamHandler:    teamHandler,
		userHandler:    userHandler,
		prHandler:      prHandler,
		metricsService: metricsService,
		auth:           auth,
		metrics:        metrics,
		logger:         logger,
	}

	s.setupRouter()

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      s.router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	return s
}

func (s *Server) setupRouter() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(LoggingMiddleware(s.logger))
	r.Use(MetricsMiddleware(s.metrics))
	r.Use(middleware.Timeout(60 * time.Second))

	// Маршруты метрик
	r.Get("/health", s.healthCheck)
	r.Get("/metrics", promhttp.Handler().ServeHTTP)

	// Маршруты для команд
	r.Post("/team/add", s.teamHandler.CreateTeam)
	r.With(AuthMiddleware(s.auth, s.logger, false)).Get("/team/get", s.teamHandler.GetTeam)
	r.With(AuthMiddleware(s.auth, s.logger, true)).Post("/team/deactivateUsers", s.teamHandler.DeactivateTeamUsers)

	// Маршруты для пользователей
	r.With(AuthMiddleware(s.auth, s.logger, true)).Post("/users/setIsActive", s.userHandler.SetIsActive)
	r.With(AuthMiddleware(s.auth, s.logger, false)).Get("/users/getReview", s.userHandler.GetReviews)

	// Маршруты для pull request
	r.With(AuthMiddleware(s.auth, s.logger, true)).Post("/pullRequest/create", s.prHandler.CreatePR)
	r.With(AuthMiddleware(s.auth, s.logger, true)).Post("/pullRequest/merge", s.prHandler.MergePR)
	r.With(AuthMiddleware(s.auth, s.logger, true)).Post("/pullRequest/reassign", s.prHandler.ReassignReviewer)

	r.Get("/stats", s.getStats)

	s.router = r
}

func (s *Server) Start() error {
	s.logger.Info("HTTP server listening", "port", s.cfg.Server.Port)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}

func (s *Server) Router() *chi.Mux {
	return s.router
}
