---
phase: 34-dart-conversion-port
plan: 06
subsystem: mobile-app
tags: [flutter, dart, riverpod, dio, offline-first, gpx, valhalla, route-planner]

# Dependency graph
requires:
  - phase: 34-dart-conversion-port
    plan: 05
    provides: "transcodeToGpx, buildLocalTrail, buildDraftTrail - all three capture paths build their draft Trail entirely on-device via trailFromGpx"
provides:
  - "resolveTrackSaveOptions(WidgetRef, BuildContext) in track_save_options_util.dart - the single online gate around showTrackSaveOptionsSheet, shared by all three capture paths (D-15)"
  - "buildFinalPlannedGpx(WidgetRef, {refetchAllHeights, snapCosting}) - per-leg road-snap with mandatory anchor re-pin, then per-leg height refetch, both skippable/off by default"
  - "finishPlanning({..., recalcHeights, followRoads}) - forwards the resolved toggles into buildFinalPlannedGpx, deriving snapCosting from the session's own travel bucket"
  - "route_planner_screen.dart's _onFinish resolving the shared gate ahead of the forward-push branch, with the prior offline error-toast dead end removed (D-16)"
  - "trail_import_util.dart's importTrailFile gated by the same sheet, offline/declined-both importing a .gpx end to end with zero HTTP requests"
affects: [34-07]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single connectivity gate covering two independently-toggled network transforms (recalcHeights, followRoads) — resolveTrackSaveOptions reads onlineStatusProvider once and returns (false, false) for both 'offline' and 'declined both online', so callers never need to distinguish the two outcomes"
    - "Snap-then-heights ordering, mirrored across all three capture paths (navigation_screen.dart's _saveRecordedTrack was already this way; route_planner_handoff_util.dart and trail_import_util.dart now match it) so elevation always reflects the final (possibly snapped) shape"
    - "Exact-coordinate anchor re-pin after a network transform: any step that can move the boundary of a `trkseg`-per-leg structure (here, road-snapping) must force the leg's first/last point back onto the pre-transform anchor coordinate, because anchorsFromTrack/segmentPolylinesFromTrack locate anchors by exact match, not nearest-point"
    - "Literal-substring grep-gate discipline continues from 34-05: doc comments referencing `showTrackSaveOptionsSheet`, `resolveTrackSaveOptions`, `onlineStatusProvider` or `trail/convert` must avoid the exact string a plan's own acceptance-criteria grep counts, or they trip it themselves — every doc comment added in this plan was worded to reference the symbol descriptively (e.g. 'the bottom sheet defined in track_save_options_sheet.dart') rather than repeating the literal identifier"

key-files:
  created:
    - app/lib/util/track_save_options_util.dart
    - app/test/util/track_save_options_util_test.dart
  modified:
    - app/lib/routes/navigation_screen.dart
    - app/lib/routes/route_planner_screen.dart
    - app/lib/util/route_planner_handoff_util.dart
    - app/lib/util/trail_import_util.dart
    - app/test/util/route_planner_handoff_util_test.dart
    - app/test/util/trail_import_util_test.dart

key-decisions:
  - "route_planner_screen.dart's _onFinish now sets `_finishing` separately inside each branch (immediately for edit-mode, after the online gate resolves for the forward-push branch) rather than once before the try block, so the gate can run ahead of the guard for the branch that needs it (D-15) while edit-mode's ordering is functionally unchanged."
  - "Added a `navContext.mounted` guard between the online gate resolving and the subsequent `setState`/`buildDraftTrail` work in both route_planner_screen.dart and trail_import_util.dart (Rule 2: the reference implementation, _saveRecordedTrack, does not have this guard at that exact point, but both of these call sites can legitimately outlive the awaited sheet - e.g. the user backgrounds the app or pops the screen mid-sheet - and calling setState/using a stale context after that would throw)."
  - "trail_import_util.dart now imports route_planner_handoff_util.dart (for snapShapeToRoads/fetchHeightsForShape/mergeHeightsIntoGpx) while route_planner_handoff_util.dart already imports trail_import_util.dart (for buildLocalTrail/pendingImportedTrail) — a legal Dart circular import between two files in the same package; verified with `flutter analyze` producing no errors or import-cycle diagnostics."
  - "A snapped leg's cached elevations are invalidated unconditionally (set to null) as soon as its points are replaced, regardless of `refetchAllHeights` — this reuses buildFinalPlannedGpx's existing 'fetch only what's missing' pending-list logic to guarantee a snapped leg is always re-heighted, without needing a second flag or a separate code path."

patterns-established:
  - "resolveTrackSaveOptions as the mandatory single call site for showTrackSaveOptionsSheet — any future capture path must route through it, not call the sheet directly."

