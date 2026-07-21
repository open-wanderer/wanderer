---
phase: 260721-eob
verified: 2026-07-21T00:00:00Z
status: human_needed
score: 6/6 must-haves verified
overrides_applied: 0
---

# Quick Task 260721-eob: Add a bottom modal sheet on track save Verification Report

**Task Goal:** Add a bottom modal sheet on track save from navigation screen with 'Recalculate heights' and 'Follow roads' toggle options; Follow roads snaps recorded path via Valhalla trace_route through a new SvelteKit proxy API route before passing trail to trail_create_screen
**Verified:** 2026-07-21
**Status:** human_needed
**Re-verification:** No — initial verification (task went through executor -> code review -> post-review fixes, all on this branch; this is the first VERIFICATION pass)

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Saving a recorded track first opens a bottom sheet with 'Recalculate heights' and 'Follow roads' toggles, both off by default. | VERIFIED | `_saveRecordedTrack` (`app/lib/routes/navigation_screen.dart:708-710`) calls `await showTrackSaveOptionsSheet(context)` before anything else; `if (options == null) return;`. `track_save_options_sheet.dart` defines `_recalcHeights = false` / `_followRoads = false` as initial `StatefulWidget` fields, each bound to a `SwitchListTile` inside a bordered `_ToggleCard`. |
| 2 | Enabling 'Follow roads' snaps the recorded path to the road network via Valhalla trace_route before the trail is handed to trail_create_screen. | VERIFIED | `navigation_screen.dart:740-757`: `if (followRoads && workingShape.length >= 2) { ... workingShape = await snapShapeToRoads(ref, buildNavShape(breadcrumbPoints), costing); }` runs before `buildDraftTrail`/`context.pushReplacement('/trail/create/edit', ...)`. `snapShapeToRoads` (`route_planner_handoff_util.dart:102-132`) POSTs to `/valhalla/trace-route`, which is implemented server-side and forwards to Valhalla's `trace_route` action with `shape_match: "map_snap"` (`web/src/routes/api/v1/valhalla/trace-route/+server.ts:85-89`). |
| 3 | Enabling 'Recalculate heights' replaces recorded GPS elevations with Valhalla /height values computed on the final (possibly snapped) shape. | VERIFIED | `navigation_screen.dart:759-764`: heights fetch (`fetchHeightsForShape`) runs on `workingShape` AFTER the follow-roads branch has potentially replaced it, then `gpx = mergeHeightsIntoGpx(workingShape, heights)`. `fetchHeightsForShape` (`route_planner_handoff_util.dart:145-166`) batches `/valhalla/height` in ≤500-point chunks over the full shape and concatenates results 1:1 (post-review CR-01 fix — no longer downsampled to Valhalla's request cap). |
| 4 | Cancelling/dismissing the sheet aborts the save with no change to the session. | VERIFIED | `showTrackSaveOptionsSheet` returns `null` on dismiss/back (bare `showModalBottomSheet<(bool,bool)>` with no explicit pop on barrier dismiss); `_saveRecordedTrack` returns immediately on `options == null` before `_savingTrack` is even set or any state is touched. |
| 5 | A trace_route or /height failure (or a truncated snap) falls back silently to the pre-transformation track — no error toast, no blocked save. | VERIFIED | `snapShapeToRoads` wraps the network call in try/catch returning the original `shape` on any error, and additionally rejects an accepted-but-truncated result via `snapResultAcceptable`'s bbox-diagonal guard (`< 0.6x` original → reject). `fetchHeightsForShape` wraps its chunk loop in try/catch, returning `const []` on any failure or chunk-length mismatch — `mergeHeightsIntoGpx` degrades gracefully to null `ele` in that case. Neither path throws into `_saveRecordedTrack`'s outer try/catch (which is reserved for the `/trail/convert` failure toast), so these fallbacks never surface an error toast. |
| 6 | Both the exit-dialog Save and the completion-banner Save route through the sheet. | VERIFIED | Both call sites invoke the same `_saveRecordedTrack` function, which opens the sheet unconditionally at its top: exit-dialog branch at `navigation_screen.dart:1254` (`if (context.mounted) _saveRecordedTrack();`), completion-banner button at `navigation_screen.dart:1416` (`onPressed: _savingTrack ? null : _saveRecordedTrack`). |

**Score:** 6/6 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/routes/api/v1/valhalla/trace-route/+server.ts` | Authenticated POST proxy to Valhalla trace_route; decodes trip.legs[].shape and returns `{ shape }` | VERIFIED | Exists, `if (!event.locals.user) return error(401, ...)` auth gate; validates body with `TraceRouteRequestSchema.parse`; forwards `{shape, costing, shape_match:"map_snap"}`; decodes `trip.legs[].shape` via `decodePolyline`, correctly reorders `[lng,lat] → {lat, lon:lng}`; returns typed `TraceRouteResponse`. |
| `web/src/lib/models/api/valhalla_trace_route_schema.ts` | Zod `TraceRouteRequestSchema` (shape 2-500 bounded, costing enum) | VERIFIED | `.min(2, "at_least_two_shape_points").max(500)` on shape array; `costing: z.enum(["pedestrian","bicycle"]).default("pedestrian")`. Also exports `TraceRouteRequest`, `TraceRouteShapePoint`, `TraceRouteResponse` types, all consumed by `+server.ts` (IN-01 fixed — no longer dead exports). |
| `app/lib/components/navigation/track_save_options_sheet.dart` | `showTrackSaveOptionsSheet` returning `(bool recalcHeights, bool followRoads)?` | VERIFIED | Function signature matches exactly; both toggles default false; confirm button pops the tuple; dismiss/back yields `null`. |
| `app/lib/util/route_planner_handoff_util.dart` | `snapShapeToRoads` best-effort helper + `snapResultAcceptable` pure truncation guard | VERIFIED | Both present with documented behavior matching plan spec; plus `fetchHeightsForShape` (post-review addition, CR-01 fix) that batches full-resolution height requests. |
| `web/src/lib/server/url.ts` | `VALHALLA_TRACE_ROUTE_URL` added to `ExternalServiceUrlKey` union | VERIFIED | `grep` confirms presence in the union type. |
| `web/src/lib/server/valhalla.ts` | `getValhallaTraceRouteUrl()` getter | VERIFIED | Present, mirrors `getValhallaNavigateUrl`. |
| `app/lib/i18n/app_en.arb` | New copy keys | VERIFIED | `save_recording_options`, `recalculate_heights(_description)`, `follow_roads(_description)`, `save` all present; `flutter gen-l10n` getters confirmed generated in `app_localizations.dart`. |
| `app/test/util/route_planner_handoff_util_test.dart` | Unit tests for `snapResultAcceptable` (+ post-review `fetchHeightsForShape` tests) | VERIFIED | 4 `snapResultAcceptable` cases + 5 `fetchHeightsForShape` cases (empty, exact-500, >500 multi-chunk concatenation, network failure, length-mismatch) all present and passing. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `app/lib/routes/navigation_screen.dart` | `app/lib/components/navigation/track_save_options_sheet.dart` | `showTrackSaveOptionsSheet` gates `_saveRecordedTrack` | WIRED | Called as the very first line of `_saveRecordedTrack`; both save entry points funnel through it. |
| `app/lib/routes/navigation_screen.dart` | `/valhalla/trace-route` | `snapShapeToRoads → apiProvider.post` | WIRED | `snapShapeToRoads` posts to `'/valhalla/trace-route'` via `ref.read(apiProvider)`; invoked from `_saveRecordedTrack`'s follow-roads branch. |
| `web/src/routes/api/v1/valhalla/trace-route/+server.ts` | `VALHALLA_TRACE_ROUTE_URL` | `getValhallaTraceRouteUrl` env upstream | WIRED | Route calls `getValhallaTraceRouteUrl()`, 400s cleanly when unset, otherwise `event.fetch`es the resolved URL. |
| `app/lib/routes/navigation_screen.dart` | `/valhalla/height` | `fetchHeightsForShape` on the (possibly snapped) shape | WIRED | `fetchHeightsForShape` posts to `/valhalla/height` in chunks; invoked from `_saveRecordedTrack`'s recalc-heights branch on `workingShape` (post-snap value). |

### Post-Review Fix Verification (not just SUMMARY claims — re-checked against current file state)

| Finding | Claimed Fix | Verified Against Code |
|---------|-------------|------------------------|
| CR-01 (data loss: >500-point recordings silently truncated + all timestamps dropped by any transform) | `fetchHeightsForShape` added; `workingShape` built from full-resolution breadcrumb, not `buildNavShape`'s downsampled form; `buildNavShape`'s cap applied only to the outbound `trace_route` hint | CONFIRMED. `navigation_screen.dart:736-738` builds `workingShape` directly from all `breadcrumbPoints` (no cap). `buildNavShape` is invoked only as the argument to `snapShapeToRoads` (line 754), i.e. only for the outbound Valhalla request; the network call's *response* replaces `workingShape` (matched path, which is the intended "Follow roads" behavior, not a truncation defect). `fetchHeightsForShape` (`route_planner_handoff_util.dart:145-166`) chunks the full shape at ≤500 points per request and concatenates 1:1, rather than capping the saved geometry. 5 new tests exercise this directly and pass, including an explicit >500-point multi-chunk-concatenation case. Point-count truncation defect is fixed. Timestamp loss on ANY transform path (snap and/or heights) remains — this is the plan's own explicitly-designed and disclosed tradeoff (`must_haves.truths` item 3, PLAN.md:136 "any transform path ... yields a timeless track"), not a residual defect. |
| WR-01 (missing `context.mounted` guard before `_saveRecordedTrack()` in the exit-dialog callback) | Guarded with `if (context.mounted) _saveRecordedTrack();` | CONFIRMED at `navigation_screen.dart:1254`. |
| WR-02 (Follow-roads always costs as pedestrian for GPS-recording sessions since `originalTrail` is always null) | Acknowledged, left as documented out-of-scope limitation | CONFIRMED still present as designed: `originalTrail = ref.read(trailProvider(widget.id)).value` resolves null for a trail-less recording session, so `costingForCategory(null)` returns `'pedestrian'` always. Inline comment at `navigation_screen.dart:741-746` documents this exact limitation. This is an accepted, disclosed gap (not silently hidden), consistent with the review's disposition. |
| IN-01 (unused Zod-inferred types drift risk) | `+server.ts` now types its `shape` array and final response with `TraceRouteShapePoint[]`/`TraceRouteResponse` | CONFIRMED at `+server.ts:108,116`. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| quick-260721-eob | 260721-eob-PLAN.md | Track-save options sheet + Valhalla trace-route proxy | SATISFIED | All 6 must-have truths, all artifacts, and all key links verified above. |

### Anti-Patterns Found

None. Scanned all newly created/modified files (`route_planner_handoff_util.dart`, `navigation_screen.dart`, `track_save_options_sheet.dart`, `+server.ts`, `valhalla_trace_route_schema.ts`) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/stub-return patterns — no matches. `flutter analyze` on the three touched Dart files surfaces only 2 pre-existing `use_build_context_synchronously` info-level lints at lines 779 and 789, confirmed (via the surrounding code) to be inherited from the pre-existing `pendingImportedTrail`/`context.pushReplacement` and error-toast pattern, not newly introduced by this task's transform logic.

### Automated Checks Re-Run By Verifier

| Check | Command | Result |
|-------|---------|--------|
| Flutter unit tests | `flutter test test/util/route_planner_handoff_util_test.dart` | 32/32 passing, including 4 `snapResultAcceptable` + 5 `fetchHeightsForShape` cases |
| Flutter analyze (3 touched files) | `flutter analyze lib/routes/navigation_screen.dart lib/components/navigation/track_save_options_sheet.dart lib/util/route_planner_handoff_util.dart` | 2 pre-existing info-level lints only, no errors/warnings |
| Svelte-check (web) | `npm run check` | 2458 files, 0 errors, 0 warnings |
| Debt-marker scan | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` on all 5 touched/created files | No matches |

### Human Verification Required

### 1. On-device save flow (exit dialog)

**Test:** Start a GPS recording on a physical device (or emulator with mock location), record for a bit, then trigger the exit dialog and tap "Save track".
**Expected:** The two-toggle sheet appears with both toggles off; confirming with no toggles enabled proceeds to `trail_create_screen` with the raw recorded breadcrumb (all points, all timestamps) unchanged from pre-task behavior.
**Why human:** Requires a live GPS/mock-location session and visual confirmation of sheet appearance/styling — not resolvable via static analysis.

### 2. On-device save flow (completion banner)

**Test:** Let a recording session complete/stop, use the completion-banner "Save" button instead of the exit dialog.
**Expected:** Same sheet appears; behaves identically to the exit-dialog path.
**Why human:** Same as above — requires a live session and visual/interactive confirmation of a second entry point sharing identical behavior.

### 3. Follow-roads produces a visibly road-snapped path

**Test:** Record an off-road-adjacent or slightly-wandering track, save with "Follow roads" ON only, inspect the resulting trail's path on the map in `trail_create_screen`/after import.
**Expected:** The saved path visibly hugs the road/trail network rather than the raw GPS jitter; point count may differ from the recorded breadcrumb (expected, per `snapResultAcceptable`'s design) but the path should span the same geographic extent as the original recording (no truncation).
**Why human:** Requires visual map inspection of an actual Valhalla `trace_route` response against a real recorded path — cannot be confirmed by reading code alone.

### 4. Recalculate-heights produces changed elevation values

**Test:** Save a track with "Recalculate heights" ON only (no follow-roads), inspect the resulting GPX/trail elevation profile against the raw recorded elevation.
**Expected:** Elevation values differ from the raw GPS-recorded elevation (typically GPS elevation is noisy; Valhalla `/height` should read as a smoother/plausible elevation profile) and the point count matches the full original breadcrumb (per the CR-01 fix — heights recalc alone should NOT change geometry/point count).
**Why human:** Requires visual/numeric comparison of elevation data pre/post transform on a real device recording — not something grep/static analysis can assert.

### 5. Cancel aborts cleanly

**Test:** Trigger save (either entry point), then dismiss the sheet via back-gesture or tapping outside it (not the confirm button).
**Expected:** No navigation occurs, no toast appears, the recording session continues exactly as if Save had never been tapped (can still resume/exit/save again).
**Why human:** Requires interactive dismiss-gesture testing and observing the app remains in its prior state — not verifiable from source alone.

### 6. Network-down silent fallback

**Test:** Disable network connectivity, then save with both toggles ON.
**Expected:** Save still completes (proceeds to `trail_create_screen`) using the raw recorded breadcrumb with no error toast — silent fallback per design.
**Why human:** Requires an actual network-disabled runtime environment to exercise the try/catch fallback paths in `snapShapeToRoads`/`fetchHeightsForShape` end-to-end; static code reading confirms the fallback logic exists but not that it behaves correctly against real Dio/Valhalla timeout behavior.

### Gaps Summary

No gaps found. All 6 observable truths, all 8 required artifacts, and all 4 key links are verified against the CURRENT state of the codebase (post-review-fix commits `ef2c7c4c` and `efe1fe5e`, not just the executor's original 3 commits). The critical CR-01 data-loss defect identified in code review (500-point truncation + timestamp loss on any transform) has been substantively fixed — the full-resolution breadcrumb is now preserved as `workingShape`'s base, `buildNavShape`'s downsampling is confined to the outbound Valhalla request hint only, and `fetchHeightsForShape` chunks height requests without capping the saved point count. This is corroborated by both direct code reading (not the SUMMARY's/REVIEW's narrative) and 5 newly-passing regression tests exercising the >500-point case. WR-01 (missing mounted guard) and IN-01 (unused types) are also confirmed fixed by direct inspection. WR-02 (pedestrian-only costing for recording-mode Follow-roads) remains a real, disclosed, out-of-scope limitation — not a regression, and explicitly acknowledged rather than silently left unexplained.

Status is `human_needed` rather than `passed` solely because this feature's core value (visual road-snapping correctness, elevation-profile correctness, sheet UX, and network-fallback behavior) can only be conclusively confirmed via on-device interaction — all six items above are genuinely un-verifiable through static code/test inspection alone, not a sign of an unresolved code gap.

---

_Verified: 2026-07-21_
_Verifier: Claude (gsd-verifier)_
