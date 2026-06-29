# Requirements: Wanderer Trail Navigation

**Defined:** 2026-06-29
**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

## v1.3 Requirements

Requirements for milestone v1.3 — Category Redesign. Each maps to roadmap phases.

### Category Data Layer

- [x] **CAT-01**: Category model gains `icon`, `short_name`, and `translations` fields; display name is resolved by active locale with fallback to English then raw `name`
- [x] **CAT-02**: Subcategory freezed model created with `id`, `category` (parent ID), `name`, `short_name`, `icon`, `badge_icon`, `translations`
- [x] **CAT-03**: SubcategoryEntity added to ObjectBox with indexed `id` and `category` fields
- [x] **CAT-04**: SubcategoryNotifier Riverpod provider fetches all subcategories from `/subcategory`
- [x] **CAT-05**: `Settings.category` field removed from the freezed model and all call sites (replaced by category preferences)

### Trail Filters

- [ ] **FILTER-01**: `TrailFilter` freezed model gains a `subcategory` list alongside the existing `category` list
- [ ] **FILTER-02**: TrailFilterScreen shows a subcategory chip section that appears when ≥1 category is selected, listing subcategories belonging to those selected categories
- [ ] **FILTER-03**: Selecting/deselecting subcategory chips updates `TrailFilter.subcategory` and the filter payload sent to the API
- [ ] **FILTER-04**: Category chips in TrailFilterScreen display locale-resolved names (translation fallback chain from CAT-01)
- [ ] **FILTER-05**: Quick filter bar (trail_quick_filter_bar.dart) Category chip bottom sheet supports selecting both categories and subcategories
- [ ] **FILTER-06**: TrailFilterScreen and quick filter bar omit categories marked `visible: false` in the user's category preferences from the selectable chip options
- [ ] **FILTER-07**: TrailFilterScreen subcategory chips omit subcategories marked `visible: false` in the user's subcategory preferences from the selectable chip options

> **Note:** Trail results are already filtered server-side — all SvelteKit trail API routes apply `withTrailPreferenceFilter` using the user's auth token; Flutter receives pre-filtered results automatically.

### Settings Categories

- [ ] **SETCAT-01**: SettingsScreen gains a "Categories" list tile (FontAwesome tag or layer-group icon) navigating to `/settings/categories`
- [ ] **SETCAT-02**: go_router registers `/settings/categories` route pointing to `SettingsCategoriesScreen`
- [x] **SETCAT-03**: `CategoryPreference` and `SubcategoryPreference` freezed models created with `id?`, `user`, `category`/`subcategory`, `visible?`, `priority?`
- [x] **SETCAT-04**: CategoryPreferenceNotifier provider fetches from `GET /user-category-preference` and upserts via `PUT /user-category-preference`
- [x] **SETCAT-05**: SubcategoryPreferenceNotifier provider fetches from `GET /user-subcategory-preference` and upserts via `PUT /user-subcategory-preference`
- [ ] **SETCAT-06**: SettingsCategoriesScreen lists categories sorted by priority (ascending, alphabetical for ties), each row shows category icon and locale-resolved name
- [ ] **SETCAT-07**: Each category row has a visibility SwitchListTile; toggling sends PUT to `/user-category-preference` with `visible: false/true` and auto-saves
- [ ] **SETCAT-08**: Each category row is expandable (ExpansionTile) to reveal its subcategories, each with its own visibility SwitchListTile
- [ ] **SETCAT-09**: SettingsCategoriesScreen uses ReorderableListView; completing a drag calls `POST /user-category-preference/reorder` with the new ordered list of category IDs

## Future Requirements

### Trail Create / Edit

- **TRAILFORM-01**: Category and subcategory picker in trail create/edit form
- **TRAILFORM-02**: Category pre-selected based on highest-priority category preference

### Bulk Edit

- **BULK-01**: Bulk-edit modal for category/subcategory/difficulty on multiple trails (web-only for now)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Drag-to-reorder with pointer events (web style) | ReorderableListView is the Flutter-idiomatic equivalent |
| Category/subcategory picker in trail form | Requires trail form rework; deferred to v1.4 |
| Bulk-edit modal | Web-only feature; out of scope for mobile |
| Subcategory reordering within a category | Lower priority; can add in future milestone |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| CAT-01 | Phase 10 | Complete |
| CAT-02 | Phase 10 | Complete |
| CAT-03 | Phase 10 | Complete |
| CAT-04 | Phase 10 | Complete |
| CAT-05 | Phase 10 | Complete |
| FILTER-01 | Phase 11 | Pending |
| FILTER-02 | Phase 11 | Pending |
| FILTER-03 | Phase 11 | Pending |
| FILTER-04 | Phase 11 | Pending |
| FILTER-05 | Phase 11 | Pending |
| FILTER-06 | Phase 11 | Pending |
| FILTER-07 | Phase 11 | Pending |
| SETCAT-01 | Phase 12 | Pending |
| SETCAT-02 | Phase 12 | Pending |
| SETCAT-03 | Phase 10 | Complete |
| SETCAT-04 | Phase 10 | Complete |
| SETCAT-05 | Phase 10 | Complete |
| SETCAT-06 | Phase 12 | Pending |
| SETCAT-07 | Phase 12 | Pending |
| SETCAT-08 | Phase 12 | Pending |
| SETCAT-09 | Phase 12 | Pending |

**Coverage:**

- v1.3 requirements: 21 total
- Mapped to phases: 21
- Unmapped: 0 ✓

**By phase:**

- Phase 10 (Category & Subcategory Data Layer): 8 requirements (CAT-01..05, SETCAT-03/04/05 — preference models + providers moved here so Phase 11 can use them)
- Phase 11 (Trail Filter Subcategory Support): 7 requirements (FILTER-01..07)
- Phase 12 (Settings Categories Screen): 6 requirements (SETCAT-01/02/06/07/08/09 — UI only, providers come from Phase 10)

---
*Requirements defined: 2026-06-29*
*Last updated: 2026-06-29 after adding FILTER-06/07 (hidden-category visibility in filter UI) — SETCAT-03/04/05 moved to Phase 10 so Phase 11 can read preferences*
