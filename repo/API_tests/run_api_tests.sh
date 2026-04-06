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
  local out="${5:-$TMP_DIR/body.json}"
  local headers="${6:-$TMP_DIR/headers.txt}"
  local code

  if [ -n "$cookie_file" ]; then
    if [ -n "$data" ]; then
      code=$(curl -sS -D "$headers" -o "$out" -w "%{http_code}" -X "$method" "$url" -H "Content-Type: application/json" -d "$data" -b "$cookie_file" -c "$cookie_file")
    else
      code=$(curl -sS -D "$headers" -o "$out" -w "%{http_code}" -X "$method" "$url" -b "$cookie_file" -c "$cookie_file")
    fi
  else
    if [ -n "$data" ]; then
      code=$(curl -sS -D "$headers" -o "$out" -w "%{http_code}" -X "$method" "$url" -H "Content-Type: application/json" -d "$data")
    else
      code=$(curl -sS -D "$headers" -o "$out" -w "%{http_code}" -X "$method" "$url")
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
  sed -n 's/.*"booking_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$1"
}

current_session_cookie() {
  local file="$1"
  awk '($0 !~ /^#/ && $6 == "trainingops_session") {print $7}' "$file"
}

LEARNER_COOKIE="$TMP_DIR/learner.cookie"
COORD_COOKIE="$TMP_DIR/coord.cookie"
INSTR_COOKIE="$TMP_DIR/instructor.cookie"
ADMIN_COOKIE="$TMP_DIR/admin.cookie"
BETA_COOKIE="$TMP_DIR/beta.cookie"

echo "[api] unauthenticated admin endpoint denied"
code=$(request "GET" "$BASE/admin/tenants")
assert_code "$code" "401" "unauthenticated admin endpoint"

echo "[api] login learner1"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"learner1","password":"LearnerPass12"}' "$LEARNER_COOKIE")
assert_code "$code" "200" "learner login"

echo "[api] login coordinator"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"coordinator","password":"CoordPass1234"}' "$COORD_COOKIE")
assert_code "$code" "200" "coordinator login"

echo "[api] login instructor"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"instructor","password":"InstrPass1234"}' "$INSTR_COOKIE")
assert_code "$code" "200" "instructor login"

echo "[api] login admin"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"admin","password":"AdminPass1234"}' "$ADMIN_COOKIE")
assert_code "$code" "200" "admin login"

