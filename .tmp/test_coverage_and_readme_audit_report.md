# Test Coverage Audit

## Project Type Detection
- README explicitly declares `Project Type: fullstack` at `repo/README.md:3`.
- Project type used for strict checks: **fullstack**.

## Backend Endpoint Inventory
Source: `repo/backend/cmd/server/main.go:98-406`.

1. `GET /health`
2. `POST /api/v1/auth/login`
3. `POST /api/v1/auth/logout`
4. `GET /api/v1/auth/me`
5. `POST /api/v1/security/upload/validate`
6. `GET /api/v1/calendar/availability/:session_id`
7. `POST /api/v1/calendar/time-slots`
8. `PUT /api/v1/calendar/time-slots/:rule_id`
9. `POST /api/v1/calendar/blackouts`
10. `PUT /api/v1/calendar/blackouts/:blackout_id`
11. `POST /api/v1/calendar/terms`
12. `PUT /api/v1/calendar/terms/:term_id`
13. `POST /api/v1/bookings/hold`
14. `POST /api/v1/bookings/:booking_id/confirm`
15. `POST /api/v1/bookings/:booking_id/reschedule`
16. `POST /api/v1/bookings/:booking_id/cancel`
17. `POST /api/v1/bookings/:booking_id/check-in`
18. `POST /api/v1/content/uploads/start`
19. `PUT /api/v1/content/uploads/:upload_id/chunks/:chunk_index`
20. `POST /api/v1/content/uploads/:upload_id/complete`
21. `GET /api/v1/content/documents/:document_id/preview`
22. `GET /api/v1/content/documents/:document_id/download`
23. `POST /api/v1/content/documents/:document_id/share-links`
24. `GET /api/v1/content/documents/:document_id/versions`
25. `GET /api/v1/content/documents/search`
26. `POST /api/v1/content/documents/bulk`
27. `POST /api/v1/content/documents/duplicates/detect`
28. `PATCH /api/v1/content/documents/duplicates/:duplicate_id/merge-flag`
29. `POST /api/v1/content/ingestion/sources`
30. `GET /api/v1/content/ingestion/sources`
31. `POST /api/v1/content/ingestion/proxies`
32. `POST /api/v1/content/ingestion/user-agents`
33. `POST /api/v1/content/ingestion/run-due`
34. `POST /api/v1/content/ingestion/sources/:source_id/run`
35. `GET /api/v1/content/ingestion/runs`
36. `POST /api/v1/content/ingestion/sources/:source_id/manual-review`
37. `POST /api/v1/planning/plans`
38. `GET /api/v1/planning/plans/:plan_id/tree`
39. `POST /api/v1/planning/plans/:plan_id/milestones`
40. `POST /api/v1/planning/milestones/:milestone_id/tasks`
41. `POST /api/v1/planning/tasks/:task_id/dependencies`
42. `DELETE /api/v1/planning/tasks/:task_id/dependencies/:depends_on_task_id`
43. `PATCH /api/v1/planning/plans/:plan_id/reorder-milestones`
44. `PATCH /api/v1/planning/milestones/:milestone_id/reorder-tasks`
45. `PATCH /api/v1/planning/tasks/bulk`
46. `GET /api/v1/dashboard/overview`
47. `GET /api/v1/dashboard/today-sessions`
48. `POST /api/v1/dashboard/refresh`
49. `POST /api/v1/dashboard/feature-store/nightly-batch`
50. `GET /api/v1/dashboard/feature-store/learners`
51. `GET /api/v1/dashboard/feature-store/cohorts`
52. `GET /api/v1/dashboard/feature-store/reporting-metrics`
53. `GET /api/v1/observability/workflow-logs`
54. `POST /api/v1/observability/scraping-errors`
55. `POST /api/v1/observability/anomalies/detect`
56. `GET /api/v1/observability/anomalies`
57. `POST /api/v1/observability/report-schedules`
58. `POST /api/v1/observability/report-schedules/run-due`
59. `POST /api/v1/observability/report-schedules/:schedule_id/run`
60. `GET /api/v1/observability/report-exports`
61. `GET /api/v1/admin/tenants`
62. `POST /api/v1/admin/tenants`
63. `PUT /api/v1/admin/tenants/:tenant_id`
64. `GET /api/v1/admin/permissions/matrix`
65. `PUT /api/v1/admin/permissions/matrix`
66. `GET /api/v1/admin/users/roles`
67. `POST /api/v1/admin/users/:user_id/roles`
68. `DELETE /api/v1/admin/users/:user_id/roles/:role`
69. `GET /api/v1/content/share/:token/download`

