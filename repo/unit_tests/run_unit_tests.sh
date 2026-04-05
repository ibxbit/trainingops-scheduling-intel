#!/usr/bin/env bash
set -euo pipefail

echo "[unit] running Go unit tests"
docker compose exec -T unit_runner sh -c "cd /app && go test ./internal/auth ./internal/security ./internal/booking ./internal/calendar ./internal/content ./internal/dashboard ./internal/planning ./internal/observability"

echo "[unit] checking offline-only API usage in frontend"
if docker compose exec -T tester sh -lc "grep -R \"https://\|http://\" /workspace/frontend/src/api | grep -v '/api/v1'"; then
  echo "[unit] unexpected external API URL found"
  exit 1
fi

echo "[unit] running frontend unit/component tests"
docker compose exec -T frontend sh -c "npm test -- --run"

echo "[unit] passed"
