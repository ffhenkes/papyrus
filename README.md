# Papyrus

**Papyrus** is a tool to analyze and explain PDF (and text) documents using [Ollama](https://ollama.ai/) — a local LLM runtime. Extract insights, summaries, and answers from your documents using customizable prompts, local language models, optional RAG-powered semantic retrieval, and neural text-to-speech.

## Table of Contents

- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Session Persistence](#session-persistence)
- [RAG (Retrieval-Augmented Generation)](#rag-retrieval-augmented-generation)
- [Text-to-Speech (TTS)](#text-to-speech-tts)
- [Configuration](#configuration)
- [Available Commands](#available-commands)
- [Architecture](#architecture)
- [Troubleshooting](#troubleshooting)

## Quick Start

Get Papyrus running in 2 steps:

### Option 1: Docker-based (Recommended)

```bash
# 1. Navigate to the project
cd papyrus

# 2. Configure your PDF file and model (edit .env or inline)
export OLLAMA_MODEL=qwen3:8b
export PDF_FILE=pdfs/test.pdf
export CUSTOM_PROMPT="Summarize this document"

# 3. Start the full stack (Ollama + Piper TTS)
make up

# 4. Analyze a PDF using the containerized CLI
make run-cli PDF_FILE=pdfs/test.pdf ARGS="--tts"
```

To stop:

```bash
make down
```

### Option 2: Local binary (requires local Ollama)

```bash
# Prerequisites: Ollama must be running on your machine (http://localhost:11434)

# Build the binary
make build

# Run it
make run ARGS="pdfs/test.pdf 'Summarize this document'"
```

## Prerequisites

- **Docker & Docker Compose:** v1.29.0 or higher
  - [Install Docker](https://docs.docker.com/get-docker/)
  - Docker Compose comes with Docker Desktop
- **Go** (optional, only for local development): v1.24.1 or higher
- **System Resources:**
  - Minimum 8GB RAM (for running Ollama + Papyrus)
  - At least 4GB free disk space for Ollama models

## Installation

### Docker-based (Recommended)

No additional installation needed beyond Docker and Docker Compose. The `make up` command handles everything.

### Local Development

For development without Docker:

```bash
# Install Go dependencies
make deps

# Run linter
make lint

# Run tests
make test

# Build the binary locally
make build

# Run against local Ollama (must be running on localhost:11434)
make run ARGS="pdfs/test.pdf 'Summarize this'"
```

**Note:** Local mode requires Ollama installed and running separately on your machine. Docker mode (recommended) includes Ollama automatically.

## Usage

### Two Usage Modes

**Mode 1: Docker (Recommended)** — Everything isolated in containers
- Use: `make run-cli PDF_FILE=pdfs/test.pdf`
- Ollama runs inside Docker: `http://ollama:11434`
- Piper TTS runs inside Docker: `http://piper:5000`
- Models stored in `./ollama_data/` (persistent)
- No local dependencies needed

**Mode 2: Local Binary** — Run natively on your machine
- Use: `make run ARGS="pdfs/test.pdf --tts"`
- Requires: Ollama running locally on `http://localhost:11434`
- Useful for: development, integration with local tools
- Note: Must start Ollama separately (not included)

### Supported Input Formats

Papyrus supports two input file formats:

- **PDF files** (`.pdf`) — Text is extracted page-by-page using the `ledongthuc/pdf` library. Scanned/image-only PDFs are not supported (see [Troubleshooting](#pdf-parsing-fails-or-scanned-image-pdf-error)).
- **Text files** (`.txt`) — Read directly as plain text. Useful for pre-processed documents or OCR output.

### Docker-based Analysis (Recommended)

```bash
# One-shot analysis (starts everything, runs, then exits)
make run-cli PDF_FILE=pdfs/myfile.pdf CUSTOM_PROMPT="List the main topics"

# Or use persistent stack (Ollama + Piper)
make up                              # Start background services
make run-cli PDF_FILE=pdfs/file1.pdf ARGS="--tts"  # Run CLI with speech
make run-cli PDF_FILE=pdfs/file2.pdf               # Run CLI without speech
make down                            # Clean up
```

### Local Binary Analysis

```bash
# Requires: Ollama running on localhost:11434
make build
make run ARGS="pdfs/test.pdf"

# With TTS enabled (requires Piper running)
make run ARGS="pdfs/test.pdf --tts"

# With RAG enabled (requires ChromaDB running)
make run ARGS="pdfs/test.pdf --rag"
```

## Session Persistence

Papyrus automatically saves your conversation to disk so you can **resume analysis later** without re-processing the PDF.

Sessions are stored in `~/.papyrus/sessions/` as JSON files. Each session includes the full document text and conversation history.

### Session Flags

```bash
# List all saved sessions
papyrus --list
papyrus --sessions   # alias

# Resume a saved session (enter REPL directly, no PDF needed)
papyrus --session <session-id>

# Delete a saved session
papyrus --delete <session-id>

# Advanced Flags
papyrus --export                     # Export to MD and exit
papyrus --no-cache                   # Disable semantic cache
papyrus --max-context 8192           # Restrict token history (default: 8192)
papyrus --tts                        # Enable text-to-speech (with SSML support)
papyrus --rag                        # Enable RAG (retrieval-augmented generation)
```

**Example workflow:**

```bash
# First run: analyze PDF, auto-saved on exit
papyrus pdfs/report.pdf

# Later: resume where you left off
papyrus --list                           # shows session IDs
papyrus --session report-abc123def456    # resume
```

### Interactive REPL Commands

While in interactive mode, the following commands are available:

| Command | Description |
|---------|-------------|
| `history` | Show all messages in the current session |
| `stats` | Show cumulative token usage statistics (input, output, total) |
| `export` | Export the current session to a Markdown file |
| `save` | Explicitly save the session to disk |
| `session info` | Show session metadata (ID, file, timestamps, message count, doc size) |
| `exit` / `quit` | Save and exit |

## RAG (Retrieval-Augmented Generation)

Papyrus supports RAG for improved accuracy on large documents. Instead of sending the entire document text to the LLM, RAG chunks the document, stores embeddings in a vector database, and retrieves only the most relevant chunks for each query.

### How it Works

1. **Chunking** — The document is split into overlapping chunks using BPE token counting (`cl100k_base` encoding via [tiktoken-go](https://github.com/pkoukk/tiktoken-go)).
2. **Embedding** — Each chunk is embedded using an Ollama embedding model (default: `nomic-embed-text`) via the `/api/embed` endpoint.
3. **Storage** — Embeddings and chunk text are stored in [ChromaDB](https://www.trychroma.com/) with a collection name derived from a SHA-256 hash of the document text.
4. **Retrieval** — For each user query, the query is embedded and the top-K most similar chunks are retrieved and injected into the system prompt.
5. **Caching** — If the document has already been ingested (same hash), ingestion is skipped.

### Prerequisites

RAG requires a running ChromaDB instance. The Docker Compose stack includes one:

```bash
# Start the full stack including ChromaDB
make up

# Or start just ChromaDB for local development
docker compose --profile rag up -d chromadb
```

### Usage

```bash
# Docker mode with RAG
make run-cli PDF_FILE=pdfs/large-report.pdf ARGS="--rag"

# Local mode with RAG
make run ARGS="pdfs/large-report.pdf --rag"

# Customize retrieval parameters
make run ARGS="pdfs/report.pdf --rag --top-k 10 --chunk-size 256"
```

### RAG Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--rag` | `false` | Enable RAG mode |
| `--vectordb-url` | `http://localhost:8000` | ChromaDB URL (overrides `VECTORDB_URL` env) |
| `--embed-model` | `nomic-embed-text` | Ollama embedding model (overrides `EMBED_MODEL` env) |
| `--top-k` | `5` | Number of chunks to retrieve per query |
| `--chunk-size` | `512` | Tokens per chunk for document ingestion |

### RAG Environment Variables

| Variable | Docker Default | Local Default | Description |
|----------|----------------|---------------|-------------|
| `VECTORDB_URL` | `http://chromadb:8000` | `http://localhost:8000` | ChromaDB API endpoint |
| `EMBED_MODEL` | `nomic-embed-text` | `nomic-embed-text` | Ollama model for generating embeddings |

## Text-to-Speech (TTS)

Papyrus can generate speech (audio) for model responses using [Piper](https://github.com/rhasspy/piper) with SSML support implemented in Papyrus.

### How it Works

1. **Neural Voice Quality:** Papyrus uses Piper for high-quality, fast, local neural TTS synthesis.
2. **Markdown-to-SSML Conversion:** Papyrus automatically converts LLM markdown output to SSML, adding prosody adjustments for headers (slightly faster, higher pitch), bold text (slower, emphasized), list items, and block quotes. Code blocks are silently removed.
3. **SSML Parsing and Synthesis:** Papyrus parses the SSML markup into individual segments and synthesizes each with Piper separately. This enables tags like `<speak>`, `<break time="500ms"/>`, `<voice>`, `<prosody>`, and `<s>` for precise control over speech pacing, pauses, and voice switching — without requiring SSML support in Piper itself.
4. **Audio Mixing:** Individual WAV segments (including generated silence for `<break>` tags) are concatenated into a single WAV file.
5. **Activation:** Use the `--tts` flag to enable text-to-speech.
6. **Output:** Audio files are saved as `.wav` files in the `voice/` directory.
    - Initial explanation: `voice/<session-id>_initial.wav`
    - REPL responses: `voice/<session-id>_<message-index>.wav`

### Configuration for TTS

| Variable | Default | Description |
|----------|---------|-------------|
| `PIPER_URL` | `http://piper:5000` (Docker) / `http://localhost:5000` (local) | Piper HTTP endpoint |
| `PIPER_VOICE` | `en_US-lessac-medium` | Piper voice ID (e.g., `pt_BR-faber-medium`, `en_US-lessac-medium`) |
| `PIPER_VOICE_URL` | *(see .env.example)* | Download URL for the Piper voice model (used by Docker) |

**Example:**
```bash
# Analyze with TTS (SSML support enabled in Papyrus)
make run-cli PDF_FILE=pdfs/report.pdf ARGS="--tts"
```

### Changing the TTS Voice

You can change the language or voice used by Piper by setting `PIPER_VOICE` and `PIPER_VOICE_URL` in your `.env` file. You can find more voices on the [Piper HuggingFace repository](https://huggingface.co/rhasspy/piper-voices).

**English (US) Example (default):**
```bash
PIPER_VOICE=en_US-lessac-medium
PIPER_VOICE_URL=https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx?download=true
```

**Portuguese (Brazil) Example:**
```bash
PIPER_VOICE=pt_BR-faber-medium
PIPER_VOICE_URL=https://huggingface.co/rhasspy/piper-voices/resolve/main/pt/pt_BR/faber/medium/pt_BR-faber-medium.onnx?download=true
```

Simply update the variables and run `make up` to restart the Piper service with the new model.

## Configuration

All settings are managed via the `.env` file. Create one from the template:

```bash
cp .env.example .env
```

### Environment Variables

| Variable | Docker Default | Local Default | Description |
|----------|----------------|---------------|-------------|
| `OLLAMA_URL` | `http://ollama:11434` | `http://localhost:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `qwen3:8b` | `qwen3:8b` | LLM model to use (must be installed in Ollama) |
| `PDF_FILE` | `pdfs/test.pdf` | N/A | Path to PDF file to analyze |
| `CUSTOM_PROMPT` | `"Explain this document."` | N/A | Custom prompt for document analysis |
| `PIPER_URL` | `http://piper:5000` | `http://localhost:5000` | Piper TTS API endpoint |
| `PIPER_VOICE` | `en_US-lessac-medium` | `en_US-lessac-medium` | Piper voice ID |
| `PIPER_VOICE_URL` | *(HuggingFace URL)* | N/A | Download URL for the voice model (Docker only) |
| `VECTORDB_URL` | `http://chromadb:8000` | `http://localhost:8000` | ChromaDB API endpoint (for RAG) |
| `EMBED_MODEL` | `nomic-embed-text` | `nomic-embed-text` | Ollama model for embeddings (for RAG) |

**Important:**
- **Docker mode** uses internal service names (`http://ollama:11434`, `http://piper:5000`, `http://chromadb:8000`)
- **Local binary mode** (`make run`) automatically sets `OLLAMA_URL=http://localhost:11434` and `PIPER_URL=http://localhost:5000`
- **To run locally**, start Ollama on your machine and it will listen on `http://localhost:11434`

### Persistent Data Directories

| Directory | Purpose |
|-----------|---------|
| `./ollama_data/` | Ollama model cache (Docker volume) |
| `./chroma_data/` | ChromaDB vector data (Docker volume, RAG mode) |
| `./voice/` | Generated TTS audio files (`.wav`) |
| `./pdfs/` | Input PDF files (mounted as `/pdfs` in containers) |
| `~/.papyrus/sessions/` | Session persistence (JSON files) |
| `~/.papyrus/cache/` | LLM response cache (per-session, 24h TTL) |

### Supported Ollama Models

Common models available in Ollama (install/pull them beforehand):
- `qwen3:8b` — Fast, general-purpose (recommended default). Supports reasoning (`<think>` blocks).
- `llama2:7b` — General-purpose, well-rounded
- `mistral:7b` — Fast, excellent for summaries
- `neural-chat:7b` — Optimized for conversations
- `gemma:7b` — Lightweight, good performance
- `deepseek-r1:14b` — Reasoning-focused (higher resource needs)

For more models, visit [Ollama Model Library](https://ollama.ai/library).

### Customizing Prompts

Replace the default prompt with your own use case. Edit `.env` or pass `CUSTOM_PROMPT` on the command line:

```bash
# Extract structured data (JSON format)
make run-cli PDF_FILE=pdfs/test.pdf CUSTOM_PROMPT="Extract all JSON data"

# Generate specific summary with speech
make run-cli PDF_FILE=pdfs/test.pdf CUSTOM_PROMPT="Create summary" ARGS="--tts"

# Q&A mode
make run-cli PDF_FILE=pdfs/test.pdf CUSTOM_PROMPT="What are the risks?"
```

The system prompt used is:
> You are an expert document analyst. When given document content, you:
> 1. Identify the document type and purpose
> 2. Summarize the key topics and main points clearly
> 3. Highlight important details, data, or findings
> 4. Explain any technical concepts in accessible language
> 5. Note the document structure and how it's organized

## Available Commands

### Makefile Targets

| Command | Description |
|---------|-------------|
| `make help` | Show all available targets |
| `make up` | Start full infra stack: Ollama + Piper (`docker-compose up -d --build`) |
| `make up-ollama` | Start only the Ollama LLM service |
| `make up-piper` | Start only the Piper TTS service |
| `make down` | Stop and remove all containers (`docker-compose down`) |
| `make run-cli PDF_FILE=... [CUSTOM_PROMPT=...] [ARGS=...]` | Run CLI in container |
| `make build` | Build binary locally (default: `linux/amd64`, override with `GOOS`/`GOARCH`) |
| `make run ARGS="..."` | Build and run binary locally (sets `OLLAMA_URL` and `PIPER_URL` to `localhost`) |
| `make test` | Run automated tests |
| `make lint` | Run code linter ([golangci-lint](https://golangci-lint.run/), auto-installed if missing) |
| `make deps` | Download/tidy Go dependencies |
| `make clean` | Remove `bin/` directory |
| `make docker-build` | Build Docker image manually |

### CLI Flags

| Flag | Description |
|------|-------------|
| `papyrus <file> [prompt]` | Analyze a PDF or TXT file and enter interactive REPL |
| `--list` / `--sessions` | List all saved sessions |
| `--session <id>` | Resume a saved session |
| `--delete <id>` | Delete a saved session |
| `--export` | Analyze, export conversation to Markdown, and exit |
| `--no-cache` | Disable local response caching |
| `--max-context N` | Maximum tokens in conversation history before pruning (default: `8192`) |
| `--tts` | Enable text-to-speech for responses (Piper + SSML) |
| `--rag` | Enable RAG support (requires ChromaDB) |
| `--vectordb-url URL` | ChromaDB URL (default: `http://localhost:8000`) |
| `--embed-model MODEL` | Ollama model for embeddings (default: `nomic-embed-text`) |
| `--top-k N` | Number of chunks to retrieve per RAG query (default: `5`) |
| `--chunk-size N` | Tokens per chunk for RAG ingestion (default: `512`) |

## Architecture

```
papyrus/
├── cmd/papyrus/             # CLI entry point and flag parsing
│   └── main.go
├── internal/config/         # Default configuration constants
│   └── config.go
├── pkg/
│   ├── chunker/             # Token-aware text chunking (tiktoken BPE)
│   │   └── chunker.go
│   ├── conversation/        # Multi-turn conversation, session persistence, export
│   │   ├── conversation.go       # Conversation struct and session ID generation
│   │   ├── context_manager.go    # History pruning to stay under token limits
│   │   ├── sessions.go           # Save/load/list/delete sessions as JSON
│   │   └── exporter.go           # Export conversation to Markdown
│   ├── embeddings/          # Text embedding via Ollama /api/embed
│   │   ├── embeddings.go         # Embedder interface
│   │   └── ollama_embedder.go    # Ollama implementation (batch support)
│   ├── llm/                 # Ollama LLM client with streaming and caching
│   │   ├── client.go             # Chat API client, streaming, RAG integration
│   │   ├── cache.go              # SHA-256 keyed response cache (24h TTL)
│   │   └── tokenizer.go          # BPE token counting and stats formatting
│   ├── pdf/                 # PDF text extraction
│   │   └── extract.go            # Page-by-page text extraction
│   ├── repl/                # Interactive read-eval-print loop
│   │   └── repl.go               # REPL commands, TTS integration, auto-save
│   ├── tts/                 # Text-to-speech pipeline
│   │   ├── tts.go                # PiperClient, synthesis orchestration
│   │   ├── ssml_parser.go        # SSML tag parsing (speak, break, voice, prosody, s)
│   │   ├── markdown_to_ssml.go   # Markdown → SSML conversion with prosody config
│   │   └── audio_mixer.go        # WAV format handling, PCM concatenation, silence gen
│   └── vectordb/            # Vector database abstraction
│       ├── vectordb.go           # Retriever interface
│       └── chroma.go             # ChromaDB v2 API implementation
├── docker-compose.yml       # Service definitions (Ollama, Piper, ChromaDB, Papyrus)
├── Dockerfile               # Multi-stage Go build (golang:1.24-alpine → alpine:3.19)
├── ollama.Dockerfile        # Custom Ollama image (auto-pulls model on start)
├── papyrus.sh               # Container entrypoint (waits for Ollama, runs CLI)
├── ollama-entrypoint.sh     # Ollama container entrypoint (serve + pull model)
├── Makefile                 # Build, run, and infrastructure targets
└── .golangci.yml            # Linter config (errcheck, govet, staticcheck, gosec, etc.)
```

### Docker Compose Services

| Service | Image | Profile | Description |
|---------|-------|---------|-------------|
| `papyrus` | Built from `Dockerfile` | `cli` | The Papyrus CLI application |
| `ollama` | Built from `ollama.Dockerfile` | *(always)* | Ollama LLM server (auto-pulls configured model) |
| `piper` | `artibex/piper-http` | *(always)* | Piper neural TTS server |
| `chromadb` | `chromadb/chroma:latest` | `rag`, `cli` | ChromaDB vector database (for RAG) |
| `voice-downloader` | `alpine:latest` | `cli` | Downloads Piper voice model files |

### Key Design Decisions

- **Document context is sent in the system prompt**, not re-embedded with each user message. This keeps follow-up queries lightweight.
- **Reasoning model support**: Models like `qwen3` that emit `reasoning_content` in their streaming response are handled transparently — reasoning is wrapped in `<think>` blocks and stripped before TTS synthesis.
- **SSML is generated client-side** by converting markdown to SSML in Papyrus, not by instructing the LLM to output SSML. This keeps LLM output clean and predictable.
- **Token counting** uses BPE encoding (`cl100k_base` via tiktoken-go) for accurate estimation, with a word-count heuristic fallback.
- **Response caching** uses SHA-256 hashed keys with a 24-hour TTL, persisted per-session as JSON files.

## Troubleshooting

### "Connection refused" or "Cannot connect to Ollama"

**Problem:** Can't connect to Ollama service.

**Solutions:**
- **If using Docker mode:** Ensure Ollama container is running: `docker-compose logs ollama`
- **If using local binary mode:** Ensure Ollama is running on your machine:
  ```bash
  ollama serve  # Start Ollama if not running
  ```
  Then verify it's accessible: `curl http://localhost:11434/api/tags`

### "Model not found" error

**Problem:** The specified model is not installed in Ollama.

**Solution:**
- Pull the model manually:
  ```bash
  # If using Docker
  docker-compose exec ollama ollama pull qwen3:8b
  
  # If running Ollama locally  
  ollama pull qwen3:8b
  ```
- Or change to an available model in `.env` or via command line

### PDF parsing fails or "scanned image PDF" error

**Problem:** "Could not extract any text from the PDF (scanned image PDF?)"

**Reason:** PDF parser only handles text-based PDFs, not scanned images.

**Workaround:**
1. Use OCR tools to convert scanned PDFs first: `tesseract scanned.pdf output.txt`
2. Feed the resulting `.txt` file to Papyrus: `papyrus output.txt`
3. Ensure PDFs are not encrypted/password-protected
4. Try a different PDF

### Docker memory errors ("Cannot allocate memory")

**Problem:** Out of memory errors when processing large PDFs or using large models.

**Solutions:**
- Increase Docker memory allocation:
  - **Docker Desktop:** Settings → Resources → Memory (set to 16GB+ for large models)
- Use a smaller model: `OLLAMA_MODEL=mistral:7b` instead of larger models
- Reduce PDF file size or process files sequentially

### Slow performance

**Problem:** PDF analysis takes too long.

**Causes & Solutions:**
- **Large PDF:** Use a smaller model or break into smaller files. Enable `--rag` for large documents.
- **Large model (13B+):** Switch to 7B parameter models: `mistral:7b`, `llama2:7b`
- **Slow disk:** Ensure project is on SSD, not network/USB drive
- **Low memory:** Increase Docker/system memory allocation
- **Other processes:** Check system resource usage

### "make: command not found"

**Problem:** GNU Make is not installed.

**Solution:**
- **Linux:** `sudo apt-get install make`
- **macOS:** `xcode-select --install` (Xcode Command Line Tools)
- **Windows:** Install [Windows Subsystem for Linux (WSL2)](https://docs.microsoft.com/en-us/windows/wsl/) or [Git Bash](https://git-scm.com/download/win)

### PDF_FILE path issues in Docker

**Problem:** "File not found" or "Permission denied" errors.

**Details:**
- PDFs must be in the `./pdfs/` directory (it's mounted as `/pdfs` in containers)
- Use relative paths: `make run-cli PDF_FILE=pdfs/myfile.pdf` (not absolute paths)
- Files should be readable: `chmod 644 pdfs/yourfile.pdf`

### RAG ingestion errors

**Problem:** Vector DB errors when using `--rag`.

**Solutions:**
- Ensure ChromaDB is running: `docker-compose --profile rag up -d chromadb`
- Verify ChromaDB is accessible: `curl http://localhost:8000/api/v2/heartbeat`
- Ensure the embedding model is pulled: `ollama pull nomic-embed-text`
- Check that `VECTORDB_URL` and `EMBED_MODEL` are correctly configured

---

## Development

For more information on local development, testing, and contributing, see the project structure and Makefile targets above.

The linter configuration (`.golangci.yml`) enables: `errcheck`, `govet`, `staticcheck`, `unused`, `bodyclose`, `gosec`, and `errorlint`.

## License

See LICENSE file for details.
