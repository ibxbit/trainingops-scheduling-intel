# TrainingOps

Project Type: fullstack

TrainingOps is a multi-tenant training operations platform. The backend is a Go/Echo HTTP API backed by PostgreSQL. The frontend is a React + Vite single-page app. Everything runs in Docker and is orchestrated by `docker-compose`.

## Quick Start (Docker)

Prerequisite: Docker Desktop (or any Docker Engine + Compose v2).

```bash
docker-compose up
```

That single command:

1. Starts PostgreSQL 16 with the `trainingops` database.
2. Runs all SQL migrations (including seed data for demo accounts and sessions).
3. Builds and starts the Go backend API on port `8000`.
4. Builds and serves the React frontend on port `3001`.

No host-side Go, Node, or PostgreSQL install is required. Source code is mounted at build time; the stack is self-contained.

## Access

- Frontend (web UI): `http://localhost:3001`
- Backend (REST API): `http://localhost:8000`
- Health endpoint: `http://localhost:8000/health`

## Demo Accounts

All accounts belong to tenant slug `acme-training` unless noted otherwise.

| Role                 | Username     | Email                                | Password        |
| -------------------- | ------------ | ------------------------------------ | --------------- |
| Administrator        | `admin`       | `admin@acme-training.local`          | `AdminPass1234` |
| Program Coordinator  | `coordinator` | `coordinator@acme-training.local`    | `CoordPass1234` |
| Instructor           | `instructor`  | `instructor@acme-training.local`     | `InstrPass1234` |
| Learner              | `learner1`    | `learner1@acme-training.local`       | `LearnerPass12` |
| Learner (secondary)  | `learner2`    | `learner2@acme-training.local`       | `LearnerPass12` |
| Learner (other tenant `beta-training`) | `learnerx` | `learnerx@beta-training.local` | `LearnerPass12` |

The backend authenticates by `(tenant_slug, username, password)`; the email column is reserved for future SSO mapping.

## Verify the Stack

After `docker-compose up`, verify with the following.

### API checks (curl)

```bash
# 1. Health (no auth required)
curl -s http://localhost:8000/health
# -> {"status":"ok"}

# 2. Login as admin and capture the session cookie
curl -s -c cookie.txt -H "Content-Type: application/json" \
  -d '{"tenant_slug":"acme-training","username":"admin","password":"AdminPass1234"}' \
  http://localhost:8000/api/v1/auth/login
# -> {"data":{"status":"authenticated"}}

# 3. Identify the session (tenant_id / user_id / roles)
curl -s -b cookie.txt http://localhost:8000/api/v1/auth/me
# -> {"data":{"tenant_id":"...","user_id":"...","roles":["administrator"]}}

# 4. List tenant settings (admin-only)
curl -s -b cookie.txt http://localhost:8000/api/v1/admin/tenants

# 5. Today's sessions (any role)
curl -s -b cookie.txt http://localhost:8000/api/v1/dashboard/today-sessions
```

A Postman/Insomnia collection can be built from the same base URL (`http://localhost:8000/api/v1`) using cookie-based auth.

### Web UI check

1. Open `http://localhost:3001` in a browser.
2. Log in with `admin` / `AdminPass1234` against tenant `acme-training`.
3. Confirm the Administrator landing page loads and Tenant Settings can be saved.
4. Log out, log in as `learner1` / `LearnerPass12`, and confirm the admin routes are blocked.
5. Upload a document from the Content Library, then log out and log in as a second user: the upload-resume cache should NOT carry over (state isolation on user switch).

## Running Tests

```bash
bash ./run_tests.sh
```

This boots the full Docker stack (db + migrations + api + frontend + tester), waits for the API health endpoint, runs the Go unit tests and frontend tests, and executes `API_tests/run_api_tests.sh` inside the `tester` container (real HTTP against the real API container).

## Stop the Stack

```bash
docker-compose down
```

Add `-v` to also remove the database volume.

## Repository Layout

```text
repo/
  backend/                  Go API service (Echo + pgx + migrations)
  frontend/                 React + Vite SPA
  API_tests/                Real-HTTP integration tests
  unit_tests/               Unit test entrypoint
  tester/                   Alpine image running integration tests
  docker-compose.yml        Full stack orchestration
  run_tests.sh              All tests via Docker
```

## Acceptance Evidence

- Runnability consistency (ports, proxy, docker-compose): `docker-compose.yml`, `backend/internal/config/config.go`, `frontend/vite.config.ts`.
- Administrator scope (tenant settings + permission matrix + role assignments, API + UI + guards):
  - Backend: `backend/internal/admin/*.go`, `backend/migrations/013_admin_controls.sql`, `backend/cmd/server/main.go`
  - Frontend: `frontend/src/features/admin/AdminPage.tsx`, `frontend/src/api/endpoints.ts`, `frontend/src/app/route-config.tsx`, `frontend/src/auth/policy.ts`
  - Frontend tests: `frontend/src/test/admin.access-and-flows.test.tsx`
- Security logging hardening (path-token redaction + IP anonymization): `backend/internal/security/logging.go`, `backend/internal/security/logging_test.go`.
- Frontend state isolation on user switch/logout: `frontend/src/state/upload-resume-cache.ts`, `frontend/src/features/content/ContentLibraryPage.tsx`, `frontend/src/App.tsx`, `frontend/src/test/state.isolation.test.tsx`.
- High-risk backend integration tests (real API + real DB path, no transport/service mocks): `API_tests/run_api_tests.sh` — lockout, session rotation/invalidation, admin authz/isolation, booking concurrency, full 69-endpoint coverage.
