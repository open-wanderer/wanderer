---
phase: 10-category-subcategory-data-layer
plan: 04
subsystem: flutter-providers
tags: [riverpod, objectbox, cache-first, category, subcategory, codegen, cleanup]
dependency_graph:
  requires:
    - phase: 10-01
      provides: "Category/Subcategory freezed models"
    - phase: 10-02
      provides: "CategoryEntity/SubcategoryEntity ObjectBox entities"
  provides:
    - "CategoryNotifier persists every /category fetch into ObjectBox (overwrite-all, D-02)"
    - "SubcategoryNotifier — cache-first keepAlive provider: synchronous cached read on build, background /subcategory refresh, cache overwrite, state update (CAT-04, D-03/D-04)"
    - "Settings model + SettingsEntity with the deprecated category field removed (CAT-05, D-09)"
  affects:
    - "Phase 11 (trail filters read cached subcategories on startup)"
    - "Phase 12 (Settings Categories screen)"
tech_stack:
  added: []
  patterns:
    - "Cache-first synchronous build(): box.getAll() -> map(toModel) returned immediately, background refresh via Future.microtask"
    - "Overwrite-all cache write: box.removeAll() + box.putMany(items.map(Entity.fromModel)) inside try block so failed fetch keeps prior cache"
    - "Background refresh swallows errors (try/catch) for offline-resilient startup; foreground fetch (CategoryNotifier) rethrows"
    - "Compiler-driven field removal: delete field + codegen regen + flutter analyze to find call sites, not a blind file-list edit"
key_files:
  created:
    - app/lib/provider/trail/subcategory_provider.dart
    - app/lib/provider/trail/subcategory_provider.g.dart
  modified:
    - app/lib/provider/trail/category_provider.dart
    - app/lib/provider/trail/category_provider.g.dart
    - app/lib/models/settings.dart
    - app/lib/models/settings.freezed.dart
    - app/lib/models/settings.g.dart
    - app/lib/entities/settings_entity.dart
    - app/lib/objectbox-model.json
    - app/lib/objectbox.g.dart
decisions:
  - "objectbox import dropped from category_provider — box type is inferred via ref.read(objectBoxProvider).box<CategoryEntity>(), so the direct objectbox import was unused (analyzer warning)"
  - "SubcategoryNotifier refresh swallows errors (vs CategoryNotifier rethrow) so a missing network at startup leaves cached rows usable (T-10-09)"
metrics:
  duration_min: 6
  completed: 2026-06-29
requirements: [CAT-04, CAT-05]
---

# Phase 10 Plan 04: Category & Subcategory Data Layer (Provider Integration) Summary

Wired the category data layer end to end: `CategoryNotifier` now overwrites all `CategoryEntity` rows on every successful `/category` fetch, a new cache-first `SubcategoryNotifier` returns cached subcategories synchronously on startup then refreshes from `/subcategory` in the background, and the deprecated `Settings.category` favourite-sport field was removed from the model and entity with a clean compiler-driven sweep.

## What Was Built

- **`CategoryNotifier` ObjectBox write-on-fetch** — after the existing `ListResult.fromJson(...).items` parse, the notifier reads `ref.read(objectBoxProvider).box<CategoryEntity>()`, calls `box.removeAll()` then `box.putMany(items.map(CategoryEntity.fromModel).toList())`, all inside the existing try block so a failed fetch leaves the prior cache intact (D-02). The `/category` ListResult parse is unchanged.
- **`SubcategoryNotifier`** (new, `@Riverpod(keepAlive: true)`, generated `subcategoryProvider`) — synchronous `build()` returning `List<Subcategory>`: reads the cache first via `box.getAll().map((e) => e.toModel())`, kicks off a non-blocking background `_refresh()` via `Future.microtask`, and returns the cached list immediately (D-03, CAT-03). `_refresh()` GETs `/subcategory`, parses the ListResult, overwrites the box (`removeAll` + `putMany`), and sets `state = items`. Refresh failures are swallowed so startup stays usable offline (T-10-09).
- **`Settings.category` removal** — deleted the `String? category` field from the `Settings` freezed factory, and the `category` field + ctor param + `fromModel`/`toModel` lines from `SettingsEntity` (CAT-05, D-09). Regenerated codegen; the ObjectBox generator emitted `Property SettingsEntity.category not found in the code, removing from the model` and dropped the property from `objectbox-model.json` (no migration script needed — ObjectBox discards it on next app open).

