#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="$ROOT_DIR/.logs"
BACKEND_PORT="${BACKEND_PORT:-8000}"
TEST_API_PORT="${TEST_API_PORT:-18000}"
mkdir -p "$LOG_DIR"
API_STARTED=0

cleanup() {
  if [ "$API_STARTED" -eq 1 ] && [ -f "$LOG_DIR/local-tests-backend.pid" ]; then
    kill "$(cat "$LOG_DIR/local-tests-backend.pid")" >/dev/null 2>&1 || true
    rm -f "$LOG_DIR/local-tests-backend.pid"
  fi
}
trap cleanup EXIT

if [ -n "${DATABASE_URL:-}" ]; then
  echo "[local-tests] running migrations"
  (cd "$ROOT_DIR/backend" && go run ./cmd/migrate)
fi

echo "[local-tests] backend unit tests"
(cd "$ROOT_DIR/backend" && go test ./...)

echo "[local-tests] frontend unit/component tests"
(cd "$ROOT_DIR/frontend" && npm install && npm test -- --run)

if [ -z "${DATABASE_URL:-}" ] || [ -z "${ENCRYPTION_KEY:-}" ]; then
  echo "[local-tests] DATABASE_URL and ENCRYPTION_KEY are required for API tests"
  exit 1
fi

echo "[local-tests] starting temporary backend for API tests on :${TEST_API_PORT}"
(
  cd "$ROOT_DIR/backend"
  HTTP_ADDR=":${TEST_API_PORT}" SESSION_SECURE_COOKIE=false SESSION_ROTATE_EVERY=1s \
    nohup go run ./cmd/server >"$LOG_DIR/local-tests-backend.log" 2>&1 &
  echo $! >"$LOG_DIR/local-tests-backend.pid"
)
API_STARTED=1
for _ in $(seq 1 40); do
  if curl -sS "http://localhost:${TEST_API_PORT}/health" >/dev/null; then
    break
  fi
  sleep 1
done
if ! curl -sS "http://localhost:${TEST_API_PORT}/health" >/dev/null; then
  echo "[local-tests] failed to start local backend; see $LOG_DIR/local-tests-backend.log"
  exit 1
fi

echo "[local-tests] running API tests"
BASE="http://localhost:${TEST_API_PORT}/api/v1" bash "$ROOT_DIR/API_tests/run_api_tests.sh"

echo "[local-tests] passed"
