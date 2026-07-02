---
phase: 12-settings-categories-screen
plan: 03
subsystem: ui
tags: [flutter, riverpod, l10n, reorderable-list, category-preferences]

# Dependency graph
requires:
  - phase: 12-settings-categories-screen
    plan: 01
    provides: SubcategoryPreferenceNotifier.reorder, sortedSubcategoriesByPreference, subcategoryVisible, ownTrailCount, settings-categories l10n keys
  - phase: 12-settings-categories-screen
    plan: 02
    provides: SettingsCategoriesScreen structural reference (row/dialog/reorder patterns, D-06)
provides:
  - SettingsSubcategoriesScreen (per-category subcategory visibility + reorder settings screen)
affects: [12-04 router-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "AsyncLoader wraps only the async preference provider; the synchronous subcategoryProvider List is read directly and filtered to the parent category id"
    - "Parent-scoped reorder: reorder(widget.category.id, orderedIds) posts {category, subcategories} (SETCAT-10)"
    - "Leaf rows (no body-tap navigation) mirror the sibling category screen's drag-handle + Switch layout (D-06)"

key-files:
  created:
    - app/lib/routes/settings_subcategories_screen.dart
  modified: []

key-decisions:
  - "subcategoryProvider is synchronous (List<Subcategory>), so AsyncLoader wraps only subcategoryPreferenceProvider; subcategories are filtered client-side to widget.category.id"
  - "Own handle resolved as '@${authProvider.value!.preferredUsername}' inline (matches sibling + profile_screen.dart:38); null-username guard shows error toast and aborts rather than building '@null'"
  - "Empty state reuses shared settings_categories_empty_* keys; screen stays reachable (D-07)"

patterns-established:
  - "Pattern: leaf settings row = ReorderableDragStartListener handle + avatar + Expanded name + trailing Switch (no InkWell body nav)"

requirements-completed: [SETCAT-08, SETCAT-10, SETCAT-11]

# Metrics
duration: 5min
completed: 2026-07-01
---

# Phase 12 Plan 03: SettingsSubcategoriesScreen Summary

**The leaf subcategory settings screen — parent-scoped list with priority sort, per-row visibility Switch (auto-save, error-toast-only), drag-handle reorder scoped to the parent category, an empty state, and the own-trail confirm-before-disable dialog whose "View trails" action opens the user's own `@`-handle profile trail list pre-filtered to the subcategory.**

## Performance

- **Duration:** ~5 min
- **Completed:** 2026-07-01
- **Tasks:** 2 code tasks + 1 human-verify checkpoint (auto-approved, end-of-phase mode)
- **Files created:** 1
- **Files modified:** 0

## Accomplishments
- `SettingsSubcategoriesScreen` is a `ConsumerStatefulWidget` taking a required parent `Category`; the AppBar title is that category's `displayName(locale)` (D-06).
- Subcategories are filtered to `sub.category == widget.category.id`, sorted via `sortedSubcategoriesByPreference` (Plan 01), and seeded into a local `_orderedIds` drag working copy.
- Rows render `ReorderableDragStartListener` handle + `subcategoryFilterAvatar(context, sub, parent, locale)` + name + trailing `Switch` (not a switch-tile widget); rows are leaf (no body-tap navigation).
- Empty state (shared `settings_categories_empty_*` keys) renders for categories with no subcategories; the screen stays reachable (D-07).
- ON-toggle auto-saves via `upsert(id, true)`; OFF-toggle runs the own-trail guarded flow. `_save` surfaces an error toast only (no success toast, D-08) with a `mounted` guard.
- Reorder uses `ReorderableListView.builder(buildDefaultDragHandles: false)` with the `if (newIndex > oldIndex) newIndex -= 1` shift, `ValueKey(sub.id)`, optimistic `setState`, and `reorder(widget.category.id, _orderedIds)` (parent id FIRST, SETCAT-10). Failure reverts to the pre-drag snapshot + error toast.
- OFF-toggle lazily calls `ownTrailCount(ref, isSubcategory: true, id: sub.id)` (D-12); count 0 saves directly, count > 0 shows the `settings_categories_confirm_disable_subcategory_title` dialog with body count, "View trails", "Disable anyway", "Cancel". Confirm saves `upsert(id, false)`; cancel/dismiss reverts. ON never checks (D-11).
- "View trails" resolves the `@`-prefixed own handle, seeds `trailFilterProvider('profile_trail_$handle')` via `updateFilter((f) => f.copyWith(category: const [], subcategory: [sub]))`, and pushes `/profile/$handle/trails` — real navigation, no stub. A null username shows the error toast and aborts instead of constructing `'@null'`.

## Task Commits

Each code task was committed atomically:

1. **Task 1: Build subcategory list, empty state, ON-toggle auto-save** - `a14c682b` (feat)
2. **Task 2: Wire parent-scoped reorder + own-trail confirm dialog** - `84adff13` (feat)

## Files Created/Modified
- `app/lib/routes/settings_subcategories_screen.dart` - New: per-category subcategory visibility + reorder settings screen (SETCAT-08/10/11 subcategory half).

## Decisions Made
- `subcategoryProvider` returns a synchronous `List<Subcategory>` (unlike the sibling's async `categoryProvider`), so `AsyncLoader` wraps only `subcategoryPreferenceProvider`; subcategories are filtered client-side to the parent id. This diverges from Plan 02's dual-async `AsyncValue` combine but is the correct shape for these providers.
- Own handle resolved inline as `'@${ref.read(authProvider).value!.preferredUsername}'` (matches `profile_screen.dart:38`, Plan 01, and the sibling screen); a `null` `preferredUsername` triggers an error toast + early return, never a `'@null'` handle.
- Empty state reuses the shared `settings_categories_empty_*` keys (the empty-title copy is "No subcategories").

## Deviations from Plan

None - plan executed exactly as written. (The single-async `AsyncLoader` wrapping is not a deviation but the correct adaptation to `subcategoryProvider` being synchronous, which the plan's `read_first` notes flag as `List<Subcategory>`.)

## Checkpoints

- **Task 3 (human-verify):** Auto-approved under active auto-mode with `human_verify_mode: end-of-phase`. No code changes. Live-device verification (empty state, parent-scoped reorder persistence + revert, OFF-only confirm dialog, "View trails" opening the pre-filtered own-trail list with the `@`-prefixed handle) is deferred to the end-of-phase human-verify gate.

## Issues Encountered
- Two grep gates in the plan require single-line forms: `ownTrailCount(ref, isSubcategory: true, id: sub.id)` and the `'@...preferredUsername'` handle assignment. Initial multi-line formatting failed those literal/regex gates; reformatted to single lines (with `// ignore: lines_longer_than_80_chars`) to satisfy the gates while keeping the analyzer clean. No behavior change.

## Verification
- `dart analyze lib/routes/settings_subcategories_screen.dart` → "No issues found!"
- All Task 1 and Task 2 grep gates pass (ReorderableListView + custom handle + index-shift + ValueKey; `reorder(widget.category.id, ...)`; `ownTrailCount(ref, isSubcategory: true, ...)`; subcategory confirm dialog key; empty-state key; `@`-prefixed handle + `context.push('/profile/$handle/trails')` + `trailFilterProvider('profile_trail_$handle')` seed with `copyWith(category: const [], subcategory: [sub])`; no `@null`, no `plugin`, no stub comment, no `SwitchListTile`).

## Next Phase Readiness
- `SettingsSubcategoriesScreen` is ready for Plan 04 router wiring (accepts a `Category category` via go_router `extra`; the sibling screen already pushes `/settings/categories/subcategories` with the category as `extra`).
- No blockers.

---
*Phase: 12-settings-categories-screen*
*Completed: 2026-07-01*

## Self-Check: PASSED

Created file `app/lib/routes/settings_subcategories_screen.dart` and `12-03-SUMMARY.md` exist on disk; both task commits (`a14c682b`, `84adff13`) present in git history.
