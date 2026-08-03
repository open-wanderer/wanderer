---
phase: 36-local-first-recording-automatic-upload
plan: 13
subsystem: ui
tags: [flutter, riverpod, go_router, widget-testing, build_runner]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload (36-11)
    provides: "trailMapLocation(TrailSummary) route helper and localTrailProvider(localId)"
  - phase: 36-local-first-recording-automatic-upload (36-12)
    provides: "the /trail/local/:localId/map sub-route and the own-trails-list divert to trail_detail_screen"
provides:
  - "TrailDropdown's Show-on-map entry resolved through trailMapLocation, disabled when a trail is unaddressable"
  - "Post-edit invalidation that targets localTrailProvider for an unsynced trail, trailProvider otherwise"
  - "Behavioural widget-test coverage of D-14 (delete gating) / D-17 (download hiding) that opens the real PopupMenuButton"
  - "A codegen fixpoint across app/lib -- two consecutive build_runner runs produce byte-identical .g.dart output"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Priming a lazily-read @riverpod provider (one only touched inside a PopupMenuButton's itemBuilder, not the widget's own build()) via an inert Consumer in the test harness, mirroring how a sibling widget already primes it in production"

key-files:
  created:
    - app/test/components/trail/trail_dropdown_menu_test.dart
  modified:
    - app/lib/components/trail/trail_dropdown.dart
    - app/test/components/trail/trail_dropdown_delete_gate_test.dart
    - app/lib/provider/router_provider.g.dart

key-decisions:
  - "Test harness primes authProvider with a throwaway Consumer in the Scaffold body, mirroring trail_panel.dart's real `.requireValue!` watch on the same screen -- otherwise the PopupMenuButton's itemBuilder (which reads authProvider lazily, only when the menu opens) sees a still-AsyncLoading provider on its very first synchronous build and renders Edit/Delete as absent regardless of the actual auth state."
  - "Only router_provider.g.dart needed regeneration; profile_trails_provider.g.dart and trail_sync_provider.g.dart were already at their fixpoint despite 36-10's deliberate no-codegen edits, so the plan's speculative files_modified list was wider than what actually changed."

patterns-established: []

requirements-completed: [REC-03, SYNC-02]

# Metrics
duration: 25min
completed: 2026-08-03
---

# Phase 36 Plan 13: Dropdown reachability fix + test-quality gap closure Summary

**Retargeted the trail dropdown's map entry and post-edit refresh through the local-trail-aware route/provider helpers, added the missing behavioural widget-test signal for D-14/D-17 menu gating, and reconciled the whole package's generated Riverpod code to a fixpoint.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-08-03T11:09:00Z (approx, from init)
- **Completed:** 2026-08-03T11:34:19Z
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- `TrailDropdown`'s Show-on-map entry now resolves through `trailMapLocation`, disabling itself (greyed icon + label) when a trail has no addressable route, instead of interpolating a raw `'/trail/${trail.id}/map'` that resolved to a no-route error page for an unsynced trail.
- The Edit item's post-edit refresh now invalidates `localTrailProvider(localId)` for an unsynced trail and `trailProvider(trail.id)` otherwise, instead of always invalidating a meaningless `trailProvider('')` family instance.
- New `trail_dropdown_menu_test.dart` opens the real `PopupMenuButton` inside a real `TrailDropdown` mounted in an `AppBar` (matching its only production call site) and proves, behaviourally: Download/"Available offline" are absent for an unsynced trail (with a synced-trail control proving the harness does render Download when it should); the unrecoverable vs. reversible delete confirmation copy; Delete disabled while the trail is mid-drain; and Show on map enabled for an unsynced trail now that 36-12's route exists.
- `trail_dropdown_delete_gate_test.dart` keeps every pre-existing source-text assertion (branch order in `_deleteTrail`), now explicitly scoped via a header note to that invariant only, since reachability is covered behaviourally by the new file.
- Ran a whole-package `build_runner build --delete-conflicting-outputs` (safe here because 36-13 is the only plan in wave 4) and proved a codegen fixpoint: two consecutive runs produce byte-identical `.g.dart` output across `app/lib`.

## Task Commits

