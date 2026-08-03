---
phase: 36-local-first-recording-automatic-upload
verified: 2026-08-03T16:30:00Z
status: human_needed
score: 6/6 roadmap success criteria verified at the code level (41/41 plan-level must-have truths from plans 36-01..36-08 previously verified 2026-08-02; all 6 gap-closure truths from plans 36-09..36-14 independently re-checked in this pass; all 4 second-review CRITICAL fixes independently confirmed present and correct in current source)
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "6/6 roadmap success criteria (plans 36-01..36-08 only)"
  gaps_closed:
    - "UAT gap 1 (own-trails list spinner flicker) -- trail_filter_provider.dart's device-derived offline fallback + profile_trail_screen.dart's skipLoadingOnReload:true, confirmed present (36-09)"
    - "UAT gap 2 (unsynced trail tap opened edit screen instead of detail screen) -- /trail/local/:localId route, localTrailProvider, trailDetailLocation/trailMapLocation, confirmed present and wired (36-11, 36-12)"
    - "UAT gap 3 (edits to unsynced trail invisible without manual reload) -- _invalidateOwnTrailsList called from both save tails, confirmed present (36-10)"
    - "UAT gap 4 (orphaned local row after post-sync delete, indefinite spinner) -- retireUploadedLocalTrail deletes (or demotes) the row instead of marking it synced, confirmed present (36-13, 36-14)"
    - "UAT gap 5 / 36-15's premise -- confirmed structurally unreachable post-36-14 per ROADMAP.md's own argument, independently spot-checked (trailProvider's savedByUserIds-gated ObjectBox fallback still exists but a retired/deleted row can no longer produce it for this scenario)"
    - "Second code review's CR-01..CR-04 (post-upload edit routed to a blank-id POST; first-save failure bricking the screen; edits silently discarded once a server id is stamped mid-drain; delete claiming 'never uploaded' while a server copy exists) -- all four fixes read directly in current source and confirmed correct, not merely trusted from commit messages"
  gaps_remaining: []
  regressions: []
