# Phase 12: Settings Categories Screen - Context

**Gathered:** 2026-07-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Build `SettingsCategoriesScreen` (category visibility + priority reordering) and a new `SettingsSubcategoriesScreen` (per-category subcategory visibility + reordering), wired into SettingsScreen and go_router at `/settings/categories`. Providers already exist from Phase 10 — this phase is UI only. Includes a confirm-before-disable flow when a user's own trails would be hidden.

**Scope amendment made during this discussion:** REQUIREMENTS.md and ROADMAP.md were updated in place — see `<specifics>` for what changed and why.

</domain>

<decisions>
## Implementation Decisions

### Reordering
- **D-01:** Category rows use `ReorderableListView.builder` with a dedicated drag-handle icon (`Icons.drag_handle`) — not whole-row drag. Ruled out up/down arrow buttons.
- **D-02:** Drag reorder posts to `POST /user-category-preference/reorder` with the full ordered list of category IDs.
- **D-03:** Subcategories are reorderable too (scope amendment) — but only inside `SettingsSubcategoriesScreen`, via the same drag-handle `ReorderableListView` pattern, posting to `POST /user-subcategory-preference/reorder` (endpoint already exists server-side, confirmed in `web/src/routes/api/v1/user-subcategory-preference/reorder/+server.ts`).
- **D-04:** On reorder failure (either endpoint): revert the list to its pre-drag order and show the same error-toast pattern used elsewhere, rather than leaving the list in an unconfirmed state.

### Navigation / Row Layout (redesigned from ExpansionTile)
- **D-05:** No inline `ExpansionTile`. Category rows are: drag handle, icon, locale-resolved name, visibility `Switch`. Tapping the row body (not the switch, not the drag handle) navigates to `SettingsSubcategoriesScreen`, passing the category.
- **D-06:** `SettingsSubcategoriesScreen` reuses the same list pattern as the category screen: AppBar title = parent category's locale-resolved name; body = `ReorderableListView` of subcategories, each with drag handle + visibility `Switch` (no further nesting).
- **D-07:** Categories with zero subcategories are still tappable and navigate to `SettingsSubcategoriesScreen`, which will render an empty state (no dedicated "disable navigation" case was requested).

### Save / Error Feedback
- **D-08:** Visibility toggles (category and subcategory) reuse the existing settings pattern: try/catch around the provider save call, error toast via `toastProvider`/`ToastMessage(type: error)` on failure (mirrors `settings_notifications_screen.dart`), no success toast. Optimistic UI via the watched provider.
- **D-09:** Reorder failures revert + toast per D-04.

