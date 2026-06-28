---
phase: 03-fanout-and-safety
plan: "01"
subsystem: federation
tags: [activitypub, go, pocketbase, fanout, instance-actor]

requires:
  - phase: 01-instance-actor
    provides: instance actor record in activitypub_actors, followerInboxes JOIN helper in activity.go
  - phase: 02-follow-lifecycle
    provides: follows collection with status field (accepted/pending), ProcessFollowActivity idempotency pattern

provides:
  - instanceFollowerInboxes(app core.App) ([]string, error) helper in db/federation/activity.go
  - Unit tests proving startup-safety, accepted-follower filtering, and pending-follower exclusion

affects:
  - 03-fanout-and-safety/03-02 (create.go fanout injection uses instanceFollowerInboxes)
  - 03-fanout-and-safety/03-03 (delete.go fanout injection uses instanceFollowerInboxes)

tech-stack:
  added: []
  patterns:
    - "startup-safe not-found guard: errors.Is(err, sql.ErrNoRows) returns (nil, nil) instead of error"
    - "delegate pattern: instanceFollowerInboxes wraps followerInboxes with IRI lookup"

key-files:
  created:
    - db/federation/activity_test.go
  modified:
    - db/federation/activity.go

key-decisions:
  - "instanceFollowerInboxes returns (nil, nil) on sql.ErrNoRows — startup-safe per D-02"
  - "Delegates accepted-follower resolution to existing followerInboxes JOIN — no duplicated SQL"
  - "No deduplication against user-level inboxes — PostActivity slices.Sort + slices.Compact handles it"

patterns-established:
  - "startup-safe guard: errors.Is(err, sql.ErrNoRows) -> return nil, nil in instanceFollowerInboxes"
  - "test isolation: reuse newInboxTestApp and createTestActor; no symbol redefinition across test files"

requirements-completed: [SYNC-01, SYNC-02, SYNC-03]

duration: 2min
completed: "2026-06-26"
---

# Phase 3, Plan 01: instanceFollowerInboxes Helper Summary

**instanceFollowerInboxes helper added to activity.go: resolves instance actor by ORIGIN-derived IRI, delegates to followerInboxes JOIN, returns (nil, nil) when actor not yet seeded**

## Performance

- **Duration:** 2 min
- **Started:** 2026-06-26T08:57:08Z
- **Completed:** 2026-06-26T08:59:20Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added `instanceFollowerInboxes(app core.App) ([]string, error)` to `db/federation/activity.go` after the existing `followerInboxes` function
- Extended activity.go imports with `"database/sql"` and `"errors"` (no existing imports disturbed)
- Created `db/federation/activity_test.go` with 3 passing tests covering accepted-followers return, no-instance-actor startup-safety, and pending-follower exclusion
- All 3 tests pass; full federation test suite (9 tests) remains green

## Task Commits

Each task was committed atomically:

1. **Task 1: Add instanceFollowerInboxes helper to activity.go** - `78c3a074` (feat)
2. **Task 2: Unit-test instanceFollowerInboxes** - `6c4f869c` (test)

## Files Created/Modified
- `db/federation/activity.go` - Added `instanceFollowerInboxes` function + `database/sql` and `errors` imports
- `db/federation/activity_test.go` - New test file with `TestInstanceFollowerInboxes_ReturnsAcceptedFollowers`, `TestInstanceFollowerInboxes_NoInstanceActor`, `TestInstanceFollowerInboxes_OnlyAcceptedFollowers`

## Decisions Made
- Followed D-01/D-02 exactly: `instanceFollowerInboxes` delegates to `followerInboxes(app, instanceActor.Id)` and returns `(nil, nil)` on `sql.ErrNoRows`
- No deduplication against user-level inboxes — `PostActivity` already applies `slices.Sort` + `slices.Compact` (activity.go lines 101-102)
- Test helper `newFollowRecord` was not needed — used `core.NewRecord(followsCol)` directly (same pattern as production code in follow.go)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `instanceFollowerInboxes` is ready for consumption by Plan 02 (create.go fanout injection) and Plan 03 (delete.go fanout injection)
- No blockers

## Self-Check

Files exist:
- `db/federation/activity.go` — FOUND (modified with instanceFollowerInboxes)
- `db/federation/activity_test.go` — FOUND (created with 3 tests)

Commits exist:
- `78c3a074` — feat(03-01): add instanceFollowerInboxes helper to activity.go — FOUND
- `6c4f869c` — test(03-01): add unit tests for instanceFollowerInboxes — FOUND

Verification:
- `go build ./federation/` — PASS
- `go test ./federation/ -run TestInstanceFollowerInboxes -count=1` — PASS (3/3)
- `grep -n "func instanceFollowerInboxes" db/federation/activity.go` — 1 match at line 60

## Self-Check: PASSED

---
*Phase: 03-fanout-and-safety*
*Completed: 2026-06-26*
