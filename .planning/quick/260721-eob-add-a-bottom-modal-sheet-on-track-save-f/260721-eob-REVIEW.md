---
phase: 260721-eob
reviewed: 2026-07-21T00:00:00Z
depth: quick
files_reviewed: 9
files_reviewed_list:
  - app/lib/components/navigation/track_save_options_sheet.dart
  - app/lib/i18n/app_en.arb
  - app/lib/routes/navigation_screen.dart
  - app/lib/util/route_planner_handoff_util.dart
  - app/test/util/route_planner_handoff_util_test.dart
  - web/src/lib/models/api/valhalla_trace_route_schema.ts
  - web/src/lib/server/url.ts
  - web/src/lib/server/valhalla.ts
  - web/src/routes/api/v1/valhalla/trace-route/+server.ts
findings:
  critical: 1
  warning: 2
  info: 1
  total: 4
status: issues_found
outcomes:
  CR-01: fixed
  WR-01: fixed
  WR-02: acknowledged_out_of_scope
  IN-01: fixed
---

# Phase 260721-eob: Code Review Report

**Reviewed:** 2026-07-21T00:00:00Z
**Depth:** quick (escalated to a full read of every listed file — the feature is small enough that a full trace was cheaper than pattern-matching and materially more reliable)
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Reviewed the new track-save options bottom sheet, the `snapShapeToRoads`/`snapResultAcceptable` pipeline, the rewired `_saveRecordedTrack`, and the new authenticated `/valhalla/trace-route` SvelteKit proxy (plus its Zod schema and env wiring).

The Valhalla proxy itself is solid: authenticated, input-bounded (`min(2).max(500)`, lat/lon range-checked, `costing` enum-restricted — no injection surface), and consistent with the sibling `/valhalla/navigate` proxy's conventions. The pure helpers (`snapResultAcceptable`, `mergeHeightsIntoGpx`) are well-tested.

The one significant defect is in how `_saveRecordedTrack` was rewired: turning on **either** toggle now routes the recorded breadcrumb through `buildNavShape` (a helper designed for the Route Planner's much sparser anchor-derived shapes), which silently downsamples the track to at most 500 points and permanently discards every point's recorded `time`. Prior to this change, `_saveRecordedTrack` always preserved the full breadcrumb via `buildGpxFromPoints`. Any real hiking/biking recording of more than a few tens of minutes routinely exceeds 500 GPS fixes, so this isn't an edge case — it's the common case for anyone who taps "Recalculate heights" or "Follow roads" on a normal-length recording. Filed as a Blocker below given the data-loss nature.

## Critical Issues

### CR-01: Enabling either save-track toggle silently truncates the recorded track to ≤500 points and drops all timestamps

**File:** `app/lib/routes/navigation_screen.dart:709-759` (see also `app/lib/util/route_planner_handoff_util.dart:25-45` for `mergeHeightsIntoGpx`, which has no `time` field at all)

**Issue:** Before this change, `_saveRecordedTrack`'s no-transform path (`buildGpxFromPoints(navState.breadcrumb)`) was the *only* path, and it preserved every recorded point 1:1 including `time`. This change adds:

```dart
if (recalcHeights || followRoads) {
  final breadcrumbPoints = [ ... ];
  var workingShape = buildNavShape(breadcrumbPoints);   // <-- downsamples to <=500 pts
  ...
  gpx = mergeHeightsIntoGpx(workingShape, heights);      // <-- Wpt has no `time` field
}
```

`buildNavShape` was written for the Route Planner's anchor-derived shapes (naturally short) and caps its output at ~500 points by dropping intermediate points entirely (`app/lib/util/gpx_util.dart:28-49`, not in this review's file list but load-bearing here). A recorded GPS breadcrumb sampled every few seconds over a multi-hour hike/ride will commonly contain several thousand points, so toggling **either** "Recalculate heights" or "Follow roads" — options whose UI copy only promises an elevation/road change — will silently:
1. Discard the majority of the recorded trkpts (real geometry loss, not just resampling for display), and
2. Discard every remaining point's `time`, even when the user never asked to modify anything but elevation.

Neither behavior is disclosed to the user, and there is no test covering `_saveRecordedTrack`'s new branches (only the pure helpers it calls are unit-tested), so this regression has no guard against recurring.

**Fix:** Don't reuse the downsampled `workingShape` as the final track geometry. Use it only to query Valhalla (which is where the 500-point cap matters), then merge the results back onto the **full-resolution** breadcrumb — e.g. nearest-index interpolation of `heights`/snapped-shape back onto every original point — and carry the original `Wpt.time` through instead of rebuilding `Wpt`s from bare `{lat,lon}` maps:

```dart
// keep the full-resolution points around for the final merge
final fullPoints = navState.breadcrumb; // has .time already
...
// after computing heights/snapped shape from the (possibly downsampled) workingShape,
// map results back onto fullPoints by nearest fractional index instead of
// replacing geometry with workingShape:
gpx = mergeHeightsIntoFullBreadcrumb(fullPoints, workingShape, heights);
```
At minimum, add a regression test exercising `_saveRecordedTrack`-equivalent logic (or a testable extraction of it) with a >500-point breadcrumb and assert the resulting GPX still has breadcrumb.length trkpts, not `workingShape.length`.

