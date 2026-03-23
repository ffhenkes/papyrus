# Makefile for Papyrus

-include .env
export

# Build defaults — override via env or .env
GOOS   ?= linux
GOARCH ?= amd64
BIN_DIR = bin/$(GOOS)-$(GOARCH)
BINARY  = $(BIN_DIR)/papyrus

.DEFAULT_GOAL := help

.PHONY: help deps lint test build clean up run-pdf run ship

help: ## Show available targets
	@printf '\033[36m  %-10s\033[0m %s\n' \
		"help"  "Show available targets" \
		"deps"  "Download and tidy dependencies" \
		"lint"  "Run linter (installs golangci-lint if missing)" \
		"test"  "Run tests" \
		"build" "Build binary for GOOS/GOARCH (default: linux/amd64)" \
		"clean" "Remove bin directory" \
		"up"    "Run docker-compose (v1.29.0) stack using .env" \
		"run-pdf" "Run papyrus on a PDF file (e.g., make run-pdf PDF_FILE=pdfs/test.pdf)" \
		"run"   "Build and run locally" \
		"ship"  "Build and tag Docker image"

up: ## Run docker-compose stack
	@echo "Cleaning up previous containers and running docker-compose (v1.29.0) stack..."
	@docker-compose down -v
	@docker-compose up --build

run-pdf: ## Run papyrus on a PDF file
	@if [ -z "$(PDF_FILE)" ]; then echo "Error: PDF_FILE is not set. Usage: make run-pdf PDF_FILE=pdfs/file.pdf"; exit 1; fi
	@docker-compose run --rm papyrus /$(PDF_FILE)

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
	@GOARCH=$(GOARCH) GOOS=$(GOOS) CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) .

clean: ## Remove bin directory
	@rm -rf bin

run: build ## Build and run locally
	@echo "Running $(BINARY)..."
	@./$(BINARY) $(ARGS)
