---
phase: 36-local-first-recording-automatic-upload
plan: 20
subsystem: testing
tags: [flutter, riverpod, testing, gate-strengthening, gap-closure]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload (plans 01-19)
    provides: local-first trail capture, deferred-upload drain, TrailSync/local_trail_store,
      delete-path hardening, retired-id carry-forward and refusal message, local-row
      reconciliation after a network save, WR-02/03/04/09 gap closure (CR-01/CR-02/CR-03/
      WR-02/03/04/08/09/10/13/14/15/16/17, 36-15..18)
provides:
  - "trail_create_screen_local_save_gate_test.dart: the CR-01 and CR-03 gates now assert an
    EFFECT of the guard (a return; that actually fires, a message that actually names the
    recoverable state, an outcome token that actually sits inside the live if condition) --
    not only token presence/order. Both were verified to fail against the exact falsifying
    rewrite the review named."
  - "Two new gates covering what the networkUpdate branch PASSES to _saveViaNetwork (a
    resolved targetId via copyWith, never the possibly-blank updatedTrail.id) and what
    _saveViaNetwork does with a successful result (reconciles the local row strictly between
    adopting result.trail and invalidating the own-trails list) -- the review's central
    finding that no gate in the file constrained this."
  - "local_trail_retirement_gate_test.dart: a new group pinning the retired-id carry-forward
    as an effect (the return value is assigned and consumed by the memo, not discarded; the
    localTrailProvider invalidation runs after retirement) and a new serverIdForRetiredBody()
    group pinning the account-scoping comparison as load-bearing, not merely present. All
    three new gates verified to fail against their named falsifying rewrite."
  - "trail_detail_screen_retired_redirect_test.dart: a behavioural widget test (real GoRouter,
    real TrailDetailScreen) proving the WR-01 post-retirement redirect on rendered output --
    the redirect firing with a resolved server id, the dead end staying without one, and a
    present local trail never consulting serverIdForRetired at all."
affects: [mobile-trail-create-flow, mobile-sync, mobile-trail-detail-flow]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Source-slicing gates for effects, not tokens: when a gate cannot open a live Store or
      mount a screen, slice the guarded region and assert the guard's CONSEQUENCE (a return;,
      a specific message string, a value flowing into the exact argument position of the call
      it protects) rather than only that two substrings appear in the expected order. A gate
      that only checks order passes against an empty guard body or a debugPrint mentioning the
      same token."
    - "Falsify-observe-restore as a required proof step for any gate a review names a specific
      falsifying rewrite for: apply the exact rewrite to the real source file, run the gate,
      record the failure output, then restore and verify a clean diff before committing. A
      gate that was never watched to fail is not a proven gate."
    - "When a plan's suggested test substitution turns out not to hold against the actual
      source (verified empirically, not assumed), replace it with a substitution that proves
      an equivalent or stronger property, and record the discrepancy and its evidence in both
      the test file's own comment and the plan's SUMMARY -- never silently drop the case."

key-files:
  created:
    - app/test/routes/trail_detail_screen_retired_redirect_test.dart
  modified:
    - app/test/routes/trail_create_screen_local_save_gate_test.dart
    - app/test/store/local_trail_retirement_gate_test.dart