Each task was committed atomically:

1. **Task 1: Retarget the dropdown's map entry and its post-edit invalidation** - `612f41a9` (feat)
2. **Task 2: Behavioural coverage for the D-14/D-17 menu gating** - `8bf8433d` (test)
3. **Task 3: Reconcile generated code with its sources, once, at the end of the phase** - `cc21a105` (chore)

_Task 2 is a single test-file commit, not a classic TDD RED/GREEN/REFACTOR sequence: the behaviour under test (D-14/D-17 gating) was already implemented by prior plans in this phase. This plan's job was adding the missing behavioural signal proving that implementation is actually live, not implementing new behaviour._

## Files Created/Modified
- `app/lib/components/trail/trail_dropdown.dart` - Show-on-map now resolves via `trailMapLocation`/disables when unaddressable; Edit's post-edit invalidation branches on `isUnsynced` between `localTrailProvider` and `trailProvider`
- `app/test/components/trail/trail_dropdown_menu_test.dart` - New behavioural widget-test file (253 lines) covering all six must-have behaviours by opening the real menu
- `app/test/components/trail/trail_dropdown_delete_gate_test.dart` - Header note scoping its remaining purpose to branch-order only; no assertions removed
- `app/lib/provider/router_provider.g.dart` - Regenerated `_$routerHash()` to match `router_provider.dart`'s current source (stale since 36-12's no-codegen edit)

## Decisions Made
- Primed `authProvider` in the new test harness with an inert `Consumer` in the `Scaffold` body, reproducing the priming `trail_panel.dart` already does in production on the same screen. Without it, the `PopupMenuButton`'s `itemBuilder` -- which reads `authProvider` lazily, only when the menu opens -- observed a still-`AsyncLoading` provider on its first synchronous evaluation and rendered Edit/Delete as absent regardless of the stubbed auth state, since Riverpod providers build lazily on first read and the async stub's `Future` hadn't resolved by the time the (synchronous) `itemBuilder` ran.
- Confirmed via `git status` after the `build_runner` pass that only `router_provider.g.dart` needed regeneration; `profile_trails_provider.g.dart` and `trail_sync_provider.g.dart` (flagged as possibly stale in the plan's context) were already current. Committed only the file that actually changed.

## Deviations from Plan

None - plan executed exactly as written. The one test-harness addition (the priming `Consumer`) was necessary scaffolding to make the plan's specified test assertions pass reliably; it does not change any of the six specified behavioural assertions, fixtures, or scope.

## Issues Encountered
- Initial test run failed 4 of 6 new assertions ("Edit"/"Delete" not found) because `authProvider` -- read only inside `TrailDropdown`'s `PopupMenuButton.itemBuilder`, not in its own `build()` -- was still `AsyncLoading` at the exact synchronous moment the popup menu's item list was constructed, even after an extra `tester.pump()`. Root-caused to Riverpod's lazy provider evaluation (nothing had read `authProvider` before the menu opened) combined with `itemBuilder` being a one-shot synchronous callback, not a reactively-rebuilt widget. Resolved by adding a throwaway `Consumer` that watches `authProvider` in the harness's `Scaffold.body`, mirroring how `trail_panel.dart` already primes the same provider in the real app before `TrailDropdown` is ever tapped.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

This is the last plan in Phase 36 (v1.8 gap-closure wave). All of 36-09 through 36-13 have landed; `flutter analyze --no-pub` is clean (only pre-existing `info`-level lints in unrelated vendor files) and the full `flutter test` suite passes (849 passed, 1 pre-existing skip, 0 failures). UAT Test 3 (blocked solely on the dropdown menu being unreachable for an unsynced trail) is now unblocked and ready for its on-device re-run per the plan's `<verification>` section -- open the dropdown on an unsynced trail's detail screen, confirm Download is absent and Delete's confirm states the deletion is unrecoverable, confirm Delete greys out mid-upload, and confirm an ordinary downloaded trail's menu is unchanged.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED

- FOUND: app/test/components/trail/trail_dropdown_menu_test.dart
- FOUND commit: 612f41a9
- FOUND commit: 8bf8433d
- FOUND commit: cc21a105
