#!/bin/sh
# Starts Ollama and pulls the default model if not already present.
# Runs as the Ollama container entrypoint.

set -e

MODEL="${OLLAMA_MODEL:-llama3.2}"

echo "[ollama-init] starting ollama server..."
ollama serve &
SERVER_PID=$!

# Wait for the API to be ready
echo "[ollama-init] waiting for ollama to be ready..."
until ollama ls > /dev/null 2>&1; do
  sleep 1
done

# Pull model only if not already present
if ollama list | grep -q "^${MODEL}"; then
  echo "[ollama-init] model ${MODEL} already present, skipping pull"
else
  echo "[ollama-init] pulling model ${MODEL}..."
  ollama pull "${MODEL}"
  echo "[ollama-init] model ${MODEL} ready"
fi

echo "[ollama-init] ollama is ready"
wait $SERVER_PID
