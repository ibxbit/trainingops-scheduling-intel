#!/usr/bin/env bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "[tests] docker command not found. Use bash ./run_tests_local.sh for Docker-free tests."
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "[tests] docker daemon is not running. Start Docker Desktop or use bash ./run_tests_local.sh."
  exit 1
fi

echo "[tests] starting stack"
docker compose down -v --remove-orphans >/dev/null 2>&1 || true
docker compose up -d --build

echo "[tests] waiting for api health"
for _ in $(seq 1 60); do
  if docker compose ps --status running api >/dev/null 2>&1 && \
    docker compose exec -T api wget -q -O - http://localhost:8000/health >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo "[tests] running unit tests"
bash ./unit_tests/run_unit_tests.sh

echo "[tests] running api tests"
docker compose exec -T tester sh -c "tr -d '\r' < /workspace/API_tests/run_api_tests.sh > /tmp/run_api_tests.sh && bash /tmp/run_api_tests.sh"

echo "[tests] all tests passed"
