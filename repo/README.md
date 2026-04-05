# TrainingOps (Offline-First)

This project now supports two reproducible paths:

1. Docker path (`docker compose`) for one-command full-stack parity.
2. Local Docker-free path (Go + Node + local PostgreSQL) using `start_local.sh` and `run_tests_local.sh`.

## Prerequisites

- Go 1.22+
- Node 20+
- PostgreSQL 16+
- Bash shell

Environment required by backend:

- `DATABASE_URL`
- `ENCRYPTION_KEY` (exactly 32 bytes)
- `SESSION_SECURE_COOKIE=false` for local HTTP usage

## Quick Start (Docker)

```bash
docker compose up
```

Services:

- Frontend: `http://localhost:3001`
- Backend: `http://localhost:8000`
- PostgreSQL: internal compose network (`db:5432`)

## Quick Start (Local / Docker-Free)

Start API + frontend:

```bash
DATABASE_URL=postgres://... ENCRYPTION_KEY=0123456789abcdef0123456789abcdef SESSION_SECURE_COOKIE=false bash ./start_local.sh
```

Stop local processes:

```bash
bash ./stop_local.sh
```

Local process logs:

- API log: `.logs/backend.log`
- Frontend log: `.logs/frontend.log`

## Test Commands

Docker harness:

```bash
bash ./run_tests.sh
```

Local harness (Docker-free):

```bash
bash ./run_tests_local.sh
```

Notes:

- `run_tests_local.sh` expects API health on `http://localhost:8000/health`.
- API tests run against local backend via `BASE=http://localhost:8000/api/v1`.

## Demo Accounts

Tenant `acme-training`:

- `admin` / `AdminPass1234`
- `coordinator` / `CoordPass1234`
- `instructor` / `InstrPass1234`
- `learner1` / `LearnerPass12`

Tenant `beta-training`:

- `learnerx` / `LearnerPass12`

## Operational Logging and Retention

Application log outputs:

- API request/security logs are written to stdout (or `.logs/backend.log` for local script start).
- Frontend dev/build logs are written to stdout (or `.logs/frontend.log` for local script start).

Database-backed observability logs:

- `workflow_logs`
- `scraping_errors`
- `anomaly_events`
- `report_exports`

Retention policy:

- 90-day retention sweep is applied at backend startup by `observabilitySvc.ApplyRetention(..., 90)`.
- Implementation: `backend/internal/observability/repository.go`, `backend/internal/observability/service.go`, `backend/cmd/server/main.go`.

Troubleshooting:

- API not reachable: check `.logs/backend.log` and `DATABASE_URL` connectivity.
- Frontend not reachable: check `.logs/frontend.log` and Node dependency install output.
- API tests failing locally: verify migrations and seeds ran (`go run ./backend/cmd/migrate`).

## Acceptance Evidence Map

- Planning drag-and-drop flow:
  - UI: `frontend/src/features/planning/PlanningPage.tsx`
  - Test: `frontend/src/test/planning.drag-drop.test.tsx`
- Analytics + feature store flows:
  - UI and actions: `frontend/src/features/dashboard/DashboardPage.tsx`
  - API client support: `frontend/src/api/endpoints.ts`
  - Test: `frontend/src/test/dashboard.feature-store.test.tsx`
- Content ingestion error handling + fallback:
  - Service fallback/manual-review/rate-limit logic: `backend/internal/content/ingestion_service.go`
  - Tests: `backend/internal/content/ingestion_fallback_test.go`
- Document upload edge cases + versioning:
  - Upload/session/version/share UI: `frontend/src/features/content/ContentLibraryPage.tsx`
  - Upload/version APIs: `frontend/src/api/endpoints.ts`
  - Validation tests: `frontend/src/test/content.upload-edge.test.tsx`
  - Metadata persistence and version finalize path: `backend/internal/content/service.go`, `backend/internal/content/repository.go`
- Security boundaries and tenant isolation:
  - Request DB tenant context for RLS (`app.tenant_id`): `backend/internal/access/middleware.go`, `backend/internal/dbctx/context.go`
  - Booking object authorization (learner vs tenant-wide coordinator/admin): `backend/internal/booking/service.go`, `backend/internal/booking/repository.go`
  - API matrix test updates: `API_tests/run_api_tests.sh`

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