requirements-completed: [PORT-03]

# Metrics
duration: ~26min
completed: 2026-08-01
---

# Phase 34 Plan 06: Extend showTrackSaveOptionsSheet to Planner and Import, Fix Offline Dead End Summary

**`resolveTrackSaveOptions` is now the single, online-only gate in front of `showTrackSaveOptionsSheet` for all three capture paths — recording (unchanged behaviour), route planner (offline dead end removed, D-16), and file import (now offers the same two toggles, and completes offline with zero network calls) — with the planner's road-snap step re-pinning each leg's boundary onto its original anchor so multi-anchor routes survive a snap/re-edit round trip.**

## Performance

- **Duration:** ~26 min
- **Started:** 2026-08-01T09:46:13+02:00
- **Completed:** 2026-08-01T10:11:57+02:00
- **Tasks:** 3/3 completed
- **Files modified:** 8 (2 new, 6 modified)

## Accomplishments

- New `app/lib/util/track_save_options_util.dart`: `resolveTrackSaveOptions(WidgetRef, BuildContext)` reads `onlineStatusProvider` once and either returns `(false, false)` immediately (offline, no UI touched) or shows `showTrackSaveOptionsSheet` and returns its result verbatim (online) — D-15's single code path for "offline" and "user declined both toggles online"
- `navigation_screen.dart`'s `_saveRecordedTrack` now calls `resolveTrackSaveOptions` instead of `showTrackSaveOptionsSheet` directly; every other line of the method (the `null`-abort, the `_savingTrack` double-tap guard placed after it, the transform branch, the catch-and-toast, the `finally`) is byte-identical to before this plan
- `buildFinalPlannedGpx` (`route_planner_handoff_util.dart`) gained `refetchAllHeights`/`snapCosting`, both defaulting off and both fully skipping their network step when off. When `snapCosting` is set, each leg with ≥2 points is snapped via the trace-route proxy and then has its first/last point forced back onto its pre-snap anchor coordinates — the mandatory re-pin that keeps `anchorsFromTrack`/`segmentPolylinesFromTrack` able to reconstruct the route on re-edit (T-34-28). A snapped leg's elevations are invalidated so the existing "fetch what's missing" logic re-heights it regardless of `refetchAllHeights`. Snapping always runs before heights, matching the recording path's ordering.
- `finishPlanning` gained `recalcHeights`/`followRoads`, forwarding them into `buildFinalPlannedGpx` as `refetchAllHeights`/`snapCosting` — the costing string for `snapCosting` comes from the session's own `bucketForState`, falling back to `'pedestrian'` (the same fallback `costingForTrail` itself uses)
- `route_planner_screen.dart`'s `_onFinish` resolves the shared gate ahead of the `_finishing` guard in the forward-push (non-edit-mode) branch only, aborting on a cancelled/dismissed sheet with no state change; the edit-mode branch (pop back to the caller, no persisted trail) is unchanged. Offline, the finish now reaches `trail_create_screen` instead of the previous error-toast dead end — D-16 is fixed by removing the offline path's unconditional network calls (34-05's local conversion + this plan's flag-gated transforms), not by deleting the `catch`, which remains as a genuine last-resort guard
- `trail_import_util.dart`'s `importTrailFile` now resolves the shared gate right after parsing the GPX and before building the local trail. `(false, false)` (offline or declined-both) uses the parsed track unchanged with zero network calls — the single most important branch, since it is what makes an offline `.gpx` import work end to end. When either toggle is on, the function mirrors `_saveRecordedTrack`'s pipeline (full-resolution shape, snap before heights, original first/last trkpt times preserved) and re-serialises the transformed `Gpx` so the saved `gpxData` matches the trail's own computed metrics rather than the untransformed original text
- Verified end to end (new permanent test, `trail_import_util_test.dart`): with `onlineStatusProvider` forced false and an `apiProvider` mock that rejects every request, importing a `.gpx` performs zero HTTP requests and still produces a trail with non-empty `gpxData`

## Task Commits

1. **Task 1: Add the shared online gate and route the recording path through it** - `07794d24` (feat)
2. **Task 2: Gate the route planner's post-capture transforms and remove its offline dead end** - `90f29cfd` (feat)
3. **Task 3: Gate the file-import path with the same sheet** - `2918ad6f` (feat)

## Files Created/Modified

