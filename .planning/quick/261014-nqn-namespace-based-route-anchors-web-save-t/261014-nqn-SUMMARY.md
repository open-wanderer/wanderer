---
phase: quick-261014-nqn
plan: 01
subsystem: web/route-editor
tags: [gpx, route-anchors, save-path]
requires: [ae31bb26e (#1370)]
provides: [routeToStoredGpx]
affects: [web/src/lib/util/gpx_segment_util.ts, web/src/routes/trail/edit/[id]/+page.svelte]
key-files:
  modified:
    - web/src/lib/util/gpx_segment_util.ts
    - web/src/lib/util/gpx_util.test.ts
    - web/src/routes/trail/edit/[id]/+page.svelte
decisions:
  - Clone via GPX.parse(gpx.toString()) and join only the clone; valhallaStore.route is never mutated
  - Exact numeric equality join rule kept; mismatched legs stay separate
status: complete
commits: 1
plan_head_before: e645c43012e2ff27d9ed5ffd1382bf670aae60c7
plan_head_after: 1779fa306
actuals:
  commits: 1
  tasks: 2
completed: 2026-10-14
---

# Quick 261014-nqn: Store drawn routes as one marked track segment

Saving a drawn route in the web editor now uploads one trkseg per track with `<wanderer:anchor/>` at each leg junction and `xmlns:wanderer` on the root, via the new `routeToStoredGpx(gpx)` helper.

## Commit

- Worktree `/Users/christianbeutel/Documents/svelte/wanderer-route-anchors`, branch `fix/route-anchor-marks-on-save`
- `1779fa306` fix(web): store drawn routes as one marked track segment (headline only, not pushed, no PR)

## What changed

- `gpx_segment_util.ts`: `GPX` import changed from type-only to value import; added `routeToStoredGpx` (parse a clone, `joinContiguousTrackSegments`, `toString`); header comment reworded to cover save as well as export.
- `+page.svelte`: save site uses `routeToStoredGpx(valhallaStore.route)`; both `splitTrackSegmentsAtAnchors` load-path calls unchanged.
- `gpx_util.test.ts`: new `describe("routeToStoredGpx")` with 6 tests (A one marked segment, B round-trip restores legs, C input not mutated, D in-memory constructor-built legs, E mismatched endpoints stay separate, F empty route).

## Verification

- vitest `src/lib/util/gpx_util.test.ts`: 19 passed (6 new, RED confirmed first with 6 failures).
- svelte-check baseline (before edits): `COMPLETED 2556 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`
- svelte-check after: `COMPLETED 2556 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS`
- Page greps: `routeToStoredGpx(valhallaStore.route)` x1, `splitTrackSegmentsAtAnchors(` x2, `valhallaStore.route.toString` x0.
- HEAD commit touches exactly the three planned files; `web/src/lib/util/gpx_util.ts` is not in the diff.
- Not run: the human-check in the dev server (draw, save, download, undo after save, reload).

## D-03 audit

No post-save consumer needs per-leg segments from the saved string. It only builds `gpxFile`; the server model replaces the form afterwards, and the map draws `valhallaStore.route`, which is unchanged.

## D-04 review of #1370

#1370 had no dead code. The export-time join in `trail2gpx` stays (legacy and app-saved trails are still multi-segment; on an already-joined file it is a no-op) and both load-path splits stay (now essential). Only the header comment in `gpx_segment_util.ts` was reworded. Nothing removed.

## Deviations from Plan

None. Tests A-F were added together in one RED step rather than split across the two tasks, and both tasks went into the single planned commit. No `routeToStoredGpx` guard for the empty route was needed (Test F passes as is).

## Known Stubs

None.

## Self-Check: PASSED

- Commit 1779fa306 is HEAD of `fix/route-anchor-marks-on-save`; the three files exist and are modified as listed.