echo "[api] me endpoint"
code=$(request "GET" "$BASE/auth/me" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "auth/me with learner"

echo "[api] role guard learner cannot refresh dashboard"
code=$(request "POST" "$BASE/dashboard/refresh" '{}' "$LEARNER_COOKIE")
assert_code "$code" "403" "learner dashboard refresh forbidden"

echo "[api] admin endpoints deny non-admin"
code=$(request "GET" "$BASE/admin/tenants" "" "$LEARNER_COOKIE")
assert_code "$code" "403" "learner denied admin tenant settings"

echo "[api] admin tenant settings list"
code=$(request "GET" "$BASE/admin/tenants" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "admin tenant settings list"

echo "[api] admin tenant settings update"
code=$(request "PUT" "$BASE/admin/tenants/11111111-1111-1111-1111-111111111111" '{"tenant_slug":"acme-training","tenant_name":"Acme Training Updated","allow_self_registration":false,"require_mfa":true,"max_active_bookings_per_learner":4}' "$ADMIN_COOKIE")
assert_code "$code" "200" "admin tenant settings update"

echo "[api] tenant isolation on admin tenant path"
code=$(request "PUT" "$BASE/admin/tenants/22222222-2222-2222-2222-222222222222" '{"tenant_slug":"beta-training","tenant_name":"Should Fail","allow_self_registration":false,"require_mfa":false,"max_active_bookings_per_learner":3}' "$ADMIN_COOKIE")
assert_code "$code" "403" "cross-tenant admin update denied"

echo "[api] role-permission matrix view and update"
code=$(request "GET" "$BASE/admin/permissions/matrix" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "matrix view"
code=$(request "PUT" "$BASE/admin/permissions/matrix" '{"assignments":[{"role":"program_coordinator","permission":"rbac.matrix.view","allowed":true}]}' "$ADMIN_COOKIE")
assert_code "$code" "200" "matrix update"

echo "[api] role assignment management"
code=$(request "GET" "$BASE/admin/users/roles" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "list user roles"
code=$(request "POST" "$BASE/admin/users/11111111-1111-1111-1111-111111111104/roles" '{"role":"instructor"}' "$ADMIN_COOKIE")
assert_code "$code" "200" "assign role"
code=$(request "DELETE" "$BASE/admin/users/11111111-1111-1111-1111-111111111104/roles/instructor" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "revoke role"

echo "[api] auth lockout after 5 failed attempts"
for _ in 1 2 3 4 5; do
  code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"learner2","password":"WrongPass123"}')
  assert_code "$code" "401" "failed login attempt"
done
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"learner2","password":"LearnerPass12"}')
assert_code "$code" "423" "account lockout"

echo "[api] session rotation check"
cookie_before=$(current_session_cookie "$ADMIN_COOKIE")
sleep 2
code=$(request "GET" "$BASE/auth/me" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "admin me"
cookie_after=$(current_session_cookie "$ADMIN_COOKIE")
if [ -z "$cookie_before" ] || [ -z "$cookie_after" ] || [ "$cookie_before" = "$cookie_after" ]; then
  echo "[api] FAIL: expected session token rotation"
  exit 1
fi

echo "[api] session invalidation check"
code=$(request "POST" "$BASE/auth/logout" '{}' "$ADMIN_COOKIE")
assert_code "$code" "200" "logout"
code=$(request "GET" "$BASE/auth/me" "" "$ADMIN_COOKIE")
assert_code "$code" "401" "invalidated session denied"

echo "[api] today sessions endpoint"
code=$(request "GET" "$BASE/dashboard/today-sessions" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "dashboard today sessions"

echo "[api] availability check"
code=$(request "GET" "$BASE/calendar/availability/11111111-1111-1111-1111-111111115001" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "availability endpoint"

echo "[api] capacity conflict"
code=$(request "POST" "$BASE/bookings/hold" '{"session_id":"11111111-1111-1111-1111-111111115002"}' "$LEARNER_COOKIE")
assert_code "$code" "409" "capacity reached conflict"

echo "[api] booking concurrency: no oversell on simultaneous holds"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"admin","password":"AdminPass1234"}' "$ADMIN_COOKIE")
assert_code "$code" "200" "admin relogin for concurrency"

HOLD_OUT_1="$TMP_DIR/hold_1.json"
HOLD_OUT_2="$TMP_DIR/hold_2.json"
HOLD_OUT_3="$TMP_DIR/hold_3.json"

curl -sS -o "$HOLD_OUT_1" -w "%{http_code}" -X POST "$BASE/bookings/hold" -H "Content-Type: application/json" -d '{"session_id":"11111111-1111-1111-1111-111111115001"}' -b "$LEARNER_COOKIE" -c "$LEARNER_COOKIE" >"$TMP_DIR/hold_1.code" &
pid1=$!
curl -sS -o "$HOLD_OUT_2" -w "%{http_code}" -X POST "$BASE/bookings/hold" -H "Content-Type: application/json" -d '{"session_id":"11111111-1111-1111-1111-111111115001"}' -b "$COORD_COOKIE" -c "$COORD_COOKIE" >"$TMP_DIR/hold_2.code" &
pid2=$!
curl -sS -o "$HOLD_OUT_3" -w "%{http_code}" -X POST "$BASE/bookings/hold" -H "Content-Type: application/json" -d '{"session_id":"11111111-1111-1111-1111-111111115001"}' -b "$ADMIN_COOKIE" -c "$ADMIN_COOKIE" >"$TMP_DIR/hold_3.code" &
pid3=$!

wait "$pid1" "$pid2" "$pid3"

code1=$(cat "$TMP_DIR/hold_1.code")
code2=$(cat "$TMP_DIR/hold_2.code")
code3=$(cat "$TMP_DIR/hold_3.code")

ok_count=0
conflict_count=0
for code in "$code1" "$code2" "$code3"; do
  if [ "$code" = "201" ]; then
    ok_count=$((ok_count + 1))
  fi
  if [ "$code" = "409" ]; then
    conflict_count=$((conflict_count + 1))
  fi
done
if [ "$ok_count" -ne 2 ] || [ "$conflict_count" -ne 1 ]; then
  echo "[api] FAIL: expected exactly 2 hold successes and 1 overflow conflict; got [$code1, $code2, $code3]"
  exit 1
fi

BOOKING_ID=""
for file in "$HOLD_OUT_1" "$HOLD_OUT_2" "$HOLD_OUT_3"; do
  id=$(extract_booking_id "$file")
  if [ -n "$id" ]; then
    BOOKING_ID="$id"
    break
  fi
done
if [ -z "$BOOKING_ID" ]; then
  echo "[api] FAIL: booking id not found in concurrency hold responses"
  exit 1
fi

echo "[api] booking concurrency: no double confirm"
curl -sS -o "$TMP_DIR/confirm_1.json" -w "%{http_code}" -X POST "$BASE/bookings/$BOOKING_ID/confirm" -H "Content-Type: application/json" -d '{}' -b "$LEARNER_COOKIE" -c "$LEARNER_COOKIE" >"$TMP_DIR/confirm_1.code" &
cpid1=$!
curl -sS -o "$TMP_DIR/confirm_2.json" -w "%{http_code}" -X POST "$BASE/bookings/$BOOKING_ID/confirm" -H "Content-Type: application/json" -d '{}' -b "$COORD_COOKIE" -c "$COORD_COOKIE" >"$TMP_DIR/confirm_2.code" &
cpid2=$!
wait "$cpid1" "$cpid2"

confirm_code_1=$(cat "$TMP_DIR/confirm_1.code")
confirm_code_2=$(cat "$TMP_DIR/confirm_2.code")
if [ "$confirm_code_1" = "$confirm_code_2" ]; then
  echo "[api] FAIL: expected one confirm to win and one to fail, got [$confirm_code_1, $confirm_code_2]"
  exit 1
fi

echo "[api] instructor cannot hold booking"
code=$(request "POST" "$BASE/bookings/hold" '{"session_id":"11111111-1111-1111-1111-111111115001"}' "$INSTR_COOKIE")
assert_code "$code" "403" "instructor hold forbidden"

echo "[api] cancellation cutoff enforced"
code=$(request "POST" "$BASE/bookings/11111111-1111-1111-1111-111111116002/cancel" '{}' "$LEARNER_COOKIE")
assert_code "$code" "409" "cancellation cutoff"

echo "[api] tenant isolation"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"beta-training","username":"learnerx","password":"LearnerPass12"}' "$BETA_COOKIE")
assert_code "$code" "200" "beta learner login"
code=$(request "POST" "$BASE/bookings/$BOOKING_ID/confirm" '{}' "$BETA_COOKIE")
assert_code "$code" "404" "cross-tenant booking access denied"

echo "[api] passed"
