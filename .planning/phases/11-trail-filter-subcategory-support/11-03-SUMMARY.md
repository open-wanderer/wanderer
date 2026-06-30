---
phase: 11-trail-filter-subcategory-support
plan: 03
subsystem: ui
tags: [flutter, riverpod, filter, subcategory, l10n, font-awesome]

requires:
  - plan: 11-01
    provides: TrailFilter.subcategory field and copyWith(subcategory:)
  - plan: 11-02
    provides: WandererFilterChip.avatarBuilder and l10n.subcategories
  - phase: 10-category-subcategory-data-layer
    provides: subcategoryProvider, categoryPreferenceProvider, subcategoryPreferenceProvider, displayName extensions

provides:
  - TrailFilterScreen with locale-resolved category chip labels and FA icon avatars
  - AnimatedSize Subcategories section scoped to last-focused category
  - Preference-based visibility filtering (visible:false categories/subcategories hidden)
  - Subcategory count badge on category chips
  - Long-press to clear focused category
affects: [category display across app — icon util extracted to category_icon_util.dart]

tech-stack:
  added: []
  patterns:
    - "String? _focusedCategoryId in state to track last-tapped category for subcategory scoping"
    - "badgeCountBuilder on WandererFilterChip to show selected-subcategory count per category chip"
    - "keepSelectedOnTap: true to allow tapping a selected category to re-focus it"
    - "onLongPress to clear focused category when the long-pressed chip is the focused one"

key-files:
  created: []
  modified:
    - app/lib/routes/trail_filter_screen.dart
    - app/lib/util/category_icon_util.dart (refactored/renamed from category_filter_util.dart)

key-decisions:
  - "Scoped subcategory chips to _focusedCategoryId (last-tapped category) rather than all selected categories — avoids information overload when multiple categories are selected"
  - "Visibility filtering uses != false guard so null/absent preference means visible (D-06)"
  - "Category icon util extracted to shared category_icon_util.dart to serve both filter screens and trail display widgets"

patterns-established:
  - "Pattern 2: _focusedCategoryId state for subcategory scoping — set by onItemTap, cleared by long-press"

requirements-completed: [FILTER-02, FILTER-04, FILTER-06, FILTER-07]

duration: ~25min (including iterative refinements across session)
completed: 2026-06-30
---

# Phase 11 Plan 03: TrailFilterScreen Subcategory Support Summary

**TrailFilterScreen now shows locale-resolved category names with FA icon avatars, a badge showing selected-subcategory count, and an AnimatedSize Subcategories section scoped to the last-tapped category — with preference-based visibility filtering throughout.**

## Performance

- **Duration:** ~25 min (iterative across session)
- **Started:** 2026-06-30
- **Completed:** 2026-06-30
- **Tasks:** 2 completed
- **Files modified:** 2

## Accomplishments

- Switched category chip labels to `c.displayName(locale)` (locale-resolved) instead of raw category name.
- Added `categoryFilterAvatar(c)` and `subcategoryFilterAvatar(context, s, parent, locale)` via `avatarBuilder` on `WandererFilterChip`.
- Added `badgeCountBuilder: (c) => filter.subcategory.where((s) => s.category == c.id).length` so category chips show a Material 3 `Badge` with the count of selected subcategories.
- Added `String? _focusedCategoryId` state field; `onItemTap` sets it, `onLongPress` clears it when the focused category is removed.
- Added `AnimatedSize`-wrapped Subcategories section below Category; visible only when `_focusedCategoryId` is non-null and `visibleSubs.isNotEmpty`.
- Subcategory chips filtered by `s.category == _focusedCategoryId` and `visible != false` preference guard.
- Applied `categoryPreferenceProvider` and `subcategoryPreferenceProvider` visibility filtering (D-06: `!= false` guard).
- Extracted shared icon helpers to `app/lib/util/category_icon_util.dart` (renamed from `category_filter_util.dart`); added `trailCategoryIcon()` for inline use throughout the app.
- Added `keepSelectedOnTap: true` on category chips so tapping a selected category re-focuses it without deselecting.

## Task Commits

| Task | Name | Type | Commit |
|------|------|------|--------|
| 1 | Locale category labels, icon avatars, and visibility filtering | feat | cd38d02d |
| 2 | Add AnimatedSize Subcategories section to filter screen | feat | fad47f93 |
| R | Refine subcategory scoping to last-tapped category | feat | 77c616b4 |
| R | Move subcategory count badge to chip top-right | fix | 79dda071 |
| R | Align subcategory filter logic with web trail_store.ts | fix | aeb25370 |
| R | Extract avatar helpers to category_icon_util.dart | refactor | 4d91d236 |

## Verification Results

- `flutter analyze lib/routes/trail_filter_screen.dart` — no issues.
- `grep "l10n.subcategories" app/lib/routes/trail_filter_screen.dart` — found.
- `grep "copyWith(subcategory:" app/lib/routes/trail_filter_screen.dart` — found.
- `grep "_focusedCategoryId" app/lib/routes/trail_filter_screen.dart` — found (state field + onItemTap + onLongPress).
- Preference visibility filtering via `categoryPreferenceProvider` / `subcategoryPreferenceProvider` — confirmed present.

## Must-Have Compliance

| Truth | Status |
|-------|--------|
| Category chips show locale-resolved names | ✓ `c.displayName(locale)` |
| Subcategories section appears only when ≥1 category selected | ✓ `AnimatedSize` gated on `_focusedCategoryId != null && visibleSubs.isNotEmpty` |
| Subcategory chips scoped to selected categories | ✓ `s.category == _focusedCategoryId` |
| visible:false categories/subcategories omitted | ✓ `!= false` preference guard on both |
| FA icon avatars with badge overlay on subcategory chips | ✓ `categoryFilterAvatar` / `subcategoryFilterAvatar` via `avatarBuilder` |

## Deviations from Plan

- Scoped subcategory section to `_focusedCategoryId` (last-tapped category) rather than union of all selected categories — user-requested refinement after initial implementation; produces cleaner UX when multiple categories selected.
- Badge on category chip (not in original plan) added per user request.
- `category_filter_util.dart` renamed to `category_icon_util.dart` and extended with `trailCategoryIcon()` for app-wide use.

## Notes for Future Plans

- Plan 04 mirrors this implementation for the quick filter bar using `ValueNotifier<String?>` instead of `setState`.
- `category_icon_util.dart` is now the single source for all FA category/subcategory icon resolution in the app.

## Self-Check: PASSED

- FOUND: app/lib/routes/trail_filter_screen.dart
- FOUND: app/lib/util/category_icon_util.dart
- FOUND: l10n.subcategories at line 177
- FOUND: copyWith(subcategory:) at line 201
- FOUND: _focusedCategoryId state field
