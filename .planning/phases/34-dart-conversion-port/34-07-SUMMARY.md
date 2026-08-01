---
phase: 34-dart-conversion-port
plan: 07
subsystem: api
tags: [sveltekit, gpx, openapi, vitest, activitypub-adjacent-endpoint]

# Dependency graph
requires:
  - phase: 34-dart-conversion-port
    plan: 05
    provides: "The app's sole /trail/convert caller (transcodeToGpx) already tolerant of a raw-GPX-string response, added in advance of this plan's hard break"
provides:
  - "POST /api/v1/trail/convert transcodes kml/kmz/tcx/fit/gpx to GPX and returns the raw document (application/gpx+xml) - no trail computed, no reverse-geocode performed"
  - "web/src/routes/api/v1/trail/convert/convert.test.ts - handler-level Vitest coverage of all three input branches, both 400 guards, and a D-06 response-shape regression guard"
  - "Regenerated web/static/docs/api/wanderer.openapi.json (gitignored, not committed) describing the new GPX response"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Transcode-only endpoint pattern: fromFile() stays the only server-side conversion step; everything downstream (trail computation, geocoding) moves to the client per D-05/D-06/D-07"

key-files:
  created:
    - web/src/routes/api/v1/trail/convert/convert.test.ts
  modified:
    - web/src/routes/api/v1/trail/convert/+server.ts

key-decisions:
  - "The `name` multipart field stays accepted-and-ignored (not a 400) per the plan's own instruction - rejecting it would break older app builds harder than necessary for no benefit"
  - "web/static/docs/api/wanderer.openapi.json was regenerated and verified (Trail $ref removed from the convert route, moving_duration survived, two runs byte-identical) but NOT committed - it is gitignored per web/.gitignore:6"

patterns-established: []

requirements-completed: [PORT-04, PORT-05]

# Metrics
duration: ~20min
completed: 2026-08-01
---

# Phase 34 Plan 07: Reduce Convert Endpoint to Transcode-Only Summary

**`POST /api/v1/trail/convert` no longer computes a `Trail` or reverse-geocodes anything - it transcodes kml/kmz/tcx/fit/gpx to a GPX document and returns it raw with `Content-Type: application/gpx+xml`, closing out the phase's server-side half of the port.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2/2 completed
- **Files modified:** 2 (1 rewritten, 1 new test file); OpenAPI JSON regenerated but gitignored

## Accomplishments

