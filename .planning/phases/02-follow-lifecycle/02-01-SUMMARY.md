---
phase: 02-follow-lifecycle
plan: 01
subsystem: database
tags: [pocketbase, migration, activitypub, federation, follows]

# Dependency graph
requires: []
provides:
  - follows.status select field accepts "rejected" as a valid value
  - PocketBase migration 1782290001 that appends "rejected" to follows.status
affects: [02-follow-lifecycle plan 03 — Reject{Follow} delivery path]

# Tech tracking
tech-stack:
  added: []
  patterns: [idempotent PocketBase select-field migration with guard against double-apply]

key-files:
  created:
    - db/migrations/1782290001_add_rejected_to_follows_status.go
  modified: []

key-decisions:
  - "Idempotent up-function: iterates existing values and skips append if 'rejected' already present, preventing double-apply on re-run"
  - "Down-function is tolerant: returns nil early if collection, field, or type assertion fails, so rollback never aborts startup"
  - "Used FindCollectionByNameOrId with the stable collection id '8obn1ukumze565i' rather than the name 'follows' to match the snapshot migration pattern"

patterns-established:
  - "PocketBase select-field migration pattern: FindCollectionByNameOrId → GetByName → type-assert to *core.SelectField → mutate Values → app.Save"

requirements-completed: [FLCL-04]

# Metrics
duration: 5min
completed: 2026-06-25
---

# Phase 02 Plan 01: Add "rejected" to follows.status — Summary

**PocketBase migration 1782290001 appends "rejected" to follows.status select field (collection 8obn1ukumze565i), enabling the Reject{Follow} lifecycle path without breaking existing pending/accepted values**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-06-25T00:00:00Z
- **Completed:** 2026-06-25T00:05:00Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Created migration `1782290001_add_rejected_to_follows_status.go` that extends the `follows.status` select field with the value `"rejected"`
- Implemented idempotent up-function that guards against double-apply by checking existing values before appending
- Implemented tolerant down-function that removes `"rejected"` without failing if the field or collection is missing

## Task Commits

Each task was committed atomically:

1. **Task 1: Create migration adding "rejected" to follows.status** - `055d1c19` (feat)

**Plan metadata:** (see below)

## Files Created/Modified
- `db/migrations/1782290001_add_rejected_to_follows_status.go` — New PocketBase migration that appends "rejected" to follows.status allowed values; idempotent up and tolerant down

## Decisions Made
- Used the stable collection id `8obn1ukumze565i` (not the name `follows`) to match the snapshot migration pattern and avoid name-change fragility
- Idempotent guard in up-function: iterates existing Values and returns nil if "rejected" is already present — prevents duplicate values on repeated migration runs
- Down-function returns nil on any missing field rather than returning an error — rollback must be tolerant since the field state on rollback is uncertain

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `go build ./...` and `go vet ./migrations/` both passed on first attempt.

## User Setup Required

None - no external service configuration required. The migration runs automatically on next PocketBase startup.

## Next Phase Readiness
- Schema prerequisite for FLCL-04 is satisfied: `follows.status` now accepts `"rejected"`
- Plan 02 (Accept{Follow} delivery) and Plan 03 (Reject{Follow} delivery) can proceed; both depend on this migration being present

---
*Phase: 02-follow-lifecycle*
*Completed: 2026-06-25*
