# TrainingOps Frontend Architecture (React)

This structure is intentionally architecture-first and UI-light.

## Goals

- Role-aware rendering for: `administrator`, `program_coordinator`, `instructor`, `learner`
- Centralized API access with auth/session handling
- Predictable state strategy optimized for server data
- Feature modules for core screens:
  - Dashboard
  - Calendar
  - Booking flow
  - Content library
  - Tasks/Planning

## Recommended Stack

- React + TypeScript
- React Router (route config + guards)
- TanStack Query for server state
- Zustand for small client/UI state

## Folder Layout

```
frontend/
  src/
    app/
      route-config.tsx         # Route definitions + role requirements
      navigation.ts            # Left-nav/top-nav config by role
    auth/
      roles.ts                 # Role enum and helpers
      access-control.tsx       # Role gate and route guard helpers
    api/
      http-client.ts           # Central API client wrapper
      endpoints.ts             # Typed endpoint helpers per domain
    state/
      session-store.ts         # Auth/session and current user
      ui-store.ts              # Small UI state (filters, panel state)
    features/
      dashboard/
        DashboardPage.tsx
      calendar/
        CalendarPage.tsx
      booking/
        BookingFlowPage.tsx
      content/
        ContentLibraryPage.tsx
      planning/
        PlanningPage.tsx
```

## Role-Based Rendering Strategy

- Route-level protection via route metadata (`allowedRoles`).
- Component-level gating using `AccessGate` for conditional controls.
- Keep role logic centralized in `auth/access-control.tsx`; avoid scattered role checks.

## API Strategy

- `api/http-client.ts` is the single transport layer.
- Consistent request/response/error normalization.
- Session-aware requests (`credentials: include`) for cookie-based auth.
- Domain functions in `api/endpoints.ts`; components never call `fetch` directly.

## State Strategy

- TanStack Query:
  - All server data (dashboard metrics, bookings, content lists, planning trees)
  - Caching and refetch windows tuned per screen
- Zustand:
  - Local-only state (active filters, selected tabs, draft form state)
- Keep role/user session in one store (`session-store.ts`) to simplify guards.

## Screen Ownership (Module Boundaries)

- `dashboard`: KPI/overview and feature-store read views
- `calendar`: availability and slot views
- `booking`: hold/confirm/reschedule/cancel/check-in flows
- `content`: library search, versions, share links, ingestion ops (admin/coordinator)
- `planning`: plans, milestones, tasks, dependency visualization

## Non-Goals For This Step

- No full visual UI implementation
- No design system implementation
- No detailed form components
