---
phase: quick-260719-n8g
plan: 01
subsystem: mobile-navigation
tags: [flutter, riverpod, go_router, maplibre, objectbox, gps-recording]

requires:
  - phase: quick-260719-fjw
    provides: buildDraftTrail save-to-trail-create-screen handoff (_saveRecordedTrack)
  - phase: quick-260712-m9v
    provides: ActiveNavigationEntity resume pattern (_maybeResume, resume dialog structure)
provides:
  - Trail-less GPS recording mode on NavigationScreen via a single isRecording flag
  - Top-level /record route with optional ActiveNavigationEntity resume seed
  - Wired "Record trail" entry card (was a dead _comingSoon stub)
  - ActiveSessionType.rec resume-after-kill path in main.dart
affects: [mobile-navigation, trail-creation]

tech-stack:
  added: []
  patterns:
    - "isRecording constructor flag threaded through 4 branch points instead of a parallel RecordScreen"
    - "Empty NavigateResponse(maneuvers: [], shape: []) + sentinel id: '' as the minimal-diff way to reuse a trail-bound screen in a trail-less mode"

key-files:
  created: []
  modified:
    - app/lib/routes/navigation_screen.dart
    - app/lib/provider/router_provider.dart
    - app/lib/routes/trail_source_select_screen.dart
    - app/lib/main.dart
    - app/lib/i18n/app_en.arb (+ all app_*.arb locale files)
    - app/test/provider/navigation_provider_test.dart

key-decisions:
  - "Reused NavigationScreen directly via isRecording flag rather than a separate RecordScreen — every maneuver/route-dependent UI already null-guards to nothing on empty input"
  - "Recording's only finish trigger is the existing 3-option exit dialog (Cancel/Exit/Save), reused verbatim — isArrived is structurally always false with empty maneuvers so there is no auto-arrival banner"
  - "Sentinel id: '' for widget.id in recording mode (not a nullable-id refactor) — every trailProv(widget.id) read was already null-guarded"

patterns-established:
  - "Recording-mode button row: [pause, stop(red), elevation] vs nav's [exit, pause, elevation] — pause/elevation FABs extracted into _buildPauseFab/_buildElevationFab shared helpers"

requirements-completed: [RECORD-MODE, RECORD-ENTRY, RECORD-RESUME]

duration: ~20min
completed: 2026-07-19
---

# Quick Task 260719-n8g: Implement the missing route recorder Summary

**Trail-less GPS recording reusing NavigationScreen via an `isRecording` flag: red Stop button opens the existing 3-option save dialog, wired from the dead "Record trail" card, with `ActiveSessionType.rec` resume-after-kill support.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-19
- **Tasks:** 2
- **Files modified:** 32 (2 core route/screen files + 1 entry-card file + main.dart + 14 arb files + regenerated app_localizations*.dart + 1 test file + router_provider.g.dart)

## Accomplishments
- `NavigationScreen` now supports a trail-less "recording mode" via a single `isRecording` flag — no separate `RecordScreen`, no new provider, no schema migration
- The "Record trail" card on `trail_source_select_screen.dart` is now a real entry point (was `_comingSoon` dead stub): requests location permission, then pushes the new top-level `/record` route
- A killed recording session resumes via the pre-existing `ActiveSessionType.rec` enum value, mirroring the shipped `.nav` resume UX
- Extended `navigation_provider_test.dart` with the one genuinely unit-testable slice: empty-`NavigateResponse` state + breadcrumb-still-appends behavior

## Task Commits

Each task was committed atomically:

1. **Task 1: Add isRecording recording-mode branches to NavigationScreen + i18n keys** - `20316c47` (feat)
2. **Task 2: Wire the Record entry card, /record route, and rec-session resume** - `c41b757d` (feat)

_No TDD-cycle commits required beyond the extended provider test bundled into Task 1's commit (this quick task's `tdd="true"` behavior block only covered the one provider-level slice specified in the plan)._

## Files Created/Modified

