---
phase: quick-261014-nqn
plan: 02
subsystem: app-gpx
tags: [flutter, gpx, route-planner, wanderer-anchor]
requires:
  - phase: quick-261014-nqn plan 01
    provides: web stored-route format (one marked trkseg, xmlns:wanderer)
provides:
  - parseGpxSafely splits a web-saved marked segment into the planner's per-leg layout
  - serializePlannedRoute stores planner routes as one marked segment
affects: [route planner, trail create/edit, trail export]
key-files:
  created:
    - app/lib/util/gpx/route_anchor_marks.dart
    - app/test/util/gpx/route_anchor_marks_test.dart
  modified:
    - app/lib/util/gpx/conversion.dart
    - app/lib/util/route/planner_handoff.dart
    - app/test/util/route/planner_handoff_test.dart
decisions:
  - buildDraftTrail gets a named plannedRoute bool selecting between two top-level serializers for compute()
  - mergeRouteIntoTrail switches unconditionally to serializePlannedRoute
  - import_trail_file.dart left unchanged (see follow-ups)
status: complete
requirements: [ANCHOR-05, ANCHOR-06, ANCHOR-07, ANCHOR-08]
commits: 2
plan_head_before: f23b3966cb86cc5701b6a832f97ee2229eabb3f3
plan_head_after: 2ffb6b401b9238669fd2d74d10582b39845fca63
---

# Quick 261014-nqn Plan 02: App reads and writes the marked-segment route format

The app now parses a web-saved single marked trkseg back into per-leg segments and stores planner-built routes as one marked trkseg with `xmlns:wanderer` on the root.

## Commits (local, not pushed)

1. `7bfe34f92` fix(app): restore route legs from wanderer:anchor marks when parsing GPX (cherry-pick of d2ed6049c, message unchanged, no adjustment needed)
2. `2ffb6b401` fix(app): store planned routes as one marked track segment

## Verification

- `flutter analyze`: baseline "13 issues found." before, "13 issues found." after (all pre-existing, in vendor/tiptap_flutter etc.).
- `flutter test test/util/gpx/ test/util/route/planner_handoff_test.dart`: all passed.
- Full `flutter test`: All tests passed (+1315, ~1 skipped).
- RED was observed for both new test groups before implementation.

## What changed

- `route_anchor_marks.dart`: added `routeAnchorNamespace`, a private `_copyWpt` (all 21 Wpt fields), and top-level `serializePlannedRoute(Gpx)`. It builds a copy with one Trkseg per Trk, marks the first point of every non-empty leg after the first on a copied Wpt, sets `xmlns:wanderer` only if something was marked, and never mutates its input. Doc comments now describe per-leg in memory vs one marked segment stored.
- `planner_handoff.dart`: `buildDraftTrail` has `plannedRoute` (default false); `finishPlanning` passes true; `mergeRouteIntoTrail` uses `serializePlannedRoute(finalGpx)`. Doc comments for `buildFinalPlannedGpx`, `anchorsFromTrack`, `mergeRouteIntoTrail` updated.
- Tests: serializePlannedRoute group (S1-S7: one segment + namespace, round-trip with ele/time, no mutation, interior 1-point leg, nothing-to-mark byte equality, empty legs skipped, re-store of a web file), plus H1-H4 in planner_handoff_test.dart (planned vs recording storage, mergeRouteIntoTrail, storage round-trip of buildFinalPlannedGpx).

## Serialization-site audit

1. `planner_handoff.dart` buildDraftTrail: shared by finishPlanning and the navigation_screen recording save, so made conditional via `plannedRoute`. navigation_screen.dart is untouched and keeps the plain writer.
2. `planner_handoff.dart` mergeRouteIntoTrail: switched unconditionally.
3. `import_trail_file.dart` (~188): unchanged.

## Deviations from Plan

- [Rule 3 - Blocking] The test file needed `import 'package:gpx/gpx.dart' hide GpxReader;` because it also imports the vendored `GpxReader` (name clash), and a local `_points` helper replaced the plan's `allPoints` (it is a method on a mapping util, not a top-level function). Test-only, no behavior effect.
- `dart format` reformatted two unrelated tests in planner_handoff_test.dart; those hunks were reverted so the diff holds only this plan's changes.

## Follow-ups (not done here)

1. import_trail_file.dart ~188: with the "recalculate heights" toggle on, a web-saved drawn route is split by parseGpxSafely, mergeHeightsIntoGpx keeps the split legs, and serializeGpxToXml stores them as unmarked app-layout segments, so the web's export join cannot rejoin them. The default import path uploads the original text with marks intact. Fixing it needs a split variant that reports whether it split, so the import can choose serializePlannedRoute without marking a file's real gaps.
2. A trailing 1-point leg cannot be marked (readers ignore marks on a segment's last point); shared with the web, only arises for a degenerate final polyline.
3. The Task 3 human check (plan 3+ anchors, save, open in web editor, download GPX, re-open in planner, edit-mode round trip, record with a pause gap) was not run; the user builds and installs.

## Known Stubs

None.

## Threat Flags

None.

## Self-Check: PASSED

- Both commits exist and are ancestors of HEAD; HEAD subjects are single-line with no trailer.
- Commit 2 touches exactly the four planned files; navigation_screen.dart and import_trail_file.dart are unchanged.
- No build, install, or push was run.
