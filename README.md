# Papyrus

**Papyrus** is a tool to analyze and explain PDF documents using [Ollama](https://ollama.ai/) — a local LLM runtime. Extract insights, summaries, and answers from your PDFs using customizable prompts and local language models.

## Table of Contents

- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [Session Persistence](#session-persistence)
- [Configuration](#configuration)
- [Text-to-Speech (TTS)](#text-to-speech-tts)
- [Available Commands](#available-commands)
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

# 3. Analyze a PDF using the containerized CLI (requires the stack from step 2)
make run-cli PDF_FILE=pdfs/test.pdf ARGS="--tts"
```

This automatically pulls the latest models and runs the analysis. To start the full stack and keep it running:

```bash
# Start Ollama + Papyrus stack in background
make up

# In another terminal, analyze PDFs using the containerized CLI
make run-cli PDF_FILE=pdfs/myfile.pdf

# Stop when done
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
- Models stored in `./ollama_data/` (persistent)
- No local dependencies needed

**Mode 2: Local Binary** — Run natively on your machine
- Use: `make run ARGS="pdfs/test.pdf --tts"`
- Requires: Ollama running locally on `http://localhost:11434`
- Useful for: development, integration with local tools
- Note: Must start Ollama separately (not included)

### Docker-based Analysis (Recommended)

```bash
# One-shot analysis (starts everything, runs, then exits)
make run-cli PDF_FILE=pdfs/myfile.pdf CUSTOM_PROMPT="List the main topics"

# Or use persistent stack (Ollama + Piper)
make up                              # Start background services
make run-cli PDF_FILE=pdfs/file1.pdf ARGS="--tts" # Run CLI with speech
make run-cli PDF_FILE=pdfs/file2.pdf # Run CLI without speech
make down                            # Clean up
```

### Local Binary Analysis

```bash
# Requires: Ollama running on localhost:11434
make build
make run ARGS="pdfs/test.pdf"
```


## Session Persistence

Papyrus automatically saves your conversation to disk so you can **resume analysis later** without re-processing the PDF.

Sessions are stored in `~/.papyrus/sessions/` as JSON files.

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
papyrus --max-context 4096           # Restrict token history
papyrus --tts                       # Enable text-to-speech (with SSML support)
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
| `stats` | Show precise token usage statistics |
| `export` | Export the current session to a Markdown file |
| `save` | Explicitly save the session to disk |
| `session info` | Show session metadata (ID, file, timestamps, message count) |
| `exit` / `quit` | Save and exit |

## Text-to-Speech (TTS)

Papyrus can generate speech (audio) for model responses using [Piper](https://github.com/rhasspy/piper) with SSML support implemented in Papyrus.

### How it works

1.  **Neural Voice Quality:** Papyrus uses Piper for high-quality, fast, local neural TTS synthesis.
2.  **SSML Support:** Papyrus parses SSML markup, breaking it into segments and synthesizing each individually with Piper. This enables tags like `<speak>`, `<break time="500ms"/>`, and `<voice>` for precise control over speech pacing, pauses, and voice switching—without requiring SSML support in Piper itself.
3.  **Activation:** Use the `--tts` flag to enable text-to-speech.
4.  **Output:** Audio files are saved as `.wav` files in the `voice/` directory.
    - Initial explanation: `voice/<session-id>_initial.wav`
    - REPL responses: `voice/<session-id>_<message-index>.wav`

### Configuration for TTS

| Variable | Default | Description |
|----------|---|-------------|
| `PIPER_URL` | `http://localhost:5000` | Piper HTTP endpoint |
| `PIPER_VOICE` | `pt_BR-faber-medium` | Piper Voice (e.g., `pt_BR-faber-medium`, `en_US-lessac-medium`) |

**Example:**
```bash
# Analyze with TTS (SSML support enabled in Papyrus)
make run-cli PDF_FILE=pdfs/report.pdf ARGS="--tts"
```

### Changing the TTS Voice
You can change the language or voice used by Piper by setting the `PIPER_VOICE_URL` in your `.env` file. You can find more voices on the [Piper HuggingFace repository](https://huggingface.co/rhasspy/piper-voices).

**Portuguese (Brazil) Example:**
```bash
PIPER_VOICE_URL=https://huggingface.co/rhasspy/piper-voices/resolve/main/pt/pt_BR/faber/medium/pt_BR-faber-medium.onnx?download=true
```

Simply update the variable and run `make up` to restart the Piper service with the new model.

## Configuration

All settings are managed via the `.env` file. Create one from the template:

```bash
cp .env.example .env
```

### Environment Variables

| Variable | Docker Default | Local Default | Description |
|----------|---|---|-------------|
| `OLLAMA_URL` | `http://ollama:11434` | `http://host.docker.internal:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `qwen3:8b` | `qwen3:8b` | LLM model to use (must be installed in Ollama) |
| `PDF_FILE` | `pdfs/test.pdf` | N/A | Path to PDF file to analyze |
| `CUSTOM_PROMPT` | `"Explain this document."` | N/A | Custom prompt for PDF analysis |
| `PIPER_URL` | `http://localhost:5000` | `http://localhost:5000` | Piper TTS API endpoint |

**Important:** 
- **Docker mode** uses `http://ollama:11434` (internal Docker service name)
- **Local binary mode** tries to connect to `http://host.docker.internal:11434` which only works from inside Docker containers
- **To run locally**, start Ollama on your machine and it will listen on `http://localhost:11434`
- The `make run` target automatically sets `OLLAMA_URL=http://localhost:11434` for local execution

### Supported Ollama Models

Common models available in Ollama (install/pull them beforehand):
- `qwen3:8b` — Fast, general-purpose (recommended default)
- `llama2:7b` — General-purpose, well-rounded
- `mistral:7b` — Fast, excellent for summaries
- `neural-chat:7b` — Optimized for conversations
- `gemma:7b` — Lightweight, good performance

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

| Command | Description |
|---------|-------------|
| `make help` | Show all available targets |
| `make up` | Start Ollama + Papyrus Docker stack (`docker-compose up`) |
| `make down` | Stop and remove containers (`docker-compose down`) |
| `make run-cli PDF_FILE=... [CUSTOM_PROMPT=...]` | Run CLI in container |
| `make build` | Build binary locally (default: linux/amd64) |
| `make run ARGS="..."` | Build and run binary locally (requires local Ollama at localhost:11434) |
| `make test` | Run automated tests |
| `make lint` | Run code linter |
| `make deps` | Download/tidy Go dependencies |
| `make clean` | Remove bin/ directory |
| `make docker-build` | Build Docker image manually |

**Binary (session management):**

| Flag | Description |
|------|-------------|
| `papyrus <file.pdf> [prompt]` | Analyze a PDF and enter interactive REPL |
| `papyrus --list` | List all saved sessions |
| `papyrus --session <id>` | Resume a saved session |
| `papyrus --delete <id>` | Delete a saved session |
| `papyrus --export` | Analyze, format as Markdown, and exit |
| `papyrus --no-cache` | Disable local semantic caching |
| `papyrus --max-context N` | Configure conversation history limit |
| `papyrus --tts` | Enable text-to-speech for responses (with SSML) |

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
1. Use OCR tools to convert scanned PDFs first: `tesseract scanned.pdf text.pdf`
2. Ensure PDFs are not encrypted/password-protected
3. Try a different PDF

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
- **Large PDF:** Use a smaller model or break into smaller files
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

---

## Development

For more information on local development, testing, and contributing, see the project structure and Makefile targets above.

## License

See LICENSE file for details.
