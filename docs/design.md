# TrainingOps Scheduling & Content Intelligence - System Design

## 1. Purpose

Design an offline-first, tenant-aware web application for training providers to manage scheduling, bookings, content operations, planning, and analytics from one platform.

Primary goals:

- Prevent booking conflicts with strong consistency
- Provide role-specific workflows for Admin, Coordinator, Instructor, Learner
- Support structured content lifecycle (upload, versioning, tagging, search)
- Keep observability and reporting fully local/offline
- Enable controlled partner data ingestion with safe anti-bot handling

## 2. High-Level Architecture

## 2.1 Components

- Frontend: React + TypeScript SPA
- Backend: Go + Echo REST API
- Database: PostgreSQL
- File storage: Local disk with DB pointers
- Background workers: in-process scheduled jobs (cleanup, nightly features, report generation)

## 2.2 Bounded contexts

- Auth/Access
- Calendar and Booking
- Content Library + Document Lifecycle
- Planning (Milestones/Tasks)
- Dashboard/Analytics
- Observability/Reporting
- Partner Ingestion + Feature Store

## 3. Frontend Design

## 3.1 App structure

- App shell with role-aware navigation and protected routes
- Feature modules under `src/features/*`
- Shared API client and endpoint mappings under `src/api`
- Session and UI state stores under `src/state`

## 3.2 Role-driven UX

- Administrator: tenant policy, permissions, system controls
- Program Coordinator: calendar, booking conflicts/alternatives, planning board
- Instructor: content management, attendance notes, version review
- Learner: session discovery, booking checkout, approved downloads

## 3.3 Key interaction flows

- Dashboard landing with daily operations and KPI tiles
- Booking flow with hold countdown and explicit conflict reasons
- Content workflow with chunked upload and preview
- Planning board with dependencies and optional drag-drop ordering

## 4. Backend Design

## 4.1 API layer

- Echo handlers per domain (auth, booking, calendar, content, planning, dashboard, observability, ingestion)
- Middleware:
	- session authentication
	- tenant context resolution
	- RBAC authorization
	- request logging and trace IDs

## 4.2 Service layer

- Encapsulates business rules and transaction boundaries
- Performs validation, policy checks, state transitions, and audit recording

## 4.3 Repository layer

- PostgreSQL access via typed repository interfaces
- Transaction support for booking-critical operations
- Row-level lock usage for availability/seat mutations

## 4.4 Background processing

Scheduled jobs:

- Expired hold cleanup (auto-release)
- Nightly feature-store aggregations (7/30/90-day windows)
- Report generation/export jobs
- Ingestion schedule executor with randomized windows

## 5. Data Design

## 5.1 Core entities

- tenants
- users
- roles / permissions
- sessions (auth sessions)
- calendar_rules (periods/blackouts)
- classes, rooms, instructors
- bookings
- booking_state_transitions (immutable audit log)
- content_items
- categories / tags / item links
- files / file_versions / share_links
- milestones / tasks / task_dependencies
- dashboard_aggregates
- observability_events
- ingestion_jobs / ingestion_runs
- feature_store_profiles

## 5.2 Critical constraints

- Unique constraints preventing duplicate confirmed bookings for same seat/room/timeslot
- Check constraints:
	- `difficulty BETWEEN 1 AND 5`
	- `duration_minutes BETWEEN 5 AND 480`
- Foreign keys for tenant ownership and dependency consistency

## 5.3 Audit and compliance

Track who/when/why for booking state changes, permission-sensitive actions, and operational overrides.

## 6. Booking Consistency and Concurrency

## 6.1 Transaction strategy

Booking creation flow uses one DB transaction:

1. Read candidate inventory rows with row-level lock.
2. Re-evaluate availability under lock.
3. Insert hold/booking row.
4. Commit atomically.

## 6.2 Optimistic concurrency

- Calendar rule and task dependency updates require version token (`If-Match`).
- On mismatch, API returns `412 CONCURRENCY_VERSION_MISMATCH`.

## 6.3 Hold expiration policy

- Booking hold duration: 5 minutes
- Expired holds are transitioned to canceled by cleanup job
- Released inventory becomes immediately available for new requests

## 7. Security Design

## 7.1 Authentication and sessions

- Local credentials only
- Password policy min 12 chars
- Lockout after 5 failed attempts for 15 minutes
- Secure, HttpOnly cookies with session rotation

## 7.2 Authorization

- RBAC enforced in middleware and domain checks
- Tenant isolation at query and service boundary

## 7.3 Data protection

- Sensitive identifiers encrypted at rest
- PII masking in logs
- File checksum + allowlist format checks before finalization

## 8. Content and Document Lifecycle

## 8.1 Upload and integrity

- Resumable chunked upload protocol
- Per-chunk tracking and final checksum validation
- Failed integrity verification blocks publish

## 8.2 Versioning and retrieval

- Immutable file versions
- Inline preview for PDF/image/text
- Watermarked downloads for authorized contexts
- Share links with 72-hour expiration

## 8.3 Taxonomy and quality

- Multi-level categories and tags
- Bulk duplicate scan and merge queue
- Full-text search over title, metadata, tags, and extracted text

## 9. Planning and Operations

Planning supports:

- Milestones and tasks
- Dependencies and due dates
- Estimated vs actual effort
- Bulk edits and optional ordering

Operational dashboard aggregates:

- Today sessions
- Occupancy heatmap
- Pending approvals
- KPI tiles (enrollment growth, repeat attendance, study time, content conversion, community activity)

## 10. Ingestion and Feature Store

## 10.1 Controlled ingestion

- Local proxy pools and dynamic user agents
- Cookie/token session persistence
- Rate limiting and randomized scheduling
- CAPTCHA/bot detection triggers pause and manual review state

## 10.2 Normalization

- Incoming partner content normalized to canonical categories and metadata schema
- Validation and dedupe before publishing to library

## 10.3 Feature computation

- Nightly batch jobs compute learner/cohort profiles
- Time windows: 7, 30, 90 days
- Outputs persisted for segmentation and reporting

## 11. Observability and Offline Reporting

- Structured workflow logs to local files/DB
- Anomaly detection rules for booking failure spikes and ingestion errors
- Scheduled CSV/PDF report exports to admin-configured local directory

## 12. Deployment and Runtime

- Dockerized frontend, backend, and test runner
- Single-node offline-friendly deployment model
- Environment-driven configuration for paths, cookie settings, and report directory

## 13. Risks and Mitigations

- Risk: race conditions in booking under load
	- Mitigation: row-level locks + unique constraints + transactional checks
- Risk: stale client edits on schedule/task resources
	- Mitigation: optimistic concurrency tokens
- Risk: ingestion blocks from anti-bot challenges
	- Mitigation: safe pause/manual review fallback and compliance throttling
- Risk: local disk saturation from uploads/reports
	- Mitigation: retention policy and storage monitoring alerts

## 14. Acceptance Criteria Traceability

- Conflict-aware booking with alternatives: covered by booking service + availability API + conflict reason mapping
- 5-minute hold and auto-release: covered by hold lifecycle and cleanup worker
- 24-hour cancellation cutoff and max 2 reschedules: enforced in booking domain policy
- Content metadata validations and duplicate maintenance: enforced in content service and DB checks
- Tenant/RBAC enforcement and auditability: enforced in middleware + transition logs
- Offline observability/reporting and ingestion safeguards: covered by observability + ingestion modules