## Tasks

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Make CategoryNotifier write to ObjectBox on every successful fetch | 3a28e52c | category_provider.dart, category_provider.g.dart |
| 2 | Create cache-first SubcategoryNotifier provider | 16c6c1bd | subcategory_provider.dart (+ .g) |
| 3 | Remove Settings.category and run the compiler-driven sweep | a00ad085 | settings.dart (+ .freezed/.g), settings_entity.dart, objectbox-model.json, objectbox.g.dart |

## Verification

- `cd app && dart run build_runner build --delete-conflicting-outputs` — completed without errors for all three tasks; objectbox generator logged the category property removal.
- `dart analyze lib/provider/trail/category_provider.dart` and `lib/provider/trail/subcategory_provider.dart` — No issues found.
- `flutter analyze` (whole-app sweep after Task 3) — 0 errors. The 38 remaining items are all pre-existing `info`-level `deprecated_member_use` entries in `lib/util/icon_util.dart` and one unused-import warning in `test/models/feed_item_test.dart`, none related to the Settings.category removal (out of scope, left untouched).
- `grep -c 'String? category' lib/models/settings.dart` → 0.
- `grep -c 'category' lib/entities/settings_entity.dart` → 0.
- `git diff app/lib/objectbox-model.json` — the `"name": "category"` SettingsEntity property line is removed.
- Unrelated `TrailFilter.category` / `GlobalSearchState.category` / `trail.expand.category` files (`trail_filter_provider.dart`, `global_search_*`, `trail_quick_filter_bar.dart`, `trail_filter_screen.dart`, `trail_entity.dart`) are NOT in any task diff (RESEARCH Pitfall 1 confirmed — only `settings_entity.dart`/`settings.dart` referenced `Settings.category`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed unused `objectbox` import from category_provider.dart**
- **Found during:** Task 1
- **Issue:** The plan's action listed `import 'package:objectbox/objectbox.dart';` among the imports to add, but the box is obtained via `ref.read(objectBoxProvider).box<CategoryEntity>()` with an inferred type, so the direct objectbox import was unused — the analyzer flagged it as a warning.
- **Fix:** Dropped the `package:objectbox/objectbox.dart` import; kept `objectbox_store_provider.dart` and `category_entity.dart`. `dart analyze` then reported no issues.
- **Files modified:** app/lib/provider/trail/category_provider.dart
- **Commit:** 3a28e52c

## Known Stubs

None — both notifiers are wired to live API endpoints and the ObjectBox cache; no placeholder data or empty hardcoded returns.

## Threat Surface

No new threat surface beyond the plan's `<threat_model>`. T-10-09 (`/subcategory` unreachable at startup) mitigated — cache-first build returns prior rows immediately and `_refresh()` failures are swallowed. T-10-08 / T-10-10 accepted as designed (reference data only; ObjectBox drops the removed property on next open). No package installs (T-10-SC not applicable) — only existing `build_runner` codegen.

## Self-Check: PASSED

- FOUND: app/lib/provider/trail/subcategory_provider.dart
- FOUND: app/lib/provider/trail/subcategory_provider.g.dart
- FOUND: app/lib/provider/trail/category_provider.dart
- FOUND commit 3a28e52c
- FOUND commit 16c6c1bd
- FOUND commit a00ad085

---
*Phase: 10-category-subcategory-data-layer*
*Completed: 2026-06-29*