### Own-Trail Disable Confirmation (new — mirrors web)
- **D-10:** Turning a visibility switch **OFF** (category or subcategory) first checks whether the user has their own trails using that category/subcategory. If count > 0, show a confirm dialog: "{count} of your trails use this category/subcategory" + a link/action to view those trails, "disable anyway" / "cancel". Cancelling reverts the switch to on (no-op). Confirming proceeds with the normal save (D-08).
- **D-11:** Turning a switch **ON** never triggers this check — only OFF transitions.
- **D-12:** The own-trail count is **not preloaded**; it's fetched at the moment of a toggle-off attempt by querying the trail list API filtered by `author = self AND category/subcategory = id` (reuse `TrailFilter` + the existing trail search endpoint already used by `profile_trails_provider.dart` / Phase 11's `TrailFilter.subcategory`). There is no dedicated count endpoint — web computes this server-side in `+page.server.ts`; mobile must call the trail list API directly.
- **D-13:** Plugin-mapping warnings (web's `confirm-disable-*-active-plugin-mappings` messaging) are explicitly **not ported** — Integrations/plugin system is out of scope for mobile settings v1 (per PROJECT.md).

### Loading State
- **D-14:** Use the existing `AsyncLoader` component (`app/lib/components/async_loader.dart`, already used in `profile_screen.dart`, `map_screen.dart`, `list_screen.dart`) to wrap the category/preference-loading state on both screens — not a bespoke `CircularProgressIndicator` or skeleton.

### Empty state (all categories hidden)
- **D-15:** No special warning if a user ends up with zero visible categories — allowed silently, consistent with no cross-field validation elsewhere in the settings suite. (This is distinct from D-10's per-toggle own-trail warning, which fires per category/subcategory regardless of how many remain visible.)

### Claude's Discretion
- Exact confirm-dialog widget (`AlertDialog` vs a reusable app-wide confirm component, if one exists) — pick whatever matches existing app conventions found during research/planning.
- Exact drag-handle icon size/spacing and row padding — follow existing settings screen visual conventions (16/16/16/8 section padding, etc. from `settings_notifications_screen.dart`).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Web Reference (feature parity source — SettingsCategoriesScreen must mirror this)
- `web/src/routes/settings/categories/+page.svelte` — full reference implementation: reorder handlers (`reorderCategory`, `reorderSubcategory`), disable-confirm flow (`promptBeforeDisable`, `confirmDisable`, `ownTrailCountForCategory`/`ownTrailCountForSubcategory`), expand/collapse state (not ported — replaced by navigation per D-05)
- `web/src/routes/settings/categories/+page.server.ts` — confirms there is **no dedicated trail-count-by-category API**; web computes counts server-side from a full author-scoped trail list. Mobile must replicate via the trail list API (D-12), not expect a new endpoint.
- `web/src/lib/util/category_util.ts` — `sortedCategoriesByPreference()`, `sortedSubcategoriesByPreference()`, `subcategoryVisible()`, `preferenceForCategory()` — sorting/visibility logic to mirror
- `web/src/routes/api/v1/user-category-preference/reorder/+server.ts` and `web/src/routes/api/v1/user-subcategory-preference/reorder/+server.ts` — both reorder endpoints already exist server-side

### Requirements & Roadmap (amended during this discussion)
- `.planning/REQUIREMENTS.md` — SETCAT-08 redefined (subcategory screen, not ExpansionTile); SETCAT-09 clarified (drag-handle, not whole-row); SETCAT-10 added (subcategory reorder); SETCAT-11 added (own-trail confirm dialog). Read the current "Settings Categories" section, not the ROADMAP.md phase summary alone.
- `.planning/ROADMAP.md` — Phase 12 success criteria updated to match

### Flutter Files to Modify/Create
- `app/lib/routes/settings_screen.dart` — add "Categories" list tile (SETCAT-01)
- `app/lib/provider/router_provider.dart` — register `/settings/categories` route (SETCAT-02), and a route for `SettingsSubcategoriesScreen` (likely `/settings/categories/:categoryId` or pushed without a named route — planner's call)
- `app/lib/routes/settings_categories_screen.dart` — new screen (does not exist yet)
- `app/lib/routes/settings_subcategories_screen.dart` — new screen (does not exist yet)
- `app/lib/routes/settings_notifications_screen.dart` — reference for toast/save pattern (`_save`, `_onToggle`, `_sectionHeader`)
- `app/lib/components/async_loader.dart` — loading wrapper to reuse (D-14)

### Phase 10 Providers (already available, read-only in this phase)
- `app/lib/provider/category_preference_provider.dart` — `categoryPreferenceProvider`
- `app/lib/provider/subcategory_preference_provider.dart` — `subcategoryPreferenceProvider`
- `app/lib/provider/trail/category_provider.dart` — `categoryProvider`
- `app/lib/provider/trail/subcategory_provider.dart` — `subcategoryProvider`
- `app/lib/models/category.dart` / `app/lib/models/subcategory.dart` — `CategoryDisplay.displayName(Locale?)` / `SubcategoryDisplay.displayName(Locale?)`

### Trail Query Reference (for D-12's own-trail count)
- `app/lib/provider/profile/profile_trails_provider.dart` — existing pattern for author-scoped, filtered trail queries via `TrailFilter` + `trailFilterProvider`
- `app/lib/models/trail.dart` — `TrailFilter` (now has `category`/`subcategory` fields per Phase 11)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `AsyncLoader` (`app/lib/components/async_loader.dart`) — loading-state wrapper used by `profile_screen.dart`, `map_screen.dart`, `list_screen.dart`
- `toastProvider` / `ToastMessage` — error feedback pattern from `settings_notifications_screen.dart`, `settings_privacy_screen.dart`
- `CategoryDisplay.displayName()` / `SubcategoryDisplay.displayName()` — locale-resolved names (Phase 10)
- `profile_trails_provider.dart` — pattern for author-scoped filtered trail fetches, reusable for the own-trail count check

### Established Patterns
- No `ReorderableListView` exists anywhere in the app yet — this phase introduces the pattern; no prior art to copy from within `app/lib`, must be built from Flutter's standard widget
- Settings screens share `Settings` freezed model + `settingsProvider.saveToServer()`, but category/subcategory preferences use their own dedicated Phase 10 providers/endpoints, not `Settings`
- `_sectionHeader()` 16/16/16/8 padding convention in `settings_notifications_screen.dart`

### Integration Points
- `SettingsScreen` list — add new tile alongside existing Privacy/Language/Notifications tiles
- `router_provider.dart` — existing GoRoute registration pattern for `/settings/*` sub-routes

</code_context>

<specifics>
## Specific Ideas

- **Major deviation from original ROADMAP.md/REQUIREMENTS.md wording, resolved by amending both docs during this discussion:** the user rejected the inline `ExpansionTile` (SETCAT-08 as originally written) in favor of navigating to a dedicated `SettingsSubcategoriesScreen` where subcategories can be both toggled AND reordered. This also un-defers "subcategory reordering within a category," which REQUIREMENTS.md previously listed as out-of-scope/future-milestone. New requirements SETCAT-10 (subcategory reorder) and SETCAT-11 (own-trail confirm dialog) were added; REQUIREMENTS.md and ROADMAP.md are already updated to reflect this — planner/researcher should treat those files (not this summary) as the source of truth for requirement text.
- The own-trail confirm dialog (D-10..D-13) is a direct port of a real web feature the user pointed to (`+page.svelte`'s `promptBeforeDisable`/`confirmDisable`/`ownTrailCountForCategory`), scoped down to exclude plugin-mapping messaging since Integrations are mobile-out-of-scope.

</specifics>

<deferred>
## Deferred Ideas

None — the scope changes discussed (subcategory screen + reorder, own-trail confirm dialog) were folded into this phase's requirements rather than deferred, since they're refinements of SETCAT-08/09 already scoped to Phase 12, not new capabilities for a future phase.

</deferred>

---

*Phase: 12-settings-categories-screen*
*Context gathered: 2026-07-01*
