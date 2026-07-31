---
phase: 33-conversion-correctness
plan: 05
subsystem: gpx-conversion / trail-edit crop panel
tags: [gpx, crop, nan-guard, data-loss, gap-closure]
dependency_graph:
  requires:
    - "web/src/lib/models/gpx/gpx-metrics-computation.ts (33-03's cumulativeDistance raw array, 33-05 gap 2/3's source of the NaN)"
  provides:
    - "web/src/lib/models/gpx/crop.ts: testable, degenerate-safe crop interpolation (getCoordinateAtDistance, hasCropInterpolationBasis)"
    - "A croppedGPX lifecycle in +page.svelte that cannot outlive the route it was derived from"
  affects:
    - "web/src/routes/trail/edit/[id]/+page.svelte crop panel (updateCropMarkers, confirmCrop, toggleCropMarkers, resetTrail, replaceRoute, updateTrailWithRouteData)"
tech_stack:
  added: []
  patterns:
    - "Total (never-throws) pure function extraction from a Svelte component script block into a co-located Vitest-reachable module"
    - "Single mutation choke point (updateTrailWithRouteData) plus redundant explicit resets at named call sites, for a cache invalidation invariant"
key_files:
  created:
    - "web/src/lib/models/gpx/crop.ts"
    - "web/src/lib/models/gpx/crop.test.ts"
  modified:
    - "web/src/routes/trail/edit/[id]/+page.svelte"
decisions:
  - "getCoordinateAtDistance guards span > 0 (not span !== 0), so a non-non-decreasing cumulative array can never produce a negative extrapolation either, not just avoiding the NaN case"
  - "hasCropInterpolationBasis's true result does NOT imply every adjacent span is non-zero (a healthy-total route can still have a coincident leading pair), so the guard function and getCoordinateAtDistance's own span>0 check are both required -- neither alone closes the gap"
  - "croppedGPX is reset at 5 sites: 4 are logically redundant with updateTrailWithRouteData()'s reset (the single choke point all 16 route-mutation call sites already route through) but kept explicit by design, so the invariant survives if that choke point is ever refactored away"
metrics:
  duration: "~15 min (Tasks 1-2 completed in a prior session; this session verified, wired Task 3, and committed)"
  completed: "2026-07-31"
---

# Phase 33 Plan 05: Crop-panel NaN guard and croppedGPX staleness fix Summary

Closed VERIFICATION gaps 2 and 3 (CR-01 and CR-03, both BLOCKER): the trail-edit crop
panel's `updateCropMarkers()` guard checked the wrong condition
(`!Number.isFinite(rawRouteTotal)`, which is dead code) and missed
`rawRouteTotal === 0` / a merely coincident leading point pair, both of which made
`getCoordinateAtDistance()` divide `0 / 0` and return `[NaN, NaN, 1]` — an uncaught
exception inside MapLibre's `LngLat.convert` on crop-panel open. Separately,
`croppedGPX` had no reset path anywhere, so `confirmCrop()` could silently replace
the user's current route with a crop of a route they had already discarded.

## What Was Built

