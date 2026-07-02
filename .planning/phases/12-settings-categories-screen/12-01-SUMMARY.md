---
phase: 12-settings-categories-screen
plan: 01
subsystem: ui
tags: [flutter, riverpod, activitypub, l10n, meilisearch, category-preferences]

# Dependency graph
requires:
  - phase: 10-category-subcategory-data-layer
    provides: Category/Subcategory models, CategoryPreference/SubcategoryPreference models, category + preference providers, displayName(locale) extensions
  - phase: 11-trail-filter-subcategory-support
    provides: TrailFilter.toFilterText category_id/subcategory_id filter field names
provides:
  - CategoryPreferenceNotifier.reorder(List<String>) posting to /user-category-preference/reorder
  - SubcategoryPreferenceNotifier.reorder(String, List<String>) posting to /user-subcategory-preference/reorder
  - category_preference_sort.dart (sortedCategoriesByPreference, sortedSubcategoriesByPreference, categoryVisible, subcategoryVisible)
  - own_trail_count.dart (lazy author-scoped ownTrailCount)
  - Seven settings-categories l10n keys + regenerated AppLocalizations
affects: [12-02 SettingsCategoriesScreen, 12-03 SettingsSubcategoriesScreen, 12-04 router-wiring]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Provider reorder methods mirror existing upsert (apiProvider.post + invalidateSelf, no client user field)"
    - "Pure-function sort/visibility util with no Riverpod dependency preserves the Phase-10 read-only-provider boundary"
    - "Lazy own-trail count via author-scoped POST /profile/{handle}/trails reading Meilisearch totalHits with fallback chain"

key-files:
  created:
    - app/lib/util/category_preference_sort.dart
    - app/lib/util/own_trail_count.dart
  modified:
    - app/lib/provider/category_preference_provider.dart
    - app/lib/provider/subcategory_preference_provider.dart
    - app/lib/i18n/app_en.arb

key-decisions:
  - "reorder payloads never send a client user field — server injects owner from session (T-12-01, mirrors upsert)"
  - "String.compareTo on lowercased displayName stands in for localeCompare; no intl-collation package added (A4)"
  - "own-trail count reads totalHits ?? estimatedTotalHits ?? hits.length ?? 0 (A1 fallback chain)"
  - "Sort/visibility helpers are pure functions (no Riverpod) so callers pass already-fetched data"

patterns-established:
  - "Pattern: provider reorder method = apiProvider.post to /reorder endpoint + invalidateSelf, no error handling (callers toast)"
  - "Pattern: visibility = prefs.where(parentId).map(visible).firstOrNull != false (no-preference and null both mean visible)"

requirements-completed: [SETCAT-09, SETCAT-10, SETCAT-11]

# Metrics
duration: 8min
completed: 2026-07-01
---

# Phase 12 Plan 01: Category Preference Contract Layer Summary

**Two provider `reorder` methods, a pure-function priority/visibility sort helper, a lazy author-scoped own-trail count helper, and seven confirm-dialog/empty-state l10n keys — the shared contract Plans 02/03 assemble against.**

## Performance

- **Duration:** ~8 min
- **Completed:** 2026-07-01
- **Tasks:** 3
- **Files created:** 2
- **Files modified:** 3 source (+ regenerated provider .g.dart and 15 AppLocalizations files)

## Accomplishments
- `CategoryPreferenceNotifier.reorder(List<String>)` and `SubcategoryPreferenceNotifier.reorder(String, List<String>)` POST the verified payload shapes (`categories`; `category` + `subcategories`) and `invalidateSelf()`, with no client `user` field (T-12-01).
- `category_preference_sort.dart` ports `sortedCategoriesByPreference` / `sortedSubcategoriesByPreference` (priority asc, prioritized-first, locale-name tie-break) plus `categoryVisible` / `subcategoryVisible` as dependency-free pure functions.
- `own_trail_count.dart` lazily counts the current user's trails in a category/subcategory via author-scoped `POST /profile/{handle}/trails` with a `category_id`/`subcategory_id IN [...]` Meilisearch filter, reading `totalHits` with a fallback chain (SETCAT-11).
- Seven new l10n keys added to canonical `app_en.arb` (with a `{count}` int placeholder on the disable body) and `AppLocalizations` regenerated across all 14 locales.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add reorder methods to both preference notifiers** - `11fee466` (feat)
2. **Task 2: Create sort helper + own-trail count helper** - `537a0190` (feat)
3. **Task 3: Add l10n keys and regenerate AppLocalizations** - `3c277c32` (feat)

## Files Created/Modified
- `app/lib/provider/category_preference_provider.dart` - Added `reorder(List<String>)` posting to `/user-category-preference/reorder`.
- `app/lib/provider/subcategory_preference_provider.dart` - Added `reorder(String, List<String>)` posting to `/user-subcategory-preference/reorder`.
- `app/lib/util/category_preference_sort.dart` - New: priority/locale sort + visibility helpers (pure functions).
- `app/lib/util/own_trail_count.dart` - New: lazy author-scoped own-trail count.
- `app/lib/i18n/app_en.arb` - Seven new keys with `@`-metadata; regenerated `app_localizations*.dart` (15 files).
- `app/lib/provider/*.g.dart` - Regenerated via build_runner.

## Decisions Made
- reorder payloads omit `user` (server injects from session, T-12-01) — mirrors existing `upsert`.
- Handle resolved as `'@${user.preferredUsername}'` per `profile_screen.dart`, not the un-prefixed form shown in RESEARCH Pattern 5 (plan/acceptance-criteria authoritative).
- `String.compareTo` on lowercased names as locale-collation stand-in; no intl package added (A4).
- Own-trail count uses `totalHits ?? estimatedTotalHits ?? hits.length ?? 0` for Meilisearch config resilience (A1).

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- The plan's Task-2 grep gate `grep -q "profile/\$handle/trails"` relies on `$` acting as a regex anchor (the escaped `$` matches literally); an unescaped fixed-string grep initially reported 0 matches. Confirmed the path `'/profile/$handle/trails'` is correct in the file via `grep -F`. No code change needed.

## Scope Boundary Note
`dart analyze lib/provider lib/util` surfaces 32 pre-existing `info`-level `deprecated_member_use` warnings in `lib/util/icon_util.dart` (Font Awesome renamed icons). These are unrelated to this plan's files and out of scope — the four plan files report "No issues found!".

## User Setup Required
None - no external service configuration required. The `/user-category-preference/reorder` and `/user-subcategory-preference/reorder` endpoints exist server-side (v1.3 mirrors shipped web PR #1059).

## Next Phase Readiness
- SETCAT-09/10 provider `reorder` halves and SETCAT-11 count half are in place for Plans 02/03 to consume directly.
- `sortedCategoriesByPreference` / visibility helpers and `AppLocalizations` getters are ready for the two settings screens.
- No blockers.

---
*Phase: 12-settings-categories-screen*
*Completed: 2026-07-01*

## Self-Check: PASSED

All created/modified files exist on disk; all three task commits (`11fee466`, `537a0190`, `3c277c32`) present in git history.
