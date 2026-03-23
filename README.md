# Papyrus

Papyrus is a tool to analyze and explain PDF documents using Ollama.

## Usage

### Prerequisites
- `docker-compose` v1.29.0 installed.

### Configuration
Manage your settings in the `.env` file:
```
OLLAMA_MODEL=qwen3:8b
PDF_FILE=pdfs/test.pdf
CUSTOM_PROMPT="Summarize the key findings."
```

### Running with Makefile
- **Build and start stack (Ollama + Papyrus):** `make up`
- **Run on a specific PDF:** `make run-pdf PDF_FILE=pdfs/myfile.pdf`
- **Build binary locally:** `make build`