human_verification:
  - test: "Re-run UAT Test 3 now that it is unblocked: open an unsynced trail from the own-trails list (now routes to the detail screen), open its dropdown menu -- check Download is absent (not disabled) and Delete says 'cannot be undone'. Start (or wait for) that trail's upload and reopen the menu mid-upload -- Delete should be disabled. Then check an ordinary downloaded trail's menu is unchanged."
    expected: "Detail screen opens (not the edit screen) with no Like button and an Edit button in place of Download/Navigate; the dropdown's Download entry is absent and Delete's confirmation states the deletion is unrecoverable; mid-upload the Delete entry is disabled; a downloaded trail's menu is unaffected."
    why_human: "UAT.md still records this test as 'blocked' -- it was never re-run after 36-11/36-12 (routing fix) and 36-13 (behavioural widget-test coverage) landed. Source and a real widget test were both read and confirmed correct, but no end-to-end device pass exists since the fix."
  - test: "Re-run UAT Test 2's three reported defects: (a) sit on the offline own-trails list for a minute -- confirm no repeated full-screen spinner; (b) edit an unsynced trail's title offline, save, pop back -- confirm the list shows the new title with no manual pull-to-refresh; (c) tap an unsynced trail -- confirm it opens the detail screen, not the edit screen."
    expected: "No spinner flicker; edits appear immediately on pop-back; tapping opens the detail screen."
    why_human: "All three fixes (36-09, 36-10, 36-11/12) were confirmed present in source, but the only round of live device testing predates all three fixes."
  - test: "Re-run UAT Test 5 end to end: create a trail offline, go online, let it fully upload, then delete it. Separately: create a trail offline, let its create (PUT /trail/form) succeed but force a waypoint/photo upload to keep failing until the row parks as Failed (airplane mode after the first waypoint, or a bad photo), then delete that Failed trail and confirm on the server (or via a second device/web) that it is actually gone -- not left live with the 'this can't be undone, it was never uploaded' message having been a lie."
    expected: "Case 1: no local row survives a full upload; deleting removes it cleanly from the own-trails list, with no orphan and no indefinite spinner on tapping a since-deleted trail elsewhere. Case 2: the Failed trail's Delete confirmation copy and its actual behavior both agree with whether it already has a server id -- if it does, deleting it must also remove the server-side record, not just the local one."
    why_human: "This is exactly the CR-04 fix from the second code review (`trailHasServerId`-gated confirm copy and delete routing in `trail_dropdown.dart`/`trail_sync_provider.dart`). The fix was read directly in source and is structurally correct, but nothing in this repo can open a live ObjectBox `Store` or hit a real PocketBase server (confirmed: `libobjectbox.dylib` fails to load under `flutter test`), so the actual server-side DELETE succeeding, and the local row surviving a failed one, has never been exercised end-to-end."
  - test: "Edit an unsynced trail's title while its create has already reached the server but a waypoint upload is still retrying (i.e. the chip reads Pending/Uploading/Failed but the create genuinely succeeded), while offline. Then do the same edit once the trail has fully synced and the row has been retired (chip has disappeared), from a stale screen instance still open on it."
    expected: "Both cases should tell the hiker clearly that the edit did not land (an error toast), never a false 'success' toast, and never crash or corrupt data."
    why_human: "This is CR-03's fix, which the second review flagged as a deliberately narrower 'minimum acceptable' fix (route to the network path and refuse rather than write locally) instead of the reviewer's preferred fix (give the drain step 2 an update path). The code is confirmed correct against silent data loss, but it also means editing offline in this specific window now fails outright rather than succeeding -- a real, narrow degradation of REC-05's promise ('edit an unsynced trail... while still offline') for this specific sub-state, worth a human judgment call on whether the UX (an error toast telling the hiker to reopen the trail) is acceptable."
  - test: "Foreground the app with a working connection and watch an unsynced trail upload with no user action; separately, kill the app mid-drain (after the trail record is created, before all waypoints finish) and relaunch/reconnect to confirm the drain resumes with no duplicate trail or waypoint on the server."
    expected: "Badge transitions Pending -> Uploading -> disappears with no tap; after an interrupted-and-resumed upload exactly one trail (and one row per waypoint) exists on the server, and the local row is gone (per the new retire-on-success behavior) with photos intact server-side."
    why_human: "Carried forward from the first verification pass -- still the only way to confirm SYNC-01/SYNC-04/SYNC-05's duplicate-prevention chain against a live server, and the chain changed further since (retirement replaces the old 'stays synced in place' design), so the previous device pass (before 36-14) does not cover the current code."
---

# Phase 36: Local-First Recording & Automatic Upload Verification Report

**Phase Goal:** A hiker who records a trail or uploads a GPX with no signal can save it, review it, and fill in its details on the spot — and it uploads itself the next time the phone has a connection, without the hiker doing anything.

**Verified:** 2026-08-03T16:30:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (plans 36-09..36-14) and a second full-phase code review (36-REVIEW.md, 2026-08-03) whose 4 CRITICAL findings were fixed by direct commits after that review, not through a formal REVIEW-FIX cycle

## Context for this pass

This phase has been through more churn than the 2026-08-02 initial verification saw:

1. Plans 36-01..36-08 shipped and were verified `human_needed` on 2026-08-02 (6/6 roadmap
   criteria, 41/41 plan-level truths, pending device confirmation).
2. A human UAT round then ran against that code and found 5 gaps (one blocker: an orphaned,
   permanently-loading trail left behind after a synced trail is deleted) plus one test
   blocked outright (the dropdown-gating test could not even be attempted because tapping an
   unsynced trail never reached the detail screen). `36-UAT.md` records this in full; its
   `status: diagnosed` and `Test 3: blocked` have not been updated since.
