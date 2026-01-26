SHELL := /bin/bash

.PHONY: help build run test clean docker-build docker-run docker-stop deps

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	go mod download
	go mod verify

build: ## Build the server binary
	go build -o zenc-server ./cmd/server

run: ## Run the server locally
	go run ./cmd/server

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## Generate test coverage report
	go tool cover -html=coverage.out -o coverage.html

clean: ## Clean build artifacts
	rm -f zenc-server
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	docker build -t aminofox/zenc-server:latest .

docker-run: ## Run with Docker Compose
	docker-compose up -d

docker-stop: ## Stop Docker Compose
	docker-compose down

docker-logs: ## Show Docker logs
	docker-compose logs -f zenc-server

lint: ## Run linter
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	go vet ./...

all: deps fmt vet test build ## Run all checks and build
