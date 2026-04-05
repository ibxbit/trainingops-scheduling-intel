#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOG_DIR="$ROOT_DIR/.logs"

stop_pid() {
  local name="$1"
  local pid_file="$2"
  if [ -f "$pid_file" ]; then
    local pid
    pid="$(cat "$pid_file")"
    if kill "$pid" >/dev/null 2>&1; then
      echo "[local] stopped $name (pid $pid)"
    fi
    rm -f "$pid_file"
  fi
}

stop_pid "backend" "$LOG_DIR/backend.pid"
stop_pid "frontend" "$LOG_DIR/frontend.pid"

echo "[local] done"
