---
phase: 24-settings-offline-maps-regions-ui
plan: 03
subsystem: mobile-offline-maps
tags: [flutter, dart, disk_space_2, tdd]

# Dependency graph
requires:
  - phase: 24-settings-offline-maps-regions-ui (plan 02)
    provides: SettingsOfflineRegionsScreen, the UI surface where the diagnosed disk-space blocker was found in on-device UAT
  - phase: 23-tilerepositorymanager-download-engine
    provides: freeDiskSpaceBytes/hasEnoughSpace (TILE-03 fail-closed disk-space gate), TileRepositoryManager.startVectorDownload/startDemDownload call sites
provides:
  - "resolveFreeDiskSpaceBytes — injectable orchestrator implementing the path -> device-wide disk-space fallback disk_space_util.dart's own doc comment already promised"
  - "freeDiskSpaceBytes now actually falls back to a device-wide query when the path-specific query throws (e.g. a never-downloaded region's regions/<id>/ directory doesn't exist yet)"
affects: [phase-25-map-rendering, phase-26-trail-download-guard]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Async fail-closed plugin wrapper split into a thin real-plugin-wiring function (freeDiskSpaceBytes) and a pure, injectable, unit-testable orchestrator (resolveFreeDiskSpaceBytes) taking PathSpaceQuery/DeviceSpaceQuery closures — lets device-only fallback/retry logic be exercised deterministically under flutter test with no platform channel"

key-files:
  created: []
  modified:
    - app/lib/util/disk_space_util.dart
    - app/test/util/disk_space_util_test.dart
    - app/pubspec.yaml
    - app/pubspec.lock

key-decisions:
  - "meta promoted from dependency_overrides-only to a direct pubspec dependency — the new @visibleForTesting orchestrator imports package:meta/meta.dart directly, and flutter analyze flagged the override-only state as depend_on_referenced_packages"

requirements-completed: [SETUI-03, SETUI-04]

# Metrics
duration: ~6min
completed: 2026-07-22
---

# Phase 24 Plan 03: Gap-Closure — Device-Wide Disk-Space Fallback Summary

**Implemented the device-wide disk-space fallback `freeDiskSpaceBytes`'s own doc comment already promised but never delivered, fixing the Phase 24 UAT blocker where every first-ever region download (vector or DEM) was refused on a real device because the path-specific disk-space query throws on a `regions/<id>/` directory that doesn't exist yet.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-07-22T15:57:00+02:00 (approx, plan doc commit)
- **Completed:** 2026-07-22T16:02:53+02:00
- **Tasks:** 1 completed (TDD: RED then GREEN)
- **Files modified:** 4 (2 plan-scoped, 2 pubspec files as a necessary dependency fix)

## Accomplishments
- `resolveFreeDiskSpaceBytes`: new `@visibleForTesting` orchestrator taking injectable `pathQuery`/`deviceQuery` closures, implementing the fallback ordering — path query first (when a path is given), falls back to the device-wide query on any thrown exception, returns `null` (fail-closed) only when both fail
- `freeDiskSpaceBytes` reduced to a thin wrapper wiring the real `disk_space_2` plugin calls (`DiskSpace.getFreeDiskSpaceForPath`, `DiskSpace.getFreeDiskSpace`) into the orchestrator — its documented signature (`[String? forPath]`) is unchanged, so both `tile_repository_manager.dart` call sites compile and behave unchanged
- Six new unit tests covering the fallback ordering deterministically (path succeeds/device never called, path throws→device fallback, both throw→null, no-path uses device directly, no-path+device-throws→null, null-mebibytes→null-bytes) — all pass, plus all 5 pre-existing `hasEnoughSpace` tests still pass (no regression)
- `tile_repository_manager.dart` confirmed untouched (`git diff --stat` empty for that file) — the fix lives entirely in the shared wrapper as the plan required

## Task Commits

TDD RED then GREEN, per `tdd="true"`:

1. **Task 1 RED: add failing resolveFreeDiskSpaceBytes fallback tests** - `b9e07055` (test)
2. **Task 1 GREEN: implement device-wide disk-space fallback (SETUI-03/04)** - `8be39c5a` (feat)

