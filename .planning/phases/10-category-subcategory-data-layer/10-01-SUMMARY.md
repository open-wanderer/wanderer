---
phase: 10-category-subcategory-data-layer
plan: 01
subsystem: flutter-models
tags: [freezed, category, subcategory, i18n, codegen]
dependency_graph:
  requires: []
  provides:
    - "Category model with icon/short_name/translations + displayName(Locale?)"
    - "CategoryTranslation nested freezed type"
    - "Subcategory freezed model with displayName(Locale?)"
  affects:
    - "Phase 10 Plan 02+ (ObjectBox entity, provider)"
    - "Phase 11 (trail filters)"
    - "Phase 12 (Settings Categories screen)"
tech_stack:
  added: []
  patterns:
    - "@JsonSerializable(explicitToJson: true) on factory constructor for nested-map freezed (freezed 3.x)"
    - "camelCase Dart field + @JsonKey(name: 'snake_case') for snake_case JSON keys"
    - "displayName fallback chain: active locale -> 'en' -> raw name"
key_files:
  created:
    - app/lib/models/subcategory.dart
    - app/lib/models/subcategory.freezed.dart
    - app/lib/models/subcategory.g.dart
  modified:
    - app/lib/models/category.dart
    - app/lib/models/category.freezed.dart
    - app/lib/models/category.g.dart
decisions:
  - "Removed Category.img (zero call sites) in favor of icon, mirroring web category.ts"
  - "CategoryTranslation reused by Subcategory via import, not redeclared"
metrics:
  duration_min: 5
  completed: 2026-06-29
requirements: [CAT-01, CAT-02]
---

# Phase 10 Plan 01: Category & Subcategory Data Layer Summary

Brought the Flutter `Category` model to parity with web PR #1059 (locale `translations`, `icon`, `short_name`, locale-resolving `displayName`) and introduced a new `Subcategory` freezed model reusing `CategoryTranslation`.

## What Was Built

- **`CategoryTranslation`** — nested freezed type with `name` and `shortName` (`@JsonKey(name: 'short_name')`).
- **`Category`** — extended with `shortName`, `icon`, and `Map<String, CategoryTranslation>? translations`; removed the unused `img` field; `@JsonSerializable(explicitToJson: true)` placed on the factory constructor so the nested translations map serializes correctly under freezed 3.x.
- **`CategoryDisplay.displayName(Locale?)`** — extension resolving `translations?[locale]?.name ?? translations?['en']?.name ?? name` (CAT-01 fallback chain).
- **`Subcategory`** — new freezed model with `id`, parent `category`, `name`, `shortName`, `icon`, `badgeIcon` (`@JsonKey(name: 'badge_icon')`), and `translations`. Imports and reuses `CategoryTranslation` from `category.dart` rather than redeclaring it. Includes a matching `SubcategoryDisplay.displayName(Locale?)` extension.
- Regenerated `.freezed.dart` / `.g.dart` codegen for both models.

## Tasks

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Extend Category model (translations, icon, short_name, displayName) | 89050e7b | category.dart, category.freezed.dart, category.g.dart |
| 2 | Create Subcategory freezed model | 5e821384 | subcategory.dart, subcategory.freezed.dart, subcategory.g.dart |

## Verification

- `dart run build_runner build --delete-conflicting-outputs` — completed with no errors for both tasks.
- `dart analyze lib/models/category.dart lib/models/subcategory.dart` — No issues found.
- `grep -c 'String? img' lib/models/category.dart` → 0 (img removed, zero call sites confirmed; only the prior generated files referenced it).
- `grep -c 'class CategoryTranslation' lib/models/subcategory.dart` → 0 (type reused, not duplicated).
- All acceptance criteria for both tasks confirmed via grep (CategoryTranslation class, displayName extensions, translations map field, JsonKey annotations, codegen files present).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Regenerated incidental riverpod codegen hashes**
- **Found during:** Task 2
- **Issue:** Running `build_runner` refreshed two unrelated generated files (`app/lib/provider/auth_provider.g.dart`, `app/lib/provider/settings_provider.g.dart`) with hash-only diffs, leaving the working tree dirty.
- **Fix:** Committed the hash-only regenerations alongside Task 2 to keep the tree clean; no logic change.
- **Files modified:** app/lib/provider/auth_provider.g.dart, app/lib/provider/settings_provider.g.dart
- **Commit:** 5e821384

## Threat Surface

No new threat surface beyond the plan's `<threat_model>`. New fields except `id`/`name`/`category` are nullable (T-10-01 mitigation in place via freezed/json_serializable tolerance of missing keys); translations carry reference data only (T-10-02 accepted). No package installs (T-10-SC not applicable).

## Self-Check: PASSED

- FOUND: app/lib/models/category.dart, subcategory.dart, subcategory.freezed.dart, subcategory.g.dart
- FOUND commit 89050e7b
- FOUND commit 5e821384
