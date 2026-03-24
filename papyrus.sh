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

# In Docker, pdfs are mounted at /pdfs. Otherwise use pdfs/ from current directory
if [ -d /pdfs ]; then
  PDF_DEFAULT="/pdfs/test.pdf"
else
  PDF_DEFAULT="pdfs/test.pdf"
fi

PDF_FILE_PATH="${PDF_FILE:-$PDF_DEFAULT}"

# Check if the file exists
if [ ! -f "$PDF_FILE_PATH" ]; then
  echo "--------------------------------------------------------"
  echo "Error: PDF file not found at: $PDF_FILE_PATH"
  echo "Available locations checked: $PDF_DEFAULT"
  echo "--------------------------------------------------------"
  exit 0
fi

# Get the path from .env or default to current directory (.)
BIN_DIR="${PAPYRUS_PATH:-.}"

# Construct the full path to the binary
# This handles both ./bin/papyrus and ./papyrus
FULL_BIN_PATH="${BIN_DIR}/papyrus"

# Run papyrus
"$FULL_BIN_PATH" "$PDF_FILE_PATH" "${CUSTOM_PROMPT}"
