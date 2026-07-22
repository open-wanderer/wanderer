---
phase: 23-tilerepositorymanager-download-engine
plan: 01
subsystem: api
tags: [sveltekit, http-range, resumable-download, vitest, proxy]

# Dependency graph
requires:
  - phase: 21.5-region-catalog-archive-pre-build-backend
    provides: Go backend /regions/{id}/download{,-dem} routes (e.FileFS, already Range-capable)
provides:
  - Range-forwarding vector region archive download proxy (200/206 passthrough)
  - Range-forwarding DEM region archive download proxy (200/206 passthrough)
  - vitest coverage proving resumed (Range in / 206+Content-Range out) and fresh (200) request behavior for both proxies
affects: [23-04-tile-repository-manager-download-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SvelteKit proxy Range passthrough: read event.request.headers.get('Range'), conditionally spread into the upstream fetch's headers, then forward response.status and Content-Range/Accept-Ranges/Content-Length verbatim instead of hardcoding status: 200"

key-files:
  created:
    - "web/src/routes/api/v1/regions/[id]/download/server.test.ts"
    - "web/src/routes/api/v1/regions/[id]/download-dem/server.test.ts"
  modified:
    - "web/src/routes/api/v1/regions/[id]/download/+server.ts"
    - "web/src/routes/api/v1/regions/[id]/download-dem/+server.ts"

key-decisions:
  - "Kept the plan's pre-existing working-tree rename of the upstream path from /api/v1/regions/... to /regions/... untouched — only edited the Range/status/header forwarding lines"
  - "Test event objects are hand-built minimal RequestEvent shapes (cast as unknown as RequestEvent) rather than full SvelteKit test harness, matching the plan's exact prescribed test structure"

patterns-established:
  - "Range-forwarding SvelteKit proxy: forward Range header in, forward upstream status + Content-Range/Accept-Ranges/Content-Length out — reusable for any future binary-archive proxy route in this codebase"

requirements-completed: [TILE-02]

# Metrics
duration: 4min
completed: 2026-07-22
---

# Phase 23 Plan 01: SvelteKit Region Download Proxy Range Forwarding Summary

**Both region-archive SvelteKit proxy routes (vector and DEM) now forward the client's Range header inbound and the backend's actual status (200/206) plus Content-Range/Accept-Ranges outbound, proven by 4 new vitest assertions — unblocking Plan 04's Flutter resumable-download work (TILE-02).**

## Performance

- **Duration:** 4 min
- **Started:** 2026-07-22T09:18:19Z
- **Completed:** 2026-07-22T09:22:22Z
- **Tasks:** 2 completed
- **Files modified:** 4 (2 modified, 2 created)

## Accomplishments
- Vector download proxy (`download/+server.ts`) forwards `Range` in, returns upstream `status`/`Content-Range`/`Accept-Ranges` out
- DEM download proxy (`download-dem/+server.ts`) has the byte-for-byte identical fix applied
- Two colocated vitest suites (4 tests total) prove a resumed request returns 206+Content-Range with the outgoing fetch carrying the `Range` header, and a fresh request returns 200

## Task Commits

Each task was committed atomically:

1. **Task 1: Forward Range in / status + range headers out on the vector download proxy** - `8f4a2540` (fix)
2. **Task 2: Apply the identical fix to the DEM download proxy, plus vitest coverage for both** - `8cb6c19c` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `web/src/routes/api/v1/regions/[id]/download/+server.ts` - Reads incoming `Range` header, forwards it upstream, returns `response.status` and forwards `Content-Range`/`Accept-Ranges`/`Content-Length`
- `web/src/routes/api/v1/regions/[id]/download-dem/+server.ts` - Identical fix, DEM-specific URL segment and `Content-Disposition` filename preserved
- `web/src/routes/api/v1/regions/[id]/download/server.test.ts` - vitest: resumed request (Range in → 206/Content-Range out) and fresh request (200)
- `web/src/routes/api/v1/regions/[id]/download-dem/server.test.ts` - Same coverage for the DEM route

## Decisions Made
- Left the already-present working-tree rename of the upstream fetch path (`/api/v1/regions/...` → `/regions/...`) untouched, per the plan's explicit instruction, and edited only the Range/status/header-forwarding lines around it.
- Test fixtures build a minimal `RequestEvent`-shaped object (`request`, `params`, `locals.pb`, `fetch`) cast via `as unknown as RequestEvent`, matching the plan's prescribed test structure rather than spinning up a full SvelteKit test harness.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TILE-02's SvelteKit-side prerequisite is complete: both region-archive proxy routes correctly passthrough HTTP Range semantics end-to-end (Flutter → SvelteKit → Go backend `e.FileFS`).
- Plan 23-04 (Flutter `TileRepositoryManager` resumable download code) can now be verified against a real resumed download through this proxy layer.
- No blockers identified for subsequent Phase 23 plans.

---
*Phase: 23-tilerepositorymanager-download-engine*
*Completed: 2026-07-22*

## Self-Check: PASSED

All created/modified files verified present on disk; all task commit hashes (8f4a2540, 8cb6c19c, fff8688c) verified present in git log.
