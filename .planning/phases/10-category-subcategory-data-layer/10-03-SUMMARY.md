---
phase: 10-category-subcategory-data-layer
plan: 03
subsystem: flutter-providers
tags: [freezed, riverpod, preferences, codegen, anonymous-gate, security]
dependency_graph:
  requires:
    - phase: 10-01
      provides: "Category/Subcategory freezed models (pattern reference)"
    - phase: 10-02
      provides: "ObjectBox entities (not directly consumed here)"
  provides:
    - "CategoryPreference freezed model (nullable id/visible/priority, required user+category)"
    - "SubcategoryPreference freezed model (nullable id/visible/priority, required user+subcategory)"
    - "categoryPreferenceProvider — keepAlive notifier with anonymous gate + bare-array fetch + upsert"
    - "subcategoryPreferenceProvider — keepAlive notifier with anonymous gate + bare-array fetch + upsert"
  affects:
    - "Phase 11 (filter chips hide visible:false categories/subcategories)"
    - "Phase 12 (Settings Categories screen toggles + reordering)"
tech_stack:
  added: []
  patterns:
    - "Flat freezed model (no @JsonSerializable(explicitToJson:true)) — no nested objects"
    - "Anonymous auth gate: ref.watch(authProvider).value == null returns [] with no API call (D-07)"
    - "Bare-array parse: (response.data as List).map(Model.fromJson).toList() — NOT ListResult"
    - "Server-injected ownership: PUT body restricted to {category|subcategory, visible}; no client user field (Security V4 / T-10-05)"
key_files:
  created:
    - app/lib/models/category_preference.dart
    - app/lib/models/category_preference.freezed.dart
    - app/lib/models/category_preference.g.dart
    - app/lib/models/subcategory_preference.dart
    - app/lib/models/subcategory_preference.freezed.dart
    - app/lib/models/subcategory_preference.g.dart
    - app/lib/provider/category_preference_provider.dart
    - app/lib/provider/category_preference_provider.g.dart
    - app/lib/provider/subcategory_preference_provider.dart
    - app/lib/provider/subcategory_preference_provider.g.dart
  modified: []
decisions:
  - "Preference models are flat freezed (no explicitToJson) — they carry no nested objects, unlike Category"
  - "Anonymous gate uses ref.watch(authProvider).value (D-07) so the provider re-evaluates on login/logout"
metrics:
  duration_min: 4
  completed: 2026-06-29
requirements: [SETCAT-03, SETCAT-04, SETCAT-05]
---

# Phase 10 Plan 03: Category & Subcategory Preference Data Layer Summary

Created the `CategoryPreference`/`SubcategoryPreference` freezed models (SETCAT-03) and their two keepAlive Riverpod providers (SETCAT-04, SETCAT-05), which fetch the authenticated user's preferences from bare-JSON-array endpoints, upsert single records via PUT with a server-injected `user`, and return an empty list with no API call for anonymous users.

## What Was Built

- **`CategoryPreference`** — flat freezed model with `String? id`, `required String user`, `required String category`, `bool? visible`, `int? priority`. Nullable id/visible/priority tolerate the server omitting fields (RESEARCH Pitfall 4). No `@JsonSerializable(explicitToJson: true)` — no nested objects.
- **`SubcategoryPreference`** — sibling model, identical except `required String subcategory` replaces `category`.
- **`CategoryPreferenceNotifier`** (`@Riverpod(keepAlive: true)`, generated `categoryPreferenceProvider`) — `build()` reads `ref.watch(authProvider).value`; returns `[]` immediately (no API call) when the user is null (D-07). Otherwise GETs `/user-category-preference` and parses the bare array via `(response.data as List).map(...).toList()` inside a try/catch. `upsert(categoryId, visible)` PUTs `{category, visible}` then `ref.invalidateSelf()`.
- **`SubcategoryPreferenceNotifier`** — sibling provider on `/user-subcategory-preference`; `upsert(subcategoryId, visible)` PUTs `{subcategory, visible}`.
- Regenerated freezed/json_serializable codegen for both models and riverpod codegen for both providers.

## Tasks

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Create CategoryPreference + SubcategoryPreference freezed models | 0059770a | category_preference.dart (+ .freezed/.g), subcategory_preference.dart (+ .freezed/.g) |
| 2 | Create CategoryPreferenceNotifier + SubcategoryPreferenceNotifier providers | 83bcf159 | category_preference_provider.dart (+ .g), subcategory_preference_provider.dart (+ .g) |

## Verification

- `dart run build_runner build --delete-conflicting-outputs` — completed without errors for both tasks.
- `dart analyze` across all four source files — No issues found!
- `grep -c 'explicitToJson' lib/models/category_preference.dart` → 0 (flat model, no nested object).
- `grep -c 'if (user == null)' lib/provider/category_preference_provider.dart` → 1 (anonymous gate present).
- `grep -c '(response.data as List)' lib/provider/category_preference_provider.dart` → 1 (bare-array parse).
- `grep -c 'ListResult' lib/provider/category_preference_provider.dart` → 0 and subcategory → 0 (no list-wrapper parse).
- `grep -c "'user':"` on both providers → 0 (server injects user — Security V4 / T-10-05).
- Both `.g.dart` and `.freezed.dart` codegen files for models and providers exist.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Reworded a code comment containing "ListResult"**
- **Found during:** Task 2
- **Issue:** The acceptance gate `grep -c 'ListResult' <provider>` must return 0, but my explanatory comment ("not a ListResult") contained the literal word, tripping the gate even though no code parsed a ListResult.
- **Fix:** Reworded both providers' comments to "BARE JSON array (not a paginated list wrapper)". No logic change.
- **Files modified:** app/lib/provider/category_preference_provider.dart, app/lib/provider/subcategory_preference_provider.dart
- **Commit:** 83bcf159

## Threat Surface

No new threat surface beyond the plan's `<threat_model>`. T-10-05 (privilege escalation via client `user`) mitigated — PUT bodies carry only `{category|subcategory, visible}`, verified `grep "'user':"` returns 0. T-10-06 (anonymous fetch) mitigated by the D-07 gate returning `[]` with no API call. T-10-07 (malformed JSON) mitigated by nullable id/visible/priority plus try/catch on the fetch. No package installs (T-10-SC not applicable) — only existing `build_runner` codegen.

## Self-Check: PASSED

- FOUND: app/lib/models/category_preference.dart
- FOUND: app/lib/models/subcategory_preference.dart
- FOUND: app/lib/provider/category_preference_provider.dart
- FOUND: app/lib/provider/subcategory_preference_provider.dart
- FOUND commit 0059770a
- FOUND commit 83bcf159

---
*Phase: 10-category-subcategory-data-layer*
*Completed: 2026-06-29*