key-decisions:
  - "The CR-02 `_localId`-ordering gate (trail_create_screen_local_save_gate_test.dart:414-463)
    was left unchanged per the plan's explicit instruction (the review rated it acceptable).
    Its two `isNot(-1)` presence checks plus one ordering comparison do not literally contain
    a `contains(`/`allMatches(` call, which is a narrow tension with the acceptance criterion
    'every expect( on an indexOf comparison is accompanied by a contains()/allMatches()
    assertion' read as applying repo-wide -- resolved in favor of the plan's explicit 'keep
    as-is' instruction over the broader-sounding acceptance-criteria wording, since the two
    presence checks already establish both literals exist (not merely mentioned) before the
    ordering is asserted, and the review itself judged the gate sufficient."
  - "Test 3 of trail_detail_screen_retired_redirect_test.dart substitutes the plan's suggested
    fallback ('serverIdForRetired returns the empty string, dead-end text renders') with a
    call-counter proof that serverIdForRetired is never consulted when a local trail is
    present. Verified empirically (a scratch probe against the real router) that the
    suggested substitution does not hold: `router_provider.dart`'s own comment states an
    empty id 'would emit /trail/, which go_router canonicalizes to /trail, a path with no
    route', and the probe confirmed GoRouter throws `Bad state: No element` rather than
    rendering `trail_not_on_this_device`. The substituted test proves the guard's real shape
    (`if (trail == null)`, no empty-string special case) instead."
  - "Test 3 mounts a present-trail TrailDetailScreen fully expecting `_buildDetail` to throw
    (no ObjectBox Store, no native maplibre platform view in this environment) and consumes
    the exception via `tester.takeException()` rather than avoiding the mount -- this proves
    the call-count assertion holds regardless of when in `_buildDetail` the environment fails,
    which is a stronger claim than avoiding the crash entirely would have been."

requirements-completed: [REC-05, SYNC-04, SYNC-05]

# Metrics
duration: ~20min
completed: 2026-08-03
---

# Phase 36 Plan 20: Effect-asserting gates close WR-05, plus a real redirect widget test Summary

**Rewrites the three WR-05-flagged token-order-only gates in trail_create_screen_local_save_gate_test.dart to assert guard effects (a return; that fires, a message that names the recoverable state, an outcome inside the live condition), adds the previously-missing gates on what is passed to _saveViaNetwork, extends local_trail_retirement_gate_test.dart with effect assertions on the retired-id carry-forward and its account scoping, and adds a real-GoRouter behavioural widget test proving the post-retirement redirect on rendered output -- every rewritten/added gate that the review named a falsifying rewrite for was applied to the real source, observed to fail, and restored.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-03T17:58:19Z
- **Completed:** 2026-08-03T18:18:23Z
- **Tasks:** 3
- **Files modified:** 3 (2 modified, 1 created)

## Accomplishments

- Closed WR-05 in `trail_create_screen_local_save_gate_test.dart`:
  - The mis-titled test ("...ONLY on the alreadySynced escape hatch") is renamed to name what
    the guard actually checks -- `alreadySynced`, `missing` OR `alreadyUploaded` -- and now
    asserts all three outcome tokens are present, each precedes the `_saveViaNetwork(` call,
    and the guard between `if (` and that call compares `outcome` to exactly three values (so
    a silently-added fourth condition, e.g. `LocalUpdateOutcome.updated` leaking in, would also
    fail it).
  - The CR-01 blank-id-refusal gate now requires the guarded slice to contain `return;`, to
    contain `trail_uploaded_reopen_to_edit`, and to NOT contain `l10n.error_saving_trail` --
    not merely that `trailHasServerId(` precedes the network call.
  - The CR-03 `alreadyUploaded` gate now requires the outcome token to sit strictly between an
    `if (` and its matching `) {` -- not merely somewhere before `_finishLocalSave(`.
  - Two new gates close the review's central finding ("no gate in this file constrains what is
    passed to `_saveViaNetwork`"): one asserts the `networkUpdate` branch calls
    `resolveNetworkSaveTarget(` and hands `_saveViaNetwork` `updatedTrail.copyWith(id:
    targetId, ...)` (never the raw, possibly-blank `updatedTrail`), with `localPhotos: const
    []` and `existsSync()` filtering also present; the other asserts `_saveViaNetwork` reconciles
    the local row (`applyNetworkEditToLocalRow(`) strictly between adopting `result.trail` and
    calling `_invalidateOwnTrailsList()`.
  - A new WR-13 gate asserts `photosNotYetOnServer(` filters the network photo payload only --
    the `createLocal` branch (which never touches the network) must never call it.
  - The CR-02 `_localId`-ordering gate is unchanged, per the plan's explicit instruction (see
    Decisions Made for the one acceptance-criteria tension this created).
