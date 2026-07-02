---
phase: 12-settings-categories-screen
verified: 2026-07-02T10:00:00Z
status: passed
resolved: 2026-07-02T12:00:00Z
resolution: "All 8 human_verification items completed via 12-UAT.md (8/8 passed, 0 issues). Security review closed 8/8 threats (12-SECURITY.md)."
score: 5/5 must-haves verified
overrides_applied: 0
human_verification:
  - test: "Sort, icon, name, drag-handle and switch render correctly on SettingsCategoriesScreen"
    expected: "Categories listed sorted by priority ascending (alphabetical tie-break), each row shows a drag handle, category icon, locale-resolved name, and a visibility switch"
    why_human: "Visual layout, icon rendering, and locale-name resolution on a live device cannot be confirmed by static analysis"
  - test: "Drag a category row by the handle to reorder; release, leave and re-enter the screen"
    expected: "New order persists after reload; simulating a network failure (e.g. airplane mode) during a drag reverts the list to the prior order and shows an error toast"
    why_human: "Requires live gesture interaction and a real/simulated network failure to observe optimistic-update revert behavior"
  - test: "Toggle a category OFF that has the signed-in user's own trails"
    expected: "A confirm dialog appears with the trail count, 'View trails', 'Disable anyway', and 'Cancel' actions. Tapping 'View trails' opens the user's own profile trail list filtered to only that category, without saving or changing the switch. 'Cancel' reverts the switch to ON. 'Disable anyway' saves the change. Toggling ON never shows this dialog."
    why_human: "Requires a live backend, an authenticated user with owned trails, and visual/behavioral confirmation of dialog and downstream navigation"
  - test: "Tap a category row body (not the switch or drag handle)"
    expected: "Navigates to SettingsSubcategoriesScreen for that category, showing the category's localized name as the AppBar title"
    why_human: "Navigation target and independent hit-target behavior (body vs. switch vs. handle) needs live-gesture confirmation"
  - test: "Open a category with zero subcategories from SettingsCategoriesScreen"
    expected: "SettingsSubcategoriesScreen shows the empty-state copy ('No subcategories' / body text) and the screen remains reachable (no crash, no blank hang)"
    why_human: "Depends on live category/subcategory data with a genuinely empty subcategory set, and visual confirmation of the empty state"
  - test: "Drag a subcategory row by the handle to reorder within SettingsSubcategoriesScreen; release, leave and re-enter"
    expected: "New order persists, scoped to that parent category only; simulated network failure reverts the list with an error toast"
    why_human: "Requires live gesture interaction, a real/simulated network failure, and confirmation the reorder is scoped correctly per-category"
  - test: "Toggle a subcategory OFF that has the signed-in user's own trails"
    expected: "A dialog titled 'Hide this subcategory?' appears with the count, 'View trails', 'Disable anyway', 'Cancel'. 'View trails' opens the user's own profile trail list filtered to only that subcategory without saving. Cancel reverts to ON; Disable anyway saves. ON never triggers this dialog."
    why_human: "Requires a live backend, an authenticated user with owned trails in a subcategory, and visual/behavioral confirmation"
  - test: "Full end-to-end reachability: Settings tab -> Categories tile -> SettingsCategoriesScreen -> tap category row -> SettingsSubcategoriesScreen"
    expected: "The whole flow is navigable on a real device/simulator without crashes, including the CR-01 deep-link/restart fallback (state.extra as Category guard) not being hit under normal in-app navigation"
    why_human: "End-to-end navigation flow and app restart / deep-link fallback behavior require live app execution"
---

# Phase 12: Settings Categories Screen Verification Report

