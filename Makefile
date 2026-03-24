# Makefile for Papyrus

-include .env
export

# Build defaults — override via env or .env
GOOS   ?= linux
GOARCH ?= amd64
BIN_DIR = bin/$(GOOS)-$(GOARCH)
BINARY  = $(BIN_DIR)/papyrus

.DEFAULT_GOAL := help

.PHONY: help deps lint test build clean up down run-pdf run docker-build

help: ## Show available targets
	@printf '\033[36m  %-10s\033[0m %s\n' \
		"help"  "Show available targets" \
		"deps"  "Download and tidy dependencies" \
		"lint"  "Run linter (installs golangci-lint if missing)" \
		"test"  "Run tests" \
		"build" "Build binary for GOOS/GOARCH (default: linux/amd64)" \
		"clean" "Remove bin directory" \
		"up"    "Start Ollama + Papyrus stack (docker-compose up)" \
		"down"  "Stop and remove containers (docker-compose down)" \
		"run-pdf" "Analyze a PDF (runs in Docker; requires 'make up' first)" \
		"run"   "Build and run binary locally (requires local Ollama)" \
		"docker-build" "Build Docker image"

up: ## Start Ollama + Papyrus stack
	@echo "Starting docker-compose stack (Ollama + Papyrus)..."
	@docker-compose up -d --build

down: ## Stop and remove containers
	@echo "Stopping docker-compose stack..."
	@docker-compose down

run-pdf: ## Analyze a PDF with override (use: make run-pdf PDF_FILE=pdfs/myfile.pdf CUSTOM_PROMPT="Your custom prompt")
	@if [ -z "$(PDF_FILE)" ]; then echo "Error: PDF_FILE not specified. Usage: make run-pdf PDF_FILE=pdfs/myfile.pdf"; exit 1; fi
	@echo "Analyzing PDF: $(PDF_FILE)"
	@docker-compose run --rm papyrus /$(PDF_FILE) "$(CUSTOM_PROMPT)"

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
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY)..."
	@GOARCH=$(GOARCH) GOOS=$(GOOS) CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) ./cmd/papyrus

clean: ## Remove bin directory
	@rm -rf bin

run: ## Build and run binary locally (requires local Ollama on localhost:11434)
	@echo "Running locally..."
	@GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) $(MAKE) build
	@OLLAMA_URL=http://localhost:11434 ./bin/$(shell go env GOOS)-$(shell go env GOARCH)/papyrus $(ARGS)

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t papyrus:latest .
