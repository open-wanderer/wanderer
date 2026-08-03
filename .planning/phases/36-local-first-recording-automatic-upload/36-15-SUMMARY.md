---
phase: 36-local-first-recording-automatic-upload
plan: 15
subsystem: mobile-sync
tags: [flutter, riverpod, objectbox, dio, i18n, delete-flow]

# Dependency graph
requires:
  - phase: 36-local-first-recording-automatic-upload (plans 01-14)
    provides: local-first trail capture, deferred-upload drain, TrailSync/local_trail_store,
      TrailDropdown's D-14/CR-04 delete gating, recordIdDirSegment/isLocalId path-segment
      validation
provides:
  - "readLocalTrailServerId: a parse-independent, owner-scoped delete decision that cannot
    be defeated by unparseable cached GPX (closes CR-02)"
  - "resolveServerDeleteOutcome: a pure classification of a failed/404'd server DELETE
    (404 proceeds locally, offline aborts needing a connection, everything else aborts
    and reports) -- closes WR-15"
  - "UnsyncedDeleteResult: TrailSync.deleteUnsynced no longer throws for a failed server
    DELETE; every outcome is classified and mapped to a distinct localized toast"
  - "Owner-scoped deleteLocalTrailRow/resetDrainBackoff (closes the account half of WR-10)"
  - "recordIdDirSegment validation on every Dio path built from a server-supplied trail id
    (deleteUnsynced and trail_save_provider.deleteTrail) -- closes WR-17"
  - "_deleteOnServer: the server DELETE path extracted so a null-localId unsynced trail
    with a real server id routes to it instead of the silent un-download no-op (closes
    WR-08)"
affects: [36-16, mobile-trail-delete-flow]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Delete decisions read the raw persisted id column (readLocalTrailServerId), never
      through toModel(), when the decision must not depend on cached data still parsing"
    - "Classify-don't-throw: async operations with multiple legitimate non-exceptional
      outcomes (deleted/blockedInFlight/needsConnection/failed) return an enum instead of
      throwing for the caller to interpret"

key-files:
  created: []
  modified:
    - app/lib/store/local_trail_store.dart
    - app/lib/provider/trail/trail_sync_provider.dart
    - app/lib/provider/trail/trail_save_provider.dart
    - app/lib/components/trail/trail_dropdown.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_localizations.dart (+ 13 locale variants)
    - app/lib/i18n/untranslated_messages.json
    - app/test/store/local_trail_store_test.dart
    - app/test/store/local_trail_retirement_gate_test.dart
    - app/test/components/trail/trail_dropdown_menu_test.dart
    - app/test/components/trail/trail_dropdown_delete_gate_test.dart

key-decisions:
  - "No 'delete on this device only' escape hatch for an offline delete of an
    already-uploaded trail -- deliberately, per resolveServerDeleteOutcome's doc comment:
    it would destroy the device's only pointer to a possibly-public live server trail,
    recreating CR-02. The hiker is told to reconnect instead."
  - "404 on the server DELETE proceeds with the local delete (the server already agrees
    the trail is gone); every other failure (401/403/500/offline) is classified and
    reported distinctly rather than treated as either success or a uniform retryable
    failure."
  - "Validation of the server-supplied trail id (recordIdDirSegment) happens at every
    Dio request site, not at the point writeServerTrailId persists it -- a throw at
    persist time would leave the row unmarked, and the next drain pass would re-issue
    PUT /trail/form and create a duplicate trail (the SYNC-04 failure this file's step 2
    exists to prevent)."

requirements-completed: [REC-03, REC-04, SYNC-04, SYNC-05]

# Metrics
duration: 35min
completed: 2026-08-03
---

# Phase 36 Plan 15: Delete-path hardening (CR-02, WR-08, WR-10, WR-15, WR-17) Summary

**Trail delete now decides on the raw persisted server id (never through a GPX-parsing `toModel()`), classifies every server-DELETE failure into a distinct localized outcome, and never falls through to a silent un-download for a row with no local handle.**

## Performance

- **Duration:** 35 min
- **Started:** 2026-08-03T16:44:34Z
- **Completed:** 2026-08-03T17:19:37Z
- **Tasks:** 3
- **Files modified:** 20 (4 non-generated lib files, 4 test files, 12 generated i18n files)

## Accomplishments

- Closed CR-02: `deleteUnsynced`'s decision now reads `TrailEntity.id` off the column via
  the new `readLocalTrailServerId`, never through `readLocalTrail`/`toModel()` — an
  unparseable cached GPX can no longer make the method skip the server DELETE and strand a
  live, possibly-public server trail with no device left pointing at it.
- Closed WR-15: a failed or 404'd server DELETE is classified by the new pure
  `resolveServerDeleteOutcome` instead of either throwing unconditionally or being
  swallowed. A 404 (server copy already gone) proceeds with the local delete; an
  unreachable server tells the hiker to reconnect (`delete_needs_connection`, new key);
  every other failure (401/403/500) is reported as a distinct, localized, logged error.
  There is deliberately no "delete on this device only" escape hatch for the offline case.
- Closed WR-17: the server id is validated through `recordIdDirSegment` at every Dio
  request site that consumes it — `deleteUnsynced`'s DELETE path and
  `trail_save_provider.deleteTrail`'s pre-existing sibling call site — before it reaches
  the network layer. A rejected id fails closed (`UnsyncedDeleteResult.failed`) without
  touching the local row.
- Closed the account half of WR-10: `deleteLocalTrailRow` and `resetDrainBackoff` now
  require `accountId` and scope their query on `TrailEntity_.owner`, so account B can no
  longer delete or reset the backoff of account A's device-only row through a stale
  `localId`.
