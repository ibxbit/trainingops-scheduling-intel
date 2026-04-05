#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://api:8000/api/v1}"
TMP_DIR="/tmp/trainingops_api_tests"
mkdir -p "$TMP_DIR"

request() {
  local method="$1"
  local url="$2"
  local data="${3:-}"
  local cookie_file="${4:-}"
  local out="$TMP_DIR/body.json"
  local code

  if [ -n "$cookie_file" ]; then
    if [ -n "$data" ]; then
      code=$(curl -sS -o "$out" -w "%{http_code}" -X "$method" "$url" -H "Content-Type: application/json" -d "$data" -b "$cookie_file" -c "$cookie_file")
    else
      code=$(curl -sS -o "$out" -w "%{http_code}" -X "$method" "$url" -b "$cookie_file" -c "$cookie_file")
    fi
  else
    if [ -n "$data" ]; then
      code=$(curl -sS -o "$out" -w "%{http_code}" -X "$method" "$url" -H "Content-Type: application/json" -d "$data")
    else
      code=$(curl -sS -o "$out" -w "%{http_code}" -X "$method" "$url")
    fi
  fi

  echo "$code"
}

assert_code() {
  local actual="$1"
  local expected="$2"
  local message="$3"
  if [ "$actual" != "$expected" ]; then
    echo "[api] FAIL: $message (expected $expected got $actual)"
    cat "$TMP_DIR/body.json"
    exit 1
  fi
}

extract_booking_id() {
  sed -n 's/.*"booking_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$TMP_DIR/body.json" | head -n 1
}

LEARNER_COOKIE="$TMP_DIR/learner.cookie"
COORD_COOKIE="$TMP_DIR/coord.cookie"
INSTR_COOKIE="$TMP_DIR/instructor.cookie"
BETA_COOKIE="$TMP_DIR/beta.cookie"

echo "[api] login learner1"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"learner1","password":"LearnerPass12"}' "$LEARNER_COOKIE")
assert_code "$code" "200" "learner login"

echo "[api] me endpoint"
code=$(request "GET" "$BASE/auth/me" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "auth/me with learner"

echo "[api] rbac learner cannot refresh dashboard"
code=$(request "POST" "$BASE/dashboard/refresh" '{}' "$LEARNER_COOKIE")
assert_code "$code" "403" "learner dashboard refresh forbidden"

echo "[api] login coordinator"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"coordinator","password":"CoordPass1234"}' "$COORD_COOKIE")
assert_code "$code" "200" "coordinator login"

echo "[api] today sessions endpoint"
code=$(request "GET" "$BASE/dashboard/today-sessions" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "dashboard today sessions"

echo "[api] availability check"
code=$(request "GET" "$BASE/calendar/availability/11111111-1111-1111-1111-111111115001" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "availability endpoint"

echo "[api] capacity conflict"
code=$(request "POST" "$BASE/bookings/hold" '{"session_id":"11111111-1111-1111-1111-111111115002"}' "$LEARNER_COOKIE")
assert_code "$code" "409" "capacity reached conflict"

echo "[api] hold booking success"
code=$(request "POST" "$BASE/bookings/hold" '{"session_id":"11111111-1111-1111-1111-111111115001"}' "$LEARNER_COOKIE")
assert_code "$code" "201" "hold booking"
BOOKING_ID=$(extract_booking_id)
if [ -z "$BOOKING_ID" ]; then
  echo "[api] FAIL: booking id not found in hold response"
  cat "$TMP_DIR/body.json"
  exit 1
fi

echo "[api] coordinator confirms learner booking (tenant-wide object auth)"
code=$(request "POST" "$BASE/bookings/$BOOKING_ID/confirm" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "coordinator can confirm learner booking"

echo "[api] cancellation cutoff enforced"
code=$(request "POST" "$BASE/bookings/11111111-1111-1111-1111-111111116002/cancel" '{}' "$LEARNER_COOKIE")
assert_code "$code" "409" "cancellation cutoff"

echo "[api] login instructor"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"instructor","password":"InstrPass1234"}' "$INSTR_COOKIE")
assert_code "$code" "200" "instructor login"

echo "[api] instructor cannot hold booking"
code=$(request "POST" "$BASE/bookings/hold" '{"session_id":"11111111-1111-1111-1111-111111115001"}' "$INSTR_COOKIE")
assert_code "$code" "403" "instructor hold forbidden"

echo "[api] tenant isolation"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"beta-training","username":"learnerx","password":"LearnerPass12"}' "$BETA_COOKIE")
assert_code "$code" "200" "beta learner login"
code=$(request "POST" "$BASE/bookings/$BOOKING_ID/confirm" '{}' "$BETA_COOKIE")
assert_code "$code" "404" "cross-tenant booking access denied"

echo "[api] passed"
