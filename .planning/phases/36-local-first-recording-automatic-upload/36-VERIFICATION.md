---
phase: 36-local-first-recording-automatic-upload
verified: 2026-08-03T21:15:00Z
status: human_needed
score: 11/11 requirements satisfied at the code level (6/6 roadmap success criteria hold; 3/3
  second-review CRITICAL blockers independently confirmed closed; 16/17 WARNING findings closed,
  1 (WR-06) deliberately deferred and tracked, not a code gap)
overrides_applied: 0
superseded_note: >
  This pass supersedes BOTH prior entries below. Pass 1 (2026-08-03T16:30:00Z) concluded
  human_needed with the four first-review CR fixes read as correct. Pass 2's re-review
  (36-REVIEW.md, later on 2026-08-03) found two of those four fixes were correct at the call
  site but wrong end-to-end, and opened three NEW blockers (CR-01/CR-02/CR-03, restated against
  the retirement design) plus 17 warnings — status was corrected to gaps_found on that evidence.
  This pass verifies the six gap-closure plans (36-15..36-20) that were executed against that
  gaps_found report. All three blockers were independently re-derived from current source (not
  trusted from SUMMARY.md), and two of the WR-05 "the gate would pass against an empty guard
  body" claims and two of the CR-01/drain-carry-forward falsification claims were independently
  reproduced by this verifier: the exact falsifying rewrites named in 36-20-SUMMARY.md were
  applied to the real source files, `flutter test` was run, observed to fail with the same
  assertion text the SUMMARY quotes, and the files were restored via `git checkout` before
  continuing. No BLOCKER-level code gap remains. Status is `human_needed`, not `passed`,
  because 36-UAT.md's round-2 device pass (5 items, all still `result: pending`) has not been
  re-run since these fixes landed, and one of those five items is a genuine UX judgment call
  (CR-03's narrowed offline-edit promise) that only a human can accept or reject.
re_verification:
  previous_status: gaps_found
  previous_score: "3 blockers open (CR-01, CR-02, CR-03), 17 warnings open"
  gaps_closed:
    - "CR-01 (post-upload edit permanently fails with a generic error) -- retireUploadedLocalTrail now returns the server id it kept (local_trail_store.dart:666), TrailSync memoizes it account-keyed (serverIdForRetired), resolveNetworkSaveTarget picks a real target before _saveViaNetwork is ever called, and a genuine refusal shows the actionable trail_uploaded_reopen_to_edit message instead of error_saving_trail. Confirmed by direct source read AND by independently re-running the falsifying rewrite the review named against the real file and observing the gate fail with the same assertion text 36-20-SUMMARY.md quotes."
    - "CR-02 (unparseable-GPX row skips the server DELETE and strands a live server trail) -- readLocalTrailServerId reads TrailEntity.id directly off the ObjectBox column, never through toModel()/parseGpxSafely, so a cached-GPX parse failure can no longer make deleteUnsynced treat a server-stamped row as if it had no server copy. Confirmed by direct source read of local_trail_store.dart:774-810 and its call site at trail_sync_provider.dart:522-579."
    - "CR-03 (an alreadyUploaded edit reaches the server but the stale local row keeps shadowing it under mergeOwnTrails' dedupe) -- applyNetworkEditToLocalRow reconciles the row's editable metadata columns (never id/owner/localId/syncState/photos/gpxData/waypoints) immediately after _saveViaNetwork adopts result.trail and strictly BEFORE _invalidateOwnTrailsList() runs, at both call sites that reach _saveViaNetwork. Confirmed by direct source read of local_trail_store.dart:557-589 and trail_create_screen.dart:748-795, and by reading own_trails_merge_test.dart's new case documenting the ordering dependency explicitly."
    - "WR-05 (the three gates added by the first fix pass assert token order only, and would pass against an empty guard body) -- 36-20 rewrote them to assert effects (a return; that fires, a specific message string, an outcome token inside the live if-condition) and added two new gates on what is passed to _saveViaNetwork. Independently re-verified: this verifier applied the review's exact named falsifying rewrites to trail_create_screen.dart and trail_sync_provider.dart, ran the specific gate test files, observed the same failures 36-20-SUMMARY.md quotes verbatim, then restored both files via git checkout (confirmed clean diff)."
    - "WR-01, WR-02, WR-04, WR-07, WR-08, WR-09, WR-10, WR-11, WR-12, WR-13, WR-14, WR-15, WR-16, WR-17 -- all independently confirmed present and correct in current source (see Anti-Patterns/Requirements sections below); WR-03 also confirmed via the removal of `late` from TrailFilterNotifier.defaultFilter (trail_filter_provider.dart:83)."
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Re-run 36-UAT.md round 2, Test 1: open an unsynced trail from the own-trails list (routes to the detail screen), open its dropdown menu -- check Download is absent (not disabled) and Delete says 'cannot be undone'. Start (or wait for) that trail's upload and reopen the menu mid-upload -- Delete should be disabled. Then check an ordinary downloaded trail's menu is unchanged."
    expected: "Detail screen opens (not the edit screen) with no Like button and an Edit button in place of Download/Navigate; the dropdown's Download entry is absent and Delete's confirmation states the deletion is unrecoverable; mid-upload the Delete entry is disabled; a downloaded trail's menu is unaffected."
    why_human: "36-UAT.md records this as round 2, Test 1, `result: pending` -- never run on device since 36-11/36-12 (routing) and 36-13 (behavioural widget coverage) landed. Source and a real widget test (trail_dropdown_menu_test.dart) are both read and confirmed correct in this pass, but no end-to-end device pass exists."
  - test: "Re-run 36-UAT.md round 2, Test 2: (a) sit on the offline own-trails list for a minute -- confirm no repeated full-screen spinner; (b) edit an unsynced trail's title offline, save, pop back -- confirm the list shows the new title with no manual pull-to-refresh; (c) tap an unsynced trail -- confirm it opens the detail screen, not the edit screen."
    expected: "No spinner flicker; edits appear immediately on pop-back; tapping opens the detail screen."
    why_human: "36-UAT.md records this as round 2, Test 2, `result: pending`. All three underlying fixes were re-confirmed present in source in this pass, but the only device pass predates them."
  - test: "Re-run 36-UAT.md round 2, Test 3: create a trail offline, go online, let it fully upload, then delete it. Separately: create a trail offline, let its create (PUT /trail/form) succeed but force a waypoint/photo upload to keep failing until the row parks as Failed (airplane mode after the first waypoint, or a bad photo), then delete that Failed trail and confirm on the server (or a second device/web) that it is actually gone -- not left live with a 'this can't be undone, it was never uploaded' message having been a lie."
    expected: "Case 1: no local row survives a full upload; deleting removes it cleanly, no orphan. Case 2: the Failed trail's Delete confirmation copy and its actual behavior both agree with whether it already has a server id -- if it does, deleting it must also remove the server-side record."
    why_human: "This is CR-02's fix (readLocalTrailServerId) plus WR-15's classified-DELETE-outcome fix, both confirmed correct by direct source read and by tracing every caller in this pass, but nothing in this repo can open a live ObjectBox Store or hit a real PocketBase server, so the actual server-side DELETE succeeding on a row whose cached GPX is corrupt has never been exercised end-to-end. 36-UAT.md records this as round 2, Test 3, `result: pending`."
  - test: "Edit an unsynced trail's title while its create has already reached the server but a waypoint upload is still retrying (chip reads Pending/Uploading/Failed but the create genuinely succeeded), while offline. Then do the same edit once the trail has fully synced and the row has been retired (chip has disappeared), from a stale screen instance still open on it. Judge whether the resulting behavior -- an error toast telling the hiker to reopen or reconnect, rather than a silent local write -- is acceptable UX for this specific sub-state, or should be raised as a follow-up to give the drain an update path instead."
    expected: "Both cases tell the hiker clearly the edit did not land (an error toast), never a false success toast, never a crash or data corruption. This is a REQUIRED HUMAN JUDGMENT CALL, not just a functional check: the fix is code-correct (confirmed in this pass) but deliberately narrows REC-05's literal 'edit an unsynced trail... while still offline' promise for this one sub-state, per CR-03's shipped 'minimum acceptable' shape (see local_trail_store.dart:130-144's doc comment and CR-03's original review text)."
    why_human: "36-UAT.md records this as round 2, Test 4, `result: pending`. The code was independently re-read and confirmed to fail loudly (not silently) in this pass, but whether the resulting UX narrowing is acceptable for this phase's stated goal is a product decision, not something grep or a test suite can answer."
  - test: "Foreground the app with a working connection and watch an unsynced trail upload with no user action; separately, kill the app mid-drain (after the trail record is created, before all waypoints finish) and relaunch/reconnect to confirm the drain resumes with no duplicate trail or waypoint on the server. While the trail is parked mid-drain, open it offline and confirm an EARLIER waypoint's photos still render even though a LATER waypoint's upload is what's still failing (WR-09); and confirm the chip reads Pending (not Failed) after several quick background/foreground cycles (WR-04)."
    expected: "Badge transitions Pending -> Uploading -> disappears with no tap; after an interrupted-and-resumed upload exactly one trail (and one row per waypoint) exists on the server, and the local row is gone (retire-on-success) with photos intact server-side; an earlier-succeeded waypoint's photos remain visible offline while a later one is still retrying; the trail does not park as Failed purely from rapid lifecycle cycling."
    why_human: "36-UAT.md records this as round 2, Test 5, `result: pending` -- still the only way to confirm SYNC-01/SYNC-04/SYNC-05's duplicate-prevention chain, plus 36-18's WR-04/WR-09 fixes, against a live server. Nothing in this repo can open a live ObjectBox Store or reach a real PocketBase server."
---

# Phase 36: Local-First Recording & Automatic Upload Verification Report

**Phase Goal:** A hiker who records a trail or uploads a GPX with no signal can save it, review it, and fill in its details on the spot — and it uploads itself the next time the phone has a connection, without the hiker doing anything.

**Verified:** 2026-08-03T21:15:00Z
**Status:** human_needed
**Re-verification:** Yes — after the six gap-closure plans (36-15..36-20) that resolved the
`gaps_found` verdict this file previously carried (three blockers, 17 warnings, per
`36-REVIEW.md`).

## Prior Pass History (preserved per the superseded-note convention)

<details>
<summary>Pass 1 (2026-08-03T16:30:00Z) — human_needed, later superseded</summary>

superseded_note (original): "This pass concluded no code-level gaps and confirmed the four CR
fixes as 'present and correct in current source'. A post-verification re-review (36-REVIEW.md,
2026-08-03T17:xx) established that two of those fixes are correct at the call site but wrong
end-to-end, and opened three new blockers. That conclusion is retained below for the record but
is NOT current -- see gaps_remaining. Status changed human_needed -> gaps_found on that
evidence."

Full body of that pass is preserved in this file's git history (see `git log -p` on this path
around commit range for 2026-08-03T16:30:00Z) rather than reproduced here a second time — its
conclusions were superseded twice over and reproducing them again would only add noise.