- `app/lib/routes/navigation_screen.dart` - `isRecording` constructor flag; `_buildButtonRow` branches to `[pause, stop(red), elevation]` in recording mode (extracted `_buildPauseFab`/`_buildElevationFab` shared helpers); `_confirmExit` dialog content branches to `stop_recording_confirm`; `_persistNow` writes `ActiveSessionType.rec`/`trailId: null` when recording; `_buildElevationPage` short-circuits to `SizedBox.shrink()` in recording mode
- `app/lib/provider/router_provider.dart` (+ `.g.dart`) - New top-level `/record` GoRoute building `NavigationScreen(id: '', response: const NavigateResponse(maneuvers: [], shape: []), isRecording: true, resumeSession: <extra>)`
- `app/lib/routes/trail_source_select_screen.dart` - Replaced dead `_comingSoon` with `_openRecorder`: mirrors `launchNavigation`'s location-permission gate (service enabled → permission check/request → iOS WhenInUse re-prompt), then `context.push('/record')`
- `app/lib/main.dart` - `_maybeResume` gained a `.rec` branch (`_maybeResumeRecording`): skips `readCachedNav` (no trail), shows a `resume_recording_prompt` dialog with no trail name, re-pushes `/record` with the row on accept, clears on decline
- `app/lib/i18n/app_en.arb` + all 13 other locale `.arb` files - Added `stop_recording`, `stop_recording_confirm`, `resume_recording_prompt` keys (natural translations for de/fr/es/it/pt/nl/pl/ru/zh/cs/no/hu; English fallback for eu per plan's discretion clause — not confident in Basque)
- `app/lib/i18n/app_localizations*.dart` - Regenerated via `flutter gen-l10n`
- `app/test/provider/navigation_provider_test.dart` - New test group asserting `Navigation.build` with an empty `NavigateResponse` resolves without throwing and that `onPosition` still appends to the breadcrumb

## Decisions Made

- Reused `NavigationScreen` in-place via `isRecording` rather than a parallel `RecordScreen` — every maneuver/route-dependent UI branch was already null/empty-guarded to nothing, so the diff stayed minimal (per CONTEXT.md's locked decision).
- Used sentinel `id: ''` for `widget.id` in recording mode rather than refactoring `id` to nullable — every `trailProvider(widget.id)` read site was already null-guarded (research-verified before implementation).
- Extracted `_buildPauseFab`/`_buildElevationFab` as shared private methods (not inlined twice) so the nav-mode and recording-mode button rows stay byte-identical for those two buttons — reduces future maintenance surface.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' `<action>` steps were followed verbatim; the only in-flight correction was a self-inflicted syntax mistake while restructuring `_buildButtonRow` (introduced dead placeholder code mid-edit), caught and cleaned up via `flutter analyze` before committing — never reached a commit.

## Issues Encountered

None outside the self-corrected editing mistake noted above (resolved before any commit; not a deviation from the plan's intended behavior).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Feature is code-complete and passes `flutter gen-l10n`, `flutter test` (full suite, no new failures — the 3 pre-existing failures noted in Phase 18's deferred-items.md are unrelated and untouched), and `flutter analyze` (zero new issues; all reported issues are pre-existing/vendor-code, confirmed via before/after comparison).
- **Manual on-device verification still required** (widget tests for `NavigationScreen` are impractical due to native MapLibre/tracelet/sensor dependencies, per the plan's own verification section):
  1. Trail source → tap "Record trail" → grant permission → recording session opens (map centers on first fix, no maneuver banner, bottom row `[pause, stop, elevation]`).
  2. Walk a few meters → stats/timer advance; tap pause → timer freezes; tap again → resumes.
  3. Tap the red Stop button → dialog reads "Stop recording?" with Cancel / Exit without saving / Save → Save hands off to `trail_create_screen` with the recorded track prefilled.
  4. Start a recording, swipe-kill the app, relaunch → "Resume recording?" prompt → accept → breadcrumb + stats continue; decline → no prompt on next launch.

---
*Phase: quick-260719-n8g*
*Completed: 2026-07-19*

## Self-Check: PASSED

All created/modified files confirmed present on disk; both task commit hashes (`20316c47`, `c41b757d`) confirmed present in git log.
