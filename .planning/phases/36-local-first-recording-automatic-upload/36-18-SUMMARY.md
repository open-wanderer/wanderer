---
phase: 36-local-first-recording-automatic-upload
plan: 18
subsystem: mobile-sync
tags: [flutter, riverpod, objectbox, gap-closure]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload (plans 01-17)
    provides: local-first trail capture, deferred-upload drain, TrailSync/local_trail_store,
      delete-path hardening, retired-id carry-forward, local-row reconciliation after a
      network save (CR-01/CR-02/CR-03/WR-08/WR-10/WR-13/WR-14/WR-15/WR-16/WR-17, 36-15..17)
provides:
  - "TrailFilterNotifier.defaultFilter initialised at declaration -- resetFilter() cannot
    throw LateInitializationError regardless of what the last /trail/filter response was
    (closes WR-03)"
  - "hasKeylessPendingWaypoint: a pure, unit-tested invariant-break detector that lets
    _drainOne bail on a corrupt waypoint row before joining the in-flight set and before the
    try, so it costs the trail nothing from kMaxSyncAttempts (closes WR-04)"
  - "writeServerWaypointId retains a waypoint's localPhotos across its own upload success --
    a waypoint's photos now survive until the trail itself is retired or deleted, not until
    that one waypoint's create call returns (closes WR-09)"
  - "trail_create_screen._invalidateOwnTrailsList only builds the profileTrailsProvider
    family key from a non-null signed-in handle, logging (not silently no-opping) when one
    is not resolvable (closes WR-02)"
affects: [mobile-trail-create-flow, mobile-sync]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "A retry-configured AsyncNotifier's `.future` accessor does not reliably complete once
      a retry has occurred in this riverpod 3.3.1 pin -- tests that need to observe a
      settled error state after a retry-driven failure must listen for state.isLoading to
      go false rather than awaiting `.future` directly."
    - "An invariant-break bail (a row shape that no retry can ever fix) belongs BEFORE a
      drain/sync loop joins its in-flight set and enters its try/catch, mirroring the
      existing missing-UserEntity guard -- never inside the generic failure handler, which
      would burn retry budget on something retrying cannot fix."

key-files:
  created: []
  modified:
    - app/lib/provider/trail/trail_filter_provider.dart
    - app/lib/store/local_trail_store.dart
    - app/lib/provider/trail/trail_sync_provider.dart
    - app/lib/routes/trail_create_screen.dart
    - app/test/provider/trail/trail_filter_fallback_test.dart
    - app/test/store/local_trail_store_test.dart
    - app/test/store/local_trail_retirement_gate_test.dart

key-decisions:
  - "TrailFilterNotifier.defaultFilter is now a plain (non-late) field initialised at
    declaration with buildDefaultTrailFilter(kOfflineTrailFilterValues), not `late` and not
    `late final`. This removes both historical failure modes at once: the `late final`
    rebuild crash (Riverpod re-runs build() on the same Notifier instance) and the `late`
    unassigned-on-a-500 crash (WR-03) -- there is no uninitialised state left to reach."
  - "The WR-03 behavioural test cannot use `await container.read(provider.future)` after a
    non-connection failure: with trailFilterRetry configured, `.future` never completes once
    a retry has run (confirmed via a minimal reproduction, independent of this fix). The test
    instead listens for state.isLoading to go false and reads state directly."
  - "hasKeylessPendingWaypoint takes a plain record list (not WaypointEntity) so the decision
    is testable without a live ObjectBox Store, matching this file's established
    extract-the-pure-half discipline."
  - "writeServerWaypointId's localPhotos retention is unconditional -- no new flag or column
    tracks 'is the trail's upload fully done yet'; retireUploadedLocalTrail (success) and
    deleteUnsyncedPhotoDir (delete) already own reclaiming the files once the trail's fate is
    decided, so nothing new needs to track that state."

requirements-completed: [REC-05, REC-06, SYNC-01, SYNC-03]

# Metrics
duration: ~35min
completed: 2026-08-03
---

# Phase 36 Plan 18: defaultFilter never unassigned, invariant breaks skip the drain, photos survive until retirement, invalidation cannot address nothing (WR-02/03/04/09) Summary

**Four narrow-but-real warnings closed: resetFilter() can no longer throw LateInitializationError after any /trail/filter failure shape, a corrupt keyless waypoint row is skipped without burning the drain's four-attempt budget, a waypoint's photos stay reachable until its trail is retired or deleted rather than vanishing the instant that one waypoint uploads, and the own-trails list invalidation only ever targets a real signed-in handle instead of the literal string '@null'.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-03T19:20:00Z
- **Completed:** 2026-08-03T19:55:07Z
- **Tasks:** 3
- **Files modified:** 7

