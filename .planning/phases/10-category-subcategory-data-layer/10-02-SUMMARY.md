---
phase: 10-category-subcategory-data-layer
plan: 02
subsystem: database
tags: [objectbox, flutter, category, subcategory, codegen, json-blob]
requires:
  - phase: 10-01
    provides: "Category (icon/shortName/translations) and Subcategory freezed models"
provides:
  - "CategoryEntity extended with icon, shortName, translationsJson JSON-blob column"
  - "SubcategoryEntity with @Index @Unique id + @Index parent category, JSON-blob translations"
  - "Regenerated objectbox-model.json/objectbox.g.dart schema including SubcategoryEntity"
affects:
  - "Phase 10 Plan 04 (category/subcategory providers read entities on startup, overwrite on refresh)"
  - "Phase 11 (trail filters query subcategories by parent category)"
  - "Phase 12 (Settings Categories screen)"
tech-stack:
  added: []
  patterns:
    - "JSON-blob translations column: jsonEncode(map.map((k,v)=>MapEntry(k,v.toJson()))) on write, jsonDecode + CategoryTranslation.fromJson on read (mirrors SettingsEntity.notificationsJson)"
    - "ObjectBox dual index: @Index @Unique(onConflict: replace) on id + standalone @Index on parent FK column for parent-scoped queries"
key-files:
  created:
    - app/lib/entities/subcategory_entity.dart
  modified:
    - app/lib/entities/category_entity.dart
    - app/lib/objectbox-model.json
    - app/lib/objectbox.g.dart
key-decisions:
  - "Reused the SettingsEntity.notificationsJson JSON-blob encode/decode shape verbatim for translations on both entities"
patterns-established:
  - "Parent-FK ObjectBox index: indexed plain String column (not a ToOne relation) for parent-scoped queries per CAT-03"
requirements-completed: [CAT-03]
duration: 4min
completed: 2026-06-29
---

# Phase 10 Plan 02: Category & Subcategory Data Layer (ObjectBox Entities) Summary

**Extended CategoryEntity with a JSON-blob translations column and added a new SubcategoryEntity with an indexed-unique id plus an indexed parent category, regenerating the ObjectBox schema so subcategories survive app restarts.**

## Performance

- **Duration:** ~4 min
- **Completed:** 2026-06-29
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- `CategoryEntity` now persists `icon`, `shortName`, and a JSON-encoded `translationsJson` blob, round-tripping the extended `Category` model from Plan 01.
- New `SubcategoryEntity` persists with `@Index @Unique(onConflict: replace) id` and a separate `@Index category` (parent-category FK index, CAT-03), plus `name`, `shortName`, `icon`, `badgeIcon`, and a `translationsJson` blob.
- Both entities encode/decode their `Map<String, CategoryTranslation>` translations the same way as `SettingsEntity.notificationsJson`.
- Regenerated `objectbox-model.json` and `objectbox.g.dart` with the new property IDs and both subcategory indexes.

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend CategoryEntity with icon, shortName, translationsJson blob** - `8066d85f` (feat)
2. **Task 2: Create SubcategoryEntity with indexed id + indexed parent category** - `5f012739` (feat)

## Files Created/Modified

- `app/lib/entities/category_entity.dart` - Added `import 'dart:convert'`; `icon`, `shortName`, `translationsJson` fields; JSON-blob encode in `fromModel`, decode in `toModel` extension.
- `app/lib/entities/subcategory_entity.dart` - New `@Entity SubcategoryEntity` with dual indexes, `fromModel` factory, and `SubcategoryEntityMapping.toModel()` extension.
- `app/lib/objectbox-model.json` - Regenerated schema: new CategoryEntity properties + SubcategoryEntity with `id` and `category` indexes.
- `app/lib/objectbox.g.dart` - Regenerated ObjectBox bindings.

## Decisions Made

- Reused the canonical `SettingsEntity.notificationsJson` encode/decode shape verbatim for `translationsJson` on both entities, keeping a single JSON-blob convention across the entity layer.
- Modeled the subcategory parent as a plain indexed `String category` column (not a ObjectBox `ToOne` relation), matching the `id`/`category` string-FK pattern used by `Subcategory` and satisfying CAT-03's parent-index requirement.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Threat Surface

No new threat surface beyond the plan's `<threat_model>`. `translationsJson` is decoded only when non-null (T-10-04 mitigation in place, identical failure surface to `notificationsJson`); cached rows are reference data only (T-10-03 accepted). No package installs (T-10-SC not applicable) — only `build_runner` codegen.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The on-device cache now has the columns the Plan 04 providers need to store extended categories and subcategories.
- Subcategories can be queried by parent category via the new `category` index.
- No blockers.

## Self-Check: PASSED

- FOUND: app/lib/entities/category_entity.dart
- FOUND: app/lib/entities/subcategory_entity.dart
- FOUND: app/lib/objectbox-model.json
- FOUND commit 8066d85f
- FOUND commit 5f012739

---
*Phase: 10-category-subcategory-data-layer*
*Completed: 2026-06-29*