</details>

<details>
<summary>Pass 2 (2026-08-03, re-review) — gaps_found, three blockers (CR-01, CR-02, CR-03) + 17 warnings</summary>

Documented directly in `36-REVIEW.md` (reviewed 2026-08-03T17:40:00Z). Summary: the four
first-pass CR fixes (blank-id refusal, `_localId` ordering, `alreadyUploaded` outcome,
delete-decides-on-id) landed as described, but two (CR-01's refusal and CR-03's routing) were
"correct as a guard, wrong end-to-end" — the refusal fired on the phase's primary flow with no
recovery path, and the routed edit reached the server but never reconciled the stale local row.
CR-04's fix was also found reachable via an unparseable-GPX bypass (restated as CR-02). Full
detail in `36-REVIEW.md`, not reproduced here.

</details>

## This Pass: Verifying Plans 36-15..36-20 Against That gaps_found Report

Six plans executed sequentially, 28 commits, targeting the three restated blockers and all 17
warnings from `36-REVIEW.md`. This pass independently re-derives each claim from current source
— not from any plan's SUMMARY.md — and, for the two highest-risk claims (CR-01's fix and WR-05's
"the gates would now actually fail against their named falsifying rewrite"), reproduced the
falsification itself: applied the exact rewrite 36-20-SUMMARY.md names to the real source files,
ran the specific gate test files, confirmed the same failure output, then restored the files via
`git checkout` and confirmed a clean diff before continuing. This is stronger evidence than
reading the SUMMARY's claimed output, because it does not depend on trusting the executor's own
narration of what it observed.

