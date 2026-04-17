#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://api:8000/api/v1}"
ROOT="${BASE%/api/v1}"
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

assert_code_in() {
  local actual="$1"
  local message="$2"
  shift 2
  for expected in "$@"; do
    if [ "$actual" = "$expected" ]; then
      return 0
    fi
  done
  echo "[api] FAIL: $message (got $actual, expected one of: $*)"
  cat "$TMP_DIR/body.json"
  exit 1
}

assert_body_contains() {
  local file="$1"
  local pattern="$2"
  local message="$3"
  if ! grep -q -- "$pattern" "$file"; then
    echo "[api] FAIL: $message (body missing pattern '$pattern')"
    cat "$file"
    exit 1
  fi
}

extract_string_field() {
  sed -n 's/.*"'"$2"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$1" | head -n1
}

extract_booking_id() {
  sed -n 's/.*"booking_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$1"
}

current_session_cookie() {
  local file="$1"
  awk '$6 == "trainingops_session" {print $7}' "$file"
}

LEARNER_COOKIE="$TMP_DIR/learner.cookie"
COORD_COOKIE="$TMP_DIR/coord.cookie"
INSTR_COOKIE="$TMP_DIR/instructor.cookie"
ADMIN_COOKIE="$TMP_DIR/admin.cookie"
BETA_COOKIE="$TMP_DIR/beta.cookie"

##############################################################################
# SECTION 1: health + unauthenticated checks
##############################################################################

echo "[api] health endpoint (public, no auth)"
code=$(request "GET" "$ROOT/health")
assert_code "$code" "200" "GET /health"
assert_body_contains "$TMP_DIR/body.json" '"status"' "health body has status"
assert_body_contains "$TMP_DIR/body.json" '"ok"' "health body ok"

echo "[api] unauthenticated admin endpoint denied"
code=$(request "GET" "$BASE/admin/tenants")
assert_code "$code" "401" "unauthenticated admin endpoint"

##############################################################################
# SECTION 2: auth/login + me + role guards
##############################################################################

echo "[api] login learner1"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"learner1","password":"LearnerPass12"}' "$LEARNER_COOKIE")
assert_code "$code" "200" "learner login"
assert_body_contains "$TMP_DIR/body.json" '"authenticated"' "learner login body"

echo "[api] login coordinator"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"coordinator","password":"CoordPass1234"}' "$COORD_COOKIE")
assert_code "$code" "200" "coordinator login"

echo "[api] login instructor"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"instructor","password":"InstrPass1234"}' "$INSTR_COOKIE")
assert_code "$code" "200" "instructor login"

echo "[api] login admin"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"admin","password":"AdminPass1234"}' "$ADMIN_COOKIE")
assert_code "$code" "200" "admin login"

