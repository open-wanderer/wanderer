# Phase 11: Trail Filter Subcategory Support - Context

**Gathered:** 2026-06-29
**Status:** Ready for planning

<domain>
## Phase Boundary

Add subcategory selection to both filter surfaces — TrailFilterScreen and the quick filter bar — so users can narrow trail searches by subcategory. Category chips display locale-resolved names. Hidden categories/subcategories (user preference `visible: false`) are omitted from the chip options. No new API endpoints, no new screens — this phase modifies existing filter UI and the `TrailFilter` model only.

</domain>

<decisions>
## Implementation Decisions

### TrailFilter Model
- **D-01:** `TrailFilter` freezed model gains `List<Subcategory> subcategory` field alongside the existing `List<Category> category` field (FILTER-01).
- **D-02:** `TrailFilter.toFilterString()` adds a `subcategory_id IN [...]` clause using subcategory IDs (not names). Field name is `subcategory_id` — mirrors the web's filter builder in `trail_store.ts` and the PocketBase column name confirmed in `web/src/lib/models/trail.ts`.

### Subcategory Chips in TrailFilterScreen
- **D-03:** New titled **"Subcategories"** section below the Category section. Section is conditionally rendered when `filter.category.isNotEmpty`. Matches the existing labelled-block layout pattern in TrailFilterScreen.
- **D-04:** Section animates in/out with `AnimatedSize` (standard Flutter widget, zero extra dependencies). Smooth expand when categories are selected, smooth collapse when all categories are deselected.
- **D-05:** Subcategory chips display **only subcategories belonging to the currently selected categories** (filter: `subcategory.category == selectedCategory.id`).
- **D-06:** Hidden categories (`visible: false` in CategoryPreference) and hidden subcategories (`visible: false` in SubcategoryPreference) are omitted from all chip options. Missing preference record (null) = visible (default shown).

### Chip Icon Rendering
- **D-07:** `WandererFilterChip<T>` is extended with an optional `Widget? Function(T item)? avatarBuilder`. Existing call sites pass `null` (no change). New category/subcategory call sites pass a builder that returns an icon widget.
- **D-08:** Category chips show a FontAwesome icon via `avatarBuilder`. Icon name resolved from `category.icon` using `fontAwesomeIconsMap` in `icon_util.dart`. Strip `fa-` prefix before lookup (mirrors `displayCategoryIcon()` in `category_util.ts`). Fall back to `Icons.category` (Material) if icon name not found in map.
- **D-09:** Subcategory chips show primary FA icon + `badge_icon` overlay. Implementation: `Stack` in `avatarBuilder` — primary `FaIcon` (16px), badge `FaIcon` (10px) positioned `Alignment.bottomRight`. Primary icon falls back to parent category's icon if `subcategory.icon` is empty (mirrors `displaySubcategoryIcon()` from `category_util.ts`). Badge icon omitted if `subcategory.badgeIcon` is null/empty.
- **D-10:** Locale-resolved names via `CategoryDisplay.displayName(Localizations.localeOf(context))` for category chips and the equivalent for subcategory display names. Uses the Phase 10 `CategoryDisplay` helper. This is the established locale access pattern (from `profile_screen.dart`).

### Quick Filter Bar
- **D-11:** The existing **Category chip opens a single bottom sheet** — no second chip added to the bar. The sheet is extended to include a Subcategories section below the categories.
- **D-12:** Subcategory section inside the bottom sheet follows the **same AnimatedSize pattern** as TrailFilterScreen (D-04).
- **D-13:** The Category chip's active state uses `filter.category.isNotEmpty || filter.subcategory.isNotEmpty`. Activates when either category or subcategory is selected. (Mirrors `_isCategoryActive()` method — update it.)
- **D-14:** Bottom sheet initial size stays at `initialChildSize: 0.5`. The existing `maxChildSize: 0.9` and scroll controller handle overflow.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Web Reference (icon + filter patterns to mirror)
- `web/src/lib/util/category_util.ts` — `displayCategoryIcon()`, `displaySubcategoryIcon()`, `displaySubcategoryBadgeIcon()`, `subcategoryVisible()`, `sortedCategoriesByPreference()` — Flutter implementations must mirror this logic
- `web/src/lib/models/trail.ts` — confirms `subcategory_id` as the PocketBase column name for the trail subcategory relation (line 219)

