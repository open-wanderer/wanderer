---
phase: 36-local-first-recording-automatic-upload
plan: 16
subsystem: mobile-sync
tags: [flutter, riverpod, objectbox, i18n, gap-closure]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload (plans 01-15)
    provides: local-first trail capture, deferred-upload drain, TrailSync/local_trail_store,
      delete-path hardening (CR-02/WR-08/WR-10/WR-15/WR-17, 36-15)
provides:
  - "retireUploadedLocalTrail returns the server id a retired row carried, instead of
    discarding the only surviving handle on the trail (CR-01 prerequisite)"
  - "resolveNetworkSaveTarget: a pure decision picking the real save target (screen id,
    then a retired-id fallback, then null meaning refuse-and-tell-the-hiker)"
  - "TrailSync._retiredServerIds: an account-keyed, bounded (64-entry) memo exposed via
    serverIdForRetired(localId), populated the instant a drain retirement runs"
  - "localTrailProvider(localId) invalidated in the same statement block that retires the
    row, so a mounted detail screen re-reads instead of rendering a dead row (WR-01)"
  - "trail_create_screen's networkUpdate save path resolves a real server id before ever
    calling _saveViaNetwork, with an actionable refusal message
    (trail_uploaded_reopen_to_edit) when none exists -- closes CR-01"
  - "trail_detail_screen's local-id-not-found branch redirects to /trail/<serverId> when
    the drain retired the row while the screen was mounted -- closes WR-01's second half"
affects: [36-17, 36-18, 36-19, 36-20, mobile-trail-create-flow]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Retirement/removal functions that destroy a row's only identity handle must return
      that handle to the caller, not discard it -- there are no ObjectBox Query.watch()
      streams anywhere in this app, so a return value plus an explicit memo is the only
      propagation mechanism available to a still-mounted screen."
    - "A save-target resolution is a pure, three-way decision (real screen id wins, a
      retired-id fallback second, null meaning 'refuse and tell the user' third) --
      tested in isolation, not inferred from token order in a source-slicing gate."

key-files:
  created: []
  modified:
    - app/lib/store/local_trail_store.dart
    - app/lib/provider/trail/trail_sync_provider.dart
    - app/lib/routes/trail_create_screen.dart
    - app/lib/routes/trail_detail_screen.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_localizations.dart (+ 13 locale variants)
    - app/lib/i18n/untranslated_messages.json
    - app/test/store/local_trail_store_test.dart
    - app/test/store/local_trail_retirement_gate_test.dart

key-decisions:
  - "The retired-id memo (TrailSync._retiredServerIds) stores the owning accountId
    alongside the server id and serverIdForRetired re-reads currentAccountId fresh at the
    point of use -- trailSyncProvider is deliberately excluded from
    accountScopedProviders, so anything it holds outlives an account switch, and without
    the per-entry account check account B could resolve account A's retired server id
    (T-36-16-01/02)."
  - "The memo is capped at 64 entries, oldest evicted first (Dart Map insertion order) --
    it lives on a keepAlive notifier for the app's whole lifetime, so it must not grow
    unbounded across a long multi-trail session (T-36-16-03)."
  - "On a genuine refusal (no target resolves), the create screen does NOT pop the route.
    _hasUnsavedChanges is still true at that point, so popping would trigger the
    discard-changes dialog and destroy the hiker's typed edit -- the opposite of what the
    refusal message asks them to do. Accepted as a real tradeoff (T-36-16-06), flagged for
    UAT rather than solved by also suppressing the dialog."
  - "_saveViaNetwork's trailHasServerId guard is kept as a last-resort backstop (not
    removed) even though every caller is now supposed to resolve a real id first -- it
    now logs via debugPrint when reached, treating that path as an invariant break rather
    than the normal refusal route."

requirements-completed: [REC-05, SYNC-05]

# Metrics
duration: ~20min
completed: 2026-08-03
---

# Phase 36 Plan 16: Retired-trail id carry-forward and refusal message (CR-01, WR-01) Summary

**A completed upload's server id now survives the local row's retirement -- carried through a return value and an account-scoped memo -- so the record-save-fix-a-typo flow reaches the server instead of failing permanently with a generic error, and a mounted detail/create screen follows the trail to its new home instead of showing a dead end.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-03T16:53:00Z
- **Completed:** 2026-08-03T17:13:00Z
- **Tasks:** 3
- **Files modified:** 19 (4 non-generated lib files, 2 test files, 13 generated i18n files)

## Accomplishments

- Closed the CR-01 prerequisite: `retireUploadedLocalTrail` now returns the server id a
  retired row carried (captured before the row is mutated or removed), instead of
  discarding it. Its doc comment explains why this return value is the only surviving
  handle on the trail once retirement runs -- there are no ObjectBox `Query.watch()`
  streams anywhere in this app.
- Added the pure, unit-tested `resolveNetworkSaveTarget` decision: a real screen id wins;
  otherwise a non-empty retired id is used; otherwise `null` means "refuse and tell the
  hiker to re-open the trail." Four cases asserted in `local_trail_store_test.dart`.
