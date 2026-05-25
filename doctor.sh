#!/usr/bin/env bash
# Read-only environment check. Prints what's missing or wrong without
# changing anything. Exit code 0 if all required checks pass.

set -uo pipefail

# Source nvm if present, otherwise node/npm won't be on PATH inside this
# non-interactive shell. Tries the standard installer location and the two
# common Homebrew prefixes.
for nvm_sh in "$HOME/.nvm/nvm.sh" "/opt/homebrew/opt/nvm/nvm.sh" "/usr/local/opt/nvm/nvm.sh"; do
  if [[ -s "$nvm_sh" ]]; then
    export NVM_DIR="${NVM_DIR:-$HOME/.nvm}"
    # shellcheck disable=SC1090
    . "$nvm_sh" >/dev/null 2>&1 || true
    break
  fi
done

if [[ -t 1 ]]; then
  RED=$'\033[31m'; YEL=$'\033[33m'; GRN=$'\033[32m'; RST=$'\033[0m'
else
  RED=''; YEL=''; GRN=''; RST=''
fi

errors=0
warnings=0

ok()   { echo "${GRN}OK${RST}   $*"; }
warn() { echo "${YEL}WARN${RST} $*"; warnings=$((warnings + 1)); }
fail() { echo "${RED}FAIL${RST} $*"; errors=$((errors + 1)); }

# --- Node ----------------------------------------------------------------
# Angular 21 accepts Node >= 20.19 or >= 22.12.
if ! command -v node >/dev/null 2>&1; then
  fail "node: nicht gefunden"
else
  v=$(node -v | sed 's/^v//')
  major=${v%%.*}
  rest=${v#*.}
  minor=${rest%%.*}
  if   [[ $major -ge 23 ]]; then ok "node $v"
  elif [[ $major -eq 22 && $minor -ge 12 ]]; then ok "node $v"
  elif [[ $major -eq 20 && $minor -ge 19 ]]; then ok "node $v"
  else fail "node $v (benötigt >= 20.19 oder >= 22.12 für Angular 21)"
  fi
fi

# --- npm -----------------------------------------------------------------
if ! command -v npm >/dev/null 2>&1; then
  fail "npm: nicht gefunden"
else
  ok "npm $(npm -v)"
fi

# --- Go ------------------------------------------------------------------
# go.mod declares 1.24.11.
if ! command -v go >/dev/null 2>&1; then
  fail "go: nicht gefunden"
else
  v=$(go version | awk '{print $3}' | sed 's/^go//')
  major=${v%%.*}
  rest=${v#*.}
  minor=${rest%%.*}
  if [[ $major -gt 1 ]] || { [[ $major -eq 1 ]] && [[ $minor -ge 24 ]]; }; then
    ok "go $v"
  else
    fail "go $v (benötigt >= 1.24, siehe backend/go.mod)"
  fi
fi

# --- Docker --------------------------------------------------------------
if ! command -v docker >/dev/null 2>&1; then
  fail "docker: nicht gefunden (für Chroma erforderlich)"
elif ! docker info >/dev/null 2>&1; then
  fail "docker: installiert, aber Daemon läuft nicht"
else
  ok "docker $(docker --version | awk '{print $3}' | tr -d ',')"
fi

# --- claude CLI (optional) ----------------------------------------------
# Nur nötig wenn der Anbieter 'claude_cli' aktiv ist.
if ! command -v claude >/dev/null 2>&1; then
  warn "claude CLI: nicht gefunden (optional, nur für Anbieter 'claude_cli')"
else
  ok "claude $(claude --version 2>/dev/null | awk '{print $1}')"
fi

echo
if [[ $errors -gt 0 ]]; then
  echo "${RED}${errors} Fehler${RST}, ${warnings} Warnung(en)."
  exit 1
fi
echo "${GRN}Alle Pflicht-Checks bestanden${RST} (${warnings} Warnung(en))."
exit 0