echo "[api] me endpoint returns tenant, user, roles"
code=$(request "GET" "$BASE/auth/me" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "auth/me with learner"
assert_body_contains "$TMP_DIR/body.json" '"tenant_id"' "auth/me has tenant_id"
assert_body_contains "$TMP_DIR/body.json" '"user_id"' "auth/me has user_id"
assert_body_contains "$TMP_DIR/body.json" '"roles"' "auth/me has roles"

echo "[api] role guard learner cannot refresh dashboard"
code=$(request "POST" "$BASE/dashboard/refresh" '{}' "$LEARNER_COOKIE")
assert_code "$code" "403" "learner dashboard refresh forbidden"

echo "[api] admin endpoints deny non-admin"
code=$(request "GET" "$BASE/admin/tenants" "" "$LEARNER_COOKIE")
assert_code "$code" "403" "learner denied admin tenant settings"

##############################################################################
# SECTION 3: admin endpoints (tenants, permission matrix, role assignments)
##############################################################################

echo "[api] admin tenant settings list"
code=$(request "GET" "$BASE/admin/tenants" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "GET /admin/tenants"
assert_body_contains "$TMP_DIR/body.json" '"tenant_slug"' "tenant list body"

echo "[api] admin tenant settings create (upsert via CreateTenantSettings)"
code=$(request "POST" "$BASE/admin/tenants" '{"tenant_slug":"acme-training","tenant_name":"Acme Training","allow_self_registration":false,"require_mfa":false,"max_active_bookings_per_learner":3}' "$ADMIN_COOKIE")
assert_code "$code" "201" "POST /admin/tenants"
assert_body_contains "$TMP_DIR/body.json" '"max_active_bookings_per_learner"' "create tenant body"

echo "[api] admin tenant settings update"
code=$(request "PUT" "$BASE/admin/tenants/11111111-1111-1111-1111-111111111111" '{"tenant_slug":"acme-training","tenant_name":"Acme Training Updated","allow_self_registration":false,"require_mfa":true,"max_active_bookings_per_learner":4}' "$ADMIN_COOKIE")
assert_code "$code" "200" "PUT /admin/tenants/:tenant_id"
assert_body_contains "$TMP_DIR/body.json" '"require_mfa"' "update tenant body"

echo "[api] tenant isolation on admin tenant path"
code=$(request "PUT" "$BASE/admin/tenants/22222222-2222-2222-2222-222222222222" '{"tenant_slug":"beta-training","tenant_name":"Should Fail","allow_self_registration":false,"require_mfa":false,"max_active_bookings_per_learner":3}' "$ADMIN_COOKIE")
assert_code "$code" "403" "cross-tenant admin update denied"

echo "[api] role-permission matrix view and update"
code=$(request "GET" "$BASE/admin/permissions/matrix" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "GET /admin/permissions/matrix"
assert_body_contains "$TMP_DIR/body.json" '"permission"' "matrix body permission field"
assert_body_contains "$TMP_DIR/body.json" '"allowed"' "matrix body allowed field"
assert_body_contains "$TMP_DIR/body.json" '"role"' "matrix body role field"

code=$(request "PUT" "$BASE/admin/permissions/matrix" '{"assignments":[{"role":"program_coordinator","permission":"rbac.matrix.view","allowed":true}]}' "$ADMIN_COOKIE")
assert_code "$code" "200" "PUT /admin/permissions/matrix"
assert_body_contains "$TMP_DIR/body.json" '"updated"' "matrix update body"

echo "[api] role assignment management"
code=$(request "GET" "$BASE/admin/users/roles" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "GET /admin/users/roles"
assert_body_contains "$TMP_DIR/body.json" '"username"' "user roles list body"

code=$(request "POST" "$BASE/admin/users/11111111-1111-1111-1111-111111111104/roles" '{"role":"instructor"}' "$ADMIN_COOKIE")
assert_code "$code" "200" "POST /admin/users/:user_id/roles"
assert_body_contains "$TMP_DIR/body.json" '"assigned"' "assign role body"

code=$(request "DELETE" "$BASE/admin/users/11111111-1111-1111-1111-111111111104/roles/instructor" "" "$ADMIN_COOKIE")
assert_code "$code" "200" "DELETE /admin/users/:user_id/roles/:role"
assert_body_contains "$TMP_DIR/body.json" '"revoked"' "revoke role body"

##############################################################################
# SECTION 4: auth lockout and session rotation/invalidation
##############################################################################

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
assert_code "$code" "200" "POST /auth/logout"
assert_body_contains "$TMP_DIR/body.json" '"logged_out"' "logout body"
code=$(request "GET" "$BASE/auth/me" "" "$ADMIN_COOKIE")
assert_code "$code" "401" "invalidated session denied"

# Re-login admin so later sections can use it.
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"acme-training","username":"admin","password":"AdminPass1234"}' "$ADMIN_COOKIE")
assert_code "$code" "200" "admin re-login after logout"

##############################################################################
# SECTION 5: dashboard + feature-store endpoints
##############################################################################

echo "[api] dashboard refresh as coordinator (success)"
code=$(request "POST" "$BASE/dashboard/refresh" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /dashboard/refresh (coord)"
assert_body_contains "$TMP_DIR/body.json" '"refresh_id"' "refresh body refresh_id"

echo "[api] dashboard overview"
code=$(request "GET" "$BASE/dashboard/overview" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "GET /dashboard/overview"
assert_body_contains "$TMP_DIR/body.json" '"data"' "overview body"

echo "[api] dashboard today-sessions"
code=$(request "GET" "$BASE/dashboard/today-sessions" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "GET /dashboard/today-sessions"
assert_body_contains "$TMP_DIR/body.json" '"data"' "today-sessions body data"

echo "[api] dashboard feature-store nightly-batch"
code=$(request "POST" "$BASE/dashboard/feature-store/nightly-batch" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /dashboard/feature-store/nightly-batch"
assert_body_contains "$TMP_DIR/body.json" '"batch_ids"' "nightly batch body batch_ids"

echo "[api] dashboard feature-store learners"
code=$(request "GET" "$BASE/dashboard/feature-store/learners?window_days=7&limit=10" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /dashboard/feature-store/learners"
assert_body_contains "$TMP_DIR/body.json" '"data"' "learners body"

echo "[api] dashboard feature-store cohorts"
code=$(request "GET" "$BASE/dashboard/feature-store/cohorts?window_days=30&limit=5" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /dashboard/feature-store/cohorts"
assert_body_contains "$TMP_DIR/body.json" '"data"' "cohorts body"

echo "[api] dashboard feature-store reporting-metrics"
code=$(request "GET" "$BASE/dashboard/feature-store/reporting-metrics?window_days=7" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /dashboard/feature-store/reporting-metrics"
assert_body_contains "$TMP_DIR/body.json" '"data"' "reporting-metrics body"

##############################################################################
# SECTION 6: calendar management (rules, blackouts, terms)
##############################################################################

echo "[api] GET /calendar/availability/:session_id"
code=$(request "GET" "$BASE/calendar/availability/11111111-1111-1111-1111-111111115001" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "GET /calendar/availability"
assert_body_contains "$TMP_DIR/body.json" '"reason"' "availability body reason"
assert_body_contains "$TMP_DIR/body.json" '"alternatives"' "availability body alternatives"

echo "[api] calendar time-slot rule create"
code=$(request "POST" "$BASE/calendar/time-slots" '{"room_id":null,"weekday":1,"slot_start":"09:00","slot_end":"10:00","is_active":true}' "$COORD_COOKIE" "$TMP_DIR/time_slot_create.json")
assert_code "$code" "201" "POST /calendar/time-slots"
assert_body_contains "$TMP_DIR/time_slot_create.json" '"rule_id"' "time-slot create body rule_id"
RULE_ID=$(extract_string_field "$TMP_DIR/time_slot_create.json" "rule_id")
if [ -z "$RULE_ID" ]; then
  echo "[api] FAIL: could not extract rule_id"; cat "$TMP_DIR/time_slot_create.json"; exit 1
fi

echo "[api] calendar time-slot rule update (rule_id=$RULE_ID)"
code=$(request "PUT" "$BASE/calendar/time-slots/$RULE_ID" '{"room_id":null,"weekday":1,"slot_start":"09:00","slot_end":"11:00","is_active":true,"lock_version":0}' "$COORD_COOKIE")
assert_code "$code" "200" "PUT /calendar/time-slots/:rule_id"
assert_body_contains "$TMP_DIR/body.json" '"updated"' "time-slot update body"

echo "[api] calendar blackout create"
code=$(request "POST" "$BASE/calendar/blackouts" '{"room_id":null,"blackout_date":"2035-01-01","reason":"New Year","is_active":true}' "$COORD_COOKIE" "$TMP_DIR/blackout_create.json")
assert_code "$code" "201" "POST /calendar/blackouts"
assert_body_contains "$TMP_DIR/blackout_create.json" '"blackout_id"' "blackout create body"
BLACKOUT_ID=$(extract_string_field "$TMP_DIR/blackout_create.json" "blackout_id")

echo "[api] calendar blackout update (blackout_id=$BLACKOUT_ID)"
code=$(request "PUT" "$BASE/calendar/blackouts/$BLACKOUT_ID" '{"room_id":null,"blackout_date":"2035-01-02","reason":"New Year (observed)","is_active":true,"lock_version":0}' "$COORD_COOKIE")
assert_code "$code" "200" "PUT /calendar/blackouts/:blackout_id"
assert_body_contains "$TMP_DIR/body.json" '"updated"' "blackout update body"

echo "[api] calendar academic term create"
code=$(request "POST" "$BASE/calendar/terms" '{"name":"Test Term 2035","start_date":"2035-01-01","end_date":"2035-06-30","is_active":true}' "$COORD_COOKIE" "$TMP_DIR/term_create.json")
assert_code "$code" "201" "POST /calendar/terms"
assert_body_contains "$TMP_DIR/term_create.json" '"term_id"' "term create body"
TERM_ID=$(extract_string_field "$TMP_DIR/term_create.json" "term_id")

echo "[api] calendar academic term update (term_id=$TERM_ID)"
code=$(request "PUT" "$BASE/calendar/terms/$TERM_ID" '{"name":"Test Term 2035 Updated","start_date":"2035-01-01","end_date":"2035-07-31","is_active":true,"lock_version":0}' "$COORD_COOKIE")
assert_code "$code" "200" "PUT /calendar/terms/:term_id"
assert_body_contains "$TMP_DIR/body.json" '"updated"' "term update body"

##############################################################################
# SECTION 7: booking flows (capacity conflict, concurrency, reschedule, check-in)
##############################################################################

echo "[api] booking capacity conflict"
code=$(request "POST" "$BASE/bookings/hold" '{"session_id":"11111111-1111-1111-1111-111111115002"}' "$LEARNER_COOKIE")
assert_code "$code" "409" "capacity reached conflict"
assert_body_contains "$TMP_DIR/body.json" '"capacity reached"' "capacity body message"

echo "[api] booking concurrency: no oversell on simultaneous holds"

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
BOOKING_FILE=""
for file in "$HOLD_OUT_1" "$HOLD_OUT_2" "$HOLD_OUT_3"; do
  id=$(extract_booking_id "$file")
  if [ -n "$id" ]; then
    BOOKING_ID="$id"
    BOOKING_FILE="$file"
    break
  fi
done
if [ -z "$BOOKING_ID" ]; then
  echo "[api] FAIL: booking id not found in concurrency hold responses"
  exit 1
fi

echo "[api] booking hold body has booking_id + state + hold_expires_at"
assert_body_contains "$BOOKING_FILE" '"state"' "hold body state"
assert_body_contains "$BOOKING_FILE" '"hold_expires_at"' "hold body expires"

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

echo "[api] POST /bookings/:booking_id/reschedule (confirmed booking to new session)"
code=$(request "POST" "$BASE/bookings/$BOOKING_ID/reschedule" '{"session_id":"11111111-1111-1111-1111-111111115003","reason":"change"}' "$COORD_COOKIE")
assert_code "$code" "200" "booking reschedule"
assert_body_contains "$TMP_DIR/body.json" '"rescheduled"' "reschedule body"

echo "[api] POST /bookings/:booking_id/check-in (instructor)"
code=$(request "POST" "$BASE/bookings/$BOOKING_ID/check-in" '{"reason":"arrived"}' "$INSTR_COOKIE")
assert_code "$code" "200" "booking check-in"
assert_body_contains "$TMP_DIR/body.json" '"checked_in"' "check-in body"

echo "[api] cancellation cutoff enforced"
code=$(request "POST" "$BASE/bookings/11111111-1111-1111-1111-111111116002/cancel" '{}' "$LEARNER_COOKIE")
assert_code "$code" "409" "cancellation cutoff"
assert_body_contains "$TMP_DIR/body.json" '"cancellation cutoff exceeded"' "cutoff body"

##############################################################################
# SECTION 8: security upload validation (multipart)
##############################################################################

echo "[api] security upload validate missing form (expect 400)"
code=$(curl -sS -o "$TMP_DIR/body.json" -w "%{http_code}" -X POST "$BASE/security/upload/validate" -b "$ADMIN_COOKIE" -c "$ADMIN_COOKIE")
assert_code "$code" "400" "validate missing form"

echo "[api] security upload validate .txt file (expect 200 with checksum)"
UPLOAD_FILE="$TMP_DIR/sample.txt"
printf "hello world\n" > "$UPLOAD_FILE"
code=$(curl -sS -o "$TMP_DIR/body.json" -w "%{http_code}" -X POST "$BASE/security/upload/validate" -F "file=@$UPLOAD_FILE" -b "$ADMIN_COOKIE" -c "$ADMIN_COOKIE")
assert_code "$code" "200" "POST /security/upload/validate"
assert_body_contains "$TMP_DIR/body.json" '"checksum_sha256"' "validate body checksum"

echo "[api] security upload validate wrong extension (expect 400)"
BAD_FILE="$TMP_DIR/sample.exe"
printf "MZ" > "$BAD_FILE"
code=$(curl -sS -o "$TMP_DIR/body.json" -w "%{http_code}" -X POST "$BASE/security/upload/validate" -F "file=@$BAD_FILE" -b "$ADMIN_COOKIE" -c "$ADMIN_COOKIE")
assert_code "$code" "400" "validate reject .exe"
assert_body_contains "$TMP_DIR/body.json" '"error"' "validate error body"

##############################################################################
# SECTION 9: content upload/preview/download/versions/search/bulk/duplicates
##############################################################################

echo "[api] POST /content/uploads/start"
code=$(request "POST" "$BASE/content/uploads/start" '{"file_name":"hello.txt","mime_type":"text/plain","total_chunks":1,"chunk_size_bytes":12}' "$COORD_COOKIE" "$TMP_DIR/upload_start.json")
assert_code "$code" "201" "POST /content/uploads/start"
assert_body_contains "$TMP_DIR/upload_start.json" '"upload_id"' "upload start body"
UPLOAD_ID=$(extract_string_field "$TMP_DIR/upload_start.json" "upload_id")
if [ -z "$UPLOAD_ID" ]; then
  echo "[api] FAIL: could not extract upload_id"; cat "$TMP_DIR/upload_start.json"; exit 1
fi

echo "[api] PUT /content/uploads/:upload_id/chunks/0"
CHUNK_FILE="$TMP_DIR/chunk0.bin"
printf "hello world\n" > "$CHUNK_FILE"
code=$(curl -sS -o "$TMP_DIR/body.json" -w "%{http_code}" -X PUT "$BASE/content/uploads/$UPLOAD_ID/chunks/0" --data-binary "@$CHUNK_FILE" -b "$COORD_COOKIE" -c "$COORD_COOKIE")
assert_code "$code" "200" "PUT /content/uploads/:upload_id/chunks/:chunk_index"
assert_body_contains "$TMP_DIR/body.json" '"chunk_received"' "chunk body"

echo "[api] POST /content/uploads/:upload_id/complete"
code=$(request "POST" "$BASE/content/uploads/$UPLOAD_ID/complete" '{"title":"Hello Doc","summary":"smoke test","difficulty":1,"duration_minutes":5}' "$COORD_COOKIE" "$TMP_DIR/upload_complete.json")
assert_code "$code" "200" "POST /content/uploads/:upload_id/complete"
assert_body_contains "$TMP_DIR/upload_complete.json" '"document_id"' "complete body document_id"
assert_body_contains "$TMP_DIR/upload_complete.json" '"version_no"' "complete body version_no"
DOCUMENT_ID=$(extract_string_field "$TMP_DIR/upload_complete.json" "document_id")

echo "[api] GET /content/documents/:document_id/preview"
code=$(curl -sS -o "$TMP_DIR/preview.bin" -w "%{http_code}" -X GET "$BASE/content/documents/$DOCUMENT_ID/preview" -b "$LEARNER_COOKIE" -c "$LEARNER_COOKIE")
assert_code "$code" "200" "GET /content/documents/:document_id/preview"
if ! grep -q "hello world" "$TMP_DIR/preview.bin"; then
  echo "[api] FAIL: preview body missing expected content"
  cat "$TMP_DIR/preview.bin"; exit 1
fi

echo "[api] GET /content/documents/:document_id/download (watermarked)"
code=$(curl -sS -D "$TMP_DIR/download_headers.txt" -o "$TMP_DIR/download.bin" -w "%{http_code}" -X GET "$BASE/content/documents/$DOCUMENT_ID/download" -b "$LEARNER_COOKIE" -c "$LEARNER_COOKIE")
assert_code "$code" "200" "GET /content/documents/:document_id/download"
assert_body_contains "$TMP_DIR/download_headers.txt" 'X-Watermark' "download has X-Watermark"
if ! grep -q "WATERMARK" "$TMP_DIR/download.bin"; then
  echo "[api] FAIL: download body missing watermark"; cat "$TMP_DIR/download.bin"; exit 1
fi

echo "[api] GET /content/documents/:document_id/versions"
code=$(request "GET" "$BASE/content/documents/$DOCUMENT_ID/versions" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /content/documents/:document_id/versions"
assert_body_contains "$TMP_DIR/body.json" '"version_no"' "versions body version_no"

echo "[api] POST /content/documents/:document_id/share-links"
code=$(request "POST" "$BASE/content/documents/$DOCUMENT_ID/share-links" '{}' "$COORD_COOKIE" "$TMP_DIR/share_link.json")
assert_code "$code" "201" "POST /content/documents/:document_id/share-links"
assert_body_contains "$TMP_DIR/share_link.json" '"token"' "share link body"
SHARE_TOKEN=$(extract_string_field "$TMP_DIR/share_link.json" "token")

echo "[api] GET /content/share/:token/download (public)"
code=$(curl -sS -D "$TMP_DIR/share_headers.txt" -o "$TMP_DIR/share.bin" -w "%{http_code}" -X GET "$BASE/content/share/$SHARE_TOKEN/download")
assert_code "$code" "200" "GET /content/share/:token/download"
assert_body_contains "$TMP_DIR/share_headers.txt" 'X-Watermark' "share download has watermark header"

echo "[api] GET /content/documents/search"
code=$(request "GET" "$BASE/content/documents/search?q=hello" "" "$LEARNER_COOKIE")
assert_code "$code" "200" "GET /content/documents/search"
assert_body_contains "$TMP_DIR/body.json" '"data"' "search body"

echo "[api] POST /content/documents/bulk"
code=$(request "POST" "$BASE/content/documents/bulk" "{\"document_ids\":[\"$DOCUMENT_ID\"],\"archive\":false}" "$COORD_COOKIE")
assert_code "$code" "200" "POST /content/documents/bulk"
assert_body_contains "$TMP_DIR/body.json" '"updated"' "bulk body"

echo "[api] POST /content/documents/duplicates/detect"
code=$(request "POST" "$BASE/content/documents/duplicates/detect" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /content/documents/duplicates/detect"
assert_body_contains "$TMP_DIR/body.json" '"flagged"' "duplicates body flagged"

echo "[api] PATCH /content/documents/duplicates/:duplicate_id/merge-flag (expect 404 for unknown id)"
code=$(request "PATCH" "$BASE/content/documents/duplicates/00000000-0000-0000-0000-000000000000/merge-flag" '{"merge_candidate":true}' "$COORD_COOKIE")
assert_code "$code" "404" "PATCH merge-flag unknown duplicate"
assert_body_contains "$TMP_DIR/body.json" 'not found' "merge-flag 404 body"

##############################################################################
# SECTION 10: content ingestion endpoints
##############################################################################

echo "[api] POST /content/ingestion/sources"
code=$(request "POST" "$BASE/content/ingestion/sources" '{"name":"Local Health Feed","base_url":"http://api:8000/health","schedule_interval_minutes":60,"schedule_jitter_seconds":0,"rate_limit_per_minute":6,"request_timeout_seconds":10}' "$COORD_COOKIE" "$TMP_DIR/ingestion_source.json")
assert_code "$code" "201" "POST /content/ingestion/sources"
assert_body_contains "$TMP_DIR/ingestion_source.json" '"source_id"' "ingestion source body"
SOURCE_ID=$(extract_string_field "$TMP_DIR/ingestion_source.json" "source_id")

echo "[api] GET /content/ingestion/sources"
code=$(request "GET" "$BASE/content/ingestion/sources" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /content/ingestion/sources"
assert_body_contains "$TMP_DIR/body.json" '"Local Health Feed"' "ingestion sources list"

echo "[api] POST /content/ingestion/sources/:source_id/run (synchronous fetch against in-network /health)"
# NOTE: run before registering proxies/user-agents so the client does a direct in-network fetch.
code=$(request "POST" "$BASE/content/ingestion/sources/$SOURCE_ID/run" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /content/ingestion/sources/:source_id/run"
assert_body_contains "$TMP_DIR/body.json" '"status"' "ingestion run body has status"
assert_body_contains "$TMP_DIR/body.json" '"started_at"' "ingestion run body has started_at"

echo "[api] POST /content/ingestion/proxies"
code=$(request "POST" "$BASE/content/ingestion/proxies" '{"proxy_url":"http://localhost:3128"}' "$COORD_COOKIE")
assert_code "$code" "201" "POST /content/ingestion/proxies"
assert_body_contains "$TMP_DIR/body.json" '"proxy_added"' "proxy add body"

echo "[api] POST /content/ingestion/user-agents"
code=$(request "POST" "$BASE/content/ingestion/user-agents" '{"user_agent":"TestBot/1.0"}' "$COORD_COOKIE")
assert_code "$code" "201" "POST /content/ingestion/user-agents"
assert_body_contains "$TMP_DIR/body.json" '"user_agent_added"' "user-agent add body"

echo "[api] POST /content/ingestion/run-due"
# After this point a proxy is registered that may make outbound calls fail; run-due
# still returns 200 with an empty/failed runs list. We assert the endpoint contract,
# not the outbound outcome.
code=$(request "POST" "$BASE/content/ingestion/run-due?max_sources=1" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /content/ingestion/run-due"
assert_body_contains "$TMP_DIR/body.json" '"data"' "run-due body"

echo "[api] GET /content/ingestion/runs"
code=$(request "GET" "$BASE/content/ingestion/runs?limit=5" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /content/ingestion/runs"
assert_body_contains "$TMP_DIR/body.json" '"data"' "ingestion runs body"

echo "[api] POST /content/ingestion/sources/:source_id/manual-review (pause)"
code=$(request "POST" "$BASE/content/ingestion/sources/$SOURCE_ID/manual-review" '{"approve":false,"reason":"captcha detected"}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /content/ingestion/sources/:source_id/manual-review pause"
assert_body_contains "$TMP_DIR/body.json" '"paused"' "manual review paused body"

echo "[api] POST /content/ingestion/sources/:source_id/manual-review (approve)"
code=$(request "POST" "$BASE/content/ingestion/sources/$SOURCE_ID/manual-review" '{"approve":true}' "$COORD_COOKIE")
assert_code "$code" "200" "manual-review approve"
assert_body_contains "$TMP_DIR/body.json" '"approved"' "manual review approved body"

##############################################################################
# SECTION 11: planning endpoints
##############################################################################

echo "[api] POST /planning/plans"
code=$(request "POST" "$BASE/planning/plans" '{"name":"Spring Plan","description":"integration test plan","starts_on":"2035-03-01","ends_on":"2035-06-30"}' "$COORD_COOKIE" "$TMP_DIR/plan_create.json")
assert_code "$code" "201" "POST /planning/plans"
assert_body_contains "$TMP_DIR/plan_create.json" '"plan_id"' "plan create body"
PLAN_ID=$(extract_string_field "$TMP_DIR/plan_create.json" "plan_id")

echo "[api] GET /planning/plans/:plan_id/tree"
code=$(request "GET" "$BASE/planning/plans/$PLAN_ID/tree" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /planning/plans/:plan_id/tree"
assert_body_contains "$TMP_DIR/body.json" '"plan"' "plan tree body"

echo "[api] POST /planning/plans/:plan_id/milestones"
code=$(request "POST" "$BASE/planning/plans/$PLAN_ID/milestones" '{"title":"M1","description":"first milestone","due_date":"2035-04-01","sort_order":1}' "$COORD_COOKIE" "$TMP_DIR/milestone_create.json")
assert_code "$code" "201" "POST /planning/plans/:plan_id/milestones"
assert_body_contains "$TMP_DIR/milestone_create.json" '"milestone_id"' "milestone create body"
MILESTONE_ID=$(extract_string_field "$TMP_DIR/milestone_create.json" "milestone_id")

echo "[api] POST /planning/milestones/:milestone_id/tasks (task 1)"
code=$(request "POST" "$BASE/planning/milestones/$MILESTONE_ID/tasks" '{"title":"Task A","description":"","state":"todo","estimated_minutes":60,"sort_order":1}' "$COORD_COOKIE" "$TMP_DIR/task1.json")
assert_code "$code" "201" "POST /planning/milestones/:milestone_id/tasks (1)"
assert_body_contains "$TMP_DIR/task1.json" '"task_id"' "task1 body"
TASK1_ID=$(extract_string_field "$TMP_DIR/task1.json" "task_id")

echo "[api] POST /planning/milestones/:milestone_id/tasks (task 2)"
code=$(request "POST" "$BASE/planning/milestones/$MILESTONE_ID/tasks" '{"title":"Task B","description":"","state":"todo","estimated_minutes":60,"sort_order":2}' "$COORD_COOKIE" "$TMP_DIR/task2.json")
assert_code "$code" "201" "POST /planning/milestones/:milestone_id/tasks (2)"
TASK2_ID=$(extract_string_field "$TMP_DIR/task2.json" "task_id")

echo "[api] POST /planning/tasks/:task_id/dependencies"
code=$(request "POST" "$BASE/planning/tasks/$TASK1_ID/dependencies" "{\"depends_on_task_id\":\"$TASK2_ID\"}" "$COORD_COOKIE")
assert_code "$code" "200" "POST /planning/tasks/:task_id/dependencies"
assert_body_contains "$TMP_DIR/body.json" '"added"' "dependency add body"

echo "[api] PATCH /planning/plans/:plan_id/reorder-milestones"
code=$(request "PATCH" "$BASE/planning/plans/$PLAN_ID/reorder-milestones" "{\"ordered_ids\":[\"$MILESTONE_ID\"]}" "$COORD_COOKIE")
assert_code "$code" "200" "PATCH reorder-milestones"
assert_body_contains "$TMP_DIR/body.json" '"reordered"' "reorder milestones body"

echo "[api] PATCH /planning/milestones/:milestone_id/reorder-tasks"
code=$(request "PATCH" "$BASE/planning/milestones/$MILESTONE_ID/reorder-tasks" "{\"ordered_ids\":[\"$TASK2_ID\",\"$TASK1_ID\"]}" "$COORD_COOKIE")
assert_code "$code" "200" "PATCH reorder-tasks"
assert_body_contains "$TMP_DIR/body.json" '"reordered"' "reorder tasks body"

echo "[api] PATCH /planning/tasks/bulk"
code=$(request "PATCH" "$BASE/planning/tasks/bulk" "{\"task_ids\":[\"$TASK1_ID\"],\"state\":\"in_progress\"}" "$COORD_COOKIE")
assert_code "$code" "200" "PATCH /planning/tasks/bulk"
assert_body_contains "$TMP_DIR/body.json" '"updated"' "bulk tasks body"

echo "[api] DELETE /planning/tasks/:task_id/dependencies/:depends_on_task_id"
code=$(request "DELETE" "$BASE/planning/tasks/$TASK1_ID/dependencies/$TASK2_ID" "" "$COORD_COOKIE")
assert_code "$code" "200" "DELETE dependency"
assert_body_contains "$TMP_DIR/body.json" '"removed"' "remove dependency body"

##############################################################################
# SECTION 12: observability endpoints
##############################################################################

echo "[api] GET /observability/workflow-logs"
code=$(request "GET" "$BASE/observability/workflow-logs?limit=10" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /observability/workflow-logs"
assert_body_contains "$TMP_DIR/body.json" '"data"' "workflow logs body"

echo "[api] POST /observability/scraping-errors"
code=$(request "POST" "$BASE/observability/scraping-errors" '{"source_name":"test-source","error_code":"HTTP_500","error_message":"upstream failure","metadata":{"url":"http://example"}}' "$INSTR_COOKIE")
assert_code "$code" "201" "POST /observability/scraping-errors"
assert_body_contains "$TMP_DIR/body.json" '"recorded"' "scraping error body"

echo "[api] POST /observability/anomalies/detect"
code=$(request "POST" "$BASE/observability/anomalies/detect" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /observability/anomalies/detect"
assert_body_contains "$TMP_DIR/body.json" '"detected"' "anomalies detect body"

echo "[api] GET /observability/anomalies"
code=$(request "GET" "$BASE/observability/anomalies?limit=5" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /observability/anomalies"
assert_body_contains "$TMP_DIR/body.json" '"data"' "anomalies list body"

echo "[api] POST /observability/report-schedules"
code=$(request "POST" "$BASE/observability/report-schedules" '{"name":"test schedule","format":"csv","frequency":"daily","output_folder":"/tmp/reports"}' "$COORD_COOKIE" "$TMP_DIR/schedule_create.json")
assert_code "$code" "201" "POST /observability/report-schedules"
assert_body_contains "$TMP_DIR/schedule_create.json" '"schedule_id"' "schedule body"
SCHEDULE_ID=$(extract_string_field "$TMP_DIR/schedule_create.json" "schedule_id")

echo "[api] POST /observability/report-schedules/:schedule_id/run"
code=$(request "POST" "$BASE/observability/report-schedules/$SCHEDULE_ID/run" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /observability/report-schedules/:schedule_id/run"
assert_body_contains "$TMP_DIR/body.json" '"data"' "schedule run body"

echo "[api] POST /observability/report-schedules/run-due"
code=$(request "POST" "$BASE/observability/report-schedules/run-due" '{}' "$COORD_COOKIE")
assert_code "$code" "200" "POST /observability/report-schedules/run-due"
assert_body_contains "$TMP_DIR/body.json" '"data"' "run-due body"

echo "[api] GET /observability/report-exports"
code=$(request "GET" "$BASE/observability/report-exports?limit=5" "" "$COORD_COOKIE")
assert_code "$code" "200" "GET /observability/report-exports"
assert_body_contains "$TMP_DIR/body.json" '"data"' "exports body"

##############################################################################
# SECTION 13: tenant isolation (cross-tenant access denial)
##############################################################################

echo "[api] tenant isolation on booking access"
code=$(request "POST" "$BASE/auth/login" '{"tenant_slug":"beta-training","username":"learnerx","password":"LearnerPass12"}' "$BETA_COOKIE")
assert_code "$code" "200" "beta learner login"
code=$(request "POST" "$BASE/bookings/$BOOKING_ID/confirm" '{}' "$BETA_COOKIE")
assert_code "$code" "404" "cross-tenant booking access denied"

echo "[api] passed"
