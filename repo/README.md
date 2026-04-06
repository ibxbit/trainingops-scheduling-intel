# TrainingOps (Offline-First)

This repository supports both Docker and Docker-free local execution.

- Local mode is first-class and uses host PostgreSQL + Go + Vite.
- Docker mode remains available for containerized parity.

## Prerequisites

- Go 1.22+
- Node 20+
- PostgreSQL 16+
- Bash shell

## Environment Variables

Required:

- `DATABASE_URL` (example: `postgres://trainingops:trainingops@localhost:5432/trainingops?sslmode=disable`)
- `ENCRYPTION_KEY` (exactly 32 bytes)

Recommended for local HTTP:

- `SESSION_SECURE_COOKIE=false`

Optional:

- `BACKEND_PORT` (default: `8000`)
- `FRONTEND_PORT` (default: `3000`)
- `VITE_API_PROXY_TARGET` (default local mode: `http://localhost:8000`; docker mode: `http://api:8000`)
- `SESSION_ROTATE_EVERY` (default: `5m`, useful for short-interval test verification)
- `SESSION_TTL` (default: `24h`)

## Local Start (Docker-Free)

1) Export required environment:

```bash
export DATABASE_URL='postgres://trainingops:trainingops@localhost:5432/trainingops?sslmode=disable'
export ENCRYPTION_KEY='0123456789abcdef0123456789abcdef'
export SESSION_SECURE_COOKIE=false
```

2) Start backend + frontend:

```bash
bash ./start_local.sh
```

Local URLs:

- Frontend: `http://localhost:3000`
- Backend: `http://localhost:8000`
- Health: `http://localhost:8000/health`

Logs:

- `.logs/backend.log`
- `.logs/frontend.log`

## Local Stop

```bash
bash ./stop_local.sh
```

## Local Tests (Docker-Free)

```bash
export DATABASE_URL='postgres://trainingops:trainingops@localhost:5432/trainingops?sslmode=disable'
export ENCRYPTION_KEY='0123456789abcdef0123456789abcdef'
export SESSION_SECURE_COOKIE=false
bash ./run_tests_local.sh
```

What it runs:

- Backend tests: `go test ./...`
- Frontend tests: `npm test -- --run`
- API integration tests: `API_tests/run_api_tests.sh` against a temporary backend on `:18000`

## Build

```bash
cd frontend && npm run build
```

## Docker Start (Optional)

```bash
docker compose up
```

Docker URLs:

- Frontend: `http://localhost:3001`
- Backend: `http://localhost:8000`

## Quick Verification Checklist

- `curl http://localhost:8000/health` returns `{"status":"ok"}`
- Log in as `admin / AdminPass1234` on `acme-training`
- Navigate to `Administrator` page and save tenant settings
- Confirm non-admin account cannot access `/admin`
- Create a content upload, logout, then login as another user and verify upload resume state does not carry over

## Demo Accounts

Tenant `acme-training`:

- `admin` / `AdminPass1234`
- `coordinator` / `CoordPass1234`
- `instructor` / `InstrPass1234`
- `learner1` / `LearnerPass12`
- `learner2` / `LearnerPass12`

Tenant `beta-training`:

- `learnerx` / `LearnerPass12`

## Acceptance Evidence

- Runnability consistency (ports/proxy/docs/scripts):
  - `backend/internal/config/config.go`
  - `frontend/vite.config.ts`
  - `start_local.sh`
  - `run_tests_local.sh`
  - `docker-compose.yml`
- Administrator scope (tenant settings + permission matrix + role assignments, API + UI + guards):
  - Backend: `backend/internal/admin/models.go`, `backend/internal/admin/repository.go`, `backend/internal/admin/service.go`, `backend/internal/admin/handler.go`, `backend/migrations/013_admin_controls.sql`, `backend/cmd/server/main.go`
  - Frontend: `frontend/src/features/admin/AdminPage.tsx`, `frontend/src/api/endpoints.ts`, `frontend/src/app/route-config.tsx`, `frontend/src/auth/policy.ts`
  - Frontend tests: `frontend/src/test/admin.access-and-flows.test.tsx`
- Security logging hardening (path token redaction + IP anonymization):
  - `backend/internal/security/logging.go`
  - `backend/internal/security/logging_test.go`
- Frontend state isolation on user switch/logout:
  - `frontend/src/state/upload-resume-cache.ts`
  - `frontend/src/features/content/ContentLibraryPage.tsx`
  - `frontend/src/App.tsx`
  - `frontend/src/test/state.isolation.test.tsx`
- High-risk backend integration tests (real API + DB path):
  - `API_tests/run_api_tests.sh` (lockout, session rotation/invalidation, admin authz/isolation, booking concurrency)

## Repository Layout

```text
repo/
  backend/
  frontend/
  API_tests/
  unit_tests/
  run_tests.sh
  run_tests_local.sh
  start_local.sh
  stop_local.sh
```