## Goal Achievement

### The Three Blockers — Individual Verdicts

| ID | Truth | Verdict | Evidence |
|----|-------|---------|----------|
| **CR-01** | A hiker can edit a trail they recorded offline after it has uploaded, even from a screen that was already open when the upload completed | ✓ **VERIFIED — genuinely closed** | Traced the exact sequence the review named: `retireUploadedLocalTrail` now returns the server id it captured before mutating/removing the row (`local_trail_store.dart:666-701`); `_drainOne` captures that value into an account-keyed memo (`_rememberRetiredServerId`) and invalidates `localTrailProvider(localId)` in the same block (`trail_sync_provider.dart:419-426`); `trail_create_screen`'s `networkUpdate` branch resolves a real target via `resolveNetworkSaveTarget` (screen id, then the retired-id memo, then null) BEFORE ever calling `_saveViaNetwork` (`trail_create_screen.dart:492-525`); a genuine refusal shows `trail_uploaded_reopen_to_edit`, never the generic `error_saving_trail` (confirmed absent from the guarded slice); `_saveViaNetwork`'s `trailHasServerId` check is kept only as a last-resort backstop with a `debugPrint` (`:718-746`). **Independently falsified**: replaced the guard body with `{ /* TODO */ }` in the real file, ran `flutter test test/routes/trail_create_screen_local_save_gate_test.dart`, and got the exact failure the review's WR-05 finding describes ("must actually `return;` rather than fall through to the POST") — restored via `git checkout`, confirmed clean. |
| **CR-02** | Deleting a trail that already has a server record removes it from the server too, even when the row's cached GPX no longer parses | ✓ **VERIFIED — genuinely closed** | `readLocalTrailServerId` (`local_trail_store.dart:774-810`) reads `entity.id` directly off the ObjectBox column inside its own query, deliberately NOT via `readLocalTrail`/`toModel()`/`parseGpxSafely` — its own doc comment names CR-02 by number and explains why. `deleteUnsynced` (`trail_sync_provider.dart:522-579`) calls this reader, not `readLocalTrail`, so an unparseable cached GPX can no longer make the delete decision see `null` and skip the server DELETE. `deleteLocalTrailRow` only runs after the server DELETE succeeds, 404s (already gone), or the id fails safe via `recordIdDirSegment` (WR-17) — every other failure aborts before touching the local row (`resolveServerDeleteOutcome`, WR-15). |
| **CR-03** | An edit that reaches the server is what the hiker subsequently sees on the own-trails list, not a stale pre-edit row shadowing it | ✓ **VERIFIED — genuinely closed** | `applyNetworkEditToLocalRow` (`local_trail_store.dart:557-589`) writes only the hiker-editable metadata columns onto the owner-scoped row, deliberately never touching `id`/`owner`/`localId`/`syncState`/`photos`/`localPhotos`/`gpxData`/`waypoints` (verified directly against the function body, not merely its doc comment). `_saveViaNetwork` calls it immediately after `trail = result.trail` is adopted and **strictly before** `_invalidateOwnTrailsList()` runs (`trail_create_screen.dart:759-795`) — traced at BOTH call sites that reach `_saveViaNetwork` (the `networkUpdate` branch at `:527-538` and the `updateLocal`-branch `alreadyUploaded`/`alreadySynced`/`missing` escape at `:657-663`), confirming the ordering holds on every path, not just the happy one. `own_trails_merge.dart`'s dedupe (`mergeOwnTrails`, unchanged, still keys on non-empty local `id`) now sees a reconciled row when it re-reads, closing the exact window the review traced. **Independently falsified** the drain's carry-forward half of this chain (see WR-05 falsification below) and confirmed the new `own_trails_merge_test.dart` case documents the ordering dependency explicitly. |