- `app/lib/util/track_save_options_util.dart` - new: `resolveTrackSaveOptions`, the single online gate around `showTrackSaveOptionsSheet`
- `app/test/util/track_save_options_util_test.dart` - new: offline-skip, online-confirm (both untouched and both toggled), online-dismiss
- `app/lib/routes/navigation_screen.dart` - `_saveRecordedTrack` routed through `resolveTrackSaveOptions`; doc comments reworded to avoid re-tripping the plan's own literal-substring grep gates
- `app/lib/routes/route_planner_screen.dart` - `_onFinish`'s forward-push branch gated by `resolveTrackSaveOptions` ahead of `_finishing`; offline dead end removed (D-16); edit-mode branch untouched
- `app/lib/util/route_planner_handoff_util.dart` - `buildFinalPlannedGpx` gained `refetchAllHeights`/`snapCosting` with the mandatory per-leg anchor re-pin after a snap; `finishPlanning` gained `recalcHeights`/`followRoads` and derives `snapCosting` from the session's bucket
- `app/lib/util/trail_import_util.dart` - `importTrailFile` gated by `resolveTrackSaveOptions`; optional snap/height transform pipeline mirroring the recording path; re-serialises the transformed GPX before handing it to `buildLocalTrail`
- `app/test/util/route_planner_handoff_util_test.dart` - two new tests: a snapped leg whose endpoints differ from the anchors still round-trips to the original anchors via `anchorsFromTrack`; both flags off against an api that throws on any request completes with zero requests for a route whose legs already have elevations
- `app/test/util/trail_import_util_test.dart` - new `importTrailFile` group: offline end-to-end import performs zero HTTP requests and still produces a trail

## Decisions Made

See `key-decisions` in the frontmatter above (the `_finishing` restructuring, the added `mounted` guards, the trail_import_util.dart ↔ route_planner_handoff_util.dart circular import, and the unconditional elevation-invalidation-on-snap).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] Added a `navContext.mounted`/`context.mounted` guard immediately after the online gate resolves, before the subsequent `setState`/local-conversion work, in both `route_planner_screen.dart` and `trail_import_util.dart`**
- **Found during:** Task 2 and Task 3, while mirroring `_saveRecordedTrack`'s exact ordering
- **Issue:** `_saveRecordedTrack` (the reference implementation) does not guard `context.mounted` between the awaited sheet and the subsequent `setState`/work, but both of these call sites can legitimately outlive the awaited sheet (backgrounding, popping the screen mid-sheet) in ways the recording screen's own lifecycle makes less likely — calling `setState` on an unmounted `State` or using a stale `BuildContext` after that would throw
- **Fix:** Added `if (!mounted) return;` (route planner) / `if (!navContext.mounted) return;` (import) immediately after the `options == null` cancel-check, before any further state mutation
- **Files modified:** `app/lib/routes/route_planner_screen.dart`, `app/lib/util/trail_import_util.dart`
- **Verification:** `flutter analyze` clean on both files; existing and new tests pass
- **Committed in:** `90f29cfd` (Task 2), `2918ad6f` (Task 3)

**2. [Rule 3 - Blocking] A test written to exercise `importTrailFile`'s cancelled-sheet path hung indefinitely and was removed rather than fixed in place**
- **Found during:** Task 3's own test authoring, before committing
- **Issue:** `importTrailFile` performs a real `File.readAsString()` (requiring `tester.runAsync`) before showing the sheet, but interacting with the sheet's dismiss barrier requires the fake-clock `pumpAndSettle()` path; combining both in one test hung the test runner (confirmed via an isolated `--plain-name` run that timed out and had to be killed)
- **Fix:** Removed the hanging test. The cancellation contract (`options == null` aborts with no state change) is still covered: unit-tested directly for `resolveTrackSaveOptions` in `track_save_options_util_test.dart`, and the identical `if (options == null) return;` idiom is shared verbatim across all three call sites. The plan's own acceptance criteria for Task 3 do not require a dedicated cancellation test for `importTrailFile` — only the offline end-to-end behaviour check, which is covered by a permanent test
- **Files modified:** `app/test/util/trail_import_util_test.dart` (test added then removed within the same task, before commit — no hanging test ever reached a commit)
- **Verification:** `flutter test test/util/trail_import_util_test.dart` completes in ~2s with 9/9 passing, no hang
- **Committed in:** `2918ad6f` (Task 3) — the final committed version never contained the hanging test

