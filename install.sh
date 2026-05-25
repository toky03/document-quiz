#!/usr/bin/env bash
# One-shot setup: validates environment, installs dependencies,
# pre-pulls the Chroma image, and builds the backend binary so
# the first run of start-all.sh is fast.

set -euo pipefail

# Source nvm if present (non-interactive shells don't inherit it via rc files).
for nvm_sh in "$HOME/.nvm/nvm.sh" "/opt/homebrew/opt/nvm/nvm.sh" "/usr/local/opt/nvm/nvm.sh"; do
  if [[ -s "$nvm_sh" ]]; then
    export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
    # shellcheck disable=SC1090
    . "$nvm_sh" >/dev/null 2>&1 || true
    break
  fi
done

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$ROOT_DIR/backend"
FRONTEND_DIR="$ROOT_DIR/frontend"

echo "==> Umgebung prüfen"
if ! "$ROOT_DIR/doctor.sh"; then
  echo "Bitte zuerst die obigen Fehler beheben."
  exit 1
fi

echo
echo "==> Backend-Abhängigkeiten laden"
( cd "$BACKEND_DIR" && go mod download )

echo
echo "==> Frontend-Abhängigkeiten installieren (npm ci)"
( cd "$FRONTEND_DIR" && npm ci )

echo
echo "==> Chroma-Image vorab laden"
docker pull chromadb/chroma:latest

echo
echo "==> Backend-Binary bauen"
( cd "$BACKEND_DIR" && go build -o document-quiz-backend . )

echo
echo "==> start-all.sh ausführbar machen"
chmod +x "$ROOT_DIR/start-all.sh"

echo
echo "Fertig. Starten mit: ./start-all.sh"