## API Test Mapping Table
Primary API test artifact: `repo/API_tests/run_api_tests.sh` (request helper at `repo/API_tests/run_api_tests.sh:9-33`; endpoint sections at `repo/API_tests/run_api_tests.sh:90-662`).

| Endpoint | Covered | Test type | Test files | Evidence |
|---|---|---|---|---|
| `GET /health` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:95` |
| `POST /api/v1/auth/login` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:109,114,118,122,191,194,216,657` |
| `POST /api/v1/auth/logout` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:209` |
| `GET /api/v1/auth/me` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:126,200,212` |
| `POST /api/v1/security/upload/validate` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:404,410,417` |
| `GET /api/v1/calendar/availability/:session_id` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:263` |
| `POST /api/v1/calendar/time-slots` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:269` |
| `PUT /api/v1/calendar/time-slots/:rule_id` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:278` |
| `POST /api/v1/calendar/blackouts` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:283` |
| `PUT /api/v1/calendar/blackouts/:blackout_id` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:289` |
| `POST /api/v1/calendar/terms` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:294` |
| `PUT /api/v1/calendar/terms/:term_id` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:300` |
| `POST /api/v1/bookings/hold` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:309,319,321,323,381` |
| `POST /api/v1/bookings/:booking_id/confirm` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:367,369,659` |
| `POST /api/v1/bookings/:booking_id/reschedule` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:385` |
| `POST /api/v1/bookings/:booking_id/cancel` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:395` |
| `POST /api/v1/bookings/:booking_id/check-in` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:390` |
| `POST /api/v1/content/uploads/start` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:426` |
| `PUT /api/v1/content/uploads/:upload_id/chunks/:chunk_index` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:437` |
| `POST /api/v1/content/uploads/:upload_id/complete` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:442` |
| `GET /api/v1/content/documents/:document_id/preview` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:449` |
| `GET /api/v1/content/documents/:document_id/download` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:457` |
| `POST /api/v1/content/documents/:document_id/share-links` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:470` |
| `GET /api/v1/content/documents/:document_id/versions` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:465` |
| `GET /api/v1/content/documents/search` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:481` |
| `POST /api/v1/content/documents/bulk` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:486` |
| `POST /api/v1/content/documents/duplicates/detect` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:491` |
| `PATCH /api/v1/content/documents/duplicates/:duplicate_id/merge-flag` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:496` |
| `POST /api/v1/content/ingestion/sources` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:505` |
| `GET /api/v1/content/ingestion/sources` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:511` |
| `POST /api/v1/content/ingestion/proxies` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:516` |
| `POST /api/v1/content/ingestion/user-agents` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:521` |
| `POST /api/v1/content/ingestion/run-due` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:531` |
| `POST /api/v1/content/ingestion/sources/:source_id/run` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:526` |
| `GET /api/v1/content/ingestion/runs` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:536` |
| `POST /api/v1/content/ingestion/sources/:source_id/manual-review` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:541,546` |
| `POST /api/v1/planning/plans` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:555` |
| `GET /api/v1/planning/plans/:plan_id/tree` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:561` |
| `POST /api/v1/planning/plans/:plan_id/milestones` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:566` |
| `POST /api/v1/planning/milestones/:milestone_id/tasks` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:572,578` |
| `POST /api/v1/planning/tasks/:task_id/dependencies` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:583` |
| `DELETE /api/v1/planning/tasks/:task_id/dependencies/:depends_on_task_id` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:603` |
| `PATCH /api/v1/planning/plans/:plan_id/reorder-milestones` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:588` |
| `PATCH /api/v1/planning/milestones/:milestone_id/reorder-tasks` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:593` |
| `PATCH /api/v1/planning/tasks/bulk` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:598` |
| `GET /api/v1/dashboard/overview` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:229` |
| `GET /api/v1/dashboard/today-sessions` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:234` |
| `POST /api/v1/dashboard/refresh` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:133,224` |
| `POST /api/v1/dashboard/feature-store/nightly-batch` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:239` |
| `GET /api/v1/dashboard/feature-store/learners` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:244` |
| `GET /api/v1/dashboard/feature-store/cohorts` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:249` |
| `GET /api/v1/dashboard/feature-store/reporting-metrics` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:254` |
| `GET /api/v1/observability/workflow-logs` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:612` |
| `POST /api/v1/observability/scraping-errors` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:617` |
| `POST /api/v1/observability/anomalies/detect` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:622` |
| `GET /api/v1/observability/anomalies` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:627` |
| `POST /api/v1/observability/report-schedules` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:632` |
| `POST /api/v1/observability/report-schedules/run-due` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:643` |
| `POST /api/v1/observability/report-schedules/:schedule_id/run` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:638` |
| `GET /api/v1/observability/report-exports` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:648` |
| `GET /api/v1/admin/tenants` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:101,137,145` |
| `POST /api/v1/admin/tenants` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:150` |
| `PUT /api/v1/admin/tenants/:tenant_id` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | calls at `repo/API_tests/run_api_tests.sh:155,160` |
| `GET /api/v1/admin/permissions/matrix` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:164` |
| `PUT /api/v1/admin/permissions/matrix` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:168` |
| `GET /api/v1/admin/users/roles` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:173` |
| `POST /api/v1/admin/users/:user_id/roles` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:177` |
| `DELETE /api/v1/admin/users/:user_id/roles/:role` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:181` |
| `GET /api/v1/content/share/:token/download` | yes | true no-mock HTTP | `repo/API_tests/run_api_tests.sh` | call at `repo/API_tests/run_api_tests.sh:476` |

## API Test Classification
1. **True No-Mock HTTP**
   - `repo/API_tests/run_api_tests.sh` using `curl` request path (`repo/API_tests/run_api_tests.sh:9-33`) against `$ROOT`/`$BASE` URLs.
   - Executed from Docker test container against API container in `repo/run_tests.sh:30-31` after stack startup `repo/run_tests.sh:15-21`.

2. **HTTP with Mocking**
   - Frontend test suite stubs transport via `vi.stubGlobal("fetch", ...)`:
     - `repo/frontend/src/test/admin.access-and-flows.test.tsx:53,156`
     - `repo/frontend/src/test/state.isolation.test.tsx:83`
     - `repo/frontend/src/test/dashboard.feature-store.test.tsx:79`
     - `repo/frontend/src/test/planning.drag-drop.test.tsx:72`
     - `repo/frontend/src/test/booking.race-and-error.test.tsx:50`
     - `repo/frontend/src/test/critical-flow.e2e.test.tsx:90`
     - `repo/frontend/src/test/app.route-guard.test.tsx:13`
     - `repo/frontend/src/test/app.auth-role.test.tsx:63`

3. **Non-HTTP (unit/integration without HTTP)**
   - `repo/backend/internal/auth/cookie_test.go`
   - `repo/backend/internal/booking/service_test.go`
   - `repo/backend/internal/content/ingestion_fallback_test.go`
   - `repo/backend/internal/content/ingestion_service_test.go`
   - `repo/backend/internal/content/storage_test.go`
   - `repo/backend/internal/dashboard/service_test.go`
   - `repo/backend/internal/security/password_test.go`
   - `repo/backend/internal/security/logging_test.go` (synthetic echo route at `repo/backend/internal/security/logging_test.go:38`, not production router wiring)

## Mock Detection
- Transport mocking in frontend tests: `vi.stubGlobal("fetch", ...)` at locations listed above.
- Synthetic local route (bypasses production router composition): `repo/backend/internal/security/logging_test.go:38`.
- No `jest.mock`, `vi.mock`, or `sinon.stub` found in backend API test path (`repo/API_tests/run_api_tests.sh`).

## Coverage Summary
- Total endpoints: **69** (`repo/backend/cmd/server/main.go:98-406`).
- Endpoints with HTTP tests: **69**.
- Endpoints with TRUE no-mock HTTP tests: **69**.
- HTTP coverage %: **100.00%**.
- True API coverage %: **100.00%**.

## Unit Test Summary

### Backend Unit Tests
- Test files:
  - `repo/backend/internal/security/logging_test.go`
  - `repo/backend/internal/content/storage_test.go`
  - `repo/backend/internal/content/ingestion_fallback_test.go`
  - `repo/backend/internal/booking/service_test.go`
  - `repo/backend/internal/security/password_test.go`
  - `repo/backend/internal/dashboard/service_test.go`
  - `repo/backend/internal/content/ingestion_service_test.go`
  - `repo/backend/internal/auth/cookie_test.go`
- Modules covered:
  - Controllers: none directly (no handler test files in `backend/internal/*/*_test.go` for handler packages).
  - Services: partial helper/service coverage (`booking`, `dashboard`, `content/ingestion`).
  - Repositories: no direct repository unit tests detected.
  - Auth/guards/middleware: partial (`auth/cookie`, `security/logging`); `access` middleware lacks direct test file.
- Important backend modules not tested directly (unit level):
  - Handler packages such as `repo/backend/internal/admin/handler.go`, `repo/backend/internal/booking/handler.go`, `repo/backend/internal/calendar/handler.go`, `repo/backend/internal/content/handler.go`, `repo/backend/internal/planning/handler.go`, `repo/backend/internal/observability/handler.go`.
  - Repository packages such as `repo/backend/internal/admin/repository.go`, `repo/backend/internal/booking/repository.go`, `repo/backend/internal/calendar/repository.go`, `repo/backend/internal/content/repository.go`, `repo/backend/internal/observability/repository.go`.

### Frontend Unit Tests (STRICT REQUIREMENT)
- Frontend test files:
  - `repo/frontend/src/test/admin.access-and-flows.test.tsx`
  - `repo/frontend/src/test/state.isolation.test.tsx`
  - `repo/frontend/src/test/content.upload-edge.test.tsx`
  - `repo/frontend/src/test/dashboard.feature-store.test.tsx`
  - `repo/frontend/src/test/planning.drag-drop.test.tsx`
  - `repo/frontend/src/test/booking.race-and-error.test.tsx`
  - `repo/frontend/src/test/critical-flow.e2e.test.tsx`
  - `repo/frontend/src/test/app.route-guard.test.tsx`
  - `repo/frontend/src/test/app.auth-role.test.tsx`
- Frameworks/tools detected:
  - Vitest (`repo/frontend/package.json:10,28`)
  - React Testing Library (`repo/frontend/src/test/admin.access-and-flows.test.tsx:1` and peers)
  - jsdom environment (`repo/frontend/vite.config.ts:19-23`)
- Components/modules covered (direct import evidence):
  - `repo/frontend/src/App.tsx` (`repo/frontend/src/test/admin.access-and-flows.test.tsx:5`)
  - `repo/frontend/src/features/admin/AdminPage.tsx` (`repo/frontend/src/test/admin.access-and-flows.test.tsx:6`)
  - `repo/frontend/src/features/content/ContentLibraryPage.tsx` (`repo/frontend/src/test/content.upload-edge.test.tsx:4`)
  - `repo/frontend/src/features/dashboard/DashboardPage.tsx` (`repo/frontend/src/test/dashboard.feature-store.test.tsx:4`)
  - `repo/frontend/src/features/planning/PlanningPage.tsx` (`repo/frontend/src/test/planning.drag-drop.test.tsx:4`)
  - `repo/frontend/src/features/booking/BookingFlowPage.tsx` (`repo/frontend/src/test/booking.race-and-error.test.tsx:4`)
- Important frontend components/modules not tested directly:
  - `repo/frontend/src/features/calendar/CalendarPage.tsx` (no test import match found)

**Frontend unit tests: PRESENT**

### Cross-Layer Observation
- Both layers have meaningful tests.
- Backend API real-HTTP coverage is strong (100%), while frontend tests remain mostly mock-transport component tests.
- Real browser-driven FE<->BE E2E is not evidenced; however strong backend API integration coverage partially compensates.

## API Observability Check
- Endpoint + method visibility: strong (`request METHOD URL` and direct `curl` calls in `repo/API_tests/run_api_tests.sh`).
- Request input visibility: strong (JSON payloads and multipart forms shown, e.g., `repo/API_tests/run_api_tests.sh:269,410,555`).
- Response content visibility: present and improved via `assert_body_contains` (`repo/API_tests/run_api_tests.sh:60-69`) across sections.
- Verdict: **strong** (not weak).

## Tests Check
- Success paths: covered across all endpoint families (`repo/API_tests/run_api_tests.sh:90-662`).
- Failure paths: covered (401/403/404/409/423 and validation errors, e.g., `repo/API_tests/run_api_tests.sh:101-103,133-139,395-399,496-499`).
- Edge cases: covered (concurrency, lockout, session rotation/invalidation, tenant isolation: `repo/API_tests/run_api_tests.sh:189-217,313-379,656-660`).
- Validation/auth/permissions: covered on multiple protected routes and role constraints.
- Assertions: mostly meaningful, with status + key body fields.
- `run_tests.sh` Docker rule: Docker-based orchestration present (`repo/run_tests.sh:14-31`) -> **OK**.

## Test Coverage Score (0-100)
**97 / 100**

## Score Rationale
- 69/69 endpoints statically mapped to true no-mock HTTP tests.
- Broad scenario depth exists (happy/failure/edge/authz/concurrency).
- Minor deduction: no static evidence of full browser FE<->BE real E2E; frontend side is still mock-transport heavy.

## Key Gaps
- Missing direct unit-level tests for many repository/handler internals.
- `CalendarPage` frontend component lacks direct test coverage.
- FE<->BE true end-to-end browser path not evidenced.

## Confidence & Assumptions
- Confidence: **high** on endpoint coverage mapping (direct line-level matching).
- Assumptions:
  - Static inspection only; runtime pass/fail not executed.
  - API test script is intended to run via `repo/run_tests.sh` Docker flow.

**Test Coverage Audit Verdict: PASS**

---

# README Audit

## High Priority Issues
- None found under current strict hard-gate policy.

## Medium Priority Issues
- No major compliance issue; wording is concise and operational.

## Low Priority Issues
- Optional enhancement only: include a short architecture diagram reference (not required by gates).

## Hard Gate Failures
- None.

Gate evidence:
- README location exists: `repo/README.md`.
- Project type declared at top: `repo/README.md:3`.
- Required startup command present exactly: `docker-compose up` at `repo/README.md:12`.
- Access method includes URLs + ports: `repo/README.md:26-28`.
- Verification method includes API (`curl`) and web flow: `repo/README.md:49-81`.
- Environment rule (Docker-contained) satisfied: `repo/README.md:22` and no host install instructions.
- Demo credentials with roles provided: role table at `repo/README.md:34-41`.

## README Verdict (PASS / PARTIAL PASS / FAIL)
**PASS**

---

## Final Verdicts
- Test Coverage Audit: **PASS**
- README Audit: **PASS**
