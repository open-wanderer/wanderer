---
phase: 36-local-first-recording-automatic-upload
plan: 19
subsystem: ui
tags: [flutter, riverpod, l10n, objectbox, trail-panel, sync-state]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload
    provides: "TrailSyncState enum + isUnsyncedState predicate (36-01), trail_dropdown.dart's isUnsyncedState convention (36-16)"
provides:
  - "TrailPanel gates summit-log/comment tabs on isUnsyncedState(trail.syncState), not the cache-provenance isLocal flag"
  - "DefaultTabController length and _TabContent children derived from one shared showsServerTabs local, structurally preventing an out-of-range tab index"
  - "LocalTrailMetrics typedef doc corrected to state the three value lists are null-dropped and NOT row-aligned"
  - "retry_upload dead l10n key removed from app_en.arb and regenerated out of app_localizations*.dart / untranslated_messages.json"
  - "WR-06 destructive-action translation gap recorded as a tracked, prioritised todo instead of silently accepted"
affects: [trail-detail-ui, l10n]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One boolean local (showsServerTabs) governs a TabBar, its DefaultTabController.length, and a hand-rolled _TabContent's children list together, so the three can never disagree on tab count"

key-files:
  created:
    - .planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md
  modified:
    - app/lib/components/trail/trail_panel.dart
    - app/lib/util/trail/offline_filter_bounds.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_localizations.dart
    - app/lib/i18n/app_localizations_cs.dart
    - app/lib/i18n/app_localizations_de.dart
    - app/lib/i18n/app_localizations_en.dart
    - app/lib/i18n/app_localizations_es.dart
    - app/lib/i18n/app_localizations_eu.dart
    - app/lib/i18n/app_localizations_fr.dart
    - app/lib/i18n/app_localizations_hu.dart
    - app/lib/i18n/app_localizations_it.dart
    - app/lib/i18n/app_localizations_nl.dart
    - app/lib/i18n/app_localizations_no.dart
    - app/lib/i18n/app_localizations_pl.dart
    - app/lib/i18n/app_localizations_pt.dart
    - app/lib/i18n/app_localizations_ru.dart
    - app/lib/i18n/app_localizations_zh.dart
    - app/lib/i18n/untranslated_messages.json

key-decisions:
  - "WR-06 stays deferred by design: recorded as a tracked todo with the current (regenerated) key list and priority order, not fixed with machine translation"
  - "The WR-06 todo's key list was expanded beyond the plan's stated 15 keys to the full 22-key union currently in untranslated_messages.json, since trail_source_*/library_empty_* keys had also accumulated since 36-REVIEW.md was written and the plan required the todo to reflect current reality, not stale text"

patterns-established:
  - "When a hand-rolled tab-content widget (_TabContent) exists alongside a TabBar/DefaultTabController, derive the controller length and the content list from the same boolean/enum local so they cannot desync"

requirements-completed: [REC-03, REC-06]

# Metrics
duration: 12min
completed: 2026-08-03
---

# Phase 36 Plan 19: Trail-panel tab gate fix, dead l10n key removal, WR-06 handoff Summary

**Downloaded trails show summit logs and comments again via a single `showsServerTabs` predicate shared by `TabBar`, `DefaultTabController`, and `_TabContent`; the dead `retry_upload` l10n key is gone; the untranslated destructive-action copy is now a named, prioritised todo instead of silent debt.**

## Performance

- **Duration:** ~12 min
- **Started:** 2026-08-03T17:29:25Z
- **Completed:** 2026-08-03T17:36:49Z
- **Tasks:** 2/2 completed
- **Files modified:** 20 (2 source + 17 l10n + 1 new todo)

## Accomplishments