- `+server.ts`: removed the `gpx2trail(...)` call, its `Invalid GPX content` 400 wrapper, the whole reverse-geocode block (`searchLocationReverse`), and the `trail.expand`/`json(trail)` response pair. The success path is now `new Response(gpxData, { headers: { 'Content-Type': 'application/gpx+xml' } })` (D-05, D-06, D-07). Dropped the now-dead `customName` variable and the `Trail`/`searchLocationReverse`/`gpx2trail`/`json` imports.
- The three input branches (multipart via `fromFile`, JSON `gpx`/`gpxData`, raw-text fallback), the `Missing file field` and `Empty GPX data` 400 guards, and the outer `try/catch` -> `handleError` shape are byte-for-byte unchanged, so the app's existing multipart upload and error handling keep working (D-06).
- Rewrote the `@swagger` JSDoc block: new summary/description describing transcode-only behaviour, `200` response moved to `application/gpx+xml` / `type: string`, and a note recording the breaking change from the prior JSON `Trail` response.
- New `convert.test.ts` (8 tests): raw-text branch, JSON `gpx` key, JSON `gpxData` key, multipart `.gpx` transcode, multipart KML->GPX transcode (real `fromKML`/`toGeoJSON` path, proving PORT-05's "transcoding still happens server-side"), multipart with no `file` field (400), empty raw body (400), and a D-06 regression guard asserting the 200 body does not parse as JSON and contains neither `"expand"` nor `elevation_gain`.
- Regenerated `web/static/docs/api/wanderer.openapi.json`: the convert route's 200 response moved from `$ref: '#/components/schemas/Trail'` under `application/json` to `{ type: string }` under `application/gpx+xml`; plan 34-02's `moving_duration` additions (3 occurrences) survived; two consecutive `npm run openapi:generate` runs produced byte-identical output. Per `web/.gitignore:6` this file is not tracked, so it was not committed - the task's `files_modified` entry for it is a build artifact, not a source file.

## Task Commits

1. **Task 1: Make the convert endpoint transcode-only and return the raw GPX document** - `df61d581` (feat)
2. **Task 2: Cover the new contract with a handler test and regenerate the published OpenAPI** - `d76e360e` (test)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `web/src/routes/api/v1/trail/convert/+server.ts` - rewritten to transcode-only; returns raw GPX with `application/gpx+xml`
- `web/src/routes/api/v1/trail/convert/convert.test.ts` - new handler-level Vitest suite (8 tests)
- `web/static/docs/api/wanderer.openapi.json` - regenerated (gitignored, not committed)

## Decisions Made

- Kept the `name` multipart field accepted-and-ignored per the plan's explicit instruction rather than rejecting it, to avoid breaking older app builds that still send it.
- Did not commit the regenerated `wanderer.openapi.json` - it is gitignored (`web/.gitignore:6`) and the plan's own critical notes anticipated this; verified its content instead of attempting to force-add it.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' acceptance criteria and verification commands passed as specified with no auto-fixes required.

## Issues Encountered

None.

## Verification Performed

- `grep -c "gpx2trail\|searchLocationReverse" web/src/routes/api/v1/trail/convert/+server.ts` -> 0
- `grep -c "customName" web/src/routes/api/v1/trail/convert/+server.ts` -> 0
- `grep -c "application/gpx+xml" web/src/routes/api/v1/trail/convert/+server.ts` -> 2
- `grep -c "handleError" web/src/routes/api/v1/trail/convert/+server.ts` -> 2; `grep -c "ClientResponseError"` -> 3 (both 400 guards survive)
- `grep -c "fromFile" web/src/routes/api/v1/trail/convert/+server.ts` -> 2
- `grep -c "Empty GPX data" web/src/routes/api/v1/trail/convert/+server.ts` -> 1
- `git status --porcelain src/routes/api/v1/trail/upload/+server.ts src/lib/util/gpx_util.ts src/lib/models/gpx/` -> empty (untouched, D-08)
- `cd web && npx svelte-check --tsconfig ./tsconfig.json` -> 0 errors, 0 warnings (both after Task 1 and again after Task 2)
- `cd web && npx vitest run src/routes/api/v1/trail/convert/convert.test.ts` -> 8/8 pass
- `cd web && npx vitest run` (full suite) -> 9 files, 91/91 pass (was 83/83 before this plan; +8 new tests, 0 regressions)
- `grep -c "application/gpx+xml" web/static/docs/api/wanderer.openapi.json` -> 1
- Python-based JSON check: the convert route's `200.content` has no `schemas/Trail` reference; `grep -c "schemas/Trail"` for the whole file -> 28 (other routes' legitimate Trail refs, untouched)
- `grep -c moving_duration web/static/docs/api/wanderer.openapi.json` -> 3 (plan 34-02's additions survived regeneration)
- Regenerated twice via `npm run openapi:generate`; `diff` between the two runs' output -> identical (deterministic)
- `grep -rn "trail/convert" web/src/` -> only this route's own file/test and the pre-existing unrelated comment in `gpx_util.ts:79` (D-08 confirmed: web frontend still never calls this endpoint)
- End-to-end app check (code-level, in lieu of a live device run - the plan's own `<human-check>` defers on-device verification to the phase's end-of-phase UAT pass): read `app/lib/util/trail_import_util.dart`'s `transcodeToGpx` - it inspects `res.data`, taking the `data is String && data.isNotEmpty` branch first. Dio (the app's HTTP client) surfaces a non-`application/json` response body as a plain `String`, so against this endpoint's new `application/gpx+xml` response the raw-string branch is the one exercised; the `data is Map` / `expand.gpx_data` branch is now reachable only against an old server that still returns a JSON `Trail`, exactly as 34-05's summary anticipated ("34-07 can now change the endpoint's success response to raw GPX without breaking this plan's client").

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The server no longer computes trails or reverse-geocodes anything for the convert endpoint; the app's `transcodeToGpx`/`buildLocalTrail` pair (34-05) and the on-device `trailFromGpx` (34-04) are the sole producers of a trail's metrics for all three capture paths, closing PORT-04.
- This was the last plan in Phase 34 per STATE.md's "Plan 7 of 7" position. All PORT-01..05 and CONV-06 requirements are now implemented; phase-level verification/UAT is the remaining step.
- No blockers for Phase 35.

---
*Phase: 34-dart-conversion-port*
*Completed: 2026-08-01*

## Self-Check: PASSED

All modified/created files confirmed present on disk; both task commit hashes
(`df61d581`, `d76e360e`) confirmed present in `git log`.