- Closed WR-08: a null-`localId` unsynced trail (the shape `retireUploadedLocalTrail`'s
  demote branch and `TrailDownloadService`'s carry-forward can both produce for a row
  already carrying a server id) is now routed explicitly — to the newly-extracted
  `_deleteOnServer` when a server id exists, otherwise to a translated error toast — and
  can no longer fall through to the `trail.isLocal` un-download branch's silent no-op,
  which previously contradicted the `delete_trail_confirm` dialog `_confirmDelete` had
  already shown.
- Two hardcoded `'Error deleting trail'` English literals replaced with
  `l18n.error_deleting_trail`.

## Task Commits

1. **Task 1 + Task 2: parse-independent, owner-scoped delete decision + classified
   `deleteUnsynced`** - `15d699ad` (fix) — committed together per the plan's own
   acceptance-criteria note: Task 1 alone leaves call-site compile errors in
   `trail_sync_provider.dart` that only Task 2's signature updates resolve.
2. **Task 3: dropdown routes and reports every delete outcome** - `35603e63` (fix)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `app/lib/store/local_trail_store.dart` - `readLocalTrailServerId`, `ServerDeleteOutcome`/`resolveServerDeleteOutcome`, owner-scoped `deleteLocalTrailRow`/`resetDrainBackoff`
- `app/lib/provider/trail/trail_sync_provider.dart` - `UnsyncedDeleteResult`, rewritten `deleteUnsynced` (classifies instead of throwing, validates server id, owner-scoped), `retry` reads a fresh account id
- `app/lib/provider/trail/trail_save_provider.dart` - `deleteTrail` validates `trail.id` through `recordIdDirSegment` (WR-17, pre-existing sibling call site)
- `app/lib/components/trail/trail_dropdown.dart` - restructured `_deleteTrail`'s unsynced branch (null-`localId` routing, WR-08), extracted `_deleteOnServer`, switched on `UnsyncedDeleteResult`, localized both error toasts
- `app/lib/i18n/app_en.arb` (+ generated `app_localizations*.dart`, `untranslated_messages.json`) - `error_deleting_trail`, `delete_needs_connection`
- `app/test/store/local_trail_store_test.dart` - `resolveServerDeleteOutcome` group (404/null+connectionFailed/401/403/500/null+not-connectionFailed)
- `app/test/store/local_trail_retirement_gate_test.dart` - re-pointed `deleteUnsynced` group: asserts `readLocalTrailServerId(` present and `readLocalTrail(` absent, `recordIdDirSegment(` present, `deleteLocalTrailRow(` called with `accountId:`, and `resolveServerDeleteOutcome(` sits between the server DELETE and the local delete
- `app/test/components/trail/trail_dropdown_delete_gate_test.dart` - re-pointed the server-DELETE-is-downstream assertion to pin both halves of the `_deleteOnServer` extraction
- `app/test/components/trail/trail_dropdown_menu_test.dart` - added a null-`localId`-with-server-id case (shows the reversible confirm copy) and a synced-trail control case

## Decisions Made

- No "delete on this device only" escape hatch for an offline delete of an
  already-uploaded trail — see `resolveServerDeleteOutcome`'s doc comment; it would strand
  a possibly-public live server trail, recreating CR-02/CR-04.
- Validation of the server-supplied id happens at every Dio request site, not at
  `writeServerTrailId`'s persist point — a throw there would leave the row unmarked and
  cause the next drain pass to create a duplicate trail (SYNC-04).
- `retireUploadedLocalTrail`'s own lack of an owner clause was left untouched per the
  plan's explicit instruction — its doc comment already argues correctly that
  `selectDrainCandidates` scoped the row before the drain reached it; 36-16 owns that.

## Deviations from Plan

None — plan executed exactly as written. The two-commit split (Task 1+2 together, Task 3
separate) matches the plan's own acceptance-criteria note that Task 1 alone leaves
call-site errors resolved only by Task 2.

## Issues Encountered

- The two new `context`-after-await usages introduced while restructuring `_deleteTrail`
  (the `trail.isLocal` branch and the `_deleteOnServer` call) triggered
  `use_build_context_synchronously` info lints not present in the pre-existing 36-issue
  baseline. Fixed by adding explicit `if (!context.mounted) return;` guards before each,
  restoring the baseline (36 issues, 0 errors) after formatting.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Delete-path hardening for CR-02/WR-08/WR-10/WR-15/WR-17 is complete and covered by
  automated source-level gates plus a new behavioural widget-test case; `flutter analyze`
  is clean (36 pre-existing info lints, 0 errors) and `flutter test` reports 0 failures
  (904 passing, 1 skipped).
- **Not provable in this repo — device UAT owns it** (per the plan's own
  `<verification>` note): `flutter test` cannot open an ObjectBox `Store` or reach
  PocketBase, so `readLocalTrailServerId`'s query, the owner clauses, and the actual
  `DELETE /trail/{id}` round trip (404 / offline / 401-403 cases) have no behavioural
  surface in this environment. 36-VERIFICATION.md's UAT Test 3 is the proof point and
  should additionally cover: deleting a trail whose cached GPX is corrupt (server copy
  should still be removed), deleting a trail already removed from the web UI (404 -> local
  row disappears), and attempting the same delete in airplane mode (both copies survive,
  `delete_needs_connection` shown).

---
*Phase: 36-local-first-recording-automatic-upload*
*Completed: 2026-08-03*

## Self-Check: PASSED