- Closed WR-05/T-36-20-01/T-36-20-03 in `local_trail_retirement_gate_test.dart`:
  - Extended the `retirementBody()` group: `retireUploadedLocalTrail` must capture the server
    id once (`final serverId =`) before either exit mutates the row, and both exits must
    actually `return serverId;` (asserted via a `>= 2` regex count), not merely compute and
    discard it.
  - New group pinning the drain's carry-forward: the retirement return value must be assigned
    (`= retireUploadedLocalTrail(`), `_rememberRetiredServerId(` must run after that
    assignment, and `invalidate(localTrailProvider(localId))` must run after retirement --
    each is an ordering/consumption assertion, not a presence check.
  - New `serverIdForRetiredBody()` group: `currentAccountId(` must be read, and the slice
    between the memo lookup and its final `return entry.serverId;` must contain both
    `accountId` and a `!=` comparison -- so removing the per-entry account check (the
    T-36-16-01/T-36-20-03 threat: account B resolving account A's retired server id) fails the
    gate.
  - The file's header doc now states plainly which of its own assertions are effect
    assertions versus presence/absence facts, per the plan's instruction.
- Added `trail_detail_screen_retired_redirect_test.dart` (286 lines, zero source-slicing): a
  real `GoRouter` mounts the real `TrailDetailScreen` with `localTrailProvider` and
  `trailSyncProvider` overridden. Three cases: the redirect firing to `/trail/<serverId>` when
  `serverIdForRetired` resolves one; the dead end (`trail_not_on_this_device`) staying when it
  does not; and a present local trail never calling `serverIdForRetired` at all (the guard's
  real shape is `if (trail == null)`, proven via a call counter since the `trail != null`
  branch cannot be mounted in this environment -- see Decisions Made).

## Task Commits

1. **Task 1: The create-screen gates assert the guard's effect, and say what they check** -
   `df119801` (test)
2. **Task 2: The drain gates pin the retired-id carry-forward as an effect, not a mention** -
   `f45bd466` (test)
3. **Task 3: A real widget test for the post-retirement redirect** - `06acff97` (test)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `app/test/routes/trail_create_screen_local_save_gate_test.dart` - three weak gates
  strengthened to assert effects; mis-titled test renamed and widened; two new gates on what
  is passed to `_saveViaNetwork`; one new WR-13 gate
- `app/test/store/local_trail_retirement_gate_test.dart` - `retirementBody()` group extended
  with a two-exits-return assertion; new drain carry-forward group; new
  `serverIdForRetiredBody()` account-scoping group; header doc updated to label effect vs.
  presence/absence assertions
- `app/test/routes/trail_detail_screen_retired_redirect_test.dart` (new) - behavioural widget
  test for the post-retirement redirect

## Decisions Made

- The CR-02 `_localId`-ordering gate was left unchanged per the plan's explicit "keep as-is"
  instruction, even though it does not literally use `contains(`/`allMatches(` alongside its
  ordering comparison -- see key-decisions above for the reasoning.
- Test 3 of the redirect widget test substitutes the plan's suggested fallback with a
  call-counter proof after empirically confirming (via a scratch probe, discarded before
  committing) that the plan's suggested "empty string treated as no target" assertion does not
  hold against the real source -- `router_provider.dart` canonicalizes an empty id to an
  unmatched route, and GoRouter throws rather than falling back to the dead-end text.
- Test 3 mounts the present-trail case fully (expecting and consuming the environment's own
  `_buildDetail` crash via `tester.takeException()`) rather than avoiding the mount, so the
  call-count assertion is proven to hold regardless of exactly where in `_buildDetail` the
  environment gives out.

## Deviations from Plan

None (Rule-triggered) — the one adjustment (Test 3's substitution) is explicitly sanctioned by
the plan's own action text ("record the substitution and its reason in the SUMMARY rather than
silently dropping the case") and is documented above and in the test file's own header comment,
not an unplanned deviation.

## Falsification Records (required proof, per the plan's threat model T-36-20-01)

**1. CR-01 gate — `trail_create_screen_local_save_gate_test.dart`, "_saveViaNetwork refuses to
run with a blank trail id, and the refusal actually returns and names the recoverable state
(CR-01, WR-05)"**

Applied to `app/lib/routes/trail_create_screen.dart` (scratch, not committed):
```dart
if (!trailHasServerId(updatedTrail.id)) { /* TODO */ }
```
replacing the full guard block (return; toast; debugPrint). `flutter test
test/routes/trail_create_screen_local_save_gate_test.dart` result:
```
Expected: true
  Actual: <false>
The blank-id guard must actually `return;` rather than fall through to the POST below it. A
guard with an empty body (e.g. `if (!trailHasServerId(updatedTrail.id)) { /* TODO */ }`)
restores the un-routable blank-id POST while this substring check alone would previously
still have passed (WR-05).
```
Restored; `git diff` against `app/lib/routes/trail_create_screen.dart` clean afterward.

**2. CR-03 gate — "the updateLocal branch also treats LocalUpdateOutcome.alreadyUploaded as a
network case, checked inside the guard condition itself (CR-03, WR-05)" — and the widened
three-outcome test**

Applied to the same file (scratch, not committed):
```dart
debugPrint('reached with outcome LocalUpdateOutcome.alreadyUploaded check');
if (outcome == LocalUpdateOutcome.alreadySynced ||
    outcome == LocalUpdateOutcome.missing) {
```
(moving `alreadyUploaded` out of the condition into a `debugPrint`, matching the review's named
falsifier verbatim). `flutter test` result — two gates failed:
```
[the three-outcome test]
Expected: <3>
  Actual: <2>
The guard between `if (` and the _saveViaNetwork( call it gates should compare `outcome`
against exactly three values (alreadySynced, missing, alreadyUploaded). ...

[the CR-03 gate]
Expected: not <-1>
  Actual: <-1>
No `if (` precedes the LocalUpdateOutcome.alreadyUploaded reference -- it is no longer part
of a conditional at all.
```
Restored; `git diff` clean afterward; `flutter test` on the file returned to 16/16 passing.

**3. Drain carry-forward gates — `local_trail_retirement_gate_test.dart`, "assigns
retireUploadedLocalTrail's return value" and "_rememberRetiredServerId( runs after the
retirement return value is captured"**

Applied to `app/lib/provider/trail/trail_sync_provider.dart` (scratch, not committed):
```dart
retireUploadedLocalTrail(store, localId);
```
replacing `final retiredServerId = retireUploadedLocalTrail(store, localId); if
(retiredServerId != null) { _rememberRetiredServerId(...); }` (discarding the return value
entirely, per the plan's named falsifier). `flutter test` result — two gates failed:
```
[assigns the return value]
Expected: true
  Actual: <false>
Calling retireUploadedLocalTrail( for its side effect alone and discarding the return value
re-opens CR-01: the screen's trail.id snapshot stays the blank local-sentinel value forever...

[_rememberRetiredServerId ordering]
Expected: not <-1>
  Actual: <-1>
```
Restored; `git diff` clean afterward.

**4. WR-01 ordering gate — "invalidates localTrailProvider(localId) AFTER retirement runs"**

Applied to the same file (scratch, not committed):
```dart
ref.invalidate(localTrailProvider(localId));
final retiredServerId = retireUploadedLocalTrail(store, localId);
if (retiredServerId != null) { _rememberRetiredServerId(localId, retiredServerId, accountId); }
```
(moving the invalidate above the retirement call, per the plan's named falsifier). `flutter
test` result:
```
Expected: true
  Actual: <false>
The invalidation must run AFTER retireUploadedLocalTrail, or the still-mounted provider
re-reads a row that has not been retired yet and observes no change.
```
Restored; `git diff` clean afterward.

**5. Account-scoping gate — "never returns the memoized server id except from inside an
account comparison" (T-36-16-01, T-36-20-03)**

Applied to the same file (scratch, not committed):
```dart
final entry = _retiredServerIds[localId];
if (entry == null) return null;
return entry.serverId;
```
(removing the `if (entry.accountId != accountId) { ...; return null; }` block entirely, per
the plan's named falsifier). `flutter test` result:
```
Expected: true
  Actual: <false>
trailSyncProvider is deliberately excluded from accountScopedProviders
(account_scope_invalidation.dart), so this memo survives an account switch. Without a
per-entry account comparison between the memo lookup and the return, account B could resolve
account A's retired server id through a stale localId and post an edit to A's trail.
```
Restored; `git diff` clean afterward.

**Every falsification above was restored before its Task's commit, and `git diff` against the
touched `lib/` files was confirmed clean at commit time — no falsifying rewrite is present in
any committed file.**

## Issues Encountered

- Two of the new gates in Task 1 initially failed against the REAL (unmodified) source before
  the falsification round even started: the `l10n.error_saving_trail` absence check matched a
  doc comment mentioning the string (`error_saving_trail` appears in a comment at
  `trail_create_screen.dart:727` explaining the guard's own rationale) -- fixed by checking for
  the literal code usage `l10n.error_saving_trail` instead of the bare substring. The
  `copyWith` proximity-window check (`< 200` chars after the `_saveViaNetwork(` call) failed
  because a doc comment sits between the call and its argument in the real source -- widened to
  `< 350` chars, still narrow enough to reject "somewhere unrelated later in the branch." Both
  fixed and re-verified against real source before proceeding to falsification (Rule 1 --
  these were bugs in the new gates' own calibration, not scope changes).
- Task 3's control case (a present local trail rendering normally) could not be built as the
  plan's primary suggestion or its literally-suggested fallback -- see Decisions Made. Resolved
  via a call-counter substitution, verified empirically against the real router before
  committing.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- WR-05 is closed: no gate in `trail_create_screen_local_save_gate_test.dart` or
  `local_trail_retirement_gate_test.dart` passes against the falsifying rewrite the review
  named for it (5 falsifications performed, observed to fail, restored -- see above). Every
  gate's name matches what it asserts.
- The post-retirement redirect (WR-01's second half, from 36-16) now has an executable
  behavioural test, not only source-slicing coverage.
- `cd app && flutter analyze --no-pub` reports 0 errors (36 pre-existing info lints, unchanged
  baseline since 36-15).
- `cd app && flutter test` reports 941 passing, 1 skipped, 0 failures (929 baseline + 3 new
  create-screen gates + 6 new retirement gates + 3 new redirect widget tests = 941).
- **Explicitly not claimed** (per the plan's own `<verification>` note, unchanged by this
  plan): these gates do not make the phase's ObjectBox and PocketBase paths behaviourally
  tested -- they cannot, in this environment. The end-to-end proof remains the device UAT
  items accumulated across 36-15 through 36-19's `human_verification` sections. This plan adds
  no new UAT items; it strengthens the automated signal that the phase's prior UAT rounds are
  actually protected going forward.
- This closes the last plan in the 36-15..20 gap-closure run. Phase 36's `36-REVIEW.md`
  criticals (CR-01, CR-02, CR-03) and the warnings in scope for this run (WR-01 through WR-05,
  WR-08 through WR-10, WR-13 through WR-17) are all closed at the source level. WR-06's
  translation-coverage half, WR-07 (dead `retry_upload` key), WR-11 (TrailPanel tab gating) and
  WR-12 (`LocalTrailMetrics` doc wording) were out of this run's scope and remain open if a
  future gap-closure pass revisits them.

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*
