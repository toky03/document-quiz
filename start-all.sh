#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"
VECTOR_DB_DIR="$ROOT_DIR/vector_db"
CHROMA_URL="http://localhost:8000"
CHROMA_CONTAINER_NAME="document-quiz-chroma"

if [[ ! -d "$BACKEND_DIR" ]]; then
  echo "Fehler: Backend-Verzeichnis nicht gefunden: $BACKEND_DIR"
  exit 1
fi

if [[ ! -d "$FRONTEND_DIR" ]]; then
  echo "Fehler: Frontend-Verzeichnis nicht gefunden: $FRONTEND_DIR"
  exit 1
fi

if [[ ! -d "$VECTOR_DB_DIR" ]]; then
  echo "Vector-DB-Verzeichnis nicht gefunden, erstelle: $VECTOR_DB_DIR"
  mkdir -p "$VECTOR_DB_DIR"
fi

BACKEND_CMD=("go" "run" ".")
if [[ -x "$BACKEND_DIR/document-quiz-backend" ]]; then
  BACKEND_CMD=("$BACKEND_DIR/document-quiz-backend")
fi

cleanup() {
  echo
  echo "Beende alle gestarteten Komponenten ..."
  if [[ -n "${BACKEND_PID:-}" ]] && kill -0 "$BACKEND_PID" 2>/dev/null; then
    kill "$BACKEND_PID" 2>/dev/null || true
  fi
  if [[ -n "${FRONTEND_PID:-}" ]] && kill -0 "$FRONTEND_PID" 2>/dev/null; then
    kill "$FRONTEND_PID" 2>/dev/null || true
  fi
  if [[ "${CHROMA_STARTED_BY_SCRIPT:-false}" == "true" ]]; then
    docker stop "$CHROMA_CONTAINER_NAME" >/dev/null 2>&1 || true
  fi
  wait || true
}

trap cleanup INT TERM EXIT

if ! command -v docker >/dev/null 2>&1; then
  echo "Fehler: Docker wurde nicht gefunden. Chroma (Vector DB) kann nicht gestartet werden."
  exit 1
fi

CHROMA_STARTED_BY_SCRIPT=false
if docker ps --filter "name=^/${CHROMA_CONTAINER_NAME}$" --format '{{.Names}}' | grep -q "^${CHROMA_CONTAINER_NAME}$"; then
  echo "Chroma läuft bereits in Container '${CHROMA_CONTAINER_NAME}'."
else
  if docker ps -a --filter "name=^/${CHROMA_CONTAINER_NAME}$" --format '{{.Names}}' | grep -q "^${CHROMA_CONTAINER_NAME}$"; then
    docker rm "$CHROMA_CONTAINER_NAME" >/dev/null 2>&1 || true
  fi

  echo "Starte Chroma (Vector DB) ..."
  docker run -d --rm \
    --name "$CHROMA_CONTAINER_NAME" \
    -p 8000:8000 \
    -v "$VECTOR_DB_DIR:/chroma/chroma" \
    chromadb/chroma:latest >/dev/null

  CHROMA_STARTED_BY_SCRIPT=true
fi

echo "Starte Backend ..."
(
  cd "$BACKEND_DIR"
  export CHROMA_URL="$CHROMA_URL"
  export SQLITE_DB_PATH="$ROOT_DIR/quiz_data.db"
  exec "${BACKEND_CMD[@]}"
) &
BACKEND_PID=$!

echo "Starte Frontend ..."
(
  cd "$FRONTEND_DIR"
  exec npm run start
) &
FRONTEND_PID=$!

echo "Chroma URL: $CHROMA_URL"
echo "Backend PID: $BACKEND_PID"
echo "Frontend PID: $FRONTEND_PID"
echo "Zum Beenden Strg+C drücken."

wait -n "$BACKEND_PID" "$FRONTEND_PID"