**Task 1 (RED):** Extracted `getCoordinateAtDistance()` verbatim from
`+page.svelte`'s script block into a new pure module,
`web/src/lib/models/gpx/crop.ts` (no behavior change), and added
`web/src/lib/models/gpx/crop.test.ts` with two degenerate-route fixtures
(`cumulative = [0, 0, 111.19, 222.39]` and `[0, 0, 0]`, both queried at target 0)
and two normal-route fixtures. The two degenerate fixtures failed, pinning the
real shipped defect. Observed RED evidence (verbatim, reproduced this session
against the Task-1 commit's pre-fix `crop.ts` to confirm the historical run):

```
❯ src/lib/models/gpx/crop.test.ts (4 tests | 2 failed)
  × returns a finite coordinate for a leading-duplicate-point route queried at 0%
    AssertionError: expected false to be true
      28|     expect(Number.isFinite(lon)).toBe(true);
  × returns a finite coordinate for an all-identical-points route queried at 0%
    AssertionError: expected false to be true
      41|     expect(Number.isFinite(lon)).toBe(true);

 Test Files  1 failed (1)
      Tests  2 failed | 2 passed (4)
```

**Task 2 (GREEN):** Made `getCoordinateAtDistance()` total and degenerate-safe:
`const span = nextDist - prevDist; const ratio = span > 0 ? (target - prevDist) / span : 0;`
replaces the unconditional `0/0` division; the binary-search index is clamped to a
valid adjacent pair (`Math.max(1, Math.min(low, points.length - 1))`); explicit
early returns handle `points.length === 0` and `points.length < 2 || cumulative.length < 2`;
non-null assertions (`prev.$.lon!`) were replaced with `?? 0` nullish-coalescing
(defence in depth — `Waypoint` already defaults missing coordinates to `-1`, never
`undefined`). Added `hasCropInterpolationBasis(cumulative): boolean`, `true` only
when `cumulative.length >= 2 && Number.isFinite(total) && total > 0` — the `> 0`
check the shipped guard was missing.

**Task 3:** Wired `+page.svelte` to the fixed module, six edits:
1. Added `import { getCoordinateAtDistance, hasCropInterpolationBasis } from "$lib/models/gpx/crop";` (line 67); deleted the local, un-testable `getCoordinateAtDistance()` declaration.
2. `updateCropMarkers()`'s guard (line 1483) is now `!hasCropInterpolationBasis(cumulativeRoute)`; the branch also sets `croppedGPX = null` (line 1489) and calls `toggleCropMarkers(false)` before returning, so a degenerate route neither confirms a stale crop nor strands two visible pins at 0N 0E.
3. `toggleCropMarkers()`'s inactive branch resets `croppedGPX = null` (line 1436).
4. `updateTrailWithRouteData()` resets `croppedGPX = null` as its first statement (line 1535) — the single choke point all 16 route-mutation call sites in the file already call.
5. `resetTrail()` (line 1400) and `replaceRoute()` (line 1411) each add an explicit `croppedGPX = null;`, redundant by design with #4, so the invariant survives if #4 is ever refactored away.
6. `confirmCrop()` hoists `const confirmedCrop = croppedGPX;` (line 1524) before calling `updateTrailWithRouteData()` (which now nulls the field per #4), and uses `confirmedCrop` for both `setRoute()` and `initRouteAnchors()`.

## croppedGPX Reset/Reference Sites (final)

| Line | Statement | Role |
|------|-----------|------|
| 166 | `let croppedGPX: GPX | null = null;` | declaration |
| 1400 | `croppedGPX = null;` | `resetTrail()` reset |
| 1411 | `croppedGPX = null;` | `replaceRoute()` reset |
| 1436 | `croppedGPX = null;` | `toggleCropMarkers(false)` reset |
| 1489 | `croppedGPX = null;` | `updateCropMarkers()` degenerate early-return reset |
| 1511 | `croppedGPX = cropGPX(...)` | the single derivation/assignment site |
| 1524 | `const confirmedCrop = croppedGPX;` | hoisted read in `confirmCrop()`, taken before the reset at #4 fires |
| 1535 | `croppedGPX = null;` | `updateTrailWithRouteData()` reset — the choke point |

## Verification

- `cd web && npx vitest run` — 44/44 tests pass across 6 files (up from the
  pre-plan 35; `crop.test.ts` contributes 8 new tests: 2 degenerate-fixture,
  2 normal-route, 4 `hasCropInterpolationBasis` predicate cases).
- `cd web && npx svelte-check --output human --threshold error` —
  `svelte-check found 0 errors and 0 warnings`.
- `grep -c "function getCoordinateAtDistance"` (comment-excluded) on `+page.svelte`
  → 0. `grep -c "Number.isFinite(rawRouteTotal)"` (comment-excluded) → 0.
- `grep -c "hasCropInterpolationBasis(cumulativeRoute)"` → 1.
  `grep -n 'from "$lib/models/gpx/crop"'` → 1 match, line 67.
- `grep -c "croppedGPX = null"` → 6 (declaration excluded; matches the 5 reset
  sites plus the doc-comment mention inside `confirmCrop()` — see table above
  for the exact breakdown; acceptance criterion required >= 5).
- `grep -c "const confirmedCrop"` → 1; `grep -c "initRouteAnchors(confirmedCrop"` → 1.
- `grep -c "setRoute(croppedGPX"` → 0 (only `setRoute(confirmedCrop, ...)` remains).
- `git diff --stat` against the plan's starting commit (`3860706e~1`) touches
  exactly three files: `crop.ts` (new), `crop.test.ts` (new), `+page.svelte`
  (modified) — no change to `gpx.ts`, `gpx-metrics-computation.ts`, `gpx_util.ts`,
  `route_editor.svelte`, or `double_slider.svelte`.
- `span > 0` guard present (`grep -c "span > 0"` >= 1); no surviving inline
  `nextDist - prevDist)` division (`grep -c` → 0).
- `export function hasCropInterpolationBasis` present exactly once.

## Deviations from Plan

**Continuation context:** Tasks 1 and 2 (the RED/GREEN pair extracting and fixing
`getCoordinateAtDistance()`) were already committed at the start of this executor
session (commits `3860706e` and `ef698579`), and Task 3's six `+page.svelte` edits
were already present in the working tree, uncommitted. This session verified every
task's work against the plan's acceptance criteria (re-running the full test suite,
`svelte-check`, and every grep gate; reproducing the historical Task 1 RED failure
against the pre-fix `crop.ts` to confirm the documented evidence), then committed
Task 3 (`f1108d71`). No code was rewritten — all three tasks matched the plan's
specified algorithm, guard conditions, and file scope exactly. This is not a
deviation from the plan's content, only from this session's starting point.

The WR-04 start-index off-by-one and the WR-05 marker-creation reordering (both
flagged as warnings, not blockers, in `33-VERIFICATION.md`'s code review) were left
in place, out of this gap's scope, as the plan's `<action>` explicitly instructed.

## Known Stubs

None.

## Threat Flags

None — matches the plan's own threat model assessment. `crop.ts` is a pure,
type-only-import module (no new network call, endpoint, persistence path,
authentication surface, or dependency). The two mitigated threats
(T-33-05-01 DoS via uncaught MapLibre exception, T-33-05-02 tampering/data-loss
via stale `croppedGPX`) are both closed per the verification evidence above.

## Self-Check: PASSED

- `web/src/lib/models/gpx/crop.ts` — FOUND
- `web/src/lib/models/gpx/crop.test.ts` — FOUND
- `web/src/routes/trail/edit/[id]/+page.svelte` — FOUND (modified)
- Commit `3860706e` (test(33-05): pin the crop-panel NaN defect with a failing regression test) — FOUND
- Commit `ef698579` (feat(33-05): make crop interpolation degenerate-safe (GREEN)) — FOUND
- Commit `f1108d71` (feat(33-05): wire trail-edit crop panel to degenerate-safe module) — FOUND