**Phase Goal:** A user can open Settings → Categories to control which categories and subcategories appear and in what priority order, with changes saved automatically.
**Verified:** 2026-07-02T10:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | From SettingsScreen the user taps a "Categories" tile and lands on SettingsCategoriesScreen via `/settings/categories` | VERIFIED | `app/lib/routes/settings_screen.dart:53-58` — `ListTile` with `FaIcon(FontAwesomeIcons.layerGroup)`, `Text(l10n.categories)`, `onTap: () => context.push('/settings/categories')`. `app/lib/provider/router_provider.dart:196-199` registers `GoRoute(path: 'categories', builder: ... const SettingsCategoriesScreen())` under `/settings`. |
| 2 | The screen lists categories sorted by priority (ascending, alphabetical for ties), each row showing icon, locale-resolved name, visibility switch, and drag handle | VERIFIED | `app/lib/routes/settings_categories_screen.dart:118-132` calls `sortedCategoriesByPreference` (`app/lib/util/category_preference_sort.dart:22-45` — prioritized-first ascending, `displayName(locale).toLowerCase()` tie-break). Row build (`_buildRow:212-262`) renders `ReorderableDragStartListener` handle, `categoryFilterAvatar(category)`, `category.displayName(locale)`, trailing `Switch`. |
| 3 | Toggling a category's visibility switch auto-saves to `/user-category-preference`; tapping the row body (not switch/handle) navigates to SettingsSubcategoriesScreen, which lists subcategories with their own switches saving to `/user-subcategory-preference` | VERIFIED | `_onToggle`/`_save` (`settings_categories_screen.dart:266-276,53-69`) calls `categoryPreferenceProvider.notifier.upsert` which PUTs `/user-category-preference` (`category_preference_provider.dart:43-49`). `InkWell.onTap` (line 235) pushes `/settings/categories/subcategories` with `extra: category`, independent of the `Switch` (line 253-257) and drag handle (line 225-231) — three separate hit targets. `SettingsSubcategoriesScreen` mirrors the same `upsert` pattern against `subcategory_preference_provider.dart:43-49` (PUT `/user-subcategory-preference`). |
| 4 | Reordering categories persists via `POST /user-category-preference/reorder`; reordering subcategories persists via `POST /user-subcategory-preference/reorder`; both reflect on reload; a failed reorder reverts with an error toast | VERIFIED | `CategoryPreferenceNotifier.reorder` posts `{'categories': orderedCategoryIds}` to `/user-category-preference/reorder` + `invalidateSelf()` (`category_preference_provider.dart:54-60`). `SubcategoryPreferenceNotifier.reorder(categoryId, ids)` posts `{'category':.., 'subcategories':..}` to `/user-subcategory-preference/reorder` (`subcategory_preference_provider.dart:55-64`). Both server routes exist and forward to the Go backend (`web/src/routes/api/v1/user-category-preference/reorder/+server.ts`, `.../user-subcategory-preference/reorder/+server.ts`, backed by `db/main.go:183-184` → `routes.CategoryPreferencesReorder` / `routes.SubcategoryPreferencesReorder`). `_onReorder` in both screens applies the index-shift fix, optimistic `setState`, and on error `setState` back to the pre-drag snapshot + error toast (`settings_categories_screen.dart:173-207`, `settings_subcategories_screen.dart:198-229`). |
| 5 | Turning off a category/subcategory with the user's own trails shows a confirm dialog with the trail count and a link to view them before saving | VERIFIED | `_onToggleOff` in both screens calls `ownTrailCount` (wrapped in try/catch post-CR-02-fix), and if `count > 0` shows `showDialog<bool>` `AlertDialog` with title/body-with-count/"View trails"/"Disable anyway"/"Cancel" (`settings_categories_screen.dart:284-350`, `settings_subcategories_screen.dart:294-358`). "View trails" (`_viewOwnTrails`) resolves the `@`-prefixed own handle, awaits `trailFilterProvider(...).future` (post-WR-01-fix) before seeding the filter, then pushes `/profile/$handle/trails`. Cancel returns `false` and the switch is untouched (provider state never changed); Confirm saves via `upsert(id, false)`. ON-toggle skips the check entirely (`_onToggle`, line 266-276 / 276-286). |

**Score:** 5/5 truths verified

### PLAN-Level Must-Haves (all four plans)

