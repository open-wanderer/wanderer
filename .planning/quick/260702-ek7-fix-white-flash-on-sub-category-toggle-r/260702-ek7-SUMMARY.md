---
phase: quick-260702-ek7
plan: 01
subsystem: mobile-app
tags: [flutter, riverpod, ui-polish, settings]
requires:
  - "AsyncValue<T>.value (Riverpod 3.3.1 nullable getter)"
provides:
  - "AsyncLoader preserves previous value during provider refresh"
affects:
  - profile_screen.dart
  - settings_categories_screen.dart
  - map_screen.dart
  - list_screen.dart
  - settings_subcategories_screen.dart
tech-stack:
  added: []
  patterns:
    - "asyncValue.value ?? mockData fallback in AsyncData switch (prefer carried previous value over empty mock)"
key-files:
  created: []
  modified:
    - app/lib/components/async_loader.dart
decisions:
  - "Fix isolated to the single shared AsyncLoader component; all 5 consumer screens left untouched (transparent change)"
  - "Used .value nullable getter per project convention; .valueOrNull deliberately avoided (does not exist in Riverpod 3.3.1)"
metrics:
  duration: ~2min
  completed: 2026-07-02
---

# Quick Task 260702-ek7: Fix White Flash on (Sub)Category Toggle/Reorder Summary

One-line change in `AsyncLoader` so it renders the AsyncValue's carried previous value (`asyncValue.value`) instead of empty `mockData` during an `invalidateSelf()`-triggered refresh, eliminating the empty-skeleton flash on category/subcategory toggle and reorder.

## What Was Done

`category_preference_provider` / `subcategory_preference_provider` call `invalidateSelf()` inside their `upsert`/`reorder` methods, flipping the provider into `AsyncLoading<T>`. The shared `AsyncLoader` switch only matched `AsyncData<T>` and otherwise fell straight to `mockData` — an EMPTY list for both settings screens — so the builder rendered empty data inside a `Skeletonizer` until the refetch resolved (the visible flash). Riverpod 3's `AsyncLoading` carries the previous value via the nullable `.value` getter, but the switch ignored it.

The fallback arm of the data `switch` in `build()` changed from:

```dart
_ => mockData,
```

to:

```dart
_ => asyncValue.value ?? mockData,
```

The `AsyncData<T>(value: final v) => v,` arm and the `hasError` early-return are unchanged. Genuine first-load (`AsyncLoading` with a null `.value`) still resolves to `mockData` via the `??`, preserving initial skeleton behavior.

## Task Completed

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Preserve previous value over mockData during provider refresh | 8a917b4c | app/lib/components/async_loader.dart |

## Verification

- `grep -n "asyncValue.value ?? mockData" app/lib/components/async_loader.dart` → line 30 matches.
- `grep -rn "valueOrNull" app/lib/` → nothing (convention preserved).
- `dart analyze` across the component + all 5 consumer files → 0 error lines, 0 new warning lines. The only warning reported (`map_screen.dart:27` unused import `subcategory.dart`) is PRE-EXISTING and unrelated — that file was not modified by this task.
- Committed diff touches only `app/lib/components/async_loader.dart` (1 insertion, 1 deletion).

## Deviations from Plan

None - plan executed exactly as written.

## Known Stubs

None.

## Self-Check: PASSED

- FOUND: app/lib/components/async_loader.dart (modified, fallback arm at line 30)
- FOUND: commit 8a917b4c