## Warnings

### WR-01: `saveTrack` exit-dialog choice invokes `_saveRecordedTrack()` without a `context.mounted` guard

**File:** `app/lib/routes/navigation_screen.dart:1243-1259`

**Issue:** In `_confirmExit`'s `.then((choice) { switch (choice) { ... } })`, the `exit` branch checks `if (context.mounted) { context.pop(); }` before touching `context`, but the `saveTrack` branch calls `_saveRecordedTrack()` unconditionally:

```dart
case _NavExitChoice.saveTrack:
  _saveRecordedTrack();
```

`_saveRecordedTrack` immediately does `await showTrackSaveOptionsSheet(context)` using the *outer* (screen) context, and the call is fire-and-forget (not awaited, no `.catchError`) at both call sites. If `context` is ever unmounted between the dialog closing and this callback running, `showModalBottomSheet` throws, and that exception is unhandled (no try/catch wraps this call).

**Fix:** Guard the same way the sibling branch does:
```dart
case _NavExitChoice.saveTrack:
  if (context.mounted) _saveRecordedTrack();
```

### WR-02: "Follow roads" always costs as `pedestrian` for GPS-recording sessions, regardless of actual activity

**File:** `app/lib/routes/navigation_screen.dart:736-739`

**Issue:**
```dart
final costing = costingForCategory(
  originalTrail?.expand?.category?.name,
);
```
For `widget.isRecording == true`, `widget.id` has no backing trail, so `ref.read(trailProvider(widget.id)).value` resolves to `null` (the provider's fetch fails and `.value` swallows the error), making `originalTrail` always `null` here. `costingForCategory(null)` always returns `'pedestrian'` (`app/lib/util/valhalla_util.dart:48-56`), so a user recording a bike ride and enabling "Follow roads" will always get pedestrian-profile road-snapping rather than bicycle-profile, even though the same feature correctly derives bicycle costing for the (non-recording) Route Planner and trail-edit handoff paths.

**Fix:** Either surface/derive the travel profile chosen at recording start (if one is captured — e.g. via `active_nav`/`ActiveNavigationEntity`) and thread it through to `costingForCategory`, or explicitly document that recording-mode road-snapping is pedestrian-only until an activity type is captured at record time.

## Info

### IN-01: New Zod-inferred types in `valhalla_trace_route_schema.ts` are exported but never consumed

**File:** `web/src/lib/models/api/valhalla_trace_route_schema.ts:16-28`

**Issue:** `TraceRouteRequest`, `TraceRouteShapePoint`, and `TraceRouteResponse` are declared and exported, but nothing outside this file imports them. `web/src/routes/api/v1/valhalla/trace-route/+server.ts` builds its response with an inline literal type (`const shape: { lat: number; lon: number }[] = []`) instead of `TraceRouteResponse`, so the two are maintained independently and can silently drift out of sync.

**Fix:** Either have the `+server.ts` handler annotate its response with `TraceRouteResponse`/`TraceRouteShapePoint` so a shape change is caught at compile time, or drop the unused exported types if they aren't meant to be a shared contract yet.

---

## Post-Review Fixes (2026-07-21)

- **CR-01 — fixed.** Added `fetchHeightsForShape` (`app/lib/util/route_planner_handoff_util.dart`), which batches the **full-resolution** breadcrumb into ≤500-point `/valhalla/height` requests and concatenates results 1:1, instead of merging onto `buildNavShape`'s downsampled output. `_saveRecordedTrack` now builds `workingShape` directly from the full breadcrumb (no downsampling) and only applies `buildNavShape`'s cap to the outbound `trace_route` request (whose response legitimately replaces the shape — that's the point of "Follow roads"). 5 new tests cover empty-shape short-circuit, exact-500 single chunk, >500 multi-chunk concatenation ordering, network-failure fallback, and mismatched-response-length fallback. Timestamp loss on any transform path remains a disclosed, accepted consequence (see CONTEXT.md) — only the point-count truncation was the actual defect.
- **WR-01 — fixed.** `saveTrack` branch now guarded with `if (context.mounted) _saveRecordedTrack();`, matching the sibling `exit` branch.
- **WR-02 — acknowledged, left as-is (out of scope).** Confirmed real: `_openRecorder` never collects a travel profile the way the route planner's `_openPlanner` does, so `originalTrail` is always null for a GPS-recording session and "Follow roads" always costs as pedestrian regardless of actual activity. This is a pre-existing gap in the recording flow (no activity-type capture at record start), not something introduced by this task, and fixing it properly means adding that capture — out of scope for this quick task. Documented inline at the costing call site instead of silently leaving it unexplained.
- **IN-01 — fixed.** `TraceRouteRequest`/`TraceRouteShapePoint`/`TraceRouteResponse` are now actually consumed: `+server.ts` builds its `shape` array as `TraceRouteShapePoint[]` and its final response as a `TraceRouteResponse`, so a schema change is now caught at compile time instead of silently drifting from an unused parallel type.

_Reviewed: 2026-07-21T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_
