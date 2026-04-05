# TrainingOps Scheduling & Content Intelligence - API Specification

## 1. Overview

- Base URL: `/api/v1`
- Protocol: HTTPS (or HTTP for local offline deployment)
- API style: JSON REST
- Backend: Go + Echo
- Database: PostgreSQL

Primary domains:

- Authentication and session management
- Tenant and role-aware authorization
- Booking and calendar scheduling
- Content library and document management
- Planning (milestones/tasks)
- Dashboard analytics and KPIs
- Observability and report exports
- Controlled ingestion from local partner portals

## 2. Authentication and Authorization

### 2.1 Auth method

- Local username/password only
- Password minimum length: 12
- Session cookies are `HttpOnly`, `Secure`, and rotated on sensitive operations
- Lockout policy: 5 failed attempts -> account locked for 15 minutes

### 2.2 Roles

- `administrator`
- `program_coordinator`
- `instructor`
- `learner`

### 2.3 Tenant isolation

Every request is resolved in a tenant context. Cross-tenant access must return `403`.

## 3. API Conventions

### 3.1 Headers

- `Content-Type: application/json`
- `X-Request-Id: <uuid>` (optional, echoed back)
- `If-Match: <version-token>` for optimistic concurrency on mutable resources

### 3.2 Standard response envelope

```json
{
	"data": {},
	"error": null,
	"meta": {
		"request_id": "f5f6b98f-5cc2-4f4f-9f1d-1dd3ff7f6ef0"
	}
}
```

Error form:

```json
{
	"data": null,
	"error": {
		"code": "BOOKING_CONFLICT",
		"message": "Selected room is already occupied.",
		"details": {
			"reason": "room_occupied"
		}
	},
	"meta": {
		"request_id": "f5f6b98f-5cc2-4f4f-9f1d-1dd3ff7f6ef0"
	}
}
```

### 3.3 Pagination

- Query params: `page` (1-based), `page_size` (max 100)
- Meta includes `total`, `page`, `page_size`

### 3.4 Time and IDs

- Time format: ISO-8601 UTC (`2026-04-02T12:30:00Z`)
- Primary resource IDs: UUID

## 4. Core Domain Models

### 4.1 Booking

```json
{
	"id": "uuid",
	"tenant_id": "uuid",
	"learner_id": "uuid",
	"session_id": "uuid",
	"room_id": "uuid",
	"status": "held|confirmed|rescheduled|canceled|checked_in",
	"hold_expires_at": "2026-04-02T10:05:00Z",
	"reschedule_count": 0,
	"created_at": "...",
	"updated_at": "...",
	"version": 1
}
```

### 4.2 Calendar rule

```json
{
	"id": "uuid",
	"type": "period|blackout",
	"name": "Morning Block A",
	"day_of_week": "monday",
	"start_time": "08:30",
	"end_time": "10:00",
	"date": null,
	"version": 3
}
```

### 4.3 Content item

```json
{
	"id": "uuid",
	"title": "Intro to Data Literacy",
	"difficulty": 3,
	"duration_minutes": 90,
	"categories": ["Data", "Fundamentals"],
	"tags": ["beginner", "worksheet"],
	"metadata": {
		"language": "en"
	}
}
```

Validation constraints:

- `difficulty`: integer 1..5
- `duration_minutes`: integer 5..480

## 5. Endpoints

## 5.1 Health

### `GET /health`

Returns system health and build info.

## 5.2 Authentication

### `POST /auth/login`

Request:

```json
{
	"username": "user@example.local",
	"password": "min12characters"
}
```

Response:

```json
{
	"data": {
		"user": {
			"id": "uuid",
			"role": "program_coordinator",
			"tenant_id": "uuid"
		},
		"session": {
			"expires_at": "2026-04-03T10:00:00Z"
		}
	},
	"error": null,
	"meta": {}
}
```

### `POST /auth/logout`

Invalidates session.

### `GET /auth/me`

Returns current session user and effective permissions.

## 5.3 Tenant and access

### `GET /access/permissions`

Returns resolved role and permission set for current user.

## 5.4 Calendar and availability

### `GET /calendar/availability`

Query:

- `start`, `end`
- Optional: `room_id`, `instructor_id`

Response includes occupied slots, blackout dates, and free ranges.

### `POST /calendar/rules`

Creates period or blackout rules. Requires `program_coordinator` or `administrator`.

### `PUT /calendar/rules/{rule_id}`

Requires `If-Match` header with latest version token.

### `DELETE /calendar/rules/{rule_id}`

Soft delete rule with audit trail.

## 5.5 Booking

### `POST /bookings/hold`

Creates temporary hold (5 minutes).

Request:

```json
{
	"session_id": "uuid",
	"room_id": "uuid",
	"learner_id": "uuid"
}
```

Success response:

```json
{
	"data": {
		"booking_id": "uuid",
		"status": "held",
		"hold_expires_at": "2026-04-02T10:05:00Z"
	},
	"error": null,
	"meta": {}
}
```

Conflict response (`409`) includes reasons and alternatives:

```json
{
	"data": null,
	"error": {
		"code": "BOOKING_CONFLICT",
		"message": "Capacity reached",
		"details": {
			"reason": "capacity_reached",
			"alternatives": [
				{ "session_id": "uuid-1", "room_id": "uuid-a" },
				{ "session_id": "uuid-2", "room_id": "uuid-b" },
				{ "session_id": "uuid-3", "room_id": "uuid-c" }
			]
		}
	},
	"meta": {}
}
```

