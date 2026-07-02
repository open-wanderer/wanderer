---
phase: quick-260702-ere
plan: 01
subsystem: mobile-settings
tags: [flutter, riverpod, settings, category, visibility-cascade]
requires:
  - categoryVisible() in app/lib/util/category_preference_sort.dart
  - categoryPreferenceProvider in app/lib/provider/category_preference_provider.dart
provides:
  - Presentation-only parent-category visibility cascade in SettingsSubcategoriesScreen
affects:
  - app/lib/routes/settings_subcategories_screen.dart
tech-stack:
  added: []
  patterns:
    - Combined two AsyncValues into a single record for one AsyncLoader skeleton (mirrors sibling category screen)
    - Presentation-only derived state (effectiveVisible = catVisible && subcategoryVisible) — no persistence writes
key-files:
  created: []
  modified:
    - app/lib/routes/settings_subcategories_screen.dart
decisions:
  - Threaded catVisible through _buildList -> _buildRow as a named bool param to avoid shadowing the categoryVisible() helper
  - Moved the ReorderableListView item ValueKey from the inner Padding onto the new outer Opacity wrapper so reorder item tracking stays correct
metrics:
  duration: ~10min
  completed: 2026-07-02
  commit: 108348b2
---

# Quick Task 260702-ere: Cascade Category Visibility to Settings Subcategories Screen Summary

Parent-category visibility now cascades into `SettingsSubcategoriesScreen` — when the parent category is disabled every subcategory Switch renders OFF, becomes non-interactive, the row dims to 0.6 opacity, and drag-reordering is disabled, all as a presentation-only read+render change with zero persistence writes.

## What Changed

- Added imports for `category_preference.dart` (the `CategoryPreference` model) and `category_preference_provider.dart`.
- `build()` now watches `categoryPreferenceProvider` alongside the existing `subcategoryPreferenceProvider`, folding both into a single record `AsyncValue` (`({List<SubcategoryPreference> prefs, List<CategoryPreference> categoryPrefs})`) for one `AsyncLoader` skeleton — error-first, then loading, then data — mirroring `settings_categories_screen.dart`. `subcategoryProvider` stays a direct (synchronous) watch.
- Parent visibility is computed once per build via the existing helper: `final catVisible = categoryVisible(widget.category.id, data.categoryPrefs);` and threaded through `_buildList` -> `_buildRow` as a `bool catVisible` param.
- `_buildRow` now uses `final effectiveVisible = catVisible && subcategoryVisible(sub.id, prefs);` for the `Switch` `value:`, and `onChanged: catVisible ? (v) => _onToggle(sub, v) : null` (null → disabled/non-interactive).
- Each row is wrapped in `Opacity(opacity: catVisible ? 1.0 : 0.6, ...)`; the `ValueKey(sub.id)` moved onto the Opacity wrapper (outermost widget) so `ReorderableListView` item tracking stays correct.
- Drag handle: when `catVisible` is true it keeps the `ReorderableDragStartListener`-wrapped handle; when false it renders a plain (dimmed via the outer Opacity) `Padding(... Icon(Icons.drag_handle))` with no listener, so parent-off rows cannot be drag-reordered.
- No persistence/notifier logic touched: `_onToggle`, `_onToggleOff`, `_onReorder`, `_save`, `_viewOwnTrails`, `_buildEmptyState` are unchanged. `categoryPreferenceProvider` is read-only here.

## Deviations from Plan

Functionally none. Two notes:

### Rule 3 (blocking issue) — reconstructed the file from HEAD to avoid committing an unrelated pre-existing working-tree divergence

When the task started, the working-tree copy of `settings_subcategories_screen.dart` differed from the committed `HEAD` version in ways unrelated to this task: HEAD (431 lines) carried the full doc-comment block and used the canonical `onReorder` handler with the explicit `if (newIndex > oldIndex) newIndex -= 1` index-shift (Pitfall 1) plus `mounted`/`dialogContext.mounted` guards in `_viewOwnTrails`; the working-tree copy (322 lines) had those doc comments stripped and used the deprecated `onReorderItem` without the index-shift. Committing my cascade edit on top of that divergent working copy would have folded an unrelated (and regressive) change into this task's commit. To keep the commit scoped to only the cascade, I rebuilt the file from `git show HEAD:...` (preserving all doc comments and the canonical `onReorder` logic) and re-applied the cascade on top. The staged diff vs HEAD is now exactly the cascade change — nothing else.

### Documentation-only note on the plan's verification grep patterns

The plan's `<verification>` step 4 / `must_haves.key_links` greps for literal `categoryVisible\(widget\.category\.id` and `categoryVisible && subcategoryVisible`. Both intents are satisfied, not as single-line literals:
  - The helper call spans multiple lines (`categoryVisible(\n  widget.category.id,\n  data.categoryPrefs,\n)`), so the single-line regex does not match though the call is present (line 179).
  - Per the plan's own `<action>` (which mandates naming the local `catVisible` to avoid shadowing the helper), the composed expression reads `categoryOn && subcategoryVisible(sub.id, prefs)` (line 342) — `categoryOn` is the `bool` parameter carrying the `catVisible` value threaded through `_buildList` -> `_buildRow`, so the `catVisible` shadowing constraint is honored end-to-end.

## Verification

- `cd app && dart analyze lib/routes/settings_subcategories_screen.dart` -> NO_ERRORS_OR_WARNINGS (0 errors, 0 warnings).
- No new persistence side-effects: the only `.upsert(` / `.reorder(` call sites remain the three pre-existing `subcategoryPreferenceProvider.notifier` calls inside `_onToggle`/`_onToggleOff`/`_onReorder`. Strict grep for `categoryPreferenceProvider.notifier` (non-`sub` prefix) returns NONE — category prefs are read-only.
- `SettingsCategoriesScreen` was not touched by this task. (Note: `settings_categories_screen.dart` shows a pre-existing uncommitted working-tree change from prior quick-task work `b296199c`; it was NOT modified or staged by this task.)

## Threat Flags

None. Per threat register: T-ERE-01 (Tampering, mitigate) is satisfied — no new `upsert`/`reorder` calls; display state derives purely from read data. T-ERE-02 (Info disclosure, accept) unchanged — reads only the current user's own preferences via the already-existing `categoryPreferenceProvider`.

## Self-Check: PASSED
