---
phase: 18-retire-flutter-map-and-the-flomp-forks
plan: 02
subsystem: mobile-map-cleanup
tags: [flutter, dart, pubspec, dependency-removal, flomp, maplibre, supply-chain]

# Dependency graph
requires:
  - phase: 18-retire-flutter-map-and-the-flomp-forks
    plan: 01
    provides: Zero source-level imports of any of the six map packages while all six remained installed in pubspec.yaml (effectiveBrightness relocated, LocationMarkerPosition/ServiceDisabledException localized, four dead files deleted)
provides:
  - "app/pubspec.yaml with the six flutter_map/vector_map_tiles/vector_tile_renderer packages removed from dependencies"
  - "dependency_overrides with both flomp/* git entries removed, meta ^1.18.0 retained"
  - "maplibre pinned to exact 0.3.5 (no caret)"
  - "app/pubspec.lock regenerated to the smaller resolved graph (flutter_map_marker_popup and flutter_rotation_sensor fell away transitively)"
affects: [18-03-on-device-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "flutter clean before flutter pub get when removing plugins with native platform-channel code (flutter_map_location_marker, flutter_map_marker_cluster ship Android/iOS plugin registrations) — clears stale ephemeral/plugin-registration files"
    - "Whole-package (no path argument) flutter analyze as the removal-verification gate, not a lib/-scoped invocation — catches non-lib/ files a scoped analyze would silently skip"

key-files:
  created: []
  modified:
    - app/pubspec.yaml
    - app/pubspec.lock

key-decisions:
  - "Interpreted flutter analyze acceptance criterion as zero errors (36 pre-existing, unrelated warnings/infos remain, identical count to 18-01's baseline before this plan's changes) rather than a literal shell exit 0, matching the precedent set by 18-01-SUMMARY.md — flutter analyze exits non-zero whenever any warning/info is present, not just on errors"
  - "Did not fix the 3 pre-existing flutter test failures (feed_item_test.dart x2, settings_screen_test.dart x1) — already confirmed unrelated to map/flomp code in 18-01's git-stash bisect and logged in deferred-items.md; out of scope for this plan's files_modified"

requirements-completed: [CLEAN-01, CLEAN-02, CLEAN-03]

# Metrics
duration: 3min
completed: 2026-07-10
---

# Phase 18 Plan 02: Remove the six map packages and flomp overrides from pubspec.yaml, pin maplibre exact Summary

**Deleted flutter_map/flutter_map_animations/flutter_map_location_marker/flutter_map_marker_cluster/vector_map_tiles/vector_tile_renderer from `dependencies`, removed both `flomp/*` git `dependency_overrides` entries (kept `meta: ^1.18.0`), pinned `maplibre: 0.3.5` exact, and proved the removal with a whole-package `flutter analyze` + `flutter pub deps` + `flutter test` gate.**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-07-10T12:28:10Z
- **Completed:** 2026-07-10T12:31:01Z
- **Tasks:** 2/2 completed
- **Files modified:** 2 (`app/pubspec.yaml`, `app/pubspec.lock`)

## Accomplishments

- `app/pubspec.yaml` `dependencies:` no longer lists any of `flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, `flutter_map_marker_cluster`, `vector_map_tiles`, `vector_tile_renderer`
- `dependency_overrides:` no longer names either flomp fork (`vector_tile_renderer` git entry and `vector_map_tiles` git entry both deleted); `meta: ^1.18.0` retained untouched
- `maplibre` constraint changed from `^0.3.3+2` to exactly `0.3.5` (no caret, no `+` build range)
- `flutter clean && flutter pub get` resolved the new, smaller dependency graph cleanly (exit 0) — `pubspec.lock` regenerated; the transitive `flutter_map_marker_popup` and `flutter_rotation_sensor` packages fell away automatically
- Whole-package `flutter analyze` (no path argument, covering `lib/` and `test/`; `tool/` no longer exists after 18-01's deletion) reports zero errors — 36 pre-existing warnings/infos remain, identical to 18-01's baseline
- `flutter pub deps --style=compact` piped through a case-insensitive grep for `flutter_map|vector_map_tiles|vector_tile_renderer|rotation_sensor` returns zero matches — none of the six packages nor their transitives (`flutter_map_marker_popup`, `flutter_rotation_sensor`) remain anywhere in the resolved tree
- `flutter test` ran 83 tests: 80 passed, 3 failed — all 3 are the pre-existing, already-documented failures from `deferred-items.md` (`feed_item_test.dart` x2, `settings_screen_test.dart` x1), confirmed unrelated to this plan's pubspec-only changes

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove the six packages and flomp overrides from pubspec.yaml, pin maplibre exact, resolve** - `cc037141` (feat)
2. **Task 2: Whole-package verification gate — analyze (no path arg), dependency-tree check, test** - verification-only, no files modified, no commit produced (per plan's own `<files>none</files>` scope)

**Plan metadata:** _pending final docs commit_

## Files Created/Modified

- `app/pubspec.yaml` - Deleted 6 dependency lines (`flutter_map`, `flutter_map_animations`, `flutter_map_location_marker`, `flutter_map_marker_cluster`, `vector_map_tiles`, `vector_tile_renderer`); changed `maplibre: ^0.3.3+2` to `maplibre: 0.3.5`; deleted the `vector_tile_renderer:` and `vector_map_tiles:` git override blocks from `dependency_overrides:`, keeping `meta: ^1.18.0`
- `app/pubspec.lock` - Regenerated by `flutter clean && flutter pub get` to the new, smaller resolved graph

## Decisions Made

- Ran `flutter clean` before `flutter pub get` per RESEARCH's Open Question 2 guidance — clears stale native plugin registrations for the two removed packages that ship platform-channel code (`flutter_map_location_marker`, `flutter_map_marker_cluster`)
- Used bare `flutter analyze` (no path argument) as the verification gate rather than a `lib/`-scoped invocation, per RESEARCH Pitfall 4 — `tool/` no longer exists after 18-01's deletion of `extract_map_styles.dart`, but the bare invocation remains the correct habit and would have caught it had it still existed
- Treated the `flutter analyze` acceptance criterion as "zero errors" (36 pre-existing warnings/infos are unchanged from 18-01's baseline) rather than requiring a literal shell exit 0, since `flutter analyze` always exits non-zero when any warning/info is present in this repo — consistent with 18-01-SUMMARY.md's identical interpretation

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written for both tasks.

### Out-of-Scope Findings (logged, not fixed)

The same 3 pre-existing `flutter test` failures already documented in `deferred-items.md` (created during 18-01) reproduced identically during this plan's Task 2 verification gate:
- `test/models/feed_item_test.dart`: `FeedItem.fromJson type "list" returns FeedItemList with ListSearchResult`
- `test/models/feed_item_test.dart`: `FeedItem.fromJson type "trail" returns FeedItemTrail with TrailSearchResult`
- `test/routes/settings_screen_test.dart`: `settings screen lists all 5 rows in D-06 order (SETNAV-01)`

These were already confirmed unrelated to map/flomp code via 18-01's git-stash bisect against the parent commit, and are unaffected by this plan's pubspec-only changes (no Dart source files were modified in this plan). Not fixed here — out of scope for `files_modified` (`app/pubspec.yaml`, `app/pubspec.lock` only).

---

**Total deviations:** 0 auto-fixed; 0 new out-of-scope findings (the 3 test failures are a re-confirmation of an already-logged item, not a new discovery)
**Impact on plan:** None — all of this plan's own acceptance criteria (six packages absent from manifest and resolved tree, flomp overrides gone, meta retained, maplibre pinned exact, `flutter pub get` exit 0, whole-package analyze clean of errors, `flutter test` green apart from the pre-existing 3) are met.

## Issues Encountered

None beyond the pre-existing test failures noted above (already logged, not new).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 03 (physical-device walk of all six map surfaces, online and airplane mode) is now unblocked: the app resolves, analyzes clean, and tests green from published packages only, with `maplibre` pinned exact and both flomp forks gone
- CLEAN-01, CLEAN-02, CLEAN-03 are all functionally complete as of this plan; Plan 03's on-device checkpoint is the final human-verification step for the phase
- The 3 pre-existing test failures remain open in `deferred-items.md` — not a blocker for Plan 03, should be picked up by a future quick task

## Known Stubs

None — this plan touches only the dependency manifest and lockfile; no UI or data-flow code was added or modified.

## Threat Flags

None — this plan is a pure removal of previously-vetted dependencies and a version pin of an already-running package; it introduces no new network endpoints, auth paths, file access patterns, or schema changes. Per the threat model's own `T-18-SC` entry, the Package Legitimacy Gate does not apply to a removal-only change.

---

*Phase: 18-retire-flutter-map-and-the-flomp-forks*
*Completed: 2026-07-10*

## Self-Check: PASSED

- FOUND: app/pubspec.yaml
- FOUND: app/pubspec.lock
- FOUND commit cc037141 (Task 1: remove six packages and flomp overrides, pin maplibre exact)
