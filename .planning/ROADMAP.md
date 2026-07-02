# Roadmap: Wanderer Trail Navigation

## Milestones

- ✅ **v1.0 MVP** — Phases 1-3 (shipped 2026-06-13)
- ✅ **v1.1 Offline** — Phases 4-5 (shipped 2026-06-14)
- ✅ **v1.2 Settings Screens** — Phases 6-9 (shipped 2026-06-29)
- 🚧 **v1.3 Category Redesign** — Phases 10-12 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-3) — SHIPPED 2026-06-13</summary>

- [x] Phase 1: Backend API (1/1 plans) — completed 2026-06-12
- [x] Phase 2: Navigation Screen (3/3 plans) — completed 2026-06-13
- [x] Phase 3: Stats Sheet (2/2 plans) — completed 2026-06-13

See `.planning/milestones/v1.0-ROADMAP.md` for full details.

</details>

<details>
<summary>✅ v1.1 Offline (Phases 4-5) — SHIPPED 2026-06-14</summary>

- [x] Phase 4: Serialization Fix + Entity Schema (2/2 plans) — completed 2026-06-14
- [x] Phase 5: Cache Write + Fallback + UI (4/4 plans) — completed 2026-06-14

See `.planning/milestones/v1.1-ROADMAP.md` for full details.

</details>

<details>
<summary>✅ v1.2 Settings Screens (Phases 6-9) — SHIPPED 2026-06-29</summary>

- [x] Phase 6: Settings Navigation + Language & Units (4/4 plans) — completed 2026-06-20
- [x] Phase 7: Privacy (1/1 plan) — completed 2026-06-20
- [x] Phase 8: Account & Profile (3/3 plans) — completed 2026-06-20
- [x] Phase 9: Notifications (1/1 plan) — completed 2026-06-21

See `.planning/milestones/v1.2-ROADMAP.md` for full details.

</details>

### 🚧 v1.3 Category Redesign (In Progress)

**Milestone Goal:** Bring the Flutter app's category system to parity with web PR #1059 — a translations/icon/short_name-aware Category model, a new Subcategory model + provider, subcategory-aware trail filters, and a Settings → Categories screen for per-category/subcategory visibility and priority preferences.

- [x] **Phase 10: Category & Subcategory Data Layer** - Updated Category model, new Subcategory model + entity + provider, locale-aware names, remove `Settings.category`, preference models + providers (SETCAT-03/04/05) (completed 2026-06-29)
- [ ] **Phase 11: Trail Filter Subcategory Support** (2/4 plans) - `TrailFilter.subcategory`, subcategory chips in TrailFilterScreen and quick filter bar, locale-resolved category labels, hidden-category/subcategory omitted from filter chips (FILTER-06/07)
- [x] **Phase 12: Settings Categories Screen** - SettingsCategoriesScreen + SettingsSubcategoriesScreen with visibility toggles, drag-handle priority reordering, own-trail disable confirmation, router wiring (UI only — providers come from Phase 10) (completed 2026-07-01)

## Phase Details

### Phase 10: Category & Subcategory Data Layer

**Goal**: The app's category data model matches web PR #1059 — categories expose locale-aware names, subcategories are fetched and cached, preference models and providers are in place, and the deprecated favourite-sport field is gone.
**Depends on**: Phase 9 (v1.2 settings infrastructure)
**Requirements**: CAT-01, CAT-02, CAT-03, CAT-04, CAT-05, SETCAT-03, SETCAT-04, SETCAT-05
**Success Criteria** (what must be TRUE):

  1. A category's display name renders in the active locale, falling back to English then the raw `name`, and exposes its `icon` and `short_name`.
  2. Subcategories load from `/subcategory` through a Riverpod provider and persist to ObjectBox with indexed `id` and `category` fields, surviving app restarts.
  3. Each subcategory carries its parent `category` id, `name`, `short_name`, `icon`, `badge_icon`, and `translations`.
  4. CategoryPreferenceNotifier and SubcategoryPreferenceNotifier providers fetch the user's preferences from their respective API endpoints.
  5. The app builds and runs with `Settings.category` removed — no remaining references to the old favourite-sport field.**Plans**: 4 plans

**Wave 1**

  - [x] 10-01-PLAN.md — Category + Subcategory + CategoryTranslation models, locale displayName (CAT-01, CAT-02)

**Wave 2** *(blocked on Wave 1 completion)*

  - [x] 10-02-PLAN.md — CategoryEntity extension + SubcategoryEntity (indexed id/category, JSON-blob translations) (CAT-03)

**Wave 3** *(blocked on Wave 2 completion)*

  - [x] 10-03-PLAN.md — Preference models + CategoryPreferenceNotifier/SubcategoryPreferenceNotifier providers (SETCAT-03/04/05)