### Flutter Filter Screens (files to modify)
- `app/lib/models/trail.dart` — `TrailFilter` freezed model (add `subcategory` field) and `toFilterString()` (add `subcategory_id IN [...]` clause)
- `app/lib/routes/trail_filter_screen.dart` — add Subcategories section below Category section
- `app/lib/components/trail/trail_quick_filter_bar.dart` — extend Category bottom sheet with Subcategories section; update `_isCategoryActive()`

### Flutter Components (files to modify/use)
- `app/lib/components/base/wanderer_filter_chip.dart` — extend with optional `avatarBuilder` parameter
- `app/lib/util/icon_util.dart` — `fontAwesomeIconsMap` for FA icon name → `FaIconData` lookup; existing `iconsMap` for fallback

### Phase 10 Providers (already available, read-only in this phase)
- `app/lib/provider/trail/category_provider.dart` — `categoryProvider` returns `List<Category>` with locale-aware display
- `app/lib/provider/trail/subcategory_provider.dart` — `subcategoryProvider` cache-first, returns `List<Subcategory>`
- `app/lib/provider/category_preference_provider.dart` — `categoryPreferenceProvider` for visibility filtering
- `app/lib/provider/subcategory_preference_provider.dart` — `subcategoryPreferenceProvider` for visibility filtering
- `app/lib/models/category.dart` — `CategoryDisplay.displayName(Locale?)` helper
- `app/lib/models/subcategory.dart` — `SubcategoryDisplay.displayName(Locale?)` helper

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `WandererFilterChip<T>` (`app/lib/components/base/wanderer_filter_chip.dart`) — generic chip widget using `FilterChip`. Has `avatar` slot via Material. Extend with `avatarBuilder`.
- `subcategoryProvider` (Phase 10) — cache-first, `keepAlive: true`, returns `List<Subcategory>`
- `categoryPreferenceProvider` / `subcategoryPreferenceProvider` (Phase 10) — for visibility filtering
- `CategoryDisplay.displayName(Locale?)` / `SubcategoryDisplay.displayName(Locale?)` — locale fallback chain

### Established Patterns
- `Localizations.localeOf(context)` — locale access in widget context (from `profile_screen.dart:219`)
- `AnimatedSize` — standard Flutter animated expansion, no extra deps
- `DraggableScrollableSheet` in quick filter bar — already scrollable, no size change needed
- `fontAwesomeIconsMap` in `icon_util.dart` — FA icon name string → `FaIconData`. Strip `fa-` prefix before lookup.
- `filter.category.isNotEmpty` active check pattern in `trail_quick_filter_bar.dart:98` → update to OR subcategory check

### Integration Points
- `TrailFilterNotifier.updateFilter()` — existing mechanism for state updates; subcategory changes use same pattern
- `TrailFilter.toFilterString()` — add `subcategory_id IN ['id1','id2']` clause alongside existing `category IN [...]` clause
- `TrailFilterValues` (PocketBase saved filters) — check if subcategory needs to be added to the persisted filter values schema
- `_isCategoryActive()` in `trail_quick_filter_bar.dart:98` — update to also check `filter.subcategory.isNotEmpty`

</code_context>

<specifics>
## Specific Ideas

- **Icon rendering must match web:** user explicitly referenced `category_util.ts` as the spec for how icons/badges are rendered. The Flutter implementation must mirror `displaySubcategoryIcon()` (primary icon with parent fallback) and `displaySubcategoryBadgeIcon()` (badge overlay).
- **Badge overlay dimensions:** primary `FaIcon` at 16px, badge `FaIcon` at 10px, badge positioned `Alignment.bottomRight` inside a `Stack`.
- Both filter surfaces (TrailFilterScreen and quick filter bar) must behave consistently — same section layout, same AnimatedSize, same chip pattern.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 11-trail-filter-subcategory-support*
*Context gathered: 2026-06-29*
