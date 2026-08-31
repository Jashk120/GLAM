#!/usr/bin/env bash
set -e
# GLAM — one-command dev (idiot-proof)
# Starts: Go server (8080) + Vite student client (5173) + Next teacher app (3000)
# Env overrides:
#   PORT / SERVER_PORT        → Go server (default 8080)
#   VITE_PORT / CLIENT_PORT   → Vite client (default 5173)
#   TEACHER_PORT              → Next teacher (default 3000)
#   VITE_API_BASE_URL / NEXT_PUBLIC_API_BASE_URL → API target (default http://localhost:$SERVER_PORT)
ROOT="$(cd "$(dirname "$0")" && pwd)"
SERVER_PORT="${PORT:-${SERVER_PORT:-8080}}"
CLIENT_PORT="${VITE_PORT:-${CLIENT_PORT:-5173}}"
TEACHER_PORT="${TEACHER_PORT:-3000}"
SERVER_LOG="/tmp/glam-server.log"
CLIENT_LOG="/tmp/glam-client.log"
TEACHER_LOG="/tmp/glam-teacher.log"

cyan()  { printf "\033[36m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow(){ printf "\033[33m%s\033[0m\n" "$*"; }
red()   { printf "\033[31m%s\033[0m\n" "$*"; }
dim()   { printf "\033[2m%s\033[0m\n" "$*"; }

cleanup() {
  yellow "→ shutting down..."
  jobs -p | xargs -r kill 2>/dev/null || true
  pkill -P $$ 2>/dev/null || true
  # also kill anything still on our ports
  for p in "$SERVER_PORT" "$CLIENT_PORT" "$TEACHER_PORT"; do
    lsof -ti :"$p" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
  done
  green "bye — logs kept at $SERVER_LOG $CLIENT_LOG $TEACHER_LOG"
}
trap cleanup EXIT INT TERM

cyan "▓▓ GLAM dev — one command (server + student + teacher) ▓▓"
cyan "root: $ROOT"
dim "  server :$SERVER_PORT  •  student (Vite) :$CLIENT_PORT  •  teacher (Next) :$TEACHER_PORT"

if ! command -v go >/dev/null 2>&1; then red "✗ go not found — install Go 1.22+"; exit 1; fi
if ! command -v node >/dev/null 2>&1; then red "✗ node not found — install Node 18+"; exit 1; fi
if ! command -v npm >/dev/null 2>&1; then red "✗ npm not found"; exit 1; fi

if [ ! -f "$ROOT/server/.env" ] && [ ! -f "$ROOT/.env" ]; then
  yellow "⚠ no .env found — creating from example (you MUST edit it!)"
  cp "$ROOT/server/.env.example" "$ROOT/server/.env"
  yellow "  → $ROOT/server/.env created (placeholder key)"
  yellow "  → set OPENROUTER_API_KEY=sk-or-... then re-run ./dev.sh"
fi

if grep -q "sk-your-key-here\|sk-or-your-key-here" "$ROOT/server/.env" 2>/dev/null; then
  yellow "⚠ server/.env still has placeholder key — /api/scenario/generate will return 500 until you set a real key"
  yellow "  example scenarios + validation + Phaser still work without a key"
fi

if [ ! -d "$ROOT/client/node_modules" ]; then
  cyan "→ installing client deps (client/node_modules missing)..."
  (cd "$ROOT/client" && npm install)
else
  cyan "→ client deps ok"
fi

if [ ! -d "$ROOT/teacher-interface/node_modules" ]; then
  cyan "→ installing teacher deps (teacher-interface/node_modules missing)..."
  (cd "$ROOT/teacher-interface" && npm install)
else
  cyan "→ teacher deps ok"
fi

if ! (cd "$ROOT/server" && go vet ./... >/dev/null 2>&1); then
  yellow "→ go vet warnings — continuing (run 'go vet ./...' in server/ for details)"
fi

cyan "→ freeing ports :$SERVER_PORT :$CLIENT_PORT :$TEACHER_PORT ..."
for p in "$SERVER_PORT" "$CLIENT_PORT" "$TEACHER_PORT"; do
  lsof -ti :"$p" 2>/dev/null | xargs -r kill -9 2>/dev/null || true
done
sleep 0.5

export PATH="$HOME/go/bin:$HOME/.local/bin:$PATH"
AIR_BIN="$(command -v air 2>/dev/null || echo "$HOME/go/bin/air")"
if [ ! -x "$AIR_BIN" ]; then
  yellow "⚠ air not found — installing github.com/air-verse/air@latest ..."
  go install github.com/air-verse/air@latest 2>&1 | tail -n 3 || true
  AIR_BIN="$HOME/go/bin/air"
fi

USE_AIR="0"
if [ -x "$AIR_BIN" ]; then
  USE_AIR="1"
  green "✓ air hot-reload enabled ($AIR_BIN)"
else
  yellow "⚠ air unavailable — falling back to 'go run .' (no hot reload)"
fi

: > "$SERVER_LOG"
: > "$CLIENT_LOG"
: > "$TEACHER_LOG"

cyan "→ starting Go server on :$SERVER_PORT ... ($([ "$USE_AIR" = "1" ] && echo "air hot-reload" || echo "go run"))"
if [ "$USE_AIR" = "1" ]; then
  (cd "$ROOT/server" && GLAM_ROOT="$ROOT" PORT="$SERVER_PORT" "$AIR_BIN" 2>&1 | sed 's/^/[server] /' | tee "$SERVER_LOG") &
else
  (cd "$ROOT/server" && GLAM_ROOT="$ROOT" PORT="$SERVER_PORT" go run . 2>&1 | sed 's/^/[server] /' | tee "$SERVER_LOG") &
fi
SERVER_PID=$!

cyan "→ starting Vite student client on :$CLIENT_PORT ... (Vite HMR)"
# Vite reads VITE_PORT / PORT_CLIENT; also proxy /api → server
(cd "$ROOT/client" && VITE_PORT="$CLIENT_PORT" PORT_CLIENT="$CLIENT_PORT" VITE_API_BASE_URL="http://localhost:$SERVER_PORT" npm run dev -- --port "$CLIENT_PORT" --host 2>&1 | sed 's/^/[client] /' | tee "$CLIENT_LOG") &
CLIENT_PID=$!

cyan "→ starting Next teacher app on :$TEACHER_PORT ... (Next HMR)"
# Next respects PORT env; pass API + client URLs so teacher can link to student
(cd "$ROOT/teacher-interface" && PORT="$TEACHER_PORT" NEXT_PUBLIC_API_BASE_URL="http://localhost:$SERVER_PORT" NEXT_PUBLIC_CLIENT_URL="http://localhost:$CLIENT_PORT" npm run dev 2>&1 | sed 's/^/[teacher] /' | tee "$TEACHER_LOG") &
TEACHER_PID=$!

sleep 2

for i in 1 2 3 4 5 6 7 8 9 10; do
  if curl -s --max-time 1 "http://localhost:$SERVER_PORT/health" 2>/dev/null | grep -q ok; then
    green "✓ server  http://localhost:$SERVER_PORT/health → ok (pid $SERVER_PID)"
    break
  fi
  if [ "$i" -eq 10 ]; then yellow "⚠ server not yet healthy — check $SERVER_LOG (air first build can take ~8s)"; fi
  sleep 1
done

# Give Vite/Next a moment to report ready (don't hard-fail if still compiling)
sleep 3
for i in 1 2 3 4 5; do
  if curl -s --max-time 1 "http://localhost:$CLIENT_PORT" >/dev/null 2>&1; then
    green "✓ student http://localhost:$CLIENT_PORT → up (pid $CLIENT_PID)"
    break
  fi
  if [ "$i" -eq 5 ]; then yellow "⚠ student not yet up — check $CLIENT_LOG (Vite may still be compiling)"; fi
  sleep 1
done
for i in 1 2 3 4 5 6 7 8; do
  if curl -s --max-time 1 "http://localhost:$TEACHER_PORT" >/dev/null 2>&1; then
    green "✓ teacher http://localhost:$TEACHER_PORT → up (pid $TEACHER_PID)"
    break
  fi
  if [ "$i" -eq 8 ]; then yellow "⚠ teacher not yet up — check $TEACHER_LOG (Next first compile can take ~15s)"; fi
  sleep 1
done

green "──────────────────────────────────────────"
green "  GLAM running — all 3 services"
green "  • Teacher (Next)  → http://localhost:$TEACHER_PORT  ← create worlds here"
green "  • Student (Vite)  → http://localhost:$CLIENT_PORT  ← play worlds here"
green "  • Server (Go)     → http://localhost:$SERVER_PORT"
green "  • Health          → http://localhost:$SERVER_PORT/health"
green "  • Assets          → http://localhost:$SERVER_PORT/api/assets"
green "  • Scenarios       → http://localhost:$SERVER_PORT/api/scenarios"
cyan  "  Flow: Teacher types prompt → Generate → copies ?scenario=ID link → Student opens link or picks from dropdown"
dim   "  Override ports: PORT=8081 TEACHER_PORT=3001 VITE_PORT=5174 ./dev.sh"
yellow "  Logs: tail -f $SERVER_LOG $CLIENT_LOG $TEACHER_LOG"
yellow "  Stop: Ctrl+C (kills all 3)"
green "──────────────────────────────────────────"
echo ""
cyan "  Quick test:"
echo "    curl http://localhost:$SERVER_PORT/health"
echo "    curl http://localhost:$SERVER_PORT/api/scenarios | jq ."
echo "    open http://localhost:$TEACHER_PORT  # teacher"
echo "    open http://localhost:$CLIENT_PORT   # student"
echo ""

wait
