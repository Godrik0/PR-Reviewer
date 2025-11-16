.PHONY: build run test lint clean docker-up docker-down load-test docker-up-dev docker-down-volumes docker-restart docker-restart-volumes

APP_NAME=pr-reviewer
DOCKER_COMPOSE=docker-compose
GO=go

build:
	@echo "Building $(APP_NAME)..."
	$(GO) build -o $(APP_NAME) ./cmd/app

run:
	@echo "Running $(APP_NAME)..."
	$(GO) run ./cmd/app

test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

test-short:
	@echo "Running short tests..."
	$(GO) test -short -v ./...

lint:
	@echo "Running linter..."
	golangci-lint run

docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-up:
	@echo "Starting services with docker-compose..."
	$(DOCKER_COMPOSE) up -d
	@echo "Services started. API available at http://localhost:8080"
	@echo "Prometheus available at http://localhost:9090"
	@echo "Grafana available at http://localhost:3000 (admin/admin)"

docker-up-dev:
	@echo "Starting services in development mode with docker-compose..."
	$(DOCKER_COMPOSE) -f docker-compose.dev.yml up -d
	@echo "Services started in development mode. API available at http://localhost:8080"
	@echo "Prometheus available at http://localhost:9090"
	@echo "Grafana available at http://localhost:3000 (admin/admin)"

docker-down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down

docker-down-volumes:
	@echo "Stopping services and removing volumes..."
	$(DOCKER_COMPOSE) down -v

docker-restart: docker-down docker-up

docker-restart-volumes: docker-down-volumes docker-up

load-test:
	@echo "Running load tests with k6..."
	docker-compose run --rm k6 run /scripts/main-load-test.js --out experimental-prometheus-rw

clean:
	@echo "Cleaning..."
	rm -f $(APP_NAME)
	rm -f coverage.out coverage.html
	$(GO) clean

deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	goimports -w .
	

.DEFAULT_GOAL := help