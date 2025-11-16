package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pr-reviewer/internal/config"
	"pr-reviewer/internal/domain"
	"pr-reviewer/internal/infrastructure/auth"
	"pr-reviewer/internal/infrastructure/http"
	"pr-reviewer/internal/infrastructure/http/handlers"
	"pr-reviewer/internal/infrastructure/logger"
	"pr-reviewer/internal/infrastructure/metrics"
	"pr-reviewer/internal/infrastructure/storage/postgres"
	"pr-reviewer/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := logger.NewSlogLogger(cfg.LogLevel)
	logger.Info("Starting PR Reviewer Service")

	metricsCollector := metrics.NewPrometheusMetrics()

	var repo http.Repository
	var txManager domain.TransactionManager

	postgresRepo, err := postgres.NewPostgresRepository(cfg.Storage.PostgresURL)
	if err != nil {
		logger.Error("Failed to initialize postgres repository", slog.Any("error", err))
		os.Exit(1)
	}
	repo = postgresRepo
	txManager = postgres.NewGormTransactionManager(postgresRepo.GetDB())
	defer postgresRepo.Close()
	logger.Info("Using PostgreSQL storage")

	var authenticator auth.Authenticator
	authenticator = auth.NewStaticTokenAuth(cfg.Auth.AdminToken, cfg.Auth.UserToken)

	teamService := usecase.NewTeamService(repo, txManager, logger)
	userService := usecase.NewUserService(repo, txManager, logger)
	prService := usecase.NewPRService(repo, txManager, logger)
	metricsService := usecase.NewMetricsService(repo, txManager, logger)

	teamHandler := handlers.NewTeamHandler(teamService, logger)
	userHandler := handlers.NewUserHandler(userService, logger)
	prHandler := handlers.NewPRHandler(prService, logger)

	srv := http.NewServer(
		cfg,
		teamHandler,
		userHandler,
		prHandler,
		metricsService,
		authenticator,
		metricsCollector,
		logger,
	)

	go func() {
		logger.Info("Starting HTTP server", slog.String("address", fmt.Sprintf(":%d", cfg.Server.Port)))
		if err := srv.Start(); err != nil {
			logger.Error("Server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", slog.Any("error", err))
	}

	logger.Info("Server exited")
}
