# Makefile for Papyrus

-include .env
export

# Build defaults — override via env or .env
GOOS   ?= linux
GOARCH ?= amd64
BINARY  = papyrus-$(GOOS)-$(GOARCH)

.DEFAULT_GOAL := help

.PHONY: help deps lint test build run ship

help: ## Show available targets
	@printf '\033[36m  %-10s\033[0m %s\n' \
		"help"  "Show available targets" \
		"deps"  "Download and tidy dependencies" \
		"lint"  "Run linter (installs golangci-lint if missing)" \
		"test"  "Run tests" \
		"build" "Build binary for GOOS/GOARCH (default: linux/amd64)" \
		"run"   "Build and run locally" \
		"ship"  "Build and tag Docker image"

deps: ## Download and tidy dependencies
	@echo "Tidying dependencies..."
	@go mod tidy
	@echo "Downloading dependencies..."
	@go mod download

lint: ## Run linter (installs golangci-lint if missing)
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
	fi
	@echo "Running linter..."
	@golangci-lint run

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

build: ## Build binary for GOOS/GOARCH (default: linux/amd64)
	@echo "Building $(BINARY)..."
	@GOARCH=$(GOARCH) GOOS=$(GOOS) CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) .

run: build ## Build and run locally
	@echo "Running $(BINARY)..."
	@./$(BINARY) $(ARGS)

ship: ## Build and tag Docker image
	@echo "Building Docker image..."
	@docker buildx build --platform $(GOOS)/$(GOARCH) -t papyrus:latest .