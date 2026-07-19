---
phase: quick-260719-fjw
plan: 01
subsystem: navigation
tags: [flutter, riverpod, gpx, go_router, i18n]

# Dependency graph
requires:
  - phase: 21-route-planner-handoff-entry-point
    provides: buildDraftTrail / pendingImportedTrail handoff mechanism (route_planner_handoff_util.dart, trail_import_util.dart)
provides:
  - "buildRecordedTrackTrail — pure breadcrumb -> stub Trail converter"
  - "Save track action on the navigation completion banner"
  - "Save track option in the premature-exit dialog (Cancel / Exit / Save track)"
affects: [navigation_screen, trail_create_screen]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Reuse buildDraftTrail for any future breadcrumb/track -> stub Trail conversion instead of re-deriving bounds/gpxData/gpx"
    - "pushReplacement (not push) when handing off a live navigation session so the disposed screen cannot be backed-into"

key-files:
  created:
    - app/lib/util/recorded_track_util.dart
    - app/test/util/recorded_track_util_test.dart
  modified:
    - app/lib/routes/navigation_screen.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_localizations.dart
    - app/lib/i18n/app_localizations_en.dart
    - app/lib/i18n/app_localizations_cs.dart
    - app/lib/i18n/app_localizations_de.dart
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

key-decisions:
  - "buildRecordedTrackTrail is a thin wrapper around buildGpxFromPoints + buildDraftTrail — no new bounds/XML logic, keeping the derivation single-sourced with the GPX-import and route-planner handoffs"
  - "No category passed to buildDraftTrail for a recorded track (a recorded track has no travel profile)"
  - "Saving always clears the active-session row (active_nav.clear) before either popping (null track) or pushReplacement-ing to the create screen, so no stale resume prompt appears on next launch either way"

patterns-established:
  - "_NavExitChoice enum for a 3-option dialog (Cancel/Exit/Save track) replacing showDialog<bool>, switched on in the .then callback"

requirements-completed: [QUICK-260719-fjw]

# Metrics
duration: ~15min
completed: 2026-07-19
---

# Quick Task 260719-fjw: Save Track Recorded During Navigation Summary

**Adds a `buildRecordedTrackTrail` pure helper (breadcrumb -> stub `Trail` via the existing `buildGpxFromPoints`/`buildDraftTrail` machinery) and wires a "Save track" action into both the navigation completion banner and the premature-exit dialog, handing off to `trail_create_screen` via the existing GPX-import `pendingImportedTrail` mechanism.**

## Performance

- **Duration:** ~15 min
- **Tasks:** 2 completed
- **Files modified:** 18 (2 created, 16 modified — 15 of the 16 are generated per-locale localization files from `flutter gen-l10n`)

## Accomplishments
- `buildRecordedTrackTrail(breadcrumb, {durationSeconds})` — null-guards under 2 points, otherwise reuses `buildGpxFromPoints` + `buildDraftTrail` to produce a stub `Trail` with `expand.gpx`/`expand.gpxData`, bounds, and a pre-filled `duration`. Fully unit tested.
- Completion banner (`_buildCompletionBannerContent`) now shows a `FilledButton.icon` "Save track" action beneath the arrival message.
- Premature-exit dialog (`_confirmExit`) now offers three choices (`Cancel` / `Exit` / `Save track`) via a new `_NavExitChoice` enum, replacing the old boolean dialog result; existing Exit/Cancel behavior is unchanged.
- `_saveRecordedTrack()` reads the breadcrumb/stats with the exact provider seed args used everywhere else in the file (per the file's own split-brain-provider warning), clears the active-session row either way, and either pops (nothing recorded) or `pushReplacement`s to `/trail/create/edit` with the stub trail.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): failing test for buildRecordedTrackTrail** - `2bd575f0` (test)
2. **Task 1 (GREEN): implement buildRecordedTrackTrail** - `61497acd` (feat)
3. **Task 2: wire Save track into completion banner and exit dialog** - `2b2aa687` (feat)

_TDD gate sequence confirmed: `test(...)` commit (2bd575f0) precedes the `feat(...)` commit (61497acd) implementing the same behavior._

## Files Created/Modified
- `app/lib/util/recorded_track_util.dart` - `buildRecordedTrackTrail` pure helper
- `app/test/util/recorded_track_util_test.dart` - unit tests covering the null-guard, track round-trip, gpxData/waypoints, bounds, and duration pre-fill
- `app/lib/routes/navigation_screen.dart` - `_saveRecordedTrack`, `_NavExitChoice` enum, completion-banner button, 3-option exit dialog
- `app/lib/i18n/app_en.arb` - new `save_track` key
- `app/lib/i18n/app_localizations*.dart` (all locales) - regenerated via `flutter gen-l10n`; non-English locales fall back to the English string (expected, matches how `resume_navigation_prompt` was added previously)

## Decisions Made
- `buildRecordedTrackTrail` deliberately does not pass a `category` to `buildDraftTrail` — a recorded track has no travel profile, and `buildDraftTrail` already accepts `category: null`.
- `_saveRecordedTrack` calls `active_nav.clear(_store)` unconditionally before branching on whether a trail was built, so "nothing to save" and "saved" both leave no stale resume prompt.
- Used `context.pushReplacement` (not `push`) for the create-screen handoff so the finished/abandoned navigation screen is removed from the stack (and disposed, stopping GPS/tracelet) rather than left behind for a back-navigation to return to.

## Deviations from Plan

None — plan executed exactly as written. One incidental fix found and corrected during the plan's own RED→GREEN TDD cycle: a test-fixture bug (a `minLon` assertion checked the wrong constant against the 3-point breadcrumb fixture) was caught and fixed in the same GREEN commit; the implementation itself was correct on the first pass.

## Issues Encountered
- The repo had pre-existing uncommitted work-in-progress on `navigation_provider.dart`, `navigation_screen.dart`, and `navigation_provider_test.dart` (unrelated to this quick task — a heading/speed/accuracy map-matching change plus a new untracked `route_map_matcher.dart`/`route_map_matcher_test.dart`). This was set aside via a targeted, pathspec-scoped `git stash` for the duration of this task's edits (not a worktree, so `git stash` was safe to use) and restored via `git stash pop` immediately after this task's two commits landed, verified conflict-free and byte-identical to the original diff.

## Next Phase Readiness
- Automated verification (unit tests, `flutter analyze`, localization/grep checks) all pass.
- Remaining verification is the plan's `<human-check>`: on-device confirmation that (1) the completion banner's Save track button opens `trail_create_screen` with the recorded route drawn, and (2) the exit dialog's three choices (Cancel/Exit/Save track) all behave as expected. This is a manual on-device step, consistent with prior quick tasks in this project (e.g. 260712-m9v, 260719-d6a) whose on-device checks are tracked as STATE.md pending todos.

---
*Phase: quick-260719-fjw*
*Completed: 2026-07-19*

## Self-Check: PASSED

All created files verified present on disk (`app/lib/util/recorded_track_util.dart`, `app/test/util/recorded_track_util_test.dart`, this SUMMARY.md). All three task commit hashes (`2bd575f0`, `61497acd`, `2b2aa687`) verified present in `git log`.
