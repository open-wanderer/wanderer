---
phase: 12-settings-categories-screen
plan: 02
subsystem: ui
tags: [flutter, riverpod, reorderable-list, category-preferences, l10n, navigation]

# Dependency graph
requires:
  - phase: 12-settings-categories-screen (Plan 01)
    provides: CategoryPreferenceNotifier.reorder, sortedCategoriesByPreference, categoryVisible, ownTrailCount, settings-categories l10n keys
  - phase: 10-category-subcategory-data-layer
    provides: Category model + displayName(locale), categoryProvider, categoryPreferenceProvider + upsert
  - phase: 11-trail-filter-subcategory-support
    provides: TrailFilter (category List<Category>), trailFilterProvider + updateFilter, profile_trail_$handle filterId convention
provides:
  - SettingsCategoriesScreen (category list with priority sort, per-row visibility toggle, drag-handle reorder, own-trail confirm-before-disable dialog)
affects: [12-04 router-wiring (registers /settings/categories and /settings/categories/subcategories)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ReorderableListView.builder with buildDefaultDragHandles:false + custom ReorderableDragStartListener handle (separate hit targets so body-tap navigates independently of drag/switch — Pattern 2)"
    - "Local _orderedIds working copy for optimistic reorder; reverts to pre-drag snapshot + error toast on failure (never renders reorder from the re-sorting provider mid-drag)"
    - "Lazy own-trail confirm dialog: count only fetched on OFF-toggle; view-trails action seeds trailFilterProvider then pushes the @-prefixed own profile trail list"

key-files:
  created:
    - app/lib/routes/settings_categories_screen.dart
  modified: []

key-decisions:
  - "Two async values (categoryProvider + categoryPreferenceProvider) folded into one record AsyncValue so a single AsyncLoader renders one skeleton (D-14)"
  - "onReorder + explicit `if (newIndex > oldIndex) newIndex -= 1` retained (plan-authoritative grep contract); onReorderItem deprecation suppressed with a targeted ignore to keep the index-shift explicit and testable"
  - "View-trails handle resolved as '@${authProvider.value.preferredUsername}' matching profile_screen.dart:38 and Plan 01's own_trail_count.dart; null username shows error toast and aborts (never builds a bogus @-null handle)"

patterns-established:
  - "Pattern: settings toggle auto-save via _save(op) with error-only toast (error_saving_settings), no success toast, mounted-guarded after awaits (D-08)"
  - "Pattern: reorder = optimistic setState of local id list + reorder POST, revert-on-error to snapshot + toast (D-04/D-09)"

requirements-completed: [SETCAT-06, SETCAT-07, SETCAT-09, SETCAT-11]

# Metrics
duration: 14min
completed: 2026-07-01
---

# Phase 12 Plan 02: SettingsCategoriesScreen Summary

**Category settings screen with priority-sorted rows, per-row visibility Switch (auto-save, error-toast-only), drag-handle reorder with revert-on-failure, and an own-trail confirm-before-disable dialog whose "View trails" action opens the @-prefixed own profile trail list pre-filtered to the category.**

## Performance

- **Duration:** ~14 min
- **Started:** 2026-07-01
- **Completed:** 2026-07-01
- **Tasks:** 3 (2 code + 1 auto-approved human-verify checkpoint)
- **Files modified:** 1 created

## Accomplishments
- `SettingsCategoriesScreen` (`ConsumerStatefulWidget`) loads categories + preferences, wraps the body in `AsyncLoader` (D-14), and sorts rows via `sortedCategoriesByPreference` (SETCAT-06).
- Each row is an explicit Row (not a switch-tile) with a `ReorderableDragStartListener` drag handle, `categoryFilterAvatar` + locale `displayName`, and a trailing `Switch` bound to `categoryVisible` — three independent hit targets (Pattern 2).
- Visibility toggle auto-saves: ON calls `upsert(id, true)` immediately (D-11); `_save` surfaces only an error toast on failure, no success toast (SETCAT-07/D-08).
- Drag-handle reorder POSTs the full ordered category id list via `.reorder(_orderedIds)`, optimistically updating a local working copy and reverting to the pre-drag snapshot + error toast on failure (SETCAT-09).
- OFF-toggle lazily counts the user's own trails (`ownTrailCount`, D-12); when count > 0 it shows an `AlertDialog` (title/body-with-count/view-trails/disable-anyway/cancel), where "View trails" seeds `trailFilterProvider('profile_trail_$handle')` and pushes `/profile/$handle/trails` with the `@`-prefixed own handle (SETCAT-11 category half / D-10).

## Task Commits

Each task was committed atomically:

1. **Task 1: Build the category list — load, sort, row layout, toggle** - `ef39e2fc` (feat)
2. **Task 2: Wire reorder gesture, body-tap navigation, and own-trail confirm dialog** - `ba5a00de` (feat)
3. **Task 3: Human-verify SettingsCategoriesScreen** - auto-approved (human-verify checkpoint under active auto-mode; no code changes)

## Files Created/Modified
- `app/lib/routes/settings_categories_screen.dart` - New `SettingsCategoriesScreen`: AppBar + AsyncLoader-wrapped `ReorderableListView.builder`, per-row toggle auto-save, reorder POST with revert, and own-trail confirm dialog with a real @-prefixed view-trails navigation.

## Decisions Made
- Folded the two independent async providers into one record `AsyncValue` so a single `AsyncLoader` skeleton covers the list (rather than nesting two loaders).
- Kept `onReorder` (not `onReorderItem`) with the explicit index-shift line because the plan's grep contract mandates it; suppressed the info-level `deprecated_member_use` with a scoped `// ignore` to keep `dart analyze` clean without altering the required pattern.
- View-trails resolves the handle inline as `'@${...preferredUsername}'` (matching the app-wide self `/profile/{handle}` convention) after a null-username guard that shows an error toast and aborts — never constructs a bogus null handle.

## Deviations from Plan

None - plan executed exactly as written. All deviations were formatting-only adjustments to satisfy the plan's own single-line grep gates (collapsing the subcategories `context.push`, the `updateFilter(...copyWith(category: [category]...)` call, and the handle assignment onto single lines, each with a scoped `// ignore: lines_longer_than_80_chars`), plus a scoped `// ignore: deprecated_member_use` on `onReorder`. No behavioral or scope changes.

## Issues Encountered
- Initial grep-gate failures were all cosmetic: the `! grep SwitchListTile` gate matched a code comment mentioning `SwitchListTile` (reworded the comment); and several multi-line-formatted calls (`context.push('/settings/...`, `updateFilter((f) => f.copyWith(category: [category]`, and the `'@...preferredUsername` handle) did not match the plan's single-line grep patterns (collapsed to single lines). No logic changes were needed.
- The `onReorder` API is deprecated on the pinned Flutter version (info-level only); resolved with a scoped ignore rather than switching to `onReorderItem`, preserving the plan-mandated explicit index-shift.

## User Setup Required
None - no external service configuration required. The `/settings/categories/subcategories` and `/profile/{handle}/trails` routes are registered/wired by Plan 04 and the existing profile-trails stack respectively.

## Next Phase Readiness
- `SettingsCategoriesScreen` is complete and ready for Plan 04 to register at `/settings/categories` and for the settings tile to link to it.
- The body-tap navigates to `/settings/categories/subcategories` (Plan 04 registers this route + `SettingsSubcategoriesScreen` from Plan 03).
- No blockers.

---
*Phase: 12-settings-categories-screen*
*Completed: 2026-07-01*

## Self-Check: PASSED

`app/lib/routes/settings_categories_screen.dart` and `12-02-SUMMARY.md` exist on disk; both task commits (`ef39e2fc`, `ba5a00de`) present in git history. `dart analyze` on the screen reports "No issues found!".