**All three blockers are genuinely closed, not merely guarded.** Confirmed by direct source
reading at every call site the review traced, and by reproducing two of the falsification
claims myself rather than trusting the plan SUMMARYs' narration.

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Capturing with no connection saves immediately into the own-trails list, no offline save-failure ever shown, unsynced visibly distinct from synced AND downloaded | ✓ VERIFIED | Unchanged since the last pass; local-first branches (`createLocal`/`updateLocal`) never touch the network; `SyncStatusChip` renders nothing when `synced`, otherwise Pending/Uploading/Failed. |
| 2 | Survives app restart, stays tied to the capturing account; a different account never sees/uploads it; logout never deletes it | ✓ VERIFIED | Every read/write in `local_trail_store.dart` remains `owner`-scoped, including the two newly hardened paths (`readLocalTrailServerId`, `deleteLocalTrailRow`, `resetDrainBackoff`, `applyNetworkEditToLocalRow` — all four now take `accountId` and query on it, closing WR-10's account half). `account_data_purge_util.dart` still excludes trail/waypoint entities (unchanged). |
| 3 | Open/review/edit an unsynced trail's title, description, category, photos while offline | ⚠️ VERIFIED with the same narrowed edge case as the prior pass, now with a loud (not silent) failure confirmed correct | A trail whose *create* has reached the server but is not yet fully synced (`alreadyUploaded`) still refuses a local write and routes to the network, failing with an explicit toast if offline. This is the CR-03 fix's shipped "minimum acceptable" shape, confirmed intentional and confirmed loud (never silent data loss). It is a real, narrow behavior change against REC-05's literal wording for one sub-state — flagged for human judgment in `human_verification` item 4, not silently passed. |
| 4 | Once foregrounded with connection, uploads on its own with inline per-item progress, manual retry on failure/stall | ✓ VERIFIED | Unchanged; 3 trigger sites in `main.dart`; chip's `retry()` tap-through, now owner-scoped via `resetDrainBackoff(store, localId, accountId: accountId)` (`trail_sync_provider.dart:494`). |
| 5 | An interrupted upload never produces a duplicate trail on retry; once uploaded the trail becomes ordinary in place, keeping its identity | ✓ VERIFIED (code-level; needs device+server confirmation) | The retire-on-success design (unchanged since the last pass) still holds: `retireUploadedLocalTrail` deletes or demotes the row, `writeServerTrailId` stamps identity before retirement, re-entry is via the network fetch. All three blockers that threatened this guarantee are now genuinely closed (see table above), not merely guarded — a materially stronger claim than the prior pass could make. Still device+server-confirmation-pending (human_verification #5). |
| 6 | With no connection the own-trails list still renders, shows every not-yet-uploaded trail plus authored-and-downloaded trails, states it's offline-only | ✓ VERIFIED | Unchanged; `profile_trail_screen.dart`'s offline banner and `trailFilterProvider`'s device-derived fallback (`defaultFilter`, now non-`late`, closing WR-03) both confirmed present. |

**Score:** 6/6 roadmap success criteria hold at the code level; criterion 3 still carries the
same narrow, deliberate, honestly-failing (never silently data-losing) edge case, now confirmed
by this pass to be exactly what the review's "minimum acceptable" fix intended, not a new
defect; criterion 5's chain is now genuinely sound at the code level (all three blockers closed)
but has not been exercised against a live server since these six plans landed.

### Required Artifacts (gap-closure plans 36-15..36-20)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `local_trail_store.dart`'s `readLocalTrailServerId` | Parse-independent, owner-scoped delete decision | ✓ VERIFIED | `:774-810`; reads `entity.id` off the column, never `toModel()`. |
| `local_trail_store.dart`'s `resolveServerDeleteOutcome`/`trail_sync_provider.dart`'s `UnsyncedDeleteResult` | Classified DELETE outcomes, no unconditional throw | ✓ VERIFIED | `:224-243` (pure classifier); `deleteUnsynced` switches on it (`:566-576`), no `rethrow`. |
| `local_trail_store.dart`'s owner-scoped `deleteLocalTrailRow`/`resetDrainBackoff` | Account-scoped writes | ✓ VERIFIED | Both require `accountId`, both callers (`trail_sync_provider.dart:494,579`) pass a freshly-read one. |
| `recordIdDirSegment` on every server-id-consuming Dio path | Path-segment validation | ✓ VERIFIED | `trail_sync_provider.dart:546`, `trail_save_provider.dart:209`. |
| `trail_dropdown.dart`'s null-`localId` routing (`_deleteOnServer`) | No silent un-download fallthrough for a server-backed row | ✓ VERIFIED | `:292-318`; explicit branch, never falls through to `trail.isLocal`. |
| `local_trail_store.dart`'s `retireUploadedLocalTrail` (String? return) | Server id survives retirement | ✓ VERIFIED | `:666-701`; returns `serverId` from both exits, captured before mutation. |
| `local_trail_store.dart`'s `resolveNetworkSaveTarget` | Pure three-way save-target decision | ✓ VERIFIED | `:161-` region (screen id wins, retired-id fallback, null = refuse); called at `trail_create_screen.dart:497`. |
| `trail_sync_provider.dart`'s `_retiredServerIds`/`serverIdForRetired` | Account-scoped, bounded memo | ✓ VERIFIED | Confirmed present, account-checked per entry (independently falsified — see below). |
| `local_trail_store.dart`'s `applyNetworkEditToLocalRow` | Post-network-save local reconciliation | ✓ VERIFIED | `:557-589`; owner-scoped, writes only editable metadata, never identity/sync-bookkeeping/photo columns. |
| `local_trail_store.dart`'s `resolveLocalSaveModeForRow` | Routes `synced`/real-server-id rows to `networkUpdate` BEFORE any filesystem side effect | ✓ VERIFIED | `:130-144`; called at `trail_create_screen.dart:449`, before `_copyPhotosForLocalSave` is reachable for a doomed-to-refuse save (WR-14). |
| `local_photo_store.dart`'s `photosNotYetOnServer` | Location-based (not filename) diff excluding already-uploaded photos | ✓ VERIFIED | Confirmed pure, `p.isWithin`-based; wired at both `_saveViaNetwork` call sites via `networkPhotoPaths` (`trail_create_screen.dart:463-482`), never in the local-first branches. |
| `trail_filter_provider.dart`'s `defaultFilter` | Non-`late`, initialised at declaration | ✓ VERIFIED | `:83`; `TrailFilter defaultFilter = buildDefaultTrailFilter(kOfflineTrailFilterValues);` — no uninitialised state remains. |
| `local_trail_store.dart`'s `hasKeylessPendingWaypoint` | Pure invariant-break detector, checked before the drain's in-flight join | ✓ VERIFIED | `:272-`; called at `trail_sync_provider.dart:255-266`, before `state = {...state, localId}` and before the `try`. |
| `local_trail_store.dart`'s `writeServerWaypointId` | Retains `localPhotos` across the waypoint's own upload success | ✓ VERIFIED | `:1012-1016`; `localPhotos` no longer cleared, doc comment confirms deliberate. |
| `trail_panel.dart`'s `showsServerTabs` | Single predicate governs `TabBar`/`DefaultTabController.length`/`_TabContent.children` | ✓ VERIFIED | `:71,165,344,360`; gated on `!isUnsyncedState(trail.syncState)`, not the cache-provenance `isLocal` flag; controller length and content list structurally cannot desync. |
| `app/test/routes/trail_create_screen_local_save_gate_test.dart` (WR-05 rewrite) | Effect-asserting gates, not token-order-only | ✓ VERIFIED — independently falsified by this verifier | See "WR-05 Independent Falsification" below. |
| `app/test/store/local_trail_retirement_gate_test.dart` (WR-05 extension) | Effect assertions on the retired-id carry-forward and its account scoping | ✓ VERIFIED — independently falsified by this verifier | See below. |
| `app/test/routes/trail_detail_screen_retired_redirect_test.dart` | Real behavioural widget test (real GoRouter, real TrailDetailScreen) for the WR-01 redirect | ✓ VERIFIED | Read directly: mounts a real `GoRouter`+`TrailDetailScreen`, asserts on `router.state.uri` and rendered text, not a source-slicing gate. Its third case is honestly self-limited (documents in its own header why the `trail != null` branch cannot be mounted in this environment) rather than silently passing something weaker. |
| `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md` (WR-06) | Tracked deferral, not silent debt | ✓ VERIFIED as an accepted, documented deferral | Exists, dated, itemizes all 22 backlog keys in priority order, explains why machine translation was refused. Not a code gap — see Anti-Patterns. |

### WR-05 Independent Falsification (this verifier's own reproduction, not SUMMARY narration)

Per the verification priorities, this pass did not accept 36-20-SUMMARY.md's falsification
claims on narration alone. Two of the five falsifications it records were independently
reproduced against the real, currently-committed source:

1. **CR-01 gate** (`trail_create_screen_local_save_gate_test.dart`): replaced
   `_saveViaNetwork`'s blank-id guard body with `{ /* TODO */ }` in
   `app/lib/routes/trail_create_screen.dart` (a real edit to the real file, not a copy). Ran
   `flutter test test/routes/trail_create_screen_local_save_gate_test.dart`. Result: **the gate
   failed**, with the exact assertion text the SUMMARY quotes ("The blank-id guard must actually
   `return;` rather than fall through to the POST below it..."). Restored via `git checkout --`;
   confirmed `git status --short` empty afterward.
2. **Drain carry-forward gates** (`local_trail_retirement_gate_test.dart`): replaced
   `final retiredServerId = retireUploadedLocalTrail(store, localId); if (retiredServerId !=
   null) { _rememberRetiredServerId(...); }` with a bare `retireUploadedLocalTrail(store,
   localId);` in `app/lib/provider/trail/trail_sync_provider.dart`. Ran `flutter test
   test/store/local_trail_retirement_gate_test.dart`. Result: **two gates failed** ("assigns
   `retireUploadedLocalTrail`'s return value" and "`_rememberRetiredServerId(` runs after the
   retirement return value is captured"), matching the SUMMARY's claimed output. Restored via
   `git checkout --`; confirmed clean.

Both reproductions match 36-20-SUMMARY.md's claims exactly. Combined with a full independent run
of `flutter analyze --no-pub` (0 errors, 36 pre-existing info lints — matches the claimed
baseline) and `flutter test` (941 passing, 1 skipped, 0 failures — matches the claimed count),
this pass has substantially higher confidence in the WR-05 closure than reading the SUMMARY
alone would provide.

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `trail_create_screen.dart`'s `networkUpdate` branch | `resolveNetworkSaveTarget`/`serverIdForRetired` | resolve-before-call | ✓ WIRED | `:492-525`; the raw `updatedTrail.id` is never handed to `_saveViaNetwork` — always `updatedTrail.copyWith(id: targetId, ...)`. |
| `trail_sync_provider.dart`'s retirement step | `_rememberRetiredServerId` / `localTrailProvider` invalidation | capture-then-remember-then-invalidate | ✓ WIRED | `:419-426`; ordering independently falsified (see above) and confirmed load-bearing. |
| `_saveViaNetwork` | `applyNetworkEditToLocalRow` → `_invalidateOwnTrailsList` | reconcile-then-invalidate | ✓ WIRED | `:766-795`; confirmed at both call sites reaching `_saveViaNetwork`. |
| `trail_sync_provider.dart`'s `deleteUnsynced` | `readLocalTrailServerId` → `recordIdDirSegment` → server DELETE → `resolveServerDeleteOutcome` → `deleteLocalTrailRow` | classify-then-act | ✓ WIRED | `:522-594`; every branch traced, `readLocalTrail`/`toModel()` never appears in this function. |
| `trail_create_screen.dart`'s `resolveLocalSaveModeForRow` | `_copyPhotosForLocalSave` | route-before-side-effect | ✓ WIRED | `:449-482`; an `alreadyUploaded`/`synced` row never reaches the `updateLocal` branch's photo copy (WR-14). |
| `local_photo_store.dart`'s `photosNotYetOnServer` | `_saveViaNetwork`'s `newPhotoFiles` | filter-before-upload | ✓ WIRED | `:463-482`; unused by the local-first branches (confirmed by 36-20's new negative gate and by direct read). |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `app/lib/i18n/*.arb` (13 non-English locales) | — | WR-06: destructive-action and sync-status strings remain English-only in 13 locales | ⚠️ Warning, deliberately deferred | Correctness half resolved (`_confirmDelete` now keys on `trailHasServerId`); coverage half tracked in `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md`, prioritized, not silently dropped. Non-blocking per the app's documented English-fallback behavior. This is the ONE item from `36-REVIEW.md`'s 17 warnings still open, and it is open by deliberate, recorded decision, not oversight. |
| No `TBD`/`FIXME`/`XXX` debt markers | — | Scanned every file touched by 36-15..36-20 (`local_trail_store.dart`, `trail_sync_provider.dart`, `trail_create_screen.dart`, `trail_detail_screen.dart`, `trail_dropdown.dart`, `trail_panel.dart`, `trail_filter_provider.dart`, `local_photo_store.dart`) | — | Clean. |

No stub returns, no empty-handler patterns, and no hardcoded-empty data reaching rendered UI
were found in any file this pass examined.

### Requirements Coverage

| Requirement | Status | Evidence |
|-------------|--------|----------|
| REC-01 | ✓ SATISFIED | Unchanged; local-first branches confirmed network-free. |
| REC-02 | ✓ SATISFIED | Unchanged; `/trail/local/:localId` addressing intact. |
| REC-03 | ✓ SATISFIED | `SyncStatusChip` unchanged; `TrailPanel`'s tab gate now correctly keyed on sync state (WR-11 closed this pass). |
| REC-04 | ✓ SATISFIED | Unchanged; account-scoping strengthened further this pass (WR-10's remaining half closed). |
| REC-05 | ⚠️ SATISFIED WITH THE SAME NARROWED EDGE CASE, now confirmed intentional and loud-not-silent | See Observable Truth 3. |
| REC-06 | ✓ SATISFIED | Unchanged. |
| SYNC-01 | ✓ SATISFIED | Unchanged. |
| SYNC-02 | ✓ SATISFIED | Unchanged; dropdown gating behaviourally tested. |
| SYNC-03 | ✓ SATISFIED | Unchanged; WR-04 fix (36-18) prevents a healthy trail parking as `failed` from an invariant-break waypoint. |
| SYNC-04 | ✓ SATISFIED (code-level, all three blockers now closed) | See blocker table above; device+server confirmation still pending. |
| SYNC-05 | ✓ SATISFIED | Redesigned (retire/delete) guarantee still holds by construction; all three blockers that threatened it are closed. |

No orphaned requirements — all 11 IDs assigned to Phase 36 in `.planning/REQUIREMENTS.md` are
claimed by at least one plan across the full 20-plan set.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full test suite passes | `flutter test` (independently re-run by this verifier, not relied on from the orchestrator) | 941 passing, 1 skipped, 0 failures | ✓ PASS |
| Analyzer clean | `flutter analyze --no-pub` (independently re-run) | 0 errors, 36 pre-existing `info` lints | ✓ PASS |
| CR-01 gate fails against its named falsifying rewrite | Applied rewrite to real source, ran gate test, restored | Failed with the exact assertion text claimed | ✓ PASS (independently reproduced) |
| Drain carry-forward gates fail against their named falsifying rewrite | Applied rewrite to real source, ran gate test, restored | Both failed with the exact assertion text claimed | ✓ PASS (independently reproduced) |
| Working tree clean after both falsification reproductions | `git status --short` | Empty | ✓ PASS |
| No debt markers in phase-touched files | `grep -rn TBD\|FIXME\|XXX` on 8 core files | None found | ✓ PASS |
| Live device/server upload, delete-with-server-copy, dropdown gating, narrowed-edit UX judgment | — | — | ? SKIP — see human_verification |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this project layout (Flutter/Dart). SKIPPED —
no conventional or PLAN-declared probes found.

### Human Verification Required

See frontmatter `human_verification` for the structured list. Five items, all mirroring
`36-UAT.md` round 2's still-`pending` tests, restated against this pass's confirmed code state:

1. Dropdown gating for an unsynced trail (36-UAT round 2, Test 1) — code and a real widget test
   confirmed correct, never run on device.
2. Spinner-flicker / edit-not-reflected / tap-routes-to-detail (36-UAT round 2, Test 2) — all
   three fixes confirmed in source, none re-confirmed on device.
3. Delete-after-sync and delete-a-Failed-trail-with-a-server-id (36-UAT round 2, Test 3) —
   CR-02/WR-15's fix confirmed correct in source; server-side DELETE has never been exercised
   end-to-end.
4. **Judgment call, not just a functional check:** the CR-03 "minimum acceptable" fix's narrowed
   offline-edit behavior (36-UAT round 2, Test 4) — code confirmed to fail loudly rather than
   silently, but whether that narrowing is acceptable UX for this phase's stated goal needs a
   human decision.
5. Interrupted-upload/no-duplicate device+server pass, extended this pass to also cover WR-04
   (a healthy trail should not park as Failed from rapid lifecycle cycling) and WR-09 (an
   earlier-succeeded waypoint's photos should stay visible while a later one is still retrying)
   (36-UAT round 2, Test 5).

### Gaps Summary

**No BLOCKER-level gaps remain against the codebase as it stands.** All three CRITICAL findings
from `36-REVIEW.md` (CR-01, CR-02, CR-03) were independently re-derived from current source in
this pass — not trusted from any plan's SUMMARY.md — and two of the highest-risk claims (the
CR-01 guard's actual effect, and the drain's carry-forward ordering) were proven by this verifier
personally reproducing the named falsifying rewrite against the real source files and observing
the same failure the executor claimed, then restoring a clean tree. Of the 17 WARNING findings,
16 are closed and independently confirmed; the 17th (WR-06, translation coverage) is an
explicit, tracked, non-blocking deferral with a durable todo record, not a code gap or an
oversight.

The phase's remaining risk is exactly what six plans' own `<verification>` notes and this
report's `human_verification` section say it is: nothing in this repository can open a live
ObjectBox `Store` or reach a real PocketBase server, so the code-level argument — however
thoroughly re-derived and, in two cases, personally reproduced by this verifier — has not been
exercised end-to-end since the retirement/delete/reconcile design was finalized by these six
plans. `36-UAT.md` round 2 records all five of its tests as `result: pending`. That is
`human_needed`, not a code gap, and one of those five items (CR-03's UX narrowing) is a genuine
product decision that no amount of source-reading can resolve on its own. ROADMAP.md's own
phase-36 line states the phase is "NOT complete until [gaps] are closed and the verifier
passes" — the gaps are now closed; the verifier's remaining ask is the device+server pass and
the one judgment call this file surfaces.

---

_Verified: 2026-08-03T21:15:00Z_
_Verifier: Claude (gsd-verifier)_