### `POST /bookings/{booking_id}/confirm`

Confirms a valid held booking.

### `POST /bookings/{booking_id}/reschedule`

Business rule: maximum 2 reschedules.

### `POST /bookings/{booking_id}/cancel`

Business rule: cancellation cutoff is 24 hours before start time.

### `POST /bookings/{booking_id}/check-in`

Marks attendance state as checked-in.

### `GET /bookings`

Filters:

- `status`
- `learner_id`
- `session_id`
- `start`, `end`

## 5.6 Content and documents

### `GET /content/items`

Search by title/tags/category; supports full-text query `q`.

### `POST /content/items`

Creates metadata-validated content item.

### `PUT /content/items/{item_id}`

Updates item metadata and taxonomy links.

### `POST /content/items/duplicates/scan`

Triggers duplicate detection for merge queue.

### `POST /content/items/duplicates/merge`

Merges selected duplicates into a canonical item.

### `POST /content/uploads/init`

Initializes resumable chunked upload.

### `PUT /content/uploads/{upload_id}/chunk`

Uploads a chunk with index/checksum.

### `POST /content/uploads/{upload_id}/complete`

Verifies checksum and persists file pointer.

### `GET /content/files/{file_id}/versions`

Returns version history.

### `GET /content/files/{file_id}/preview`

Inline preview for PDF/images/text.

### `POST /content/files/{file_id}/share-links`

Generates expiring share link. TTL fixed to 72 hours.

### `GET /content/files/{file_id}/download`

Returns watermarked file stream for authorized users.

## 5.7 Planning

### `GET /planning/milestones`

Lists milestones with progress metrics.

### `POST /planning/milestones`

Creates milestone.

### `GET /planning/tasks`

Lists tasks with filters: `milestone_id`, `assignee_id`, `status`, `due_before`.

### `POST /planning/tasks`

Creates task with dependency references.

### `PUT /planning/tasks/{task_id}`

Updates due date, estimate, actual effort, dependencies.

### `POST /planning/tasks/bulk-update`

Bulk status or date changes.

### `POST /planning/tasks/reorder`

Optional drag-and-drop order persistence.

## 5.8 Dashboard

### `GET /dashboard/summary`

Returns:

- Today sessions
- Occupancy heatmap data
- Pending approvals
- KPI tiles: enrollment growth, repeat attendance, study time logged, content conversion, community activity

## 5.9 Observability and reports

### `GET /observability/events`

Returns workflow/anomaly events.

### `POST /observability/reports/export`

Request:

```json
{
	"format": "csv|pdf",
	"report_type": "bookings|content|ingestion|kpi",
	"from": "2026-03-01T00:00:00Z",
	"to": "2026-03-31T23:59:59Z"
}
```

Response includes generated local file path.

## 5.10 Content ingestion

### `POST /ingestion/jobs`

Creates ingestion job with source profile and schedule constraints.

### `GET /ingestion/jobs/{job_id}`

Returns run status and bot-check/captcha flags.

### `POST /ingestion/jobs/{job_id}/pause`

Manual safe fallback trigger.

### `POST /ingestion/jobs/{job_id}/resume`

Resumes paused ingestion.

### `GET /ingestion/runs`

Lists historical runs with normalization outputs and error counts.

## 6. Booking State Machine and Audit

Allowed transitions:

- `held -> confirmed`
- `held -> canceled` (manual or timeout cleanup)
- `confirmed -> rescheduled`
- `confirmed -> canceled` (if within policy)
- `confirmed -> checked_in`
- `rescheduled -> checked_in`
- `rescheduled -> canceled`

Every transition writes immutable audit records with:

- actor
- timestamp
- previous/new state
- reason
- correlation/request ID

## 7. Error Codes

- `AUTH_INVALID_CREDENTIALS` (`401`)
- `AUTH_LOCKED` (`423`)
- `ACCESS_FORBIDDEN` (`403`)
- `VALIDATION_ERROR` (`400`)
- `BOOKING_CONFLICT` (`409`)
- `BOOKING_HOLD_EXPIRED` (`409`)
- `RESCHEDULE_LIMIT_REACHED` (`409`)
- `CANCELLATION_WINDOW_CLOSED` (`409`)
- `CONCURRENCY_VERSION_MISMATCH` (`412`)
- `FILE_CHECKSUM_MISMATCH` (`422`)
- `FILE_FORMAT_NOT_ALLOWED` (`415`)
- `INGESTION_BOT_CHECK_DETECTED` (`202` with paused/manual action)
- `INTERNAL_ERROR` (`500`)

## 8. RBAC Matrix (Summary)

- Administrator: full access, tenant setup, policy management, permissions
- Program Coordinator: calendar, bookings, planning, dashboard operational view
- Instructor: materials, attendance notes, content versions, assigned schedules
- Learner: browse catalog, reserve sessions, download approved files

## 9. Non-functional Contract Notes

- Strong consistency for booking creation via transactional availability checks
- Row-level locking and unique constraints prevent double booking/oversell
- Optimistic concurrency required for mutable scheduling and dependency resources
- Sensitive fields encrypted at rest; logs must mask personal identifiers
