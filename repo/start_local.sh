#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-all}"
ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="$ROOT_DIR/.logs"
BACKEND_PORT="${BACKEND_PORT:-8000}"
FRONTEND_PORT="${FRONTEND_PORT:-3000}"
mkdir -p "$LOG_DIR"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "[local] DATABASE_URL is required"
  exit 1
fi

if [ -z "${ENCRYPTION_KEY:-}" ]; then
  echo "[local] ENCRYPTION_KEY is required (exactly 32 bytes)"
  exit 1
fi

start_api() {
  echo "[local] running migrations"
  (cd "$ROOT_DIR/backend" && go run ./cmd/migrate)

  echo "[local] starting backend api"
  (
    cd "$ROOT_DIR/backend"
    HTTP_ADDR=":${BACKEND_PORT}" nohup go run ./cmd/server >"$LOG_DIR/backend.log" 2>&1 &
    echo $! >"$LOG_DIR/backend.pid"
  )
  echo "[local] backend started (pid $(cat "$LOG_DIR/backend.pid"))"
}

start_frontend() {
  echo "[local] installing frontend dependencies"
  (cd "$ROOT_DIR/frontend" && npm install)

  echo "[local] starting frontend dev server"
  (
    cd "$ROOT_DIR/frontend"
    VITE_API_PROXY_TARGET="${VITE_API_PROXY_TARGET:-http://localhost:${BACKEND_PORT}}" \
      nohup npm run dev -- --host 0.0.0.0 --port "$FRONTEND_PORT" >"$LOG_DIR/frontend.log" 2>&1 &
    echo $! >"$LOG_DIR/frontend.pid"
  )
  echo "[local] frontend started (pid $(cat "$LOG_DIR/frontend.pid"))"
}

case "$MODE" in
  api)
    start_api
    ;;
  frontend)
    start_frontend
    ;;
  all)
    start_api
    start_frontend
    ;;
  *)
    echo "Usage: $0 [all|api|frontend]"
    exit 1
    ;;
esac

echo "[local] backend: http://localhost:${BACKEND_PORT}"
echo "[local] frontend: http://localhost:${FRONTEND_PORT}"
echo "[local] logs: $LOG_DIR/backend.log, $LOG_DIR/frontend.log"
