# Self-Test

Date: 2026-04-06

## Commands Executed

1. `go test ./...` (workdir: `backend`)
2. `npm test -- --run` (workdir: `frontend`)
3. `npm run build` (workdir: `frontend`)
4. `bash ./run_tests_local.sh` (workdir: repo root)

## Results

- Backend tests: **PASS**
  - Packages with tests passed: `internal/auth`, `internal/booking`, `internal/content`, `internal/dashboard`, `internal/security`
  - Other backend packages: no test files.

- Frontend tests: **PASS**
  - Test files: **9 passed**
  - Tests: **12 passed**

- Frontend build: **PASS**
  - `vite build` completed successfully.

- Local harness (`run_tests_local.sh`): **PARTIAL**
  - Backend stage: pass
  - Frontend stage: pass
  - API stage: not executed due missing required env vars
  - Exact output: `[local-tests] DATABASE_URL and ENCRYPTION_KEY are required for API tests`

## Verification Boundary

- Full API integration checks in `API_tests/run_api_tests.sh` were not executed in this rerun because `DATABASE_URL` and `ENCRYPTION_KEY` were not set in the shell environment.
- No Docker commands were run.

## Reproduction (to complete API stage)

```bash
export DATABASE_URL='postgres://trainingops:trainingops@localhost:5432/trainingops?sslmode=disable'
export ENCRYPTION_KEY='0123456789abcdef0123456789abcdef'
export SESSION_SECURE_COOKIE=false
bash ./run_tests_local.sh
```