| Plan | Must-have truth | Status | Evidence |
|------|------------------|--------|----------|
| 12-01 | `reorder` methods on both notifiers with verified payload shapes, no `user` field | VERIFIED | Confirmed above; grep for `'user'` key in reorder payloads returns none. |
| 12-01 | Sort/visibility helpers with correct priority/tie-break/visible-by-default semantics | VERIFIED | `category_preference_sort.dart` — logic matches spec exactly (prioritized-first ascending, `!= false` visibility check for null-safety). |
| 12-01 | `ownTrailCount` lazy author-scoped count with fallback chain | VERIFIED | `own_trail_count.dart:13-53` — POSTs `/profile/$handle/trails` with `category_id`/`subcategory_id IN [...]` filter, reads `totalHits ?? estimatedTotalHits ?? hits.length ?? 0`, coerced defensively (post-WR-03 fix). |
| 12-01 | Seven l10n keys added, AppLocalizations regenerated | VERIFIED | All seven keys present in `app_en.arb` (lines 496-509) and exposed as getters in `app_localizations.dart` (lines 3023-3063). |
| 12-02 | SettingsCategoriesScreen: sort, toggle, reorder, confirm dialog, independent hit targets | VERIFIED | See SC 2-5 above. |
| 12-03 | SettingsSubcategoriesScreen: parent-scoped list, empty state, toggle, reorder, confirm dialog | VERIFIED | `settings_subcategories_screen.dart` filters to `sub.category == widget.category.id` (line 96), empty state at `_buildEmptyState` reachable when filtered list is empty (lines 112-116, 140-162), reorder posts `widget.category.id` FIRST (line 215), confirm dialog uses subcategory-titled key (line 328). |
| 12-04 | Settings tile + go_router registration (both routes) | VERIFIED | See SC1 above; nested `subcategories` GoRoute at `router_provider.dart:200-212`, with the CR-01 type-guard fallback (`extra is! Category` → `SettingsCategoriesScreen()`) in place. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/provider/category_preference_provider.dart` | `reorder(List<String>)` method | VERIFIED | Present, correct payload, `invalidateSelf()` called. |
| `app/lib/provider/subcategory_preference_provider.dart` | `reorder(String, List<String>)` method | VERIFIED | Present, correct payload, `invalidateSelf()` called. |
| `app/lib/util/category_preference_sort.dart` | sort + visibility helpers | VERIFIED | 93 lines, all four functions present and analyzer-clean. |
| `app/lib/util/own_trail_count.dart` | lazy own-trail count | VERIFIED | 53 lines, real network call + defensive coercion. |
| `app/lib/i18n/app_en.arb` | 7 new l10n keys | VERIFIED | All present with `@`-metadata. |
| `app/lib/routes/settings_categories_screen.dart` | Category settings screen | VERIFIED | 403 lines; `ConsumerStatefulWidget`; substantive, wired, analyzer-clean (2 info-level lints only). |
| `app/lib/routes/settings_subcategories_screen.dart` | Subcategory settings screen | VERIFIED | 413 lines; substantive, wired, analyzer-clean. |
| `app/lib/routes/settings_screen.dart` | Categories tile | VERIFIED | Tile present, wired to `context.push('/settings/categories')`. |
| `app/lib/provider/router_provider.dart` | categories + subcategories routes | VERIFIED | Both nested routes registered, CR-01 guard applied. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `settings_screen.dart` | `/settings/categories` | `context.push` | WIRED | Confirmed at line 56. |
| `router_provider.dart` | `SettingsCategoriesScreen` | `GoRoute` builder | WIRED | `const SettingsCategoriesScreen()` at `categories` path. |
| `router_provider.dart` | `SettingsSubcategoriesScreen` | `GoRoute` builder (nested) | WIRED | Type-guarded `extra as Category` with fallback (CR-01 fix applied). |
| `settings_categories_screen.dart` | `categoryProvider` + `categoryPreferenceProvider` | `ref.watch` | WIRED | Both watched, combined into one `AsyncValue` record. |
| `settings_categories_screen.dart` | `CategoryPreferenceNotifier.reorder` | `ref.read(...).reorder(...)` | WIRED | Called in `_onReorder` with pre/post-drag snapshot handling. |
| `settings_categories_screen.dart` | `ownTrailCount` | function call | WIRED | Called in `_onToggleOff`, wrapped in try/catch (CR-02 fix). |
| `settings_categories_screen.dart` | `/profile/@{handle}/trails` | `context.push` | WIRED | `_viewOwnTrails` resolves `@`-prefixed handle, seeds filter, pushes. |
| `settings_subcategories_screen.dart` | `subcategoryProvider` + `subcategoryPreferenceProvider` | `ref.watch` | WIRED | Confirmed. |
| `settings_subcategories_screen.dart` | `SubcategoryPreferenceNotifier.reorder(categoryId, ids)` | `ref.read(...).reorder(...)` | WIRED | Parent id passed first, confirmed at line 215. |
| `settings_subcategories_screen.dart` | `ownTrailCount(isSubcategory: true)` | function call | WIRED | Confirmed, try/catch applied. |
| `settings_subcategories_screen.dart` | `/profile/@{handle}/trails` | `context.push` | WIRED | Confirmed. |
| `user-category-preference/reorder` (SvelteKit) | `/category-preferences/reorder` (Go) | `pb.send` | WIRED | Confirmed route registration in `db/main.go:183`. |
| `user-subcategory-preference/reorder` (SvelteKit) | `/subcategory-preferences/reorder` (Go) | `pb.send` | WIRED | Confirmed route registration in `db/main.go:184`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `settings_categories_screen.dart` | `categoriesAsync` | `categoryProvider` → `GET /category` + ObjectBox cache fallback | Yes — real API fetch with offline cache | FLOWING |
| `settings_categories_screen.dart` | `prefsAsync` | `categoryPreferenceProvider` → `GET /user-category-preference` | Yes | FLOWING |
| `settings_subcategories_screen.dart` | `subcategories` | `subcategoryProvider` → cache-first + background `GET /subcategory` refresh | Yes | FLOWING |
| `settings_subcategories_screen.dart` | `prefsAsync` | `subcategoryPreferenceProvider` → `GET /user-subcategory-preference` | Yes | FLOWING |
| `own_trail_count.dart` | `ownTrailCount` return | `POST /profile/$handle/trails` (Meilisearch-backed, author-scoped) | Yes — real search query, not hardcoded | FLOWING |

No hollow props or static/empty-array stubs found in any Phase 12 artifact.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SETCAT-01 | 12-04 | Categories tile navigating to `/settings/categories` | SATISFIED | `settings_screen.dart:53-58` |
| SETCAT-02 | 12-04 | go_router registers `/settings/categories` route | SATISFIED | `router_provider.dart:196-212` |
| SETCAT-06 | 12-02 | Sorted category list with icon + locale name | SATISFIED | `sortedCategoriesByPreference` + row rendering |
| SETCAT-07 | 12-02 | Visibility switch auto-saves via PUT | SATISFIED | `upsert` call chain |
| SETCAT-08 | 12-03 | Dedicated SettingsSubcategoriesScreen with own switches | SATISFIED | `settings_subcategories_screen.dart` |
| SETCAT-09 | 12-01, 12-02 | Category drag-handle reorder POSTs ordered IDs | SATISFIED | `reorder(List<String>)` + `_onReorder` |
| SETCAT-10 | 12-01, 12-03 | Subcategory reorder POSTs parent id + ordered IDs | SATISFIED | `reorder(categoryId, ids)` + `_onReorder` |
| SETCAT-11 | 12-01, 12-02, 12-03 | Confirm dialog on disable with own-trail count + view-trails link | SATISFIED | `_onToggleOff` + `_viewOwnTrails` in both screens |

No orphaned requirements — all 8 phase requirement IDs (SETCAT-01, 02, 06, 07, 08, 09, 10, 11) declared across the 4 plans match exactly the phase's declared requirement set and REQUIREMENTS.md traceability table (all marked "Complete", Phase 12).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER markers found in any Phase 12 file | — | None |
| `settings_categories_screen.dart` | 398 | `use_build_context_synchronously` (info-level lint) | Info | `context.mounted` guard present at line 399 immediately before use; analyzer flags it as an info-level heuristic limitation, not a real bug — guard is correctly placed per the code review's own convention. |
| `settings_subcategories_screen.dart` | 408 | Same info-level lint | Info | Same as above. |

No blockers or warnings. `dart analyze` on all 8 modified/created Phase 12 files reports 0 errors, 0 warnings, 4 info-level lints (all `use_build_context_synchronously`, already guarded).

### Code Review Findings (12-REVIEW.md / 12-REVIEW-FIX.md)

2 critical + 4 warning findings from `12-REVIEW.md` were addressed in `12-REVIEW-FIX.md`:
- CR-01 (unguarded `extra as Category` cast) — FIXED, verified in `router_provider.dart:200-212`.
- CR-02 (unhandled `ownTrailCount` exception) — FIXED, verified via try/catch in both `_onToggleOff` methods.
- WR-01 (pre-filter silently dropped) — FIXED, verified via `await trailFilterProvider(...).future` before `updateFilter`.
- WR-02 (`_orderedIds` clobbered mid-drag) — FIXED, verified via `_dragging` guard in both screens.
- WR-03 (unsafe `as int` cast) — FIXED, verified via defensive `raw is int ? raw : (raw as num).toInt()` coercion.
- WR-04 (duplicated toggle/dialog logic across both screens) — SKIPPED, explicitly documented as an intentional deferral (non-correctness refactor suggestion, not required for phase goal achievement). This is an accepted, documented deviation — no override entry needed since it was never a must-have, only a review suggestion.

All fix commits (`cc43a35d`, `89f68a99`, `ac84d318`, `d61e904f`, `6006afc0`) are present in git history and their diffs match the claimed fixes.

### Behavioral Spot-Checks

Skipped — this is a Flutter mobile app requiring a running device/simulator and a live PocketBase backend with an authenticated user session; no runnable entry point exists for isolated command-line spot-checks. `dart analyze` (static) was used instead as the closest automatable proxy, and passed clean.

### Probe Execution

Skipped — no `scripts/*/tests/probe-*.sh` files exist for this phase, and none are referenced in the PLAN/SUMMARY files.

### Human Verification Required

The 8 items below were harvested from the `checkpoint:human-verify` tasks in `12-02-PLAN.md` and `12-03-PLAN.md`. Both were marked "auto-approved" in their SUMMARY.md under `human_verify_mode: end-of-phase` — meaning live-device confirmation was deliberately deferred to this end-of-phase gate, not actually performed by a human yet. All code-level wiring for these behaviors is verified above; only live/visual/gesture confirmation remains.

### 1. Category list sort, icon, and switch rendering

**Test:** Open Settings → Categories on a live device/simulator against a real backend.
**Expected:** Categories are listed sorted by priority ascending (alphabetical tie-break), each row shows a drag handle, category icon, locale-resolved name, and a visibility switch.
**Why human:** Visual layout and icon/locale rendering cannot be confirmed by static analysis.

### 2. Category drag-handle reorder persistence and failure revert

**Test:** Drag a category row by the handle to a new position, release, leave and re-enter the screen. Then simulate a network failure (e.g. airplane mode) during a drag.
**Expected:** New order persists after reload. On failure, list reverts to prior order with an error toast.
**Why human:** Requires live gesture interaction and real/simulated network failure.

### 3. Category own-trail confirm dialog

**Test:** Toggle a category OFF that has the signed-in user's own trails; tap "View trails"; then repeat and try Cancel, then Disable anyway; then toggle a category ON.
**Expected:** Dialog appears only on OFF with count + working "View trails" navigation to the pre-filtered own-trail list; Cancel reverts, Disable anyway saves; ON never shows the dialog.
**Why human:** Requires live backend, authenticated user with owned trails, visual/behavioral confirmation.

### 4. Category row body-tap navigation independence

**Test:** Tap a category row body (not switch, not drag handle).
**Expected:** Navigates to SettingsSubcategoriesScreen for that category (AppBar title = category's localized name); switch/handle taps do not navigate.
**Why human:** Independent hit-target behavior needs live-gesture confirmation.

### 5. Subcategory empty state

**Test:** Open a category with zero subcategories.
**Expected:** Empty-state copy ("No subcategories" / body) renders; screen remains reachable.
**Why human:** Requires live data with a genuinely empty subcategory set and visual confirmation.

### 6. Subcategory drag-handle reorder (parent-scoped) persistence and failure revert

**Test:** Drag a subcategory row to reorder within a category; release, leave and re-enter. Simulate failure.
**Expected:** New order persists, scoped only to that parent category; failure reverts with error toast.
**Why human:** Requires live gesture interaction, network failure simulation, and confirmation of parent-scoping.

### 7. Subcategory own-trail confirm dialog

**Test:** Toggle a subcategory OFF that has the signed-in user's own trails; tap "View trails"; Cancel; Disable anyway; toggle ON.
**Expected:** Dialog titled "Hide this subcategory?" with count, working "View trails" pre-filtered to the subcategory, Cancel reverts, Disable anyway saves, ON never triggers it.
**Why human:** Requires live backend, authenticated user, visual/behavioral confirmation.

### 8. End-to-end reachability

**Test:** Settings tab → Categories tile → SettingsCategoriesScreen → tap a category row → SettingsSubcategoriesScreen, on a real device/simulator.
**Expected:** Full flow navigable without crashes.
**Why human:** End-to-end live app execution required; also confirms the CR-01 fallback guard doesn't misfire under normal in-app navigation.

### Gaps Summary

No code-level gaps found. All must-haves (ROADMAP Success Criteria 1-5, all 4 plans' frontmatter must-haves, all 8 requirement IDs) are verified present, substantive, wired, and backed by real data flow — including confirmation that the 5 code-review-identified bugs (CR-01, CR-02, WR-01, WR-02, WR-03) were genuinely fixed in the codebase, not just claimed fixed. The only remaining item is live human/device verification of the interactive behaviors (drag persistence, dialog visuals, navigation gestures) that were deliberately deferred to this end-of-phase gate per the project's `human_verify_mode: end-of-phase` workflow setting, rather than skipped.

---

_Verified: 2026-07-02T10:00:00Z_
_Verifier: Claude (gsd-verifier)_
