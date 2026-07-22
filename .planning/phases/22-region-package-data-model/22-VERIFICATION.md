---
phase: 22-region-package-data-model
verified: 2026-07-22T00:00:00Z
status: passed
score: 4/4 must-haves verified
overrides_applied: 0
---

# Phase 22: Region & Package Data Model Verification Report

**Phase Goal:** The app has a region manifest — fetched from Phase 21.5's catalog API, not bundled — and an ObjectBox schema for regions and their downloadable tile packages — the foundation every later phase in this milestone builds on.
**Verified:** 2026-07-22
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | App fetches region catalog from backend API at runtime and parses it into a typed manifest model (id/name/bbox/vector URL+size/optional DEM URL+size) | ✓ VERIFIED | `app/lib/provider/region/region_repository.dart:58-67` `fetchRegionCatalog(Dio api)` calls `api.get('/regions')` (resolves to SvelteKit-proxied `/api/v1/regions`) and parses via `parseRegionCatalog`. `app/lib/models/region_catalog_entry.dart` is a `@freezed` model with `id`, `name`, `bbox` (`List<double>`), `status`, nullable `version`/`vectorUrl`/`vectorSize`/`demStatus`/`demUrl`/`demSize`/`error`, matching `db/routes/regions_get.go`'s conditional field construction. `flutter test test/models/region_catalog_entry_test.dart` (4 tests) and `test/provider/region_repository_test.dart` (9 tests) pass, covering full-ready/minimal-building/error fixtures and a fake-Dio fetch-success/fetch-failure harness. |
| 2 | ObjectBox `Region` entity persists every fetched-catalog field plus live status (notDownloaded/downloading/downloaded/updateAvailable) backed by explicit stable int constants (never `Enum.values[index]`), surviving app restart | ✓ VERIFIED | `app/lib/entities/region_entity.dart` — `RegionEntity` is `@Entity()` (ObjectBox-persisted, not in-memory), holds catalog fields (`name`, `minLon/minLat/maxLon/maxLat`, `version`, `vectorUrl/vectorSize`, `demUrl/demSize`, `error`), and `RegionStatus get status` (lines 103-119) computed from `vectorPackage.target?.status`. `dbCatalogStatus`/`dbDemStatus` shadow getters/setters (lines 59-66, 75-82) read/write `.code` via `firstWhere` value-lookup with safe fallback — confirmed zero `.index` or `Enum.values[` usage in the file (`region_status.dart` also has no `.index`). `objectbox-model.json` shows `RegionEntity` registered as a real persisted entity (id 12) with `dbCatalogStatus`/`dbDemStatus` as int-typed properties (type 6) — durable across restarts since it's an ObjectBox-managed field, not `@Transient()`. Tests in `test/entities/region_entity_test.dart` (18 tests) assert all 4 status-getter states including `updateAvailable`, and out-of-range `.code` fallback behavior. |
| 3 | ObjectBox `DownloadedTilePackage` entity tracks vector and DEM packages independently (separate path/timestamp/size/status per package) | ✓ VERIFIED | `app/lib/entities/downloaded_tile_package_entity.dart` — standalone `@Entity()` with its own `status`/`localFilePath`/`downloadedAtUtc`/`sizeBytesOnDisk`. `RegionEntity` links two independent instances via `final vectorPackage = ToOne<DownloadedTilePackageEntity>();` and `final demPackage = ToOne<DownloadedTilePackageEntity>();` (region_entity.dart:86-87) — no discriminator field, no shared row, confirming a region can show vector downloaded while DEM is not. `objectbox-model.json` registers `DownloadedTilePackageEntity` (id 11) as a distinct persisted entity with its own property set, and two separate relation entries (`vectorPackage`/`demPackage`) targeting it from `RegionEntity`. |
| 4 | App builds and runs unchanged — nothing yet reads from the new entities | ✓ VERIFIED | `flutter analyze` across the whole package reports 0 errors (46 pre-existing info-level notices, none in region files except one unrelated `unnecessary_import` info notice). `grep` for `RegionEntity`/`DownloadedTilePackageEntity`/`regionRepositoryProvider`/`RegionCatalogEntry` usage outside their own definition/generated files and `test/` returns zero matches — no other app code consumes these new artifacts yet. `objectbox-model.json` diff since before this phase shows only additive entity blocks (ids 11/12) and bumped `lastEntityId`/`lastIndexId` counters; no existing entity's properties were altered or removed. |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/models/region_status.dart` | `CatalogStatus`/`RegionStatus`/`PackageStatus` enhanced enums with `.code` | ✓ VERIFIED | All 3 enums present, each with `final int code` and explicit integer literals; no `.index` usage. |
| `app/lib/models/region_catalog_entry.dart` (+ `.freezed.dart`/`.g.dart`) | freezed parse model for `/api/v1/regions` element | ✓ VERIFIED | `factory RegionCatalogEntry.fromJson` present; generated files exist on disk (verified via `ls`). |
| `app/lib/entities/downloaded_tile_package_entity.dart` | ObjectBox `DownloadedTilePackageEntity` with `.code` shadow | ✓ VERIFIED | `int get dbStatus => status.code;` present with safe-fallback setter. |
| `app/lib/entities/region_entity.dart` | ObjectBox `RegionEntity`: catalog fields, computed status getter, `ToOne` links, `fromCatalogEntry`/`applyCatalogEntry` | ✓ VERIFIED | All present; `applyCatalogEntry` overwrites only catalog-owned fields, leaves `obxId`/`vectorPackage`/`demPackage`/`lastDownloadedVersion` untouched (matches D-01). |
| `app/lib/provider/region/region_repository.dart` (+ `.g.dart`) | `RegionRepository`, `RegionCatalogException`, fetch/parse/upsert/orphan functions, construction-only provider | ✓ VERIFIED | All present; `regionRepositoryProvider` body is exactly `return RegionRepository(ref.watch(apiProvider), ref.watch(objectBoxProvider));` — no fetch on build (D-02). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `region_catalog_entry.dart` | `CatalogStatus` | `@JsonValue`-annotated enum decode | ✓ WIRED | `status`/`demStatus` fields typed `CatalogStatus`/`CatalogStatus?`, decoded through json_serializable's generated `.g.dart`. |
| `region_entity.dart` | `downloaded_tile_package_entity.dart` | `ToOne` relation | ✓ WIRED | `ToOne<DownloadedTilePackageEntity>()` x2, confirmed registered as ObjectBox relations in `objectbox-model.json`. |
| `region_entity.dart` | `region_catalog_entry.dart` | `fromCatalogEntry`/`applyCatalogEntry` mapping | ✓ WIRED | Both methods map every catalog field 1:1, with bbox-length validation (`FormatException`). |
| `region_repository.dart` | `apiProvider` | `ref.watch(apiProvider)` | ✓ WIRED | Present in provider body. |
| `region_repository.dart` | `objectBoxProvider` | `ref.watch(objectBoxProvider)` | ✓ WIRED | Present in provider body. |
| `region_repository.dart` | `RegionEntity.applyCatalogEntry` | upsert merge | ✓ WIRED | `upsertCatalog` calls `existing.applyCatalogEntry(entry)` on found rows and `RegionEntity.fromCatalogEntry(entry)` on new rows, inside a single `runInTransaction(TxMode.write, ...)` block; zero `removeAll(` calls confirmed by grep-equivalent code read. |