## Accomplishments

- Closed WR-03: `TrailFilterNotifier.defaultFilter` is now initialised at declaration
  (`buildDefaultTrailFilter(kOfflineTrailFilterValues)`) instead of being `late`. Previously
  `build()` only assigned it on the success path and the connection-failure fallback; any
  other failure (a 500, a malformed payload) rethrew without assigning it, and
  `resetFilter()` read it unconditionally from a button callback on a `keepAlive` provider,
  throwing `LateInitializationError`. The declaration-site initialiser removes the
  uninitialised state entirely, while `build()`'s success and connection-failure paths still
  overwrite it and every other error still surfaces as `AsyncError`.
- Closed WR-04: new pure `hasKeylessPendingWaypoint` in `local_trail_store.dart` detects a
  waypoint whose id is still a local sentinel but has no `localKey` to record the server's
  returned id against -- an invariant break `WaypointEntity.fromModel` should make
  unreachable, not a network condition. `_drainOne` now bails on this BEFORE joining the
  in-flight set and before its `try`, mirroring the existing missing-`UserEntity` guard one
  block up. The `StateError` previously thrown mid-loop (which consumed one of the four
  `kMaxSyncAttempts` and could park a healthy trail as `failed` after four fast lifecycle
  passes) is gone, replaced by a non-null assertion the pre-loop guard now proves safe.
- Closed WR-09: `writeServerWaypointId` no longer clears a waypoint's `localPhotos` the
  instant its own create call succeeds. If a later waypoint in the same drain loop fails and
  the trail parks as `failed`, the earlier waypoint's photos remain reachable through the
  model instead of pointing at nothing while the JPEGs still sit on disk under
  `unsynced/<localId>/waypoints/<key>/`. `retireUploadedLocalTrail` and
  `deleteUnsyncedPhotoDir` already own reclaiming those files once the trail's fate is
  decided.
- Closed WR-02: `trail_create_screen._invalidateOwnTrailsList` no longer interpolates
  `ref.read(authProvider).value?.preferredUsername` straight into the `profileTrailsProvider`
  family key. During a token refresh in flight or a mid-logout race that value is null,
  producing the literal key `'@null'`, which matches no mounted `profile_trail_screen`
  instance -- a silent no-op the function's own doc comment already warned against. The
  invalidation now only fires when a real username resolved; the miss is logged via
  `debugPrint` instead of vanishing. `trailLibraryProvider`'s unconditional invalidation is
  unchanged.
- Added a companion regression test proving the connection-failure fallback path is
  unaffected by the WR-03 fix: a `DioExceptionType.connectionError` build still yields
  `AsyncData`, never `AsyncError`.

## Task Commits

1. **Task 1: defaultFilter is a real value from the first instant (WR-03)** - `11371dda`
   (fix)
2. **Task 2: An invariant break skips the drain instead of burning its retry budget, and
   photos survive until retirement (WR-04, WR-09)** - `47137c3b` (fix)
3. **Task 3: The own-trails invalidation cannot address a family key that matches nothing
   (WR-02)** - `a81f4add` (fix)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `app/lib/provider/trail/trail_filter_provider.dart` - `defaultFilter` changed from `late`
  to a declaration-site-initialised field; doc comment explains both historical crash modes
  it now prevents
- `app/lib/store/local_trail_store.dart` - new pure `hasKeylessPendingWaypoint`;
  `writeServerWaypointId` no longer clears `localPhotos`, doc comment updated
- `app/lib/provider/trail/trail_sync_provider.dart` - `_drainOne` gained a
  `hasKeylessPendingWaypoint` bail before the in-flight join and before the `try`; the
  waypoint-loop `StateError` removed, replaced with a proven-safe `!` assertion
- `app/lib/routes/trail_create_screen.dart` - `_invalidateOwnTrailsList` null-guards the
  resolved username before building the `profileTrailsProvider` family key
- `app/test/provider/trail/trail_filter_fallback_test.dart` - new WR-03 behavioural group
  (500 failure -> `resetFilter()` does not throw, settles on the offline default) and a
  connection-failure regression guard
- `app/test/store/local_trail_store_test.dart` - new `hasKeylessPendingWaypoint` group (5
  cases: empty, local+keyed, local+keyless, server+keyless, mixed)
