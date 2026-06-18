---
quick_id: 260618-o0r
slug: add-quick-filter-bar-to-library-and-prof
description: Add quick filter bar to library and profile trail screens with horizontal action chips opening bottom sheets for Sort, Category, Difficulty, Elevation, Date, and Completion Status filters using family trail filter provider
date: 2026-06-18
status: complete
commits:
  - c5831ae8
  - fdfd75e1
---

# Quick Task 260618-o0r — Summary

## What was done

**Task 1 — Provider family conversion (c5831ae8):**
- `TrailFilterNotifier.build()` now accepts `String filterId`, making it a family provider
- `trail_filter_provider.g.dart` hand-updated to follow the `profile_trails_provider.g.dart` family pattern with `isAutoDispose: false` (honours `keepAlive: true`)
- All existing call sites updated to `trailFilterProvider('map')`:
  - `map_screen.dart`
  - `map_trail_search_provider.dart`
  - `trail_search_provider.dart`
- `TrailFilterScreen` and `TrailSortScreen` gained a `filterId` parameter defaulting to `'map'`

**Task 2 — Filter bar + screen wiring (fdfd75e1):**
- Created `app/lib/components/trail/trail_quick_filter_bar.dart` with `TrailQuickFilterBar(filterId: ...)` widget
- Seven `ActionChip`s: Sort, Category, Difficulty, Elevation, Date, Completion Status, Reset
- Each chip opens a `DraggableScrollableSheet` reusing existing filter widgets (WandererSortChipGroup, WandererFilterChip, RangeSlider, WandererDatePicker, WandererRadioGroup)
- Chips show `primaryContainer` color when their filter section differs from default
- `applyTrailFilter()` — client-side filter for offline `Trail` objects (library)
- `applyProfileTrailFilter()` — client-side filter for `TrailSearchResult` objects (profile trails)
- `library_screen.dart`: adds `TrailQuickFilterBar(filterId: 'library')`, applies filter via `applyTrailFilter()`
- `profile_trail_screen.dart`: adds `TrailQuickFilterBar(filterId: 'profile_trail_$handle')`, applies filter via `applyProfileTrailFilter()`
