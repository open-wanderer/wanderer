---
phase: 11-trail-filter-subcategory-support
plan: 02
subsystem: ui
tags: [flutter, l10n, arb, filter-chip, riverpod, material]

# Dependency graph
requires:
  - phase: 10-category-subcategory-data-layer
    provides: Subcategory model/provider and locale-aware category names consumed by filter surfaces
provides:
  - WandererFilterChip optional avatarBuilder for rendering a leading icon avatar
  - subcategories l10n key in all 14 locale ARBs plus regenerated AppLocalizations.subcategories getter
affects: [11-03 trail-filter-screen, 11-04 trail-quick-filter-bar]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Backward-compatible widget extension via optional nullable builder callback (avatarBuilder)"
    - "ARB key added to template (en) with @metadata, value-only in the other 13 locales"

key-files:
  created: []
  modified:
    - app/lib/components/base/wanderer_filter_chip.dart
    - app/lib/i18n/app_en.arb (+13 other locale ARBs)
    - app/lib/i18n/app_localizations*.dart (regenerated)

key-decisions:
  - "Inserted subcategories key in alphabetical position (before subway_stop in en, after categories in others) to follow each file's existing sort convention rather than forcing a fixed offset"
  - "No icon color/size/IconTheme set at chip level; avatar inherits chip foreground per UI-SPEC, builder controls its own size"

patterns-established:
  - "avatarBuilder: optional Widget? Function(T item)? wired to FilterChip.avatar, null by default so existing call sites are unaffected"

requirements-completed: [FILTER-02]

# Metrics
duration: 6min
completed: 2026-06-30
---

# Phase 11 Plan 02: Shared filter chip + subcategories l10n Summary

**WandererFilterChip gains an optional avatarBuilder for leading icons, and a `subcategories` label key is added to all 14 locale ARBs with regenerated AppLocalizations bindings.**

## Performance

- **Duration:** ~6 min
- **Started:** 2026-06-30
- **Completed:** 2026-06-30
- **Tasks:** 2
- **Files modified:** 30 (chip + 14 ARBs + 15 generated dart)

## Accomplishments
- Added optional `Widget? Function(T item)? avatarBuilder` field to `WandererFilterChip<T>`, wired to `FilterChip.avatar` — backward compatible (existing call sites pass nothing, render no avatar)
- Added `subcategories` key to all 14 `app_*.arb` files with locale-appropriate translations
- Added `@subcategories` metadata to the English template ARB
- Regenerated `app_localizations*.dart`, exposing `l10n.subcategories` for Plans 03 and 04

## Task Commits

Each task was committed atomically:

1. **Task 1: Add avatarBuilder parameter to WandererFilterChip** - `56fc402f` (feat)
2. **Task 2: Add 'subcategories' l10n key to all locale ARBs and regenerate** - `87b47d87` (feat)

**Plan metadata:** see final docs commit (commit_docs disabled — see below)

## Files Created/Modified
- `app/lib/components/base/wanderer_filter_chip.dart` - Added optional avatarBuilder field, constructor param, and `avatar: avatarBuilder?.call(option)` wiring
- `app/lib/i18n/app_en.arb` - Added `subcategories` value + `@subcategories` metadata
- `app/lib/i18n/app_{cs,de,es,eu,fr,hu,it,nl,no,pl,pt,ru,zh}.arb` - Added translated `subcategories` value
- `app/lib/i18n/app_localizations*.dart` - Regenerated bindings exposing `subcategories` getter

## Decisions Made
- Placed `subcategories` in each ARB's existing alphabetical sort position (English: before `subway_stop`; others: after `categories`). Functionally equivalent to the plan's "adjacent to categories" guidance and keeps the files' sort convention intact.
- Did not set any icon color/size or IconTheme at the chip level per UI-SPEC; the avatar inherits the chip foreground and the call-site builder controls its own size.

## Deviations from Plan

None - plan executed exactly as written. (The alphabetical key placement is the plan's stated intent — adjacency to `categories` — applied per each file's sort order; not a behavioral deviation.)

## Issues Encountered
None. `flutter gen-l10n` emitted pre-existing untranslated-message warnings for ru/zh (unrelated to this change) but exited successfully; all 14 ARBs validated as JSON.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `WandererFilterChip.avatarBuilder` and `l10n.subcategories` are ready for consumption by Plan 03 (TrailFilterScreen) and Plan 04 (trail_quick_filter_bar).
- No blockers.

## Self-Check: PASSED

- FOUND: app/lib/components/base/wanderer_filter_chip.dart
- FOUND: app/lib/i18n/app_en.arb
- FOUND: .planning/phases/11-trail-filter-subcategory-support/11-02-SUMMARY.md
- FOUND: commit 56fc402f (Task 1)
- FOUND: commit 87b47d87 (Task 2)

---
*Phase: 11-trail-filter-subcategory-support*
*Completed: 2026-06-30*