- `app/test/store/local_trail_retirement_gate_test.dart` - two new `drainOneBody()`
  assertions (`hasKeylessPendingWaypoint(` precedes the in-flight join; zero `StateError(`
  occurrences) and a new `writeServerWaypointId` body-slicing group (zero `localPhotos = []`
  occurrences)

## Decisions Made

- `defaultFilter` is a plain (non-`late`) field, not `late final` -- both historical crash
  modes (the `late final` rebuild crash and the `late` unassigned-on-error crash) share one
  root cause (an uninitialised field that some code path can observe) and one fix
  (initialise at declaration, nothing left to be uninitialised).
- The WR-03 behavioural test cannot rely on `await container.read(provider.future)` after a
  non-connection failure -- confirmed via a minimal reproduction that with
  `trailFilterRetry` configured, `.future` never completes once a retry has actually run,
  independent of this plan's change. The test listens for `state.isLoading` to go false
  instead.
- `hasKeylessPendingWaypoint` takes a plain record list, not `WaypointEntity`, continuing
  this file's extract-the-pure-half discipline so the decision is unit-testable without a
  live `Store`.
- `writeServerWaypointId`'s `localPhotos` retention needed no new flag or schema change --
  `retireUploadedLocalTrail` and `deleteUnsyncedPhotoDir` already own reclaiming the files
  once the trail's fate (success or delete) is decided.

## Deviations from Plan

None — plan executed exactly as written. One additional test was added beyond the plan's
explicit instructions: a connection-failure regression guard in
`trail_filter_fallback_test.dart`, added because the plan's own acceptance criteria required
"the existing connection-failure fallback case in the same file still passes" but no such
runtime test actually existed in that file prior to this plan (only pure `buildDefaultTrailFilter`
tests that never invoke `build()` with a `DioException` at all). Added under Rule 2 (missing
critical test coverage the plan's own acceptance criteria assumed) rather than skipping the
criterion.

## Issues Encountered

- `await container.read(trailFilterProvider(id).future)` hung indefinitely (past the 30s test
  timeout) when the underlying build failed and `trailFilterRetry` triggered a real retry,
  even though the provider's `state` correctly settled to `AsyncError` within ~1.2s
  (confirmed via a `container.listen` probe). Root cause isolated to riverpod 3.3.1's
  `.future` accessor, independent of this plan's `defaultFilter` change (reproduced with an
  unmodified copy of the provider). Worked around by listening for `state.isLoading` to go
  false and reading `container.read(provider)` directly instead of awaiting `.future` in the
  new WR-03 test.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- WR-02, WR-03, WR-04 and WR-09 are closed at the source level: `flutter analyze --no-pub`
  reports 0 errors (36 pre-existing info lints, unchanged from the 36-17 baseline), and
  `flutter test` reports 929 passing, 1 skipped, 0 failures (918 baseline + 11 new cases
  across `trail_filter_fallback_test.dart`, `local_trail_store_test.dart` and
  `local_trail_retirement_gate_test.dart`).
- No `late` field remains on `TrailFilterNotifier`; no `StateError` is thrown from inside
  `_drainOne`'s `try`; `writeServerWaypointId` writes the server id and photo filenames and
  nothing else destructive; every `profileTrailsProvider` family key built in
  `trail_create_screen.dart` comes from a non-null handle -- all four of this plan's
  `<success_criteria>` hold.
- **Not provable in this repo — device UAT owns it**, per the plan's own `<verification>`
  note: WR-09's effect (a waypoint's photos still rendering offline after a LATER waypoint's
  upload fails and the trail parks as `failed`) needs a live `Store` and a controllable
  server; WR-04's effect (a healthy trail not being parked after a lifecycle flurry) needs a
  real app lifecycle. Both fold into `36-VERIFICATION.md` `human_verification` item 5's
  interrupted-upload pass: after forcing a mid-loop waypoint failure, open the trail offline
  and confirm the already-uploaded waypoint still shows its photos, and confirm the chip
  reads Pending (not Failed) after backgrounding/foregrounding the app several times in quick
  succession.
- All defects `36-REVIEW.md` still listed as open against this phase after 36-15/16/17
  (WR-02, WR-03, WR-04, WR-09) are now closed. `36-REVIEW.md`'s remaining warnings (WR-05
  through WR-08, WR-10 through WR-12, WR-17, and WR-06's coverage half) were out of this
  plan's scope and remain tracked for a future gap-closure pass if the phase revisits them.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED
