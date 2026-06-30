---
phase: 10-category-subcategory-data-layer
fixed_at: 2026-06-29T00:00:00Z
review_path: .planning/phases/10-category-subcategory-data-layer/10-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 10: Code Review Fix Report

**Fixed at:** 2026-06-29
**Source review:** .planning/phases/10-category-subcategory-data-layer/10-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6
- Fixed: 6
- Skipped: 0

## Fixed Issues

### CR-01: Non-atomic cache replacement can produce empty ObjectBox state on crash

**Files modified:** `app/lib/provider/trail/category_provider.dart`, `app/lib/provider/trail/subcategory_provider.dart`
**Commit:** a8978842
**Applied fix:** Wrapped the `box.removeAll()` + `box.putMany()` sequence in `store.runInTransaction(TxMode.write, () { ... })` in both providers. The two operations are now atomic — a process kill between them will roll back the entire transaction, leaving the cache in its prior state rather than permanently empty.

---

### CR-02: Unsafe bare cast of `response.data` to `List` in preference providers

**Files modified:** `app/lib/provider/category_preference_provider.dart`, `app/lib/provider/subcategory_preference_provider.dart`
**Commit:** 3c4d3d69
**Applied fix:** Replaced the bare `(response.data as List)` cast with an explicit `final data = response.data; if (data == null || data is! List) { throw Exception(...) }` guard in both preference providers. HTTP 204 (null body) and unexpected JSON shapes now surface a descriptive error message instead of an opaque `TypeError`.

---

### WR-01: `CategoryNotifier` writes to ObjectBox cache but never reads from it

**Files modified:** `app/lib/provider/trail/category_provider.dart`
**Commit:** f2c001b1
**Applied fix:** Added a cache fallback in the `catch` block of `CategoryNotifier.build()`. When the network fetch fails, the provider now reads all cached `CategoryEntity` rows from ObjectBox and returns them if non-empty, enabling offline cold-starts to serve previously fetched categories instead of always entering `AsyncError` state.

---

### WR-02: `keepAlive` mismatch between `CategoryNotifier` and `SubcategoryNotifier`

**Files modified:** `app/lib/provider/trail/category_provider.dart`
**Commit:** 5bce1783
**Applied fix:** Changed `@riverpod` to `@Riverpod(keepAlive: true)` on `CategoryNotifier` so its lifecycle matches `SubcategoryNotifier`. Both providers now persist for the app lifetime, preventing redundant network fetches and ObjectBox overwrites each time listeners reattach.

---

### WR-03: Background `_refresh()` microtask can mutate state on an invalidated notifier

**Files modified:** `app/lib/provider/trail/subcategory_provider.dart`
**Commit:** a6acb9bd
**Applied fix:** Added `if (!ref.isDisposed) { state = items; }` guard around the `state = items` assignment at the end of `_refresh()`. The ObjectBox write still proceeds (cache is always updated when data is fresh) but the in-memory state assignment is skipped if the notifier has been invalidated between the `build()` return and the async completion.

---

### WR-04: `SettingsEntity.fromModel` silently stores empty string for null `id`, risking record conflation

**Files modified:** `app/lib/entities/settings_entity.dart`
**Commit:** 405852df
**Applied fix:** Replaced the silent `settings.id ?? ''` fallback with an `assert(settings.id != null && settings.id!.isNotEmpty, '...')` and changed the assignment to `id: settings.id!`. Callers passing a transient `Settings` object before a server round-trip now fail fast in debug builds instead of silently overwriting a real stored record via the `@Unique(onConflict: replace)` constraint on the empty-string key.

---

_Fixed: 2026-06-29_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