No REFACTOR commit — the GREEN implementation was already clean (`flutter analyze` clean on first pass after adding `meta` as a direct dependency); no further cleanup needed.

## Files Created/Modified
- `app/lib/util/disk_space_util.dart` - added `PathSpaceQuery`/`DeviceSpaceQuery` typedefs and `resolveFreeDiskSpaceBytes`; `freeDiskSpaceBytes` reduced to a thin wrapper delegating to it; doc comments updated to describe the now-implemented fallback
- `app/test/util/disk_space_util_test.dart` - added a `resolveFreeDiskSpaceBytes` test group (6 tests) with injected fake closures; existing `hasEnoughSpace` group untouched
- `app/pubspec.yaml` - added `meta: ^1.18.0` under `dependencies:` (was previously only in `dependency_overrides:`)
- `app/pubspec.lock` - regenerated via `flutter pub get` after the `meta` dependency change (meta's resolved entry changes from `direct overridden` to `direct main`)

## Decisions Made
- **`meta` promoted to a direct dependency:** the plan's action text assumed `meta` was "already a project dependency," but it was only present in `dependency_overrides:` (pinning a transitive version), not `dependencies:`. Importing `package:meta/meta.dart` directly for `@visibleForTesting` triggered `flutter analyze`'s `depend_on_referenced_packages` lint. Fixed by adding `meta: ^1.18.0` to `dependencies:` (Rule 3 — blocking issue, required for the plan's own acceptance criterion "`flutter analyze` on both files reports no new issues" to pass) and running `flutter pub get` to regenerate the lockfile.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `meta` as a direct pubspec dependency**
- **Found during:** Task 1 (post-implementation `flutter analyze`)
- **Issue:** The plan's action text stated "`meta` is already a project dependency," but it was only present in `dependency_overrides:`. `flutter analyze` flagged `lib/util/disk_space_util.dart:2:8 • depend_on_referenced_packages` for the new `import 'package:meta/meta.dart'` needed by `@visibleForTesting`.
- **Fix:** Added `meta: ^1.18.0` under `dependencies:` in `app/pubspec.yaml`, ran `flutter pub get` to regenerate `pubspec.lock`.
- **Files modified:** `app/pubspec.yaml`, `app/pubspec.lock`
- **Verification:** `flutter analyze lib/util/disk_space_util.dart test/util/disk_space_util_test.dart` reports "No issues found!"
- **Committed in:** `8be39c5a` (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 blocking dependency-declaration fix)
**Impact on plan:** Necessary for the plan's own `flutter analyze` acceptance criterion to pass; no behavior change, no scope creep — `meta`'s resolved version (`^1.18.0`) is identical to what the pre-existing override already pinned.

## Issues Encountered
None beyond the `meta` dependency fix documented above.

The plan's `<human-check>` (on-device physical verification of UAT tests 2 and 3 for a never-before-downloaded region) is deferred to end-of-phase per this project's `human_verify_mode: end-of-phase` config — not performed by this automated execution pass.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The root-cause fix for the diagnosed Phase 24 UAT blocker (`region-download-diskspace.md`) is implemented and unit-tested; `tile_repository_manager.dart`'s existing create-directory-after-check ordering required no changes.
- Automated verification is complete: `flutter test test/util/disk_space_util_test.dart` (11/11 pass) and `flutter analyze` (clean) both pass.
- Outstanding before Phase 24 can be considered fully signed off: the end-of-phase on-device human-check pass (this plan's Task 1 `<human-check>`, folded into the existing six-point checklist from 24-02) — specifically re-running UAT tests 2 and 3 on a physical device against a never-before-downloaded region, and confirming UAT tests 4 and 5 (previously `blocked_by: prior-phase`) now unblock.
- Nothing in this plan touches map rendering or trail-download-guard code; Phase 25 can proceed once the on-device pass confirms this fix on real hardware.

---
*Phase: 24-settings-offline-maps-regions-ui*
*Completed: 2026-07-22*

## Self-Check: PASSED

Both plan-scoped files confirmed present on disk (`app/lib/util/disk_space_util.dart`, `app/test/util/disk_space_util_test.dart`); both task commit hashes (`b9e07055`, `8be39c5a`) confirmed present in `git log`.