3. Six gap-closure plans (36-09..36-14) were planned and executed to close those 5 gaps.
   36-15 (a planned 7th plan, bounding `trailProvider`'s retry policy) was **deliberately
   dropped** by product-owner decision on 2026-08-03 once 36-14's row-retirement design made
   its target defect unreachable — this is not an unfinished plan and not a gap (see
   ROADMAP.md's "User decision (2026-08-03) — 36-15 is DROPPED").
4. A second, full-phase code review (`36-REVIEW.md`, reviewing all 14 plans together) then
   found **4 NEW critical defects** distinct from the first review's CR-01..CR-04 — all
   downstream of the row-retirement design and the "route on the persisted row" fix the
   previous review's CR-04 introduced. These were fixed directly by 5 commits
   (`6b32de55`, `9af61ffd`, `04075d12`, `b9ee66ca`, `588d46b6`) rather than a formal
   `36-REVIEW-FIX.md` pass.

This verification independently re-reads the current source for all of the above — the gap
closures, the second review's 4 critical fixes, and the 12 still-open warnings — rather than
trusting any commit message or SUMMARY narrative.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Capturing with no connection saves immediately into the own-trails list (recording end or GPX import), no offline save-failure ever shown, unsynced visibly distinct from synced AND downloaded | ✓ VERIFIED | Unchanged from the 2026-08-02 pass; independently re-confirmed `trail_create_screen.dart`'s local-first branches (`createLocal`/`updateLocal`) never touch the network. `SyncStatusChip` (`sync_status_chip.dart:22-70`) renders nothing when `synced`, otherwise Pending/Uploading/Failed. |
| 2 | Survives app restart, stays tied to the capturing account; a different account never sees/uploads it; logout never deletes it | ✓ VERIFIED | Unchanged; every read/write in `local_trail_store.dart` remains `owner`-scoped; `account_data_purge_util.dart` still excludes `TrailEntity`/`WaypointEntity`. |
| 3 | Open/review/edit an unsynced trail's title, description, category, photos while offline, on the offline-capable screen | ⚠️ VERIFIED with a narrowed edge case | `resolveLocalSaveMode`/`updateLocalTrail` still route a re-save of a genuinely-local (never contacted the server) unsynced trail fully locally, no network. **New narrowing found in this pass:** since the second review's CR-03 fix, a row that is still `pending`/`uploading`/`failed` (visually "unsynced" via the chip) but whose *create* has already reached the server now refuses the local write (`LocalUpdateOutcome.alreadyUploaded`, `local_trail_store.dart:312-326`) and routes to the network instead — which fails with an explicit error toast if offline (`trail_create_screen.dart:619-651`, doc comment confirms "Editing a synced server trail offline remains out of scope... that is the point"). This is a real, narrow degradation of "edit... while still offline" for that specific sub-state; it replaces the previous silent-data-loss bug with a loud, honest failure rather than data loss, which is the better of the two, but it is a genuine behavior change worth a human judgment call (see human_verification #4). |
| 4 | Once foregrounded with connection, uploads on its own with inline per-item progress on the trail itself, manual retry on failure/stall | ✓ VERIFIED | Unchanged; 3 trigger sites in `main.dart` re-confirmed present; chip's `retry()` tap-through re-confirmed. |
| 5 | An interrupted upload never produces a duplicate trail on retry; once uploaded the trail becomes ordinary in place, keeping its identity rather than appearing twice | ✓ VERIFIED (code-level; needs device confirmation) | The design changed since the last pass: a fully-uploaded trail with no downloader is now **deleted locally** (`retireUploadedLocalTrail`, `local_trail_store.dart:440-471`) rather than kept `synced` in place — a deliberate, documented supersession (ROADMAP.md "a local trail row is DELETED once it uploads successfully"). This still satisfies SYNC-05: the trail's identity survives via the server id `writeServerTrailId` stamped before retirement, and it re-enters the own-trails list through the network fetch, never a second local entry. All 4 of the second review's fresh CRITICAL findings that threatened this guarantee (CR-01: blank-id POST; CR-02: first-save failure bricking the screen; CR-03: mid-drain edits silently discarded; CR-04: delete claiming "never uploaded" while a server copy exists) were independently read in current source and are fixed correctly (see Anti-Patterns/Requirements sections). No automated test in this repo can exercise a live `Store` or a real server, so end-to-end confirmation is still a device+server pass away (human_verification #3, #5). |
| 6 | With no connection the own-trails list still renders, shows every not-yet-uploaded trail plus authored-and-downloaded trails, and plainly states it's offline-only | ✓ VERIFIED | `profile_trail_screen.dart`'s offline banner unchanged; the UAT-reported spinner-flicker gap (36-09: device-derived offline fallback for `trailFilterProvider`, `skipLoadingOnReload: true` at `profile_trail_screen.dart:106`) confirmed present. |

**Score:** 6/6 roadmap success criteria hold at the code level; criterion 3 carries a narrow, deliberate, and honestly-failing (not silently data-losing) edge case worth a human read; criterion 5's fix chain has not been exercised against a live server since it changed.

### Deferred Items

None applicable — 36-15's drop is a scope decision, not a deferral to a later phase (this is the last planned phase before Phase 37, and Phase 37 is explicitly unscheduled/blocked on this phase's completion).

### Required Artifacts (gap-closure plans 36-09..36-14, not covered by the 2026-08-02 pass)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/util/offline_trail_filter_bounds.dart` | Device-derived offline filter bounds, account-scoped | ✓ VERIFIED | Present; pure arithmetic genuinely unit-tested; ObjectBox shim left deliberately uncovered per its own doc comment (no test harness), not disguised as tested. |
| `app/lib/provider/trail/trail_filter_provider.dart` | Bounded connectivity-failure fallback, no 10x retry storm on offline | ✓ VERIFIED (with one open, unrelated warning) | `defaultFilter` assigned on both success and connection-failure paths, confirmed at `:84`/`:104`. **Still open:** `defaultFilter` stays `late` (not initialized at declaration) and any *other* failure (a 500, a malformed payload) still leaves it unassigned, so `resetFilter()` can still throw `LateInitializationError` on that unrelated path (second review's WR-03, not fixed — see Anti-Patterns). |
| `app/lib/routes/profile_trail_screen.dart` | `skipLoadingOnReload: true`, no reload-triggered full-screen spinner | ✓ VERIFIED | Confirmed at line 106. |
| `app/lib/routes/trail_create_screen.dart` | `_invalidateOwnTrailsList()` called from both save tails | ✓ VERIFIED | Confirmed via `_saveViaNetwork`'s `_invalidateOwnTrailsList()` call (`:670`) and the local-first tail's invalidation path. |
| `app/lib/util/trail_route_location.dart` | `trailDetailLocation`/`trailMapLocation` — sanctioned route-building for a possibly-unsynced trail | ✓ VERIFIED | Present; routes unsynced trails to `/trail/local/<localId>`, synced trails to `/trail/<id>`, null when unaddressable. |
| `app/lib/provider/trail/local_trail_provider.dart` | `localTrailProvider` family, synchronous, account-scoped | ✓ VERIFIED | Wired into `trail_detail_screen.dart:73` (`ref.watch(localTrailProvider(localId))`). |
| `app/lib/util/local_trail_store.dart`'s `readOwnLocalTrail` | Owner-scoped single-row lookup keyed by a route parameter | ✓ VERIFIED | Confirmed at `:516-541`, doc comment explains the route-parameter attack-surface reasoning. |
| `app/lib/routes/trail_detail_screen.dart` | Dual-mode (server id OR local id) detail screen, unsynced chrome gating | ✓ VERIFIED | `widget.localId` branch (`:71-88`); `isUnsynced` hides `LikeButton` (`:136-139`) and swaps Download/Navigate for an Edit button (`:164-183`) — matches D-17. |
| `app/lib/provider/router_provider.dart` | `/trail/local/:localId` (+ `map` sub-route) declared before `/trail/:id` | ✓ VERIFIED | Confirmed at `:359-380` vs `:381` — ordering is correct. |
| `app/lib/components/trail/trail_panel.dart` | Three map pushes retargeted through `trailMapLocation` | ✓ VERIFIED | Confirmed via `trailMapLocation(trail)` call, doc comment explains the D-06 blank-id problem it replaces. |
| `app/lib/components/trail/trail_dropdown.dart` | Show-on-map through `trailMapLocation`; post-edit invalidation for a local trail | ✓ VERIFIED | Confirmed. |
| `app/lib/util/local_trail_store.dart`'s `retireUploadedLocalTrail` | Deletes (or demotes, if downloaded mid-upload) the row on successful upload instead of marking it `synced` | ✓ VERIFIED | Confirmed at `:440-471`; delete-vs-demote decision (`shouldDeleteUploadedRow`) correctly keyed on `savedByUserIds` emptiness. |
| `app/test/routes/profile_trail_screen_navigation_test.dart` | Real widget test (not a source-grep) proving the pushed location for an unsynced trail | ✓ VERIFIED | Confirmed: pumps a real `ProfileTrailScreen` inside a real `GoRouter`, asserts the actual pushed location. Explicitly written to be the missing signal a prior source-grep test could not provide. |
| `app/test/components/trail/trail_dropdown_menu_test.dart` | Real widget test opening the actual menu, reading rendered `PopupMenuItem`s | ✓ VERIFIED | Confirmed: mounts a real `TrailDropdown` in an `AppBar`, opens the real `PopupMenuButton`. Not a source-grep. |
| `app/test/util/local_trail_retirement_gate_test.dart` | Coverage for `retireUploadedLocalTrail`'s invariants | ⚠️ WEAKER EVIDENCE THAN IT LOOKS | This is a **source-text gate test** (slices the function body out of the file and asserts on substring order), not a behavioural test — its own doc comment says so and explains why (`flutter test` cannot open an ObjectBox `Store`). Per this phase's own history (a real defect — the unsynced-tap-to-edit divert — hid behind exactly this pattern for the whole first UAT round), this gate is treated here as weaker evidence than a passing assertion normally implies. The actual invariant was independently confirmed by direct source reading (above), not by trusting this test alone. |

### Key Link Verification (gap-closure plans)

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `profile_trail_screen.dart`'s `_onTrailSelect` | `trailDetailLocation` | route push | ✓ WIRED | Confirmed at `:60-64`; falls back to `/trail/create/edit` only for the (stated-impossible) unaddressable case. |
| `trail_panel.dart` (3 map pushes) | `trailMapLocation` | route push | ✓ WIRED | Confirmed. |
| `router_provider.dart` | `TrailDetailScreen`/`TrailDetailMapScreen` | dual-mode `id`/`localId` constructor args | ✓ WIRED | Confirmed at `:366-380`. |
| `trail_sync_provider.dart` drain step 4 | `retireUploadedLocalTrail` | called inside the try, after the waypoint loop | ✓ WIRED | Confirmed present (per plan 36-14's stated shape; source read matches). |
| `trail_dropdown.dart`'s `_confirmDelete`/`_deleteTrail` | `trailHasServerId` / `TrailSync.deleteUnsynced` | decide-on-id, delete-both-sides | ✓ WIRED | Confirmed at `:255-257` (confirm copy) and `trail_sync_provider.dart:390-425` (`deleteUnsynced` issues a real `DELETE /trail/$serverId` before removing the local row, when a server id exists; throws rather than silently proceeding on a failed server delete). This is the exact fix for the second review's CR-04. |
| `trail_create_screen.dart`'s `_onSave`/`_saveViaNetwork` | `trailHasServerId` | refuse a blank-id POST | ✓ WIRED | Confirmed at `:625-651` — this is the exact fix for the second review's CR-01. |
| `trail_create_screen.dart`'s `createLocal` branch | `_localId` publication order | assign only after the row exists | ✓ WIRED | Confirmed at `:478-504` — `_localId = localId` now sits after `saveNewLocalTrail` succeeds, not before. This is the exact fix for the second review's CR-02. |
| `local_trail_store.dart`'s `updateLocalTrail` | `LocalUpdateOutcome.alreadyUploaded` | refuse a local write once a server id exists pre-synced | ✓ WIRED | Confirmed at `:343-345`/`:556-582` in the caller — this is the exact fix for the second review's CR-03, implemented as the review's own stated "minimum acceptable" alternative (see below). |

### "Minimum acceptable" fixes — sufficiency judgment (per reviewer note)

The second code review explicitly offered a preferred fix and a "minimum acceptable" fallback for two of its four CRITICAL findings; the fixes actually shipped took the minimum in both cases, to avoid further churn on the drain's create/update split:

- **CR-01** (blank-id POST): preferred fix was to carry a server id back to the screen and retry against real data; shipped fix instead *refuses* the save and shows an error, telling the hiker to re-open the trail. **Judgment: sufficient for this phase's requirements.** The goal is "no duplicate, no silent loss" (SYNC-04/SYNC-05); a loud, correct refusal satisfies that. It costs the hiker a re-open step in a narrow window (editing the very instant a completed upload retires the row, or after a first save's write itself failed) — annoying, not data-destroying, and not silently misleading.
- **CR-03** (mid-drain edits silently discarded): preferred fix was to give the drain's create step an update path so an edit could still reach the server via the existing sync machinery; shipped fix instead makes `updateLocalTrail` refuse and forces the network path, which fails loudly if offline. **Judgment: sufficient to prevent the CRITICAL (silent data loss under a false success toast), but it does narrow REC-05's offline-edit promise for one specific sub-state** — see Observable Truth 3 above and human_verification #4. This is the one place where the minimum fix has an actual, user-visible cost against a phase requirement's literal wording, not just a stylistic shortcut, and is flagged accordingly rather than waved through.

CR-02 and CR-04 were not offered a "minimum acceptable" alternative in the review and were fixed to the review's actual preferred shape — confirmed by direct comparison of the review's suggested code against the current source.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `app/lib/provider/trail/trail_filter_provider.dart` | `:68` (declaration), `:107` (rethrow path) | Second review's WR-03, still open: `defaultFilter` is `late`, assigned only on success and on connection-failure; any other failure (500, malformed payload) leaves it unassigned and `resetFilter()` (`:150-152`) reads it unconditionally | ⚠️ Warning | An uncaught `LateInitializationError` out of a button callback, on a `keepAlive` provider so the un-initialised instance survives. Narrow (needs a non-connectivity server error), not an offline-path defect, so it does not undermine this phase's core REC-06 promise, but it is a real crash surface the phase's own review flagged and left unfixed. |
| `app/lib/provider/trail/trail_sync_provider.dart` | drain step 3 (waypoint loop) | Second review's WR-04, still open: a keyless waypoint's `StateError` is thrown inside `_drainOne`'s `try`, consuming one of `kMaxSyncAttempts` instead of being treated as a non-retryable invariant break | ⚠️ Warning | Can park an otherwise-healthy trail as `failed` after 4 quick passes, requiring a manual retry tap (SYNC-03 exists precisely for this, so the trail is not stuck, just needlessly demoted). If the create already landed server-side before this throws, CR-04's now-correct delete-decides-on-id logic still deletes the server copy correctly, so no CR-04-style data loss results. |
| `app/lib/util/local_trail_store.dart` | `writeServerWaypointId` (`:709-739`) | Second review's WR-09, still open: clears a waypoint's `localPhotos` the instant its own upload succeeds, before the *trail's* upload as a whole completes | ⚠️ Warning | If a later waypoint in the same drain pass fails and the trail parks as `failed`, this waypoint's photos become unreachable through the model even though the JPEGs still sit on disk (an orphaned-disk-space issue, not a data-loss one — nothing deletes those files until the trail itself is deleted). Narrows REC-05's "review a trail's photos while offline" for that specific waypoint in that specific window. |
| `app/lib/util/local_trail_store.dart` | `deleteLocalTrailRow` (`:394-408`) | Second review's WR-10, still open: has no `owner` clause, unlike every read path in the same file | ⚠️ Warning | A stale `Trail` from a previously signed-in account, still referenced by a backgrounded widget after an account switch, could in principle have its local row deleted by the currently signed-in account tapping Delete. Requires a `localId` collision or a retained cross-account object reference to be practically reachable; flagged because it is the one asymmetry in an otherwise consistently account-scoped file, and the project's own conventions (see MEMORY.md: "scope, don't delete user data") make this worth surfacing even though it is not blocking this phase's stated goal. |
| `app/lib/components/trail/trail_panel.dart` | `:242-254` | Second review's WR-11, still open, but **pre-existing (Phase 35), not introduced by this phase**: `TabBar` gated on `!trail.isLocal`, and `TrailEntity.toModel()` hardcodes `isLocal: true` for every cached row (downloaded trails included), hiding summit-log/comment tabs for any trail served from ObjectBox | ℹ️ Info | Confirmed via `git log -S "isLocal: true"` that this line predates Phase 36 (introduced in Phase 35's `21c4b1ee`). Real, but out of this phase's blast radius — flagged for completeness, not counted against this phase's goal. |
| `app/lib/i18n/*.arb` (12 non-English locales) | — | Second review's WR-06 (destructive-action strings still English-only), and WR-07 (dead `retry_upload` l10n key) | ⚠️ Warning | Content/hygiene gaps, non-blocking per the project's documented English-fallback l10n convention; the destructive-action string gap is arguably more urgent now that CR-04 changed *which* rows get the "cannot be undone" copy, but this is a translation/content task, not a code defect. |
| No `TBD`/`FIXME`/`XXX` debt markers | — | Scanned all ~107 files referenced across the phase's 14 plans (file paths in `files_modified`) | — | Clean. One unrelated pre-existing `TODO` in `settings_account_screen_test.dart` (predates this phase, not in its `files_modified` list). |

No stub returns, no empty-handler patterns, and no hardcoded-empty data reaching rendered UI were found in this pass.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| REC-01 | 36-01, 36-02, 36-06 | Captures with no connection save, no offline-caused failure shown | ✓ SATISFIED | Unchanged from the prior pass; local-first branches confirmed network-free. |
| REC-02 | 36-03, 36-07, 36-11, 36-12 | Saved unsynced trail appears in own-trails list immediately, addressable via its own route | ✓ SATISFIED | `readOwnLocalTrails` + `mergeOwnTrails` unchanged; `/trail/local/:localId` now makes it individually addressable, closing UAT gap 2. |
| REC-03 | 36-01, 36-02, 36-08, 36-13 | Unsynced visibly distinct from synced and from downloaded, and that distinction is actionable through the dropdown | ✓ SATISFIED | `SyncStatusChip` unchanged; dropdown gating now behaviourally tested and reachable (36-13), closing the UAT test-quality finding. |
| REC-04 | 36-01, 36-02, 36-05 | Survives restart, account-scoped, logout doesn't delete | ✓ SATISFIED | Unchanged. |
| REC-05 | 36-02, 36-03, 36-06, 36-07, 36-10, 36-11 | Open/review/edit unsynced trail's metadata offline; edits visible in the list without manual reload | ⚠️ SATISFIED WITH A NARROWED EDGE CASE | 36-10 closes UAT gap 3 (edits now invalidate the list). The second review's CR-03 fix (see above) narrows the literal "while still offline" promise for the specific sub-state where a create has landed server-side but the trail has not yet fully synced — flagged, not silently passed. |
| REC-06 | 36-02, 36-03, 36-06, 36-07, 36-09 | Own-trails list renders offline, states offline-only plainly, no spinner storm | ✓ SATISFIED | 36-09 closes UAT gap 1. |
| SYNC-01 | 36-04 | Uploads on its own on foreground with connection | ✓ SATISFIED | Unchanged. |
| SYNC-02 | 36-02, 36-08, 36-13 | Progress/failure visible inline, not a separate screen; dropdown reflects it correctly | ✓ SATISFIED | Unchanged design; gating now behaviourally tested. |
| SYNC-03 | 36-04, 36-08 | Manual retry for failed/stalled upload | ✓ SATISFIED | Unchanged. |
| SYNC-04 | 36-01, 36-03, 36-04, 36-06 | Interrupted upload never duplicates on retry | ✓ SATISFIED (code-level) | All 4 second-review CRITICAL fixes that threatened this were independently confirmed in source; live-server confirmation remains a human_verification item, unchanged in nature from the prior pass but now covering a materially different (retire-on-success) code path. |
| SYNC-05 | 36-01, 36-03, 36-04, 36-07, 36-14 | Uploaded trail keeps identity in place, not duplicated in list | ✓ SATISFIED | Redesigned (retire/delete instead of keep-in-place-as-synced) but the requirement's actual guarantee — no duplicate entry, identity preserved via server id — still holds by construction; confirmed in `retireUploadedLocalTrail` and the network-fetch re-entry path. |

No orphaned requirements — all 11 IDs assigned to this phase in `.planning/REQUIREMENTS.md` are claimed by at least one plan's `requirements` frontmatter, confirmed again in this pass across all 14 plans.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full test suite passes | `flutter test` (per orchestrator, already run) | 883 passing, 1 pre-existing skip, 0 failures | ✓ PASS (relied on per task instructions, not re-run) |
| Analyzer clean | `flutter analyze --no-pub` (per orchestrator, already run) | 0 errors, 36 pre-existing `info` lints | ✓ PASS (relied on per task instructions, not re-run) |
| Route ordering: `/trail/local/:localId` before `/trail/:id` | `grep -n "GoRoute(" router_provider.dart` | Confirmed local route declared first (lines 359-380 vs 381) | ✓ PASS |
| `retireUploadedLocalTrail`/CR-01..CR-04 fix presence | Direct source read (not grep-only) of `local_trail_store.dart`, `trail_create_screen.dart`, `trail_sync_provider.dart`, `trail_dropdown.dart` | All four fixes present and structurally correct, matching their commit descriptions | ✓ PASS |
| Live device/server upload, delete-with-server-copy, and the narrowed offline-edit case | — | — | ? SKIP — see human_verification |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this project layout (Flutter/Dart). Step 7c: SKIPPED — no conventional or PLAN-declared probes found.

### Human Verification Required

See frontmatter `human_verification` for the structured list. Five items:

1. Re-run UAT Test 3 (dropdown gating for an unsynced trail) — now unblocked, never re-run since the fix.
2. Re-run UAT Test 2's three reported defects (spinner flicker, edit-not-reflected, tap-routes-to-edit) — all three fixes confirmed in source, none re-confirmed on device.
3. Re-run UAT Test 5 (orphan-after-delete, plus the new case: deleting a Failed trail that already has a server id) — confirms the second review's CR-04 fix against a live server.
4. Judge whether the CR-03 "minimum acceptable" fix's narrowed offline-edit behavior (error instead of silent success, for a trail whose create succeeded but is not yet fully synced) is acceptable UX, or whether it should be raised as a follow-up.
5. Re-run the interrupted-upload/no-duplicate device+server pass (carried forward, now against the retire-on-success design rather than the previous keep-in-place design).

### Gaps Summary

No BLOCKER-level gaps were found against the codebase as it stands today. All 4 CRITICAL findings from the second, full-phase code review (36-REVIEW.md) were independently read in the current source — not merely trusted from their commit messages — and are fixed correctly. Of the 12 WARNING findings from that same review, none were found to break a phase requirement outright; one (WR-03, `defaultFilter` `LateInitializationError`) is a real but narrow crash surface unrelated to the offline path this phase is about, and one pair (WR-04/WR-09) narrows REC-05's photo/retry robustness in specific mid-drain windows without causing data loss. One narrowing was found by this verification pass that the review itself did not fully characterize as a requirements-level concern: the CR-03 fix's "minimum acceptable" shape genuinely narrows REC-05's literal "edit... while still offline" promise for one specific sub-state (a trail whose create has landed on the server but has not yet fully synced) — flagged plainly above rather than passed through silently.

The phase's remaining risk is exactly what the fix commits and the plan chain themselves say it is: nothing in this repository can open a live ObjectBox `Store` or hit a real PocketBase server, so the retirement/delete/duplicate-prevention chain — the mechanism this entire phase and its two review rounds exist to get right — has not been exercised end-to-end since the design changed twice (keep-in-place → retire-on-success) and since the second review's 4 fixes landed. That is `human_needed`, not a code gap: the code-level argument is sound and was independently verified by direct reading, not by trusting SUMMARY.md or commit-message claims, but it needs a device+server pass before this phase can be marked complete in ROADMAP.md, which itself explicitly states "NOT complete until they are closed and the verifier passes."

---

_Verified: 2026-08-03T16:30:00Z_
_Verifier: Claude (gsd-verifier)_