### Data-Flow Trace (Level 4)

Not applicable — this phase is explicitly "purely additive, nothing reads these entities yet" (confirmed by Truth #4's zero-consumer grep). No UI or downstream component renders this data yet, so there is no rendering data-flow to trace. The relevant flow to verify instead is fetch → parse → persist, which is covered by the unit tests (fake-Dio harness exercising the full `fetchRegionCatalog` path, and `applyCatalogEntry`/`fromCatalogEntry` preservation tests exercising the persist path).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| All new/changed unit tests pass | `flutter test test/models/region_status_test.dart test/models/region_catalog_entry_test.dart test/entities/region_entity_test.dart test/provider/region_repository_test.dart` | 38 tests, all passed | ✓ PASS |
| Static analysis clean | `flutter analyze` | 0 errors, 46 pre-existing info-level notices (deprecated icon members, vendor code style) unrelated to this phase | ✓ PASS |
| Generated codegen artifacts present | `ls lib/models/region_catalog_entry.freezed.dart lib/models/region_catalog_entry.g.dart lib/provider/region/region_repository.g.dart` | All 3 files exist | ✓ PASS |
| ObjectBox schema additive only | `git diff HEAD~10 -- lib/objectbox-model.json` | Only two new entity blocks (ids 11/12) added; no existing entity's properties removed/altered; only `lastEntityId`/`lastIndexId` counters bumped | ✓ PASS |
| No file outside `app/` touched by phase commits | `git log --oneline` (10 phase-22 commits) + `git diff --stat` on each | All 10 commits touch only `app/lib/**`, `app/test/**`, and `.planning/` docs | ✓ PASS |
| Pre-existing unrelated uncommitted changes left untouched | `git diff db/main.go`, `git diff web/src/routes/api/v1/regions/**` | Diffs are Phase 21.5 routing-comment corrections (predate Phase 22, not touched by any 22-0x commit) | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this phase and none are referenced in PLAN/SUMMARY files. Skipped — no runnable probes declared or conventional.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|--------------|------------|--------------|--------|----------|
| REGN-01 | 22-01-PLAN.md, 22-02-PLAN.md | Typed manifest model + fetch function for the catalog API | ✓ SATISFIED | `RegionCatalogEntry.fromJson` + `RegionRepository.fetchCatalog`/`refreshCatalog` implemented and tested. |
| REGN-02 | 22-01-PLAN.md | ObjectBox `Region` entity, explicit-int status enums, restart-durable | ✓ SATISFIED | `RegionEntity` + `region_status.dart` enums, `.code` shadow pattern verified, zero `.index`/positional-lookup usage. |
| REGN-03 | 22-01-PLAN.md | `DownloadedTilePackage` entity tracks vector/DEM independently | ✓ SATISFIED | `DownloadedTilePackageEntity` + two independent `ToOne` links on `RegionEntity`. |

No orphaned requirements found — REGN-01/02/03 are the full requirement set mapped to Phase 22 in ROADMAP.md, and all three are claimed and satisfied across the two plans.

### Anti-Patterns Found

None. No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any of the five phase-22 source files (`region_status.dart`, `region_catalog_entry.dart`, `downloaded_tile_package_entity.dart`, `region_entity.dart`, `region_repository.dart`). No `.index` or `Enum.values[` positional-lookup anti-pattern present (the exact anti-pattern REGN-02 was designed to avoid). No empty stub implementations, no hardcoded-empty return values feeding a render path (there is no render path yet — confirmed purely additive).

### Human Verification Required

None. This phase has no UI (explicitly "zero UI" per 22-CONTEXT.md), no runtime behavior visible to a user, and no external service integration beyond a unit-testable Dio call already covered by a fake-Dio harness. All success criteria are verifiable via static analysis, ObjectBox schema diffing, and automated tests.

### Gaps Summary

No gaps. All 4 roadmap success criteria are verified against the actual codebase (not just SUMMARY.md claims): the typed parse model exists and round-trips through tests; the ObjectBox `RegionEntity` persists catalog fields plus a computed status via explicit `.code` int constants; `DownloadedTilePackageEntity` tracks vector/DEM packages independently via two separate `ToOne` links; and the app builds/analyzes cleanly with zero consumers of the new entities, confirming the "purely additive" claim. The three pre-existing uncommitted files (`db/main.go`, `web/src/routes/api/v1/regions/**`) were confirmed untouched by any Phase 22 commit — they carry over from Phase 21.5 work as expected.

---

*Verified: 2026-07-22*
*Verifier: Claude (gsd-verifier)*
