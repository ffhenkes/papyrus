# Papyrus

A portable Go agent that extracts text from a PDF and explains its contents using a **local Ollama LLM**. No API keys, no cloud, no data leaves your machine.

## Prerequisites

| Requirement | Notes |
|---|---|
| [Docker Desktop](https://www.docker.com/products/docker-desktop) | With WSL2 on Windows |
| [Ollama](https://ollama.com/download) | Running on the host machine |
| A pulled model | e.g. `ollama pull qwen3:8b` |

## Minimum Hardware Requirements

### qwen3:8b (default, recommended)
| Component | Minimum | Recommended |
|---|---|---|
| RAM | 16 GB | 16 GB+ |
| VRAM (Nvidia) | 6 GB | 8 GB+ |
| Disk space | 10 GB free | 20 GB free |
| GPU Driver | CUDA 11.8+ | CUDA 12.x |

### deepseek-r1:14b (better quality)
| Component | Minimum |
|---|---|
| RAM | 16 GB |
| VRAM (Nvidia) | 10 GB |
| Disk space | 20 GB free |

## Quick Start

```bash
# 1. Pull a model (one time)
ollama pull qwen3:8b

# 2. Build the Docker image
docker build -t papyrus .

# 3. Explain a PDF
docker run --rm \
  -v "$(pwd):/pdfs:ro" \
  papyrus /pdfs/your-document.pdf
```

## Usage

### Basic
```bash
docker run --rm \
  -v "/path/to/dir:/pdfs:ro" \
  papyrus /pdfs/document.pdf
```

### With a custom prompt
```bash
docker run --rm \
  -v "/path/to/dir:/pdfs:ro" \
  papyrus /pdfs/document.pdf "List all technical decisions and their rationale"
```

### Using a different model
```bash
docker run --rm \
  -e OLLAMA_MODEL=deepseek-r1:14b \
  -v "/path/to/dir:/pdfs:ro" \
  papyrus /pdfs/document.pdf
```

### Using the wrapper script (easiest, Linux/macOS/Git Bash)
```bash
chmod +x papyrus.sh
./papyrus.sh document.pdf
./papyrus.sh document.pdf "Focus on financial highlights"
OLLAMA_MODEL=deepseek-r1:14b ./papyrus.sh document.pdf
```

### Using Docker Compose
```bash
mkdir -p pdfs && cp your-document.pdf pdfs/
docker compose run papyrus /pdfs/your-document.pdf
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `OLLAMA_URL` | `http://host.docker.internal:11434` | Ollama host URL |
| `OLLAMA_MODEL` | `qwen3:8b` | Model to use |

> `host.docker.internal` is the magic hostname Docker uses to reach the host machine from inside a container. It works on Windows, macOS, and Linux (Docker Desktop).

## Windows Setup (from scratch)

1. **Install Nvidia driver** — [nvidia.com/drivers](https://www.nvidia.com/drivers). Run `nvidia-smi` in PowerShell to verify.
2. **Install Ollama** — [ollama.com/download](https://ollama.com/download). Run `ollama pull qwen3:8b`.
3. **Install Docker Desktop** — [docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop). Accept WSL2 install. Reboot.
4. **Install Git** — [git-scm.com](https://git-scm.com/download/win). Choose Git Bash as default terminal.
5. Open **Git Bash**, clone this repo, build and run.

## How It Works

1. `pdfcpu` extracts text from the PDF page by page (pure Go, no system dependencies)
2. The text is sent to Ollama's local API at `http://host.docker.internal:11434`
3. The model explains the content and prints the result to stdout

## Notes

- Scanned PDFs (image-only, no embedded text) are not supported — only text-based PDFs work
- The Docker image is ~25MB
- No internet access required after the model is pulled
