---
phase: 24-settings-offline-maps-regions-ui
plan: 01
subsystem: mobile-offline-maps
tags: [flutter, riverpod, objectbox, l10n, dart]

# Dependency graph
requires:
  - phase: 22-region-package-data-model
    provides: RegionEntity, DownloadedTilePackageEntity, region_file_path.dart path builders
  - phase: 23-tile-repository-manager-download-engine
    provides: TileRepositoryManager, TileRepositoryStatus, deleteRegion (full cascade delete precedent)
provides:
  - "TileRepositoryManager.deleteDemPackage(regionId) — DEM-only cascade delete (D-01)"
  - "TileRepositoryStatus.deleteDemPackage(regionId) — notifier wrapper with ephemeral-state cleanup"
  - "regionListNotifierProvider — synchronous A-Z ObjectBox snapshot of RegionEntity"
  - "formatBytes(int) — human-readable byte formatting"
  - "regionDiskUsageBytes / totalRegionDiskUsageBytes — real on-disk byte aggregation incl. .part files"
  - "18 Phase 24 English l10n keys in app_en.arb, generated across all 14 locales"
affects: [24-02, phase-25-map-rendering]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "riverpod_generator's default class-name-derived provider naming strips a trailing 'Notifier' suffix (TrailLibraryNotifier -> trailLibraryProvider); use @Riverpod(name: '...') with the full desired identifier (verbatim, no auto-appended 'Provider') to override when a specific provider symbol name is load-bearing for a later plan"

key-files:
  created:
    - app/lib/util/byte_format_util.dart
    - app/lib/util/region_disk_usage_util.dart
    - app/test/util/byte_format_util_test.dart
    - app/test/util/region_disk_usage_util_test.dart
  modified:
    - app/lib/services/tile_repository_manager.dart
    - app/lib/provider/region/tile_repository_provider.dart
    - app/lib/provider/region/region_provider.dart
    - app/lib/provider/region/region_provider.g.dart
    - app/lib/i18n/app_en.arb

key-decisions:
  - "regionListNotifierProvider named explicitly via @Riverpod(name: 'regionListNotifierProvider') because the default riverpod_generator naming convention would have produced regionListProvider (Notifier-suffix stripping), which would have broken Plan 02's already-written literal references"
  - "Collapsed settings_offline_regions_tile into settings_offline_regions_title (plan's own action text permitted this when copy is identical, and Plan 02 only references the _title key)"

patterns-established:
  - "region_disk_usage_util.dart: final-file-else-.part-else-zero per package, always routed through assertValidRegionId/regionVectorPath/regionDemPath, never string-concatenated"

requirements-completed: [SETUI-01, SETUI-02, SETUI-04, SETUI-05]

# Metrics
duration: 15min
completed: 2026-07-22
---

# Phase 24 Plan 01: Non-UI Foundation for Offline Maps/Regions Summary

**DEM-only cascade delete, an A-Z synchronous region-list provider, real-on-disk byte formatting/aggregation utilities (incl. `.part` partial files), and all 18 Phase 24 English l10n keys — the full symbol contract Plan 02's screen is written against.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-22T11:25:00Z (approx)
- **Completed:** 2026-07-22T11:41:21Z
- **Tasks:** 3 completed
- **Files modified:** 9 (2 new lib files, 2 new test files, 5 modified; plus 14 generated `app_localizations_*.dart` locale files)

## Accomplishments
- `TileRepositoryManager.deleteDemPackage`/`TileRepositoryStatus.deleteDemPackage`: DEM-only removal that provably leaves the vector package, its file, and `lastDownloadedVersion` untouched (D-01/T-24-02)
- `regionListNotifierProvider`: synchronous, A-Z-by-name ObjectBox snapshot mirroring `TrailLibraryNotifier`'s established shape (D-09)
- `formatBytes`/`regionDiskUsageBytes`/`totalRegionDiskUsageBytes`: unit-tested byte formatting and real-on-disk usage aggregation that correctly counts `.part` partial files for downloading/paused packages (D-06, Pitfall 1)
- All 18 Phase 24 English l10n strings added to `app_en.arb`, copy sourced verbatim from `24-UI-SPEC.md`, and `flutter gen-l10n` regenerated cleanly across all 14 locales

## Task Commits

Each task was committed atomically:

1. **Task 1: DEM-only delete on TileRepositoryManager + TileRepositoryStatus (D-01)** - `7b725b2b` (feat)
2. **Task 2: regionListNotifierProvider snapshot + byte-format + disk-usage utilities** (TDD) - `7e09fcf4` (test, RED) → `93889b4f` (feat, GREEN)
3. **Task 3: Phase 24 English l10n keys** - `726cad10` (feat)

_Note: Task 2 is a TDD task — RED (`7e09fcf4`) then GREEN (`93889b4f`), both commits verified present in `git log` before the GREEN commit was made._

