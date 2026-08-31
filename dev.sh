#!/usr/bin/env bash
set -e
# Centralized ports/hosts — env-driven with defaults (mirrors client/src/config/env.ts + client/vite.config.ts + server/main.go PORT)
# Override via env: PORT (Go server, default 8080) and VITE_PORT (Vite client, default 5173)
# VITE_API_BASE_URL can override proxy target (default http://localhost:$PORT)
ROOT="$(cd "$(dirname "$0")" && pwd)"
SERVER_PORT="${PORT:-8080}"
CLIENT_PORT="${VITE_PORT:-5173}"
SERVER_LOG="/tmp/glam-server.log"
CLIENT_LOG="/tmp/glam-client.log"

cyan()  { printf "\033[36m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow(){ printf "\033[33m%s\033[0m\n" "$*"; }
red()   { printf "\033[31m%s\033[0m\n" "$*"; }

cleanup() {
  yellow "→ shutting down..."
  jobs -p | xargs -r kill 2>/dev/null || true
  pkill -P $$ 2>/dev/null || true
  green "bye"
}
trap cleanup EXIT INT TERM

cyan "▓▓ GLAM dev — one command ▓▓"
cyan "root: $ROOT"

if ! command -v go >/dev/null 2>&1; then red "go not found"; exit 1; fi
if ! command -v node >/dev/null 2>&1; then red "node not found"; exit 1; fi
if ! command -v npm >/dev/null 2>&1; then red "npm not found"; exit 1; fi

if [ ! -f "$ROOT/server/.env" ] && [ ! -f "$ROOT/.env" ]; then
  yellow "⚠ no .env found — creating from example (edit it!)"
  cp "$ROOT/server/.env.example" "$ROOT/server/.env"
  yellow "  → $ROOT/server/.env created with placeholder key"
  yellow "  → set OPENCODE_API_KEY=sk-... then re-run ./dev.sh"
fi

if grep -q "sk-your-key-here" "$ROOT/server/.env" 2>/dev/null; then
  yellow "⚠ server/.env still has placeholder key — /api/scenario/generate will return 500 until you set a real key"
  yellow "  example scenarios + validation + Phaser still work without a key"
fi

if [ ! -d "$ROOT/client/node_modules" ]; then
  cyan "→ installing client deps..."
  (cd "$ROOT/client" && npm install)
fi

if ! (cd "$ROOT/server" && go vet ./... >/dev/null 2>&1); then
  yellow "→ go vet warnings — continuing"
fi

lsof -ti :"$SERVER_PORT" | xargs -r kill -9 2>/dev/null || true
lsof -ti :"$CLIENT_PORT" | xargs -r kill -9 2>/dev/null || true

export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"
AIR_BIN="$(command -v air 2>/dev/null || echo "$HOME/go/bin/air")"
if [ ! -x "$AIR_BIN" ]; then
  yellow "⚠ air not found — installing github.com/air-verse/air@latest ..."
  go install github.com/air-verse/air@latest 2>&1 | tail -n 3
  AIR_BIN="$HOME/go/bin/air"
fi

USE_AIR="0"
if [ -x "$AIR_BIN" ]; then
  USE_AIR="1"
  green "✓ air hot-reload enabled ($AIR_BIN)"
else
  yellow "⚠ air unavailable — falling back to go run (no hot reload)"
fi

cyan "→ starting Go server on :$SERVER_PORT ... ($([ "$USE_AIR" = "1" ] && echo "air hot-reload" || echo "go run"))"
if [ "$USE_AIR" = "1" ]; then
  (cd "$ROOT/server" && PORT="$SERVER_PORT" "$AIR_BIN" 2>&1 | sed 's/^/[server] /' | tee "$SERVER_LOG") &
else
  (cd "$ROOT/server" && PORT="$SERVER_PORT" go run . 2>&1 | sed 's/^/[server] /' | tee "$SERVER_LOG") &
fi
SERVER_PID=$!

cyan "→ starting Vite client on :$CLIENT_PORT ... (Vite HMR hot-reload)"
(cd "$ROOT/client" && npm run dev -- --port "$CLIENT_PORT" --host 2>&1 | sed 's/^/[client] /' | tee "$CLIENT_LOG") &
CLIENT_PID=$!

sleep 1

for i in 1 2 3 4 5; do
  if curl -s --max-time 1 "http://localhost:$SERVER_PORT/health" | grep -q ok; then
    green "✓ server http://localhost:$SERVER_PORT/health → ok (pid $SERVER_PID)"
    break
  fi
  if [ "$i" -eq 5 ]; then yellow "⚠ server not yet healthy — check $SERVER_LOG"; fi
  sleep 1
done

green "──────────────────────────────────────────"
green "  GLAM running"
green "  • Client → http://localhost:$CLIENT_PORT"
green "  • Server → http://localhost:$SERVER_PORT"
green "  • Health → http://localhost:$SERVER_PORT/health"
green "  • Assets → http://localhost:$SERVER_PORT/api/assets"
cyan  "  Teacher flow: open client → type prompt → Generate → playable world"
yellow "  Logs: tail -f $SERVER_LOG $CLIENT_LOG"
yellow "  Stop: Ctrl+C (kills both)"
green "──────────────────────────────────────────"

wait
