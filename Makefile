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
		"up"    "Start full infra stack (Ollama + Piper + OpenTTS)" \
		"up-piper" "Start only the Piper TTS service" \
		"up-opentts" "Start only the OpenTTS service" \
		"up-ollama" "Start only the Ollama LLM service" \
		"down"  "Stop and remove all containers" \
		"run-cli" "Run the Papyrus CLI in a container (requires 'make up')" \
		"run"   "Run the Papyrus binary locally on host" \
		"docker-build" "Build the Papyrus Docker image"

up: ## Start Ollama + Piper + OpenTTS stack
	@echo "Ensuring required voice models are downloaded..."
	@docker-compose run --rm voice-downloader
	@echo "Starting full stack (Ollama + Piper + OpenTTS)..."
	@docker-compose up -d --build

up-piper: ## Start only the Piper service
	@echo "Starting Piper container..."
	@docker-compose up -d piper

up-opentts: ## Start only the OpenTTS service
	@echo "Ensuring required voice models are downloaded..."
	@docker-compose run --rm voice-downloader
	@echo "Starting OpenTTS container..."
	@docker-compose up -d opentts

up-ollama: ## Start only the Ollama service
	@echo "Starting Ollama container..."
	@docker-compose up -d ollama

down: ## Stop and remove containers
	@echo "Stopping docker-compose stack..."
	@docker-compose down

run-cli: ## Run CLI in container (use: make run-cli PDF_FILE=pdfs/myfile.pdf ARGS=--tts)
	@if [ -z "$(PDF_FILE)" ]; then echo "Error: PDF_FILE not specified. Usage: make run-cli PDF_FILE=pdfs/myfile.pdf"; exit 1; fi
	@echo "Executing Papyrus CLI for: $(PDF_FILE)"
	@docker-compose run --rm --no-deps papyrus $(ARGS)

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

run: ## Run binary locally on host (requires 'make up' to provide Ollama/Piper/OpenTTS)
	@echo "Running locally against Docker infra..."
	@$(MAKE) build
	@OLLAMA_URL=http://localhost:11434 PIPER_URL=http://localhost:5000 OPENTTS_URL=http://localhost:5500 ./bin/$(shell go env GOOS)-$(shell go env GOARCH)/papyrus $(ARGS)

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t papyrus:latest .