**3. [Rule 1 - Bug] Doc comments referencing `onlineStatusProvider`, `resolveTrackSaveOptions`, and `showTrackSaveOptionsSheet` initially re-tripped the plan's own literal-substring acceptance-criteria greps**
- **Found during:** Task 1's own acceptance-criteria verification, and the plan's overall `<verification>` block re-checked after Task 3
- **Issue:** `track_save_options_util.dart`'s doc comment used `[onlineStatusProvider]` and `[showTrackSaveOptionsSheet]`/`` `showTrackSaveOptionsSheet` `` inline, and `navigation_screen.dart`'s doc comment used `[resolveTrackSaveOptions]` — each adding a second literal match beyond the intended single code occurrence the plan's grep gates count
- **Fix:** Reworded all three to reference the symbol descriptively (e.g. "the app's connectivity status (see `online_status_provider.dart`)", "the bottom sheet defined in `track_save_options_sheet.dart`", "the shared online gate (see `track_save_options_util.dart`)") with no content loss
- **Files modified:** `app/lib/util/track_save_options_util.dart`, `app/lib/routes/navigation_screen.dart`
- **Verification:** All acceptance-criteria and plan-level `<verification>` grep commands re-run and passing exactly (see Verification Performed below)
- **Committed in:** `07794d24` (Task 1)

---

**Total deviations:** 3 auto-fixed (1 Rule 1, 1 Rule 2, 1 Rule 3)
**Impact on plan:** All three were required to satisfy either correctness (the `mounted` guards prevent a genuine crash class) or the plan's own acceptance gates; no scope creep and no runtime behaviour changed beyond what the plan specified. The removed hanging test cost no committed history — it never reached a commit.

## Issues Encountered

None beyond the deviations above, both caught and resolved during this plan's own task-level verification before committing.

## Verification Performed

- `cd app && flutter analyze` (full) - no errors; the same 33 pre-existing info-level deprecated-icon lints documented in 34-05's baseline, none in files touched by this plan
- `cd app && flutter test` (full suite) - 580 pass, 1 skip, 4 fail; the 4 failures are the documented pre-existing `test/components/route_planner/settings_tab_test.dart` icon-lookup failures (verified identical at the phase's pre-existing baseline), no new failures
- `cd app && flutter test test/util/track_save_options_util_test.dart` - 4/4 pass
- `cd app && flutter test test/util/route_planner_handoff_util_test.dart` - 46/46 pass (44 pre-existing + 2 new)
- `cd app && flutter test test/util/trail_import_util_test.dart` - 9/9 pass (8 pre-existing + 1 new)
- `grep -rn "showTrackSaveOptionsSheet" app/lib/` - exactly 2 lines: the declaration in `track_save_options_sheet.dart` and the single call inside `resolveTrackSaveOptions`
- `grep -rn "resolveTrackSaveOptions" app/lib/` - exactly 4 lines: the declaration plus one call each in `navigation_screen.dart`, `route_planner_screen.dart`, `trail_import_util.dart`
- `git status --porcelain app/lib/components/navigation/track_save_options_sheet.dart` - empty (byte-unchanged, D-15 is reuse not new UI)
- `grep -rn "trail/convert" app/lib/` - exactly 1 line, in `trail_import_util.dart`'s `transcodeToGpx` (PORT-03 unbroken)
- Task 1: `grep -c "onlineStatusProvider" app/lib/util/track_save_options_util.dart` = 1; `grep -c "if (options == null) return;"` / `grep -c "if (_savingTrack) return;"` in `navigation_screen.dart` = 1 each, in that order
- Task 2: `grep -c "bool refetchAllHeights = false"` / `"String? snapCosting"` / `"bool recalcHeights = false"` / `"bool followRoads = false"` in `route_planner_handoff_util.dart` = 1 each; `grep -c "if (_finishing) return;"` in `route_planner_screen.dart` = 1; the new snap-re-pin test asserts `anchorsFromTrack` returns the original anchor coordinates and `trksegs.length == legs.length` after a snap whose response deliberately differs from every real anchor; the new offline test asserts zero trace-route/height calls for a route whose legs already carry elevations
- Task 3: `grep -c "resolveTrackSaveOptions"` = 1; `grep -c "snapShapeToRoads\|fetchHeightsForShape\|mergeHeightsIntoGpx"` = 3; `grep -c "showError()"` = 3 (declaration + 2 call sites, unchanged from before); the new offline-import test asserts zero HTTP requests and a non-empty `gpxData`

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three capture paths (recording, route planner, file import) now share one online-only gate around `showTrackSaveOptionsSheet`; connectivity is decided in exactly one place
- D-15 and D-16 are both closed: offline behaviour is uniform across all three sources (skip the sheet, both toggles off, zero network calls), and the planner's prior offline dead end is gone
- The planner's per-leg `trkseg` structure survives a road-snap: the mandatory anchor re-pin (T-34-28) is implemented and covered by a dedicated round-trip test
- `track_save_options_sheet.dart` remains byte-unchanged — no new UI was needed, only new call sites
- No blockers for 34-07

---
*Phase: 34-dart-conversion-port*
*Completed: 2026-08-01*
