---
phase: 38-downloaded-trails-as-state-not-objects
plan: 06
subsystem: ui
tags: [flutter, riverpod, dio, widget-testing, trail-menu]

# Dependency graph
requires:
  - phase: 38-downloaded-trails-as-state-not-objects
    provides: "fetchServerTrail (38-03), the Update/Remove menu split and library-membership-derived availability (38-05)"
provides:
  - "Edit on a server-backed trail fetches the server copy before opening the editor, never handing the editor a cached model"
  - "A failed edit-fetch refuses with a ToastType.warning + edit_needs_connection toast instead of silently opening the cached copy"
  - "Widget-test coverage for both menu branches (Update/Remove vs Download), both destructive axes (D-01/D-02), the un-download confirm copy (D-04), and the edit refusal (D-15/D-17)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "availableOffline as a plain _harness constructor-bool parameter reaches both TrailDropdown menu branches with zero ObjectBox Store involvement"
    - "apiProvider Dio override (_StubApi + a throwing HttpClientAdapter) for deterministic network-failure widget tests, mirrored from trail_filter_fallback_test.dart"
    - "Reading a keepAlive notifier's state off ProviderScope.containerOf(tester.element(...)) to assert a toast fired without restructuring the harness"

key-files:
  created: []
  modified:
    - app/lib/components/trail/trail_dropdown.dart
    - app/test/components/trail/trail_dropdown_menu_test.dart

key-decisions:
  - "Edit's onTap now branches on isUnsynced first (unchanged local-model path), then for every other trail awaits fetchServerTrail before pushing the fetched model -- never widget.trail -- so the editor is structurally unable to receive a cached model (D-15)"
  - "Used context.mounted (not the State's mounted) for the guard immediately before the post-fetch context.push, since the analyzer's use_build_context_synchronously check does not treat a build()-closure-captured context as covered by a bare mounted check -- resolved a new info-level lint with zero code-shape cost"
  - "Toast.add's 4s self-removal timer required an explicit tester.pump(Duration(seconds: 5)) after the refusal assertions to satisfy the test binding's pending-timer invariant before widget disposal"

patterns-established:
  - "Dio-override refusal tests use apiProvider.overrideWith(() => _StubApi(dio)) with a throwing HttpClientAdapter, matching trail_filter_fallback_test.dart's shape exactly"

requirements-completed: [DL-02, DL-03, DL-07]

duration: 9min
completed: 2026-08-04
---

# Phase 38 Plan 06: Edit Fetches the Server Copy Summary

**Edit on a downloaded trail now awaits `fetchServerTrail` before opening the editor and refuses with a stated-reason toast on failure, structurally closing the photo-duplication bug at its root; seven new widget tests pin both menu branches, both destructive axes, and the refusal.**

## Performance

- **Duration:** 9 min
- **Started:** 2026-08-04T16:45:56Z (prior plan's completion commit)
- **Completed:** 2026-08-04T16:53:51Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Closed the live photo-duplication bug (D-15/D-18) structurally: the Edit `onTap` for a server-backed trail now awaits `fetchServerTrail(api, trail.id)` and pushes the fetched model, never `widget.trail`, so the editor can never seed its photo picker from a downloaded row's local-file-path `photos` column
- A failed fetch refuses with a `ToastType.warning` + `edit_needs_connection` toast instead of silently opening the cached copy, matching the file's existing `delete_needs_connection` refusal treatment (D-16/D-17); Edit stays enabled and visible in both cases
- Extended the existing widget-test harness with an `availableOffline` parameter and a failing-Dio `apiProvider` override, then added seven new cases covering D-08's item split, D-07's translated status line, D-02's both-axes case, D-01's authorship-not-provenance case, D-04's confirm-copy divergence, and the D-15/D-17 refusal -- all eight pre-existing cases still pass unchanged

## Task Commits

Each task was committed atomically:

1. **Task 1: Edit fetches the server copy on tap, or refuses with a stated reason** - `c189a139` (fix)
2. **Task 2: Widget tests for both menu branches, both destructive axes, and the refusal** - `b7ce9669` (test)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/components/trail/trail_dropdown.dart` - Edit `onTap` rewritten: unsynced path unchanged; server-backed path awaits `fetchServerTrail`, pushes the fetched model on success, refuses with a toast on failure
- `app/test/components/trail/trail_dropdown_menu_test.dart` - `_harness` gained `availableOffline`/`api` parameters; `_StubApi`/`_FailingAdapter` added; seven new `testWidgets` cases

## Decisions Made
- The Edit closure checks `isUnsynced` first and returns early on that branch (identical to the pre-existing code), keeping the server-fetch logic isolated to the branch where a cached model would actually be dangerous.
- `context.mounted` used instead of the `State`'s bare `mounted` for the guard immediately before the post-await `context.push`, per `flutter analyze`'s `use_build_context_synchronously` diagnostic -- this cleared an info-level lint the plan's own zero-error/zero-warning gate would have tolerated, but keeping the file lint-clean matched its existing discipline.
- `Toast.add`'s 4-second self-removal `Future.delayed` needed an explicit `tester.pump(Duration(seconds: 5))` in the new refusal test to avoid `flutter_test`'s "Timer is still pending" teardown assertion; this is a test-only addition with no production-code effect.

## Deviations from Plan

None - plan executed exactly as written. The `edit_needs_connection` l10n key (minted in plan 38-02) already existed with the exact English text the plan's acceptance criteria and this plan's Task 2 test assert against, so no new l10n work was needed here.

## Issues Encountered
- The new post-fetch `context.push` triggered an info-level `use_build_context_synchronously` lint under a bare `mounted` guard; switched that one guard to `context.mounted`, which resolved it with `flutter analyze --no-pub lib` returning to zero new issues.
- The new Dio-override refusal test left a pending `Toast.add` removal timer at test teardown, causing a `flutter_test` binding assertion failure; fixed by advancing the fake clock past the 4-second delay before the test ends.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness

This is the last plan of Phase 38. Verification per this plan's own `<verification>` block:
- `flutter analyze --no-pub lib test` — 36 issues, all pre-existing info-level (0 new errors/warnings)
- `flutter test` — 968 passing (961 pre-existing + 7 new), 1 pre-existing skip, 0 failures
- `grep -rn "isLocal" app/lib/components/trail/trail_dropdown.dart app/lib/components/trail/trail_panel.dart` — no matches
- `grep -rn "forceOffline\|offline=1" app/lib app/test` — no matches

The Task 2 `<human-check>` device pass (8 steps covering the fetched-copy edit, photo-count-unchanged verification, the offline edit refusal, and the timed-out-fetch original-report case) is deferred to the phase's end-of-phase human-verification gate (`human_verify_mode: end-of-phase` in `.planning/config.json`) rather than blocking this plan's completion.

---
*Phase: 38-downloaded-trails-as-state-not-objects*
*Completed: 2026-08-04*

## Self-Check: PASSED
