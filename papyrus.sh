#!/bin/bash

# Source .env if exists
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

# Define the endpoint
OLLAMA_HOST="${OLLAMA_URL:-http://ollama:11434}"

echo "X Waiting for Ollama ($OLLAMA_HOST) to be ready..."

# Loop until Ollama responds with a 200 OK
until curl -s "$OLLAMA_HOST/api/tags" > /dev/null; do
  sleep 2
done

echo "!! Ollama is up! Starting processing..."

# Configuration from environment (with defaults)
PDF_FILE_PATH="${PDF_FILE:-/pdfs/test.pdf}"
PROMPT_CONTENT="${CUSTOM_PROMPT:-"Analyze this document."}"

# If first positional argument exists and is not a flag, use it as PDF path
if [[ -n "$1" && ! "$1" =~ ^- ]]; then
  PDF_FILE_PATH="$1"
  shift
fi

# If second positional argument exists and is not a flag, use it as prompt
if [[ -n "$1" && ! "$1" =~ ^- ]]; then
  PROMPT_CONTENT="$1"
  shift
fi

# Run papyrus
# "$@" now contains only flags (if any were passed before or after the positional args)
# or all arguments if we shifted them properly.
"$FULL_BIN_PATH" "$@" "$PDF_FILE_PATH" "$PROMPT_CONTENT"