**Wave 4** *(blocked on Wave 3 completion)*

  - [x] 10-04-PLAN.md — CategoryNotifier ObjectBox write, cache-first SubcategoryNotifier, remove Settings.category (CAT-04, CAT-05)

### Phase 11: Trail Filter Subcategory Support

**Goal**: Users can narrow trail searches by subcategory in both the full filter screen and the quick filter bar, with category labels shown in their language and hidden categories/subcategories omitted from the picker.
**Depends on**: Phase 10
**Requirements**: FILTER-01, FILTER-02, FILTER-03, FILTER-04, FILTER-05, FILTER-06, FILTER-07
**Success Criteria** (what must be TRUE):

  1. With at least one category selected in TrailFilterScreen, the user sees subcategory chips limited to the subcategories of those selected categories.
  2. Tapping subcategory chips toggles them and changes which trails the search returns (the subcategory selection reaches the API filter payload).
  3. Category chips throughout the filter UI display locale-resolved names using the CAT-01 fallback chain.
  4. The quick filter bar's Category bottom sheet lets the user pick both categories and subcategories, and the chosen filter persists in the active trail filter.
  5. Categories and subcategories the user has marked hidden in Settings → Categories do not appear as selectable chips in the filter UI.

**Plans**: TBD
**UI hint**: yes

### Phase 12: Settings Categories Screen

**Goal**: A user can open Settings → Categories to control which categories and subcategories appear and in what priority order, with changes saved automatically.
**Depends on**: Phase 10 (preference providers come from Phase 10)
**Requirements**: SETCAT-01, SETCAT-02, SETCAT-06, SETCAT-07, SETCAT-08, SETCAT-09, SETCAT-10, SETCAT-11
**Success Criteria** (what must be TRUE):

  1. From SettingsScreen the user taps a "Categories" tile and lands on SettingsCategoriesScreen via the `/settings/categories` route.
  2. The screen lists categories sorted by priority (ascending, alphabetical for ties), each row showing the category icon, its locale-resolved name, a visibility switch, and a drag handle.
  3. Toggling a category's visibility switch auto-saves to `/user-category-preference`; tapping the row body (not the switch or drag handle) navigates to `SettingsSubcategoriesScreen` for that category, which lists its subcategories with their own visibility switches saving to `/user-subcategory-preference`.
  4. Reordering categories via drag handle persists the new order via `POST /user-category-preference/reorder`; reordering subcategories within `SettingsSubcategoriesScreen` persists via `POST /user-subcategory-preference/reorder`. Both reflect the saved order on reload, and a failed reorder reverts the list with an error toast.
  5. Turning off a category/subcategory that has the user's own trails shows a confirm dialog with the trail count and a link to view them before saving.

**Plans**: 4 plans

**Wave 1**

  - [x] 12-01-PLAN.md — Provider reorder methods + sort/visibility helpers + own-trail count helper + l10n keys (foundation)

**Wave 2** *(blocked on Wave 1 completion)*

  - [x] 12-02-PLAN.md — SettingsCategoriesScreen: sorted list, visibility toggle, drag-handle reorder, own-trail confirm dialog (SETCAT-06/07/09/11)
  - [x] 12-03-PLAN.md — SettingsSubcategoriesScreen: parent-scoped list + empty state, toggle, reorder, own-trail confirm dialog (SETCAT-08/10/11)

**Wave 3** *(blocked on Wave 2 completion)*

  - [x] 12-04-PLAN.md — Settings "Categories" tile + go_router route wiring (SETCAT-01/02)

**UI hint**: yes

## Progress

**Execution Order:**
Phases execute in numeric order: 10 → 11 → 12 (11 and 12 are independent of each other but both depend on 10).

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Backend API | v1.0 | 1/1 | Complete | 2026-06-12 |
| 2. Navigation Screen | v1.0 | 3/3 | Complete | 2026-06-13 |
| 3. Stats Sheet | v1.0 | 2/2 | Complete | 2026-06-13 |
| 4. Serialization Fix + Entity Schema | v1.1 | 2/2 | Complete | 2026-06-14 |
| 5. Cache Write + Fallback + UI | v1.1 | 4/4 | Complete | 2026-06-14 |
| 6. Settings Navigation + Language & Units | v1.2 | 4/4 | Complete | 2026-06-20 |
| 7. Privacy | v1.2 | 1/1 | Complete | 2026-06-20 |
| 8. Account & Profile | v1.2 | 3/3 | Complete | 2026-06-20 |
| 9. Notifications | v1.2 | 1/1 | Complete | 2026-06-21 |
| 10. Category & Subcategory Data Layer | v1.3 | 4/4 | Complete    | 2026-06-29 |
| 11. Trail Filter Subcategory Support | v1.3 | 0/TBD | Not started | - |
| 12. Settings Categories Screen | v1.3 | 4/4 | Complete    | 2026-07-02 |