- Fixed WR-11: `TrailPanel`'s tab gate no longer reads `trail.isLocal` (a cache-provenance flag hardcoded `true` for every ObjectBox row by `TrailEntity.toModel()`). It now reads `isUnsyncedState(trail.syncState)`, matching the convention `trail_dropdown.dart` already established. A downloaded trail — including one `TrailNotifier.build()` served from cache after a single fetch timeout — shows its Summit book and Comments tabs again.
- Closed the structural risk WR-11's fix could have introduced: `DefaultTabController.length` and `_TabContent`'s `children` list are both derived from the same `showsServerTabs` local, so the tab controller can never produce an index outside the rendered children.
- Fixed WR-12: `LocalTrailMetrics`'s typedef doc no longer claims "three parallel lists" — it now states plainly that the three lists are independent, null-dropped, and NOT row-aligned, and names the failure mode (pairing one trail's distance with another's elevation) that zipping them would cause.
- Fixed WR-07: deleted the dead `retry_upload` l10n key (present only in `app_en.arb`, referenced nowhere in `app/lib` or `app/test`) and regenerated `app_localizations*.dart` / `untranslated_messages.json` via `flutter gen-l10n`, shrinking the translation backlog by one string across all 13 non-English locales.
- Recorded WR-06 (destructive-action strings still English-only in 13 locales) as tracked work rather than leaving it silent: `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md`, with `delete_unsynced_trail_confirm` and `signout_unsynced_warning` prioritised first, regenerated against the current `untranslated_messages.json` (22 keys, up from the plan's originally-cited 15, because `trail_source_*`/`library_empty_*` keys had also accumulated since the review was written).

## Task Commits

1. **Task 1: One predicate governs the trail tabs, and the metrics typedef stops lying** - `9261ce33` (fix)
2. **Task 2: Delete the dead l10n key, and hand off the destructive-action translations** - `12affbac` (chore)

**Plan metadata:** (this commit)

## Files Created/Modified

- `app/lib/components/trail/trail_panel.dart` - `showsServerTabs` local governs `TabBar`, `DefaultTabController.length`, and `_TabContent.children` together; About column hoisted to a named `aboutTab` local so it can be reused whether or not the server tabs render
- `app/lib/util/trail/offline_filter_bounds.dart` - `LocalTrailMetrics` typedef doc rewritten to state the three lists are null-dropped and not row-aligned (doc-only, no behaviour change)
- `app/lib/i18n/app_en.arb` - `retry_upload` entry removed
- `app/lib/i18n/app_localizations.dart` + 13 per-locale variants - regenerated by `flutter gen-l10n`, `retry_upload` getter removed
- `app/lib/i18n/untranslated_messages.json` - regenerated; `retry_upload` no longer listed in any locale
- `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md` - new WR-06 handoff: exact keys (22, priority-ordered), exact locales (13), and why machine translation is refused

## Decisions Made

- WR-06's key list in the todo covers the full current `untranslated_messages.json` union (22 keys) rather than only the 15 keys the plan text originally enumerated, because the plan's own instruction was to regenerate the list against current reality (post `retry_upload` deletion, post 36-15/36-16 additions) — and by the time this plan ran, the backlog had also picked up 7 unrelated `trail_source_*`/`library_empty_*` keys that were real English-only strings and would otherwise have been silently dropped from tracking. These are called out as a separate, lower-priority group in the todo (Priority 5) so the destructive-action prioritisation the review asked for isn't diluted.

## Deviations from Plan

None — plan executed exactly as written. The only judgment call (expanding the todo's key list to the full 22-key union instead of the plan's literal 15) was explicitly directed by the plan's own instruction to "regenerate the key list from `untranslated_messages.json` **after** the `retry_upload` deletion and the gen-l10n run, so the todo does not list a key that no longer exists" — treated as guidance to reflect current reality, not a deviation from it.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `trail_panel.dart`'s tab gate is now correct and structurally safe (controller length and content list cannot desync). Per the plan's own `<verification>` note, this fix is asserted here only at the source level (the governing predicate, analyzer, full suite) because `TrailPanel` cannot be mounted in `flutter test` — it embeds a native `MapLibreMap` platform view. Behavioural proof belongs in device UAT: open a downloaded trail (both self-authored and not) and confirm Summit book/Comments are present and switchable; then open an unsynced trail and confirm only About is shown. This should be added as a new case alongside `36-VERIFICATION.md` `human_verification` item 1.
- WR-06 is tracked, not blocking: the app's English-fallback behaviour for missing keys means nothing regresses functionally; the todo is the durable record so it isn't rediscovered from scratch next review pass.
- No blockers for subsequent phase-36 plans.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED
