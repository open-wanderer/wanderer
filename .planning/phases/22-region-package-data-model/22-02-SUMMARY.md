---
phase: 22-region-package-data-model
plan: 02
subsystem: mobile-offline-data-model
tags: [flutter, dart, dio, objectbox, riverpod, region-catalog]

# Dependency graph
requires:
  - phase: 22-region-package-data-model (Plan 01)
    provides: "RegionCatalogEntry parse model, RegionEntity ObjectBox entity with fromCatalogEntry/applyCatalogEntry upsert mapping"
  - phase: 21.5-region-catalog-archive-pre-build-backend
    provides: "GET /api/v1/regions catalog endpoint (bare JSON array, no wrapper)"
provides:
  - "RegionCatalogException typed error for fetch/parse failures"
  - "parseRegionCatalog/fetchRegionCatalog pure+network parse functions"
  - "orphanedRegionIds pure set-math helper"
  - "RegionRepository (fetchCatalog/upsertCatalog/refreshCatalog)"
  - "regionRepositoryProvider construction-only Riverpod seam"
affects: [23-tile-repository-manager-download-engine, 24-settings-offline-maps-regions-ui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fetch-then-upsert split into two explicit steps (fetchCatalog then upsertCatalog) inside refreshCatalog, so a fetch failure is guaranteed to occur before any store write and can never corrupt persisted state"
    - "Upsert-by-id via a single runInTransaction(TxMode.write) block: find-or-create by unique business id, applyCatalogEntry on the found row, fromCatalogEntry on the not-found branch -- contrasts with subcategory_provider.dart/category_provider.dart's removeAll()+putMany() full-replace merge, which this phase explicitly does not copy"
    - "Construction-only Riverpod provider seam: the provider function's body is exactly a constructor call reading two existing keepAlive providers (apiProvider, objectBoxProvider) -- no fetch/refresh/microtask in build(), unlike subcategory_provider's build()-triggers-refresh shape"

key-files:
  created:
    - app/lib/provider/region/region_repository.dart
    - app/lib/provider/region/region_repository.g.dart
    - app/test/provider/region_repository_test.dart
  modified: []

key-decisions:
  - "Malformed elements are dropped at two independent layers: parseRegionCatalog catches RegionCatalogEntry.fromJson failures per-element, and upsertCatalog additionally catches FormatException from RegionEntity.fromCatalogEntry/applyCatalogEntry's bbox-length guard per-entry -- neither layer can abort the whole batch"
  - "orphan detection runs after all upserts complete in the same transaction, querying box.getAll() post-upsert rather than pre-upsert, so an entry that both already existed and is still present in the fetch is never miscounted as an orphan"

patterns-established:
  - "fetch-then-upsert two-step split (RegionRepository.refreshCatalog = upsertCatalog(await fetchCatalog())) as the sanctioned shape for any future catalog-style refresh that must never corrupt local state on a failed fetch"

requirements-completed: [REGN-01]

# Metrics
duration: 12min
completed: 2026-07-22
---

# Phase 22 Plan 02: Region Repository (Fetch + Upsert) Summary

**RegionRepository.refreshCatalog() GETs /api/v1/regions through the cookie-authenticated Dio client, parses the bare JSON array while dropping malformed elements, and upserts by business id inside a single write transaction -- preserving every region's ToOne package links and local download status, with orphaned regions flipped inCatalog=false rather than deleted**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-22 (this session)
- **Completed:** 2026-07-22
- **Tasks:** 2 completed
- **Files modified:** 3 (2 created source/test, 1 generated)

## Accomplishments
- `RegionCatalogException` typed error carries a message and optional cause, thrown by `fetchRegionCatalog`/`RegionRepository.fetchCatalog` on any network or parse failure -- never silently swallowed (D-03)
- `parseRegionCatalog` maps the bare `/api/v1/regions` JSON array into `RegionCatalogEntry` values, dropping individual malformed elements rather than aborting the whole parse (T-22-05); a non-`List` payload throws instead of silently returning an empty list
- `fetchRegionCatalog` calls `api.get('/regions')` through the existing cookie-authenticated `apiProvider` Dio client (no manual auth header) and wraps any `DioException`/parse failure in `RegionCatalogException`, preserving the original error as `cause`
- `orphanedRegionIds` is a pure set-math helper: persisted region ids not present in the latest fetch
- `RegionRepository.upsertCatalog` runs a single `runInTransaction(TxMode.write, ...)`: find-or-create by business id, `applyCatalogEntry` on an existing row (preserving `obxId`, both `ToOne` targets, `lastDownloadedVersion`) or `RegionEntity.fromCatalogEntry` for a new row; a per-entry `FormatException` (malformed bbox) is caught and that entry skipped, not fatal. Contains zero `removeAll(` calls (T-22-06, D-01)
- After upserting, every persisted region absent from the fetch is flipped `inCatalog = false` via `box.put(...)` -- no row deletion, no file access (D-08)
- `RegionRepository.refreshCatalog()` composes `fetchCatalog()` then `upsertCatalog()` -- because the fetch fully completes or throws before any write, a failed/offline fetch always leaves every persisted row untouched
- `regionRepositoryProvider` (`@Riverpod(keepAlive: true) RegionRepository regionRepository(Ref ref)`) is a construction-only seam: its body is exactly `return RegionRepository(ref.watch(apiProvider), ref.watch(objectBoxProvider));` -- no fetch/refresh/microtask call, unlike `subcategory_provider.dart`'s `build()`-triggers-refresh shape (D-02)
- 9 new unit tests all green covering `parseRegionCatalog` (full+minimal, valid+malformed, empty, non-List), `fetchRegionCatalog` (success + DioException->RegionCatalogException via a fake-Dio interceptor harness), and `orphanedRegionIds` set math
- `dart analyze` clean (0 errors; 2 pre-existing `unnecessary_import` info-level notices on the deliberately-explicit `objectbox/objectbox.dart` import, per the plan's literal import list) on both new files; `flutter analyze` reports 0 errors across the whole package

## Task Commits

Each task was committed atomically:

1. **Task 1: RegionRepository + typed error + pure parse/orphan helpers** - `3461185f` (feat, TDD)
2. **Task 2: regionRepository provider seam + phase build-health gate** - `6e7ed023` (feat)

## Files Created/Modified
- `app/lib/provider/region/region_repository.dart` - `RegionCatalogException`, `parseRegionCatalog`, `fetchRegionCatalog`, `orphanedRegionIds`, `RegionRepository`, `regionRepositoryProvider`
- `app/lib/provider/region/region_repository.g.dart` - riverpod_generator codegen output for `regionRepositoryProvider`
- `app/test/provider/region_repository_test.dart` - parse/fetch/orphan unit tests with a fake-Dio interceptor harness (no ObjectBox native store harness available in this repo, so `upsertCatalog`/`refreshCatalog`'s store-backed glue is exercised only via `dart analyze` + Plan 01's `applyCatalogEntry`/`fromCatalogEntry` preservation tests, per the plan's own note)

## Decisions Made
- Followed 22-CONTEXT.md decisions verbatim (D-01, D-02, D-03, D-08); no new gray areas surfaced during implementation
- Kept the explicit `import 'package:objectbox/objectbox.dart';` alongside `import 'package:wanderer/objectbox.g.dart';` even though `dart analyze` flags it as an unnecessary (info-level, non-blocking) import -- matches the plan's literal action-text import list

## Deviations from Plan

None - plan executed exactly as written. All acceptance criteria matched verbatim: `region_repository.dart` contains `class RegionCatalogException implements Exception`, top-level `parseRegionCatalog`/`fetchRegionCatalog`/`orphanedRegionIds`, and `class RegionRepository`; no `removeAll(` and no bare error-swallowing catch; `upsertCatalog` calls `applyCatalogEntry`/`RegionEntity.fromCatalogEntry` inside a single `runInTransaction(TxMode.write` block and sets `inCatalog = false` for orphans with no `box.remove`; the provider body is exactly the one-line construction call; `region_repository.g.dart` exists and defines `regionRepositoryProvider`; `flutter analyze` reports 0 errors.

## Issues Encountered

`dart run build_runner build` regenerated `app/lib/provider/route_anchor_provider.g.dart` and `app/lib/provider/router_provider.g.dart` again as a side effect of running codegen across the whole `app/` tree -- these are the same pre-existing generated files outside this plan's `files_modified` scope noted in Plan 01's SUMMARY (no source changes causing them). Left untouched/unstaged per the scope-boundary rule; the working tree also carries unrelated pre-existing uncommitted changes to `db/main.go` and three `web/src/routes/api/v1/regions/**` files from earlier unrelated work, which were likewise never staged or touched by this plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 23 (TileRepositoryManager -- Download Engine) can now call `RegionRepository.refreshCatalog()` (or the pure `fetchRegionCatalog`/`parseRegionCatalog` helpers directly) to keep the local region catalog in sync, and can rely on `upsertCatalog`'s upsert-by-id contract to never sever a `ToOne` package link it has already wired
- Phase 24 (Settings UI) decides when/where `regionRepositoryProvider`'s `refreshCatalog()` is actually invoked (e.g. on Settings/Regions screen open) -- this plan deliberately does not wire any call site (D-02)
- No blockers - app builds and runs unchanged; nothing reads `regionRepositoryProvider` yet, matching the plan's "purely additive, not wired to any call site" objective

---
*Phase: 22-region-package-data-model*
*Completed: 2026-07-22*

## Self-Check: PASSED

All 3 created/generated files (region_repository.dart, region_repository.g.dart, region_repository_test.dart) and the SUMMARY.md verified present on disk. Both commits (3461185f, 6e7ed023) verified present in git log.