- `TrailSync` gained an account-keyed, 64-entry-bounded memo (`_retiredServerIds`) and a
  public `serverIdForRetired(localId)` reader that re-reads `currentAccountId` fresh and
  refuses (logging) a cross-account lookup -- `trailSyncProvider` is deliberately excluded
  from `accountScopedProviders`, so this memo (like the notifier's in-flight set) survives
  an account switch and needed its own scoping.
- `_drainOne`'s retirement step now captures the return value into that memo and
  invalidates `localTrailProvider(localId)` in the same block -- closing WR-01: a hiker
  sitting on `/trail/local/<localId>` while the upload completes is no longer stuck
  rendering a row that no longer exists.
- `trail_create_screen`'s `networkUpdate` save branch resolves a target via
  `resolveNetworkSaveTarget` before ever calling `_saveViaNetwork`, falling back to
  `serverIdForRetired` when the screen's own snapshot still reads the blank
  local-sentinel id. When no target exists, the hiker sees the new
  `trail_uploaded_reopen_to_edit` message instead of the generic `error_saving_trail`, and
  the route deliberately stays open (see Decisions) rather than popping into a
  discard-changes dialog.
- `trail_detail_screen`'s local-id-not-found branch now checks `serverIdForRetired` before
  rendering `trail_not_on_this_device`, redirecting via `pushReplacement('/trail/<id>')`
  when the drain retired the row while the screen was mounted -- closing WR-01's second
  half.
- `_saveViaNetwork`'s `trailHasServerId` backstop guard is kept (not removed) as a
  last-resort invariant check, now with a `debugPrint` and the same actionable message
  instead of `error_saving_trail`.

## Task Commits

1. **Task 1: Retirement reports the server id it kept, and a pure decision picks the save
   target** - `99d7a6fa` (fix) -- includes the same-commit re-point of
   `local_trail_retirement_gate_test.dart`'s hard-coded signature literal per the plan's
   mandatory ordering requirement.
2. **Task 2: The drain memoizes the retired server id, account-keyed, and wakes the
   detail screen** - `85d36939` (fix)
3. **Task 3: The save routes to the carried-forward id, and the refusal names the real
   state** - `34240086` (fix) -- includes the new l10n key and its regenerated artifacts.

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `app/lib/store/local_trail_store.dart` - `retireUploadedLocalTrail`: `void` -> `String?`;
  new pure `resolveNetworkSaveTarget`
- `app/lib/provider/trail/trail_sync_provider.dart` - `_retiredServerIds` memo,
  `_rememberRetiredServerId`, `serverIdForRetired`; `_drainOne` captures the retirement
  return value and invalidates `localTrailProvider(localId)`
- `app/lib/routes/trail_create_screen.dart` - `networkUpdate` branch resolves a target via
  `resolveNetworkSaveTarget`/`serverIdForRetired` before `_saveViaNetwork`; new refusal
  path with `trail_uploaded_reopen_to_edit`; `_saveViaNetwork`'s backstop guard logs and
  uses the same message
- `app/lib/routes/trail_detail_screen.dart` - local-id-not-found branch redirects to
  `/trail/<serverId>` via `serverIdForRetired` before falling back to
  `trail_not_on_this_device`
- `app/lib/i18n/app_en.arb` (+ generated `app_localizations*.dart`,
  `untranslated_messages.json`) - `trail_uploaded_reopen_to_edit`
- `app/test/store/local_trail_store_test.dart` - `resolveNetworkSaveTarget` group (4 cases)
- `app/test/store/local_trail_retirement_gate_test.dart` - re-pointed `retirementBody()`'s
  signature literal to `String? retireUploadedLocalTrail(...)`, same commit as the
  return-type change

## Decisions Made

- The retired-id memo stores the owning `accountId` alongside the server id;
  `serverIdForRetired` re-reads `currentAccountId` fresh and refuses (logging) a
  cross-account lookup, since `trailSyncProvider` deliberately outlives an account switch.
- The memo is bounded to 64 entries, oldest evicted first, so a long multi-trail session
  cannot grow it unbounded on a `keepAlive` notifier.
- A genuine refusal (`targetId == null`) does not pop the create screen's route --
  `_hasUnsavedChanges` is still true at that point, and popping would trigger the
  discard-changes dialog and destroy the hiker's typed edit. This tradeoff is accepted and
  flagged for UAT rather than solved by also suppressing the dialog (T-36-16-06).
- `_saveViaNetwork`'s `trailHasServerId` guard is kept as a last-resort backstop, not
  removed -- every caller is now supposed to resolve a real id first, so reaching this
  branch is logged as an invariant break rather than treated as the normal path.

## Deviations from Plan

None -- plan executed exactly as written. Two formatting fixes were applied to satisfy
the plan's own literal grep-based acceptance criteria (not behavioural changes):
`dart format` initially re-wrapped `updatedTrail.copyWith(id: targetId, localId: null,
localPhotos: const [])` across multiple lines, breaking the plan's `copyWith(id: targetId`
substring check; wrapped in `// dart format off` / `// dart format on` markers (matching
the codebase's existing 20-05/21-01 precedent for grep-sensitive one-line expressions) to
keep it on one physical line.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- CR-01 and WR-01 are closed at the source level: `flutter analyze --no-pub` reports 0
  errors (36 pre-existing info lints, unchanged), and `flutter test` reports 908 passing,
  1 skipped, 0 failures (904 baseline + 4 new `resolveNetworkSaveTarget` cases).
- **Not provable in this repo -- device UAT owns it**, per the plan's own
  `<verification>` note: the end-to-end flow (record offline -> go online -> let it
  upload -> edit the title from the still-open screen -> confirm the change on the
  server) requires both a live ObjectBox `Store` and a real PocketBase. This is
  36-VERIFICATION.md `human_verification` item 4, re-run against this plan's behaviour,
  plus the new case: sit on `/trail/local/<localId>` while the upload completes and
  confirm the screen moves to `/trail/<serverId>` instead of showing "This trail is no
  longer on this device."
- CR-02 and CR-03 (the other two criticals from `36-REVIEW.md`) are NOT addressed by this
  plan -- it was scoped explicitly to CR-01/WR-01. They remain open for a subsequent gap-
  closure plan.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED
