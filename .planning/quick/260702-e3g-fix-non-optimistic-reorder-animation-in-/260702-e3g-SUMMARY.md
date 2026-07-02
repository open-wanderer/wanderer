---
phase: quick-260702-e3g
plan: 01
subsystem: app-settings
tags: [flutter, reorderable-list, optimistic-ui, category-preferences]
requires:
  - "SettingsCategoriesScreen / SettingsSubcategoriesScreen reorder guard (Phase 12)"
provides:
  - "Round-trip-scoped reorder guard (_reordering) covering drag→reorder() network call"
affects:
  - app/lib/routes/settings_categories_screen.dart
  - app/lib/routes/settings_subcategories_screen.dart
tech-stack:
  added: []
  patterns:
    - "Reorder guard held from onReorderStart through await reorder() resolution, cleared inside _onReorder (not onReorderEnd)"
key-files:
  created: []
  modified:
    - app/lib/routes/settings_categories_screen.dart
    - app/lib/routes/settings_subcategories_screen.dart
decisions:
  - "Renamed _dragging to _reordering; the guard now spans the whole drag→reorder() round-trip, not just the drag gesture"
  - "onReorderEnd removed entirely — its synchronous fire before await reorder() returned was the direct cause of the snap-back"
  - "Guard cleared in both _onReorder outcomes: success (mounted-guarded) and caught failure (alongside the snapshot revert)"
metrics:
  duration: ~6min
  completed: 2026-07-02
---

# Quick Task 260702-e3g: Fix Non-Optimistic Reorder Animation Summary

Extended the category/subcategory reorder guard to cover the full drag→`reorder()` network round-trip, eliminating the visible snap-back where a dropped row jumped back to its pre-drag slot until the provider re-emitted.

## What Changed

`SliverReorderableList.onReorderEnd` fires synchronously the moment `onReorder` returns — before the `await reorder()` network call resolves. The old `_dragging` guard was cleared there, so the next `build()` reseeded `_orderedIds` from the still-stale provider data, snapping the dragged item back to its original position until the server responded.

Both screens now:

- Rename `_dragging` → `_reordering` with an updated doc comment describing the two cases the guard covers (snap-back prevention + WR-02 mid-drag protection).
- Reseed `_orderedIds` in `build()` only under `if (!_reordering)`.
- Set `_reordering = true` in `onReorderStart`.
- **Remove `onReorderEnd` entirely** (the premature clear was the bug).
- Clear `_reordering = false` inside `_onReorder` after `reorder()` resolves — on success (mounted-guarded via `if (!mounted) return;`) and on caught failure (inside the same `setState` as the `_orderedIds = snapshot` revert).

The subcategory screen preserves the parent-scoped `reorder(widget.category.id, _orderedIds)` call signature (SETCAT-10). Both preserve the pre-drag `snapshot`, the `if (newIndex > oldIndex) newIndex -= 1` index-shift, the optimistic `setState(() => _orderedIds = reordered)`, the error toast, and the `// ignore: deprecated_member_use` on the `onReorder` wiring.

## Tasks

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Hold reorder guard for full round-trip in SettingsCategoriesScreen | b296199c | app/lib/routes/settings_categories_screen.dart |
| 2 | Apply identical round-trip guard to SettingsSubcategoriesScreen | 6b3e6f6b | app/lib/routes/settings_subcategories_screen.dart |

## Verification

- Grep contract passed on both files: `_reordering` present, no `_dragging`, no `onReorderEnd`, `if (!_reordering)` present, and (subcategory) `reorder(widget.category.id, _orderedIds)` preserved.
- `dart analyze` on both files reports 0 new errors/warnings. The 4 remaining issues are pre-existing info-level `use_build_context_synchronously` lints in `_viewOwnTrails` (unchanged code, not touched by this fix).

## Success Criteria Met

- [x] Dragging a category or subcategory row leaves it settled in the new slot on drop — no snap-back while `reorder()` is in flight, no late jump when the provider re-emits.
- [x] WR-02 mid-drag protection preserved — an unrelated rebuild during a drag/round-trip does not reseed the optimistic working copy.
- [x] On `reorder()` failure the row reverts to the pre-drag snapshot and an error toast shows (D-04/D-09 unchanged).
- [x] `dart analyze` on both files reports 0 new errors/warnings.

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- FOUND: app/lib/routes/settings_categories_screen.dart
- FOUND: app/lib/routes/settings_subcategories_screen.dart
- FOUND commit: b296199c
- FOUND commit: 6b3e6f6b