## Files Created/Modified
- `app/lib/services/tile_repository_manager.dart` - added `deleteDemPackage(regionId)`: selective `'$id:dem'` cancel, DEM-only row/file removal
- `app/lib/provider/region/tile_repository_provider.dart` - added `TileRepositoryStatus.deleteDemPackage(regionId)` notifier wrapper
- `app/lib/provider/region/region_provider.dart` - added `RegionListNotifier` (`@Riverpod(name: 'regionListNotifierProvider')`)
- `app/lib/provider/region/region_provider.g.dart` - generated `regionListNotifierProvider`
- `app/lib/util/byte_format_util.dart` - new `formatBytes(int)` utility
- `app/lib/util/region_disk_usage_util.dart` - new `regionDiskUsageBytes`/`totalRegionDiskUsageBytes` utilities
- `app/test/util/byte_format_util_test.dart` - 6 unit tests
- `app/test/util/region_disk_usage_util_test.dart` - 5 unit tests
- `app/lib/i18n/app_en.arb` - 18 new keys (2 with ICU placeholder metadata blocks)
- `app/lib/i18n/app_localizations*.dart` (15 files) - regenerated by `flutter gen-l10n`

## Decisions Made
- **`regionListNotifierProvider` naming override:** `riverpod_generator`'s default class-name-derived naming strips a trailing "Notifier" from the class name before appending "Provider" (confirmed against the existing `TrailLibraryNotifier` -> `trailLibraryProvider` precedent already in the codebase). A class literally named `RegionListNotifier` would therefore generate `regionListProvider`, not the `regionListNotifierProvider` symbol the plan's own text and the already-written Plan 02 depend on verbatim in multiple places. Rather than rename the class to something awkward (e.g. a doubled `RegionListNotifierNotifier`), used `@Riverpod(name: 'regionListNotifierProvider')` — `riverpod_annotation`'s `name` parameter is used as-is (no automatic "Provider" suffix), confirmed by testing (`name: 'regionListNotifier'` alone generated `regionListNotifier`, not `regionListNotifierProvider`). This keeps the class named cleanly (`RegionListNotifier`, satisfying the must_haves artifact contains-check) while producing the exact provider identifier Plan 02 requires.
- **Collapsed `settings_offline_regions_tile` into `settings_offline_regions_title`:** the plan's own action text explicitly permitted this ("executor may collapse to one key if identical"); confirmed Plan 02's action text only references `l10n.settings_offline_regions_title` for both the AppBar title and the Settings tile label, so no second key was needed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected the plan's assumed `regionListNotifierProvider` generation mechanism**
- **Found during:** Task 2 (regionListNotifierProvider snapshot)
- **Issue:** The plan's action text assumed a plain `@riverpod class RegionListNotifier extends _$RegionListNotifier { ... }` would generate a provider named `regionListNotifierProvider`. Running `dart run build_runner build` with that exact code produced `regionListProvider` instead — `riverpod_generator` strips a trailing "Notifier" suffix from the class name when deriving the default provider identifier, a behavior already visible elsewhere in this codebase (`TrailLibraryNotifier` -> `trailLibraryProvider`) but not accounted for in the plan's text.
- **Fix:** Added `@Riverpod(name: 'regionListNotifierProvider')` to the class annotation instead of the bare `@riverpod` shorthand, producing the exact symbol name Plan 02 depends on, verified by grepping the generated `region_provider.g.dart`.
- **Files modified:** `app/lib/provider/region/region_provider.dart`, `app/lib/provider/region/region_provider.g.dart`
- **Verification:** `grep -q "regionListNotifierProvider" lib/provider/region/region_provider.g.dart` passes; `flutter analyze` clean
- **Committed in:** `93889b4f` (Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug fix, naming mechanism correction — no scope change, no behavior change beyond the exact generated symbol name)
**Impact on plan:** Necessary correction to keep the exact provider-name contract Plan 02 already depends on; no scope creep.

## Issues Encountered
None beyond the deviation documented above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plan 02 can now build `SettingsOfflineRegionsScreen` against a confirmed-working `regionListNotifierProvider`, `TileRepositoryManager.deleteDemPackage`/`TileRepositoryStatus.deleteDemPackage`, `formatBytes`, `regionDiskUsageBytes`/`totalRegionDiskUsageBytes`, and every l10n key its action text references (`settings_offline_regions_title`, `regions_search_hint`, `regions_dem_toggle_label`/`_caption`, `regions_update_available`/`_action`, `regions_retry`, `regions_not_yet_available`, `regions_build_failed`, `regions_delete_confirm_title`/`_body`/`_action`, `regions_disk_usage_summary`, `regions_empty_search_title`/`_body`, `regions_empty_catalog_title`/`_body`).
- No blockers identified for Plan 02.

---
*Phase: 24-settings-offline-maps-regions-ui*
*Completed: 2026-07-22*

## Self-Check: PASSED

All 8 created/modified files confirmed present on disk; all 4 task commit hashes (`7b725b2b`, `7e09fcf4`, `93889b4f`, `726cad10`) confirmed present in `git log`.
