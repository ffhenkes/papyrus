#!/usr/bin/env bash
# papyrus.sh — convenience wrapper around the Docker container
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path-to-pdf> [custom prompt]"
  echo ""
  echo "Examples:"
  echo "  $0 report.pdf"
  echo "  $0 report.pdf 'Summarize only the financial data'"
  echo ""
  echo "Environment variables (optional):"
  echo "  OLLAMA_URL    default: http://host.docker.internal:11434"
  echo "  OLLAMA_MODEL  default: qwen3:8b"
  exit 1
fi

PDF_PATH="$(realpath "$1")"
PDF_DIR="$(dirname "$PDF_PATH")"
PDF_FILE="$(basename "$PDF_PATH")"
shift

docker run --rm \
  -e OLLAMA_URL="${OLLAMA_URL:-http://host.docker.internal:11434}" \
  -e OLLAMA_MODEL="${OLLAMA_MODEL:-qwen3:8b}" \
  -v "${PDF_DIR}:/pdfs:ro" \
  papyrus \
  "/pdfs/${PDF_FILE}" "$@"
