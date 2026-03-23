#!/bin/sh

# Source .env if exists
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

PDF_FILE_PATH="/${PDF_FILE:-pdfs/test.pdf}"

# Check if the file exists
if [ ! -f "$PDF_FILE_PATH" ]; then
  echo "--------------------------------------------------------"
  echo "Error: PDF file not found at: $PDF_FILE_PATH"
  echo "Please check your .env file and ensure the file is mapped."
  echo "--------------------------------------------------------"
  # Exit 0 to prevent the docker-compose (v1.29.0) KeyError
  exit 0
fi

# Run papyrus
./papyrus "$PDF_FILE_PATH" "${CUSTOM_PROMPT}"
