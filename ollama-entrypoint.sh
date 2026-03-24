#!/bin/sh

export OLLAMA_INSECURE=1

# Start ollama in the background
OLLAMA_INSECURE=1 ollama serve &

# Wait for ollama to be ready
echo "Waiting for ollama to be ready..."
sleep 5

echo "Pulling model: $OLLAMA_MODEL"
OLLAMA_INSECURE=1 ollama pull $OLLAMA_MODEL

# Wait for ollama process to exit
wait $!
