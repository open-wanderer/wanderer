---
phase: 04-nodeinfo
plan: 01
subsystem: api
tags: [nodeinfo, activitypub, federation, pocketbase, go]

# Dependency graph
requires:
  - phase: 03-fanout-and-safety
    provides: ActivityPub fanout infrastructure and route registration patterns
provides:
  - NodeInfo 2.1 well-known endpoint (GET /.well-known/nodeinfo/2.1)
  - NodeInfo JRD discovery endpoint (GET /.well-known/nodeinfo)
  - Pure builder functions testable without HTTP context
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Pure builder functions (no *core.RequestEvent) enable unit testing without HTTP context"
    - "PocketBase CountRecords with dbx.NewExp for filtered aggregation"
    - "Test bootstrap: PocketBase already creates users auth collection on Bootstrap(); do not reimport"

key-files:
  created:
    - db/routes/nodeinfo.go
    - db/routes/nodeinfo_test.go
  modified:
    - db/main.go

key-decisions:
  - "software.version from WANDERER_VERSION env var, falls back to dev when unset (D-01)"
  - "localPosts uses CountRecords with dbx.NewExp(public = true) — private trails excluded (D-02)"
  - "users.total uses CountRecords with nil filter — instance actor is in activitypub_actors not users (D-03)"
  - "NodeInfo21 sets Content-Type with NodeInfo schema profile parameter per spec"
  - "TDD: pure builder functions separated from HTTP handlers for clean unit testing"

patterns-established:
  - "TDD with pure builders: split business logic from HTTP layer for testability"
  - "PocketBase test bootstrap: call Bootstrap() then ImportCollectionsByMarshaledJSON for custom collections"

requirements-completed: [SAFE-04]

# Metrics
duration: 3min
completed: 2026-06-26
---

# Phase 4 Plan 01: NodeInfo Summary

**NodeInfo 2.1 and JRD discovery endpoints implemented with TDD, serving live user/post counts and software identity to peer ActivityPub instances (SAFE-04)**

## Performance

- **Duration:** 3 min
- **Started:** 2026-06-26T10:54:57Z
- **Completed:** 2026-06-26T10:58:43Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- Implemented `buildNodeInfoDiscovery` and `buildNodeInfo21` pure builder functions with full unit test coverage
- Added `NodeInfo` and `NodeInfo21` HTTP handlers wired to `GET /.well-known/nodeinfo` and `GET /.well-known/nodeinfo/2.1`
- Privacy hard constraint enforced: `localPosts` counts only `public = true` trails (D-02, T-04-02); tested with a seeded private trail that must be excluded

## Task Commits

Each task was committed atomically:

1. **Task 1: Failing tests for NodeInfo payload builders** - `91672f8f` (test — RED state)
2. **Task 2: Implement NodeInfo payload builders (GREEN)** - `fbd18d5e` (feat — GREEN + test bootstrap fix)
3. **Task 3: Wire handlers and register the two well-known routes** - `4c0ab71d` (feat)

_TDD: test commit (RED) followed by feat commit (GREEN) as required._

## Files Created/Modified

- `db/routes/nodeinfo.go` - Pure builders (`buildNodeInfoDiscovery`, `buildNodeInfo21`) + HTTP handlers (`NodeInfo`, `NodeInfo21`)
- `db/routes/nodeinfo_test.go` - Unit tests for all builder behaviors: discovery rel/href, software name/version, version fallback, localPosts privacy filter, users.total, required keys
- `db/main.go` - Two GET route registrations: `/.well-known/nodeinfo` and `/.well-known/nodeinfo/2.1`

## Decisions Made

- Used pure builder functions (no `*core.RequestEvent`) so unit tests can call them directly without spinning up an HTTP server
- Set `Content-Type: application/json; profile="..."` in NodeInfo21 handler per NodeInfo spec recommendation
- Discovery handler guards against empty `ORIGIN` env (mirrors existing pattern in `activitypub.go`)
- `software.homepage` and `software.repository` included as recommended optional fields

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] PocketBase users collection already exists after Bootstrap()**
- **Found during:** Task 2 (GREEN implementation) — first test run
- **Issue:** `newNodeInfoTestApp` tried to import a `users` auth collection via `ImportCollectionsByMarshaledJSON`, but PocketBase `Bootstrap()` already creates a default `users` auth collection, causing `UNIQUE constraint failed: _collections.name`
- **Fix:** Removed the explicit `users` collection JSON import from the test helper; the PocketBase-provided collection is used directly for counting
- **Files modified:** `db/routes/nodeinfo_test.go`
- **Verification:** All 8 NodeInfo tests pass; `go build ./...` exits 0
- **Committed in:** `fbd18d5e` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 - bug in test bootstrap)
**Impact on plan:** Fix was necessary for tests to reach GREEN state. No scope creep; no behavioral change to production code.

## Issues Encountered

None beyond the test bootstrap deviation documented above.

## User Setup Required

None - no external service configuration required. The `WANDERER_VERSION` environment variable is optional (falls back to `"dev"` when unset). `ORIGIN` must be set in production (already required by existing ActivityPub handlers).

## Next Phase Readiness

- SAFE-04 satisfied: both well-known endpoints registered and tested
- All four phase requirements complete; this is the final plan of Phase 04
- Ready for milestone close / PR

---
*Phase: 04-nodeinfo*
*Completed: 2026-06-26*
