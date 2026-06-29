---
phase: 10-category-subcategory-data-layer
reviewed: 2026-06-29T00:00:00Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - app/lib/entities/category_entity.dart
  - app/lib/entities/settings_entity.dart
  - app/lib/entities/subcategory_entity.dart
  - app/lib/models/category.dart
  - app/lib/models/category_preference.dart
  - app/lib/models/subcategory.dart
  - app/lib/models/subcategory_preference.dart
  - app/lib/models/settings.dart
  - app/lib/provider/category_preference_provider.dart
  - app/lib/provider/subcategory_preference_provider.dart
  - app/lib/provider/trail/category_provider.dart
  - app/lib/provider/trail/subcategory_provider.dart
findings:
  critical: 2
  warning: 4
  info: 1
  total: 7
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-06-29
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

This phase introduces the ObjectBox entity layer for categories and subcategories, preference models for user-level visibility toggles, and four Riverpod providers that wire these together. The models and entity mapping code are structurally sound. Two blockers were found: a non-atomic cache replacement that can corrupt the local database during an app crash, and an unguarded unsafe cast of `response.data` in both preference providers that will surface as an opaque `TypeError` when the server returns `null` or a non-list body. Four warnings cover meaningful asymmetries and quality gaps: the `CategoryNotifier` writes to ObjectBox but never reads from it (making offline startup always fail for categories), a `keepAlive` mismatch between the two related providers, potential use-after-invalidation in the background `_refresh` microtask, and a silent null-id fallback in `SettingsEntity` that can conflate unrelated records.

---

## Critical Issues

### CR-01: Non-atomic cache replacement can produce empty ObjectBox state on crash

**File:** `app/lib/provider/trail/category_provider.dart:34-35` and `app/lib/provider/trail/subcategory_provider.dart:43-44`

**Issue:** Both providers replace the ObjectBox cache with a two-step sequence — `box.removeAll()` followed by `box.putMany(items)`. These two calls are NOT wrapped in a write transaction. If the process is killed (or an exception is thrown inside `putMany`) after `removeAll` completes but before `putMany` finishes, the cache is left permanently empty. For `SubcategoryNotifier` this is serious because its `build()` reads the cache first (offline path) and will silently return an empty list on every subsequent cold start until the app can reach the network. For `CategoryNotifier` it just removes data that was never read back anyway, but the pattern is still incorrect.

**Fix:** Wrap both operations in a single ObjectBox write transaction so they are atomic:

```dart
// category_provider.dart  (same fix applies to subcategory_provider.dart)
final store = ref.read(objectBoxProvider);
store.runInTransaction(TxMode.write, () {
  final box = store.box<CategoryEntity>();
  box.removeAll();
  box.putMany(items.map(CategoryEntity.fromModel).toList());
});
```

---

### CR-02: Unsafe bare cast of `response.data` to `List` in preference providers

**File:** `app/lib/provider/category_preference_provider.dart:25` and `app/lib/provider/subcategory_preference_provider.dart:25`

**Issue:** Both providers cast `response.data` directly to `List` without a prior null or type guard:

```dart
return (response.data as List)
    .map((e) => CategoryPreference.fromJson(e as Map<String, dynamic>))
    .toList();
```

If the server returns HTTP 204 No Content, Dio sets `response.data` to `null`. If the server returns an error JSON object (e.g., `{"error": "..."}`) instead of an array, `response.data` is a `Map`. In both cases `as List` throws a `TypeError`. The `catch (e)` block wraps this as `Exception('Failed to fetch category preferences: $e')` and rethrows, so there is no crash, but the provider enters error state with an unhelpful message and the root cause (a `null` body or unexpected shape) is completely obscured. `CategoryNotifier` already guards this correctly with `if (response.data == null) throw Exception(...)`. The preference providers should do the same.

**Fix:**

```dart
final data = response.data;
if (data == null || data is! List) {
  throw Exception('Unexpected response shape from /user-category-preference: $data');
}
return data
    .map((e) => CategoryPreference.fromJson(e as Map<String, dynamic>))
    .toList();
```

Apply the same guard in `subcategory_preference_provider.dart`.

---

## Warnings

### WR-01: `CategoryNotifier` writes to ObjectBox cache but never reads from it — offline cold-start always fails

**File:** `app/lib/provider/trail/category_provider.dart:13-42`

**Issue:** `SubcategoryNotifier.build()` implements a deliberate cache-first pattern (reads ObjectBox, kicks off a background refresh). `CategoryNotifier.build()` does the opposite: it is purely network-first, only writing to ObjectBox on a successful response. The ObjectBox cache is therefore useless for categories — it is populated on success but never consulted. When the device is offline at app launch, `categoryProvider` always enters `AsyncError` state and the UI has no data even if a previous fetch populated the cache.

The `removeAll()` comment even says "a failed fetch leaves the prior cache intact" — but there is no code path that ever reads the prior cache, making that guarantee meaningless.

**Fix:** Add a cache-first read path analogous to `SubcategoryNotifier`, or at minimum fall back to the cache in the `catch` block:

```dart
} catch (e) {
  final box = ref.read(objectBoxProvider).box<CategoryEntity>();
  final cached = box.getAll();
  if (cached.isNotEmpty) return cached.map((e) => e.toModel()).toList();
  throw Exception('Failed to fetch categories: $e');
}
```

---

### WR-02: `keepAlive` mismatch between `CategoryNotifier` and `SubcategoryNotifier`

**File:** `app/lib/provider/trail/category_provider.dart:10` and `app/lib/provider/trail/subcategory_provider.dart:10`

**Issue:** `SubcategoryNotifier` is annotated `@Riverpod(keepAlive: true)` — it is never auto-disposed and its background refresh state persists for the app lifetime. `CategoryNotifier` uses plain `@riverpod` (auto-dispose), so it is torn down whenever all listeners detach. This means:

- Every time the user returns to a route that watches `categoryProvider`, a fresh network fetch fires.
- The ObjectBox write from the previous successful fetch is immediately discarded (`removeAll` is called again).
- Network traffic and parse work are unnecessarily repeated for data that changes infrequently.

Given that both datasets are used in the same filter UI and have the same staleness characteristics, they should have matching lifecycle policies.

**Fix:** Add `keepAlive: true` to `CategoryNotifier` if it should behave like `SubcategoryNotifier`, or adopt the same cache-first synchronous `build()` pattern so at minimum it can serve cached data without a network round-trip.

---

### WR-03: Background `_refresh()` microtask can mutate state on an invalidated notifier

**File:** `app/lib/provider/trail/subcategory_provider.dart:20` and `app/lib/provider/trail/subcategory_provider.dart:46`

**Issue:** `build()` calls `Future.microtask(_refresh)` before returning, and `_refresh()` assigns `state = items` after the `await`. Although `subcategoryProvider` is `keepAlive: true` (preventing auto-dispose), a caller could still call `ref.invalidate(subcategoryProvider)`. If the provider is invalidated between the `build()` return and the `state = items` assignment inside `_refresh()`, the assignment runs on the old notifier instance. In Riverpod this typically throws a `StateError` ("Cannot use ref after it was disposed") or silently has no effect depending on the version.

The error is swallowed by `catch (_)`, so callers never see it, but the cache and UI state become inconsistent: ObjectBox has been written with fresh data (`box.putMany` at line 44 already ran) but the provider state still reflects the stale snapshot from `build()`.

**Fix:** Guard state mutation after the async gap:

```dart
Future<void> _refresh() async {
  try {
    final response = await ref.read(apiProvider).get('/subcategory');
    if (response.data == null) return;

    final items = ListResult.fromJson(
      response.data,
      (json) => Subcategory.fromJson(json as Map<String, dynamic>),
    ).items;

    final box = ref.read(objectBoxProvider).box<SubcategoryEntity>();
    box.removeAll();
    box.putMany(items.map(SubcategoryEntity.fromModel).toList());

    // Only assign state if the notifier is still alive.
    if (!ref.isDisposed) {
      state = items;
    }
  } catch (_) {
    // swallow — keep cached state
  }
}
```

---

### WR-04: `SettingsEntity.fromModel` silently stores empty string for null `id`, risking record conflation

**File:** `app/lib/entities/settings_entity.dart:36`

**Issue:** `SettingsEntity.fromModel` writes `id: settings.id ?? ''`. The `id` field carries `@Unique(onConflict: ConflictStrategy.replace)`, so any two `Settings` objects that arrive with `id == null` will both be stored under the key `''` and the second write silently overwrites the first. The `Settings` model (line 96) declares `id` as `String?` making this a realistic scenario (e.g., a partially-constructed local-only `Settings` object passed to the entity factory before a server round-trip).

While in practice server-originated `Settings` records always have a non-null id, the current code provides no safeguard. A refactor that accidentally calls `SettingsEntity.fromModel` with a transient object would silently corrupt the stored settings.

**Fix:** Assert or throw on a null id rather than silently substituting an empty string:

```dart
factory SettingsEntity.fromModel(Settings settings) {
  assert(settings.id != null && settings.id!.isNotEmpty,
      'SettingsEntity requires a non-empty id');
  return SettingsEntity(
    id: settings.id!, // fail fast rather than store ''
    ...
  );
}
```

---

## Info

### IN-01: `priority` field declared in preference models but never set or read by any provider or consumer

**File:** `app/lib/models/category_preference.dart:15` and `app/lib/models/subcategory_preference.dart:15`

**Issue:** Both `CategoryPreference` and `SubcategoryPreference` declare `int? priority`. The `upsert` methods in both preference providers send only `category`/`subcategory` and `visible` — `priority` is never included in the request body. No UI code reads or sets `priority`. This is dead schema surface that may mislead future contributors into thinking priority-ordered preferences are functional when they are not.

**Fix:** Either remove `priority` from both freezed models until it is implemented, or add a TODO comment explicitly documenting that it is intentionally reserved for a future ordering feature.

---

_Reviewed: 2026-06-29_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
