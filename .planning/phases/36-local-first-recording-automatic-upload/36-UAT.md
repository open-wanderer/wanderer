---
status: complete
phase: 36-local-first-recording-automatic-upload
source: [36-VERIFICATION.md]
started: 2026-08-02T15:48:59Z
updated: 2026-08-03T21:40:00Z
round: 2
---

## Current Test

[testing complete]

## Tests

### 1. Trail dropdown gating for unsynced trails
steps: Open an unsynced trail from the own-trails list (it should now route to the detail screen). Open its dropdown menu — check Download is absent and Delete says "cannot be undone". Start (or wait for) that trail's upload and reopen the menu mid-upload. Then check an ordinary downloaded trail's menu.
expected: Detail screen opens (not edit); no Download entry; Delete confirmation states the deletion is unrecoverable; mid-upload Delete is disabled; a downloaded trail's menu is unaffected.
why_human: Round 1 recorded this as `blocked` — it was never re-run after 36-11/36-12 (routing fix) and 36-13 (behavioural widget coverage) landed. Source and a real widget test are both confirmed correct, but no device pass exists since the fix.
result: pass
reported: "pass. However trail detail screen has \"Offline\" badge instead of \"Waiting for upload\" badge"
note: Every stated assertion of this test passed (detail screen routing, absent Download, Delete copy, mid-upload gating). The badge deviation is logged separately as a minor gap — it is a surface this test did not assert on.

### 2. Round 1 Test 2's three defects, re-checked
steps: (a) Sit on the offline own-trails list for a minute. (b) Edit an unsynced trail's title offline, save, pop back. (c) Tap an unsynced trail.
expected: (a) No repeated full-screen spinner. (b) The list shows the new title with no manual pull-to-refresh. (c) It opens the detail screen, not the edit screen.
why_human: All three fixes (36-09, 36-10, 36-11/12) are confirmed present in source, but the only round of live device testing predates all three.
result: pass

### 3. Delete after sync, plus the failed-with-server-id case
steps: Create a trail offline, go online, let it fully upload, then delete it. Separately — create a trail offline, let its create (`PUT /trail/form`) succeed but force a waypoint/photo upload to keep failing until the row parks as Failed (airplane mode after the first waypoint, or a bad photo), then delete that Failed trail and confirm via the web UI or a second device that it is actually gone server-side.
expected: Case 1 — no local row survives a full upload; deleting removes it cleanly with no orphan. Case 2 — the Delete confirmation copy and the actual behaviour both agree with whether the trail already has a server id; if it does, deleting must remove the server record too, not just the local row. The "this can't be undone, it was never uploaded" wording must not be a lie.
why_human: Originally code review finding CR-04; now also covers CR-02's fix (`readLocalTrailServerId` reads the id straight off the ObjectBox column, so a corrupt cached GPX can no longer make a server-stamped row look unsynced) and WR-15's classified DELETE outcome. All confirmed correct by source read, but nothing in this repo can open a live ObjectBox `Store` or reach a real PocketBase server, so the server-side DELETE has never been exercised end to end.
result: pass

### 4. Offline edit in the create-succeeded-but-not-fully-synced window
steps: Edit an unsynced trail's title while its create has already reached the server but a waypoint upload is still retrying (chip reads Pending/Uploading/Failed but the create genuinely succeeded), while offline. Then do the same edit once the trail has fully synced and the row has been retired, from a stale screen still open on it.
expected: Both cases tell the hiker clearly that the edit did not land (error toast) — never a false success toast, never a crash or corruption.
why_human: This is code review finding CR-03, fixed with the review's stated *minimum acceptable* fix rather than its preferred one, to avoid an ObjectBox schema change. It prevents the silent data loss, but it also means editing offline in this narrow window now fails outright instead of succeeding — a real narrowing of REC-05's "edit an unsynced trail while still offline" promise for this sub-state. Needs a human judgment call on whether that UX is acceptable.
result: pass
judgment: The narrowed offline-edit behaviour in this sub-state is ACCEPTED by the user as shippable. CR-03's "minimum acceptable" fix shape stands; no follow-up is being opened to give the drain an update path.

### 5. Automatic upload and interrupted-upload resume, against the new retirement flow
steps: Foreground the app with a working connection and watch an unsynced trail upload with no user action. Separately, kill the app mid-drain (after the trail record is created, before all waypoints finish) and relaunch/reconnect to confirm the drain resumes. While the trail is parked mid-drain, open it offline and check an EARLIER waypoint's photos still render even though a LATER waypoint is what's still failing (WR-09); and check the chip still reads Pending (not Failed) after several quick background/foreground cycles (WR-04).
expected: Badge transitions Pending → Uploading → disappears with no tap. After an interrupted-and-resumed upload exactly one trail (and one row per waypoint) exists on the server, and the local row is gone (the new retire-on-success behaviour) with photos intact server-side. An earlier-succeeded waypoint's photos stay visible offline while a later one retries, and rapid lifecycle cycling alone never parks the trail as Failed.
why_human: Round 1 passed this, but against the OLD design where the local row survived as `synced`. 36-14 replaced that with retirement, so the previous pass does not cover the current code. Still the only way to confirm SYNC-01/SYNC-04/SYNC-05's duplicate-prevention chain — plus 36-18's WR-04/WR-09 fixes — against a live server.
result: pass

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0

All five of round 2's tests passed on device, including test 4's required human judgment
call. One minor gap was raised as a side observation on an otherwise-passing test 1 (the
detail screen's badge) — see Gaps. It is not a test failure and does not block the phase's
requirements; it is UI copy on a Phase 36 surface.

## Gaps

- truth: "An unsynced trail's detail screen states its upload state (Waiting to upload / Uploading… / Upload failed)"
  status: failed
  reason: "User reported: pass. However trail detail screen has \"Offline\" badge instead of \"Waiting for upload\" badge"
  severity: minor
  test: 1
  root_cause: "`TrailPanel` renders a pre-Phase-36 badge keyed on `trail.isLocal` that hardcodes `l18n.offline` (\"Offline\") at trail_panel.dart:192-227. Phase 36 introduced `SyncStatusChip` (sync_pending \"Waiting to upload\" / sync_uploading / sync_failed \"Upload failed · Tap to retry\") but wired it only into the two list surfaces — trail_list_item.dart:114 and trail_card.dart:176. The detail screen was never given the chip, so the new /trail/local/:localId route the phase added shows the generic local-trail badge and no retry affordance. Note the two badges do not mean the same thing: `isLocal` covers downloaded-and-local alike, whereas the chip is keyed on syncState."
  artifacts:
    - path: "app/lib/components/trail/trail_panel.dart"
      issue: "isLocal-keyed 'Offline' badge (lines 192-227) shadows sync state on the detail screen"
    - path: "app/lib/components/trail/sync_status_chip.dart"
      issue: "Chip exists and is correct, but has no detail-screen call site"
  missing:
    - "Render SyncStatusChip on the trail detail screen for unsynced trails, so the badge reads Waiting to upload / Uploading… / Upload failed · Tap to retry"
    - "Keep the existing isLocal 'Offline' badge for local trails that are NOT in an unsynced sync state, so downloaded-trail behaviour is unchanged"
  debug_session: ""

## Round 1 (2026-08-02 → 2026-08-03) — resolved

Round 1 ran 5 tests: 3 passed, 1 blocked, 1 issue, and produced 5 diagnosed gaps. Six
gap-closure plans (36-09..36-14) were planned and executed against them; a subsequent
full-phase code review found 4 further critical defects downstream of the redesign, all
since fixed. Resolution of each round-1 gap, confirmed by the verifier reading current
source rather than trusting SUMMARY narratives:

- truth: "Own-trails list renders stably without spurious loading states"
  status: resolved
  severity: minor
  resolved_by: "36-09 — device-derived offline fallback on trailFilterProvider plus a bounded retry policy (trailFilterRetry), and skipLoadingOnReload:true on profile_trail_screen"
  debug_session: ".planning/debug/resolved/own-trails-spinner-flicker.md"

- truth: "Tapping an unsynced trail opens the trail detail screen, from which the user can choose to edit"
  status: resolved
  severity: major
  resolved_by: "36-11/36-12 — an addressable /trail/local/:localId route backed by localTrailProvider, with id-dependent chrome gated on sync state; 36-13 added the behavioural widget coverage that the original source-grep test could not provide"
  debug_session: ".planning/debug/resolved/unsynced-trail-skips-detail-screen.md"

- truth: "Edits to an unsynced trail are reflected in the own-trails list without a manual reload"
  status: resolved
  severity: major
  resolved_by: "36-10 — _invalidateOwnTrailsList called from both save tails"
  debug_session: ".planning/debug/resolved/unsynced-trail-edit-not-reflected-in-list.md"

- truth: "Deleting a trail that was created offline and has since synced removes both the server record and the local row"
  status: resolved
  severity: blocker
  resolved_by: "36-14 — retireUploadedLocalTrail deletes (or demotes) the capture row inside the drain's success transaction, so the orphan class cannot be created. Code review CR-04 then fixed the related case of deleting a Failed row that already holds a server id."
  debug_session: ".planning/debug/resolved/orphaned-local-row-after-post-sync-delete.md"
  note: "Re-test as round 2 Test 3 — the fix is structural and unverified on device."

- truth: "Opening a trail whose server record no longer exists fails fast with a clear error, not a chromeless spinner"
  status: dissolved
  severity: major
  resolved_by: "Not fixed — the defect's reproduction was eliminated rather than repaired. This gap's root_cause is the orphan verbatim (a capture row in the box under the server id, excluded by the savedByUserIds clause, rethrowing into defaultRetry). 36-14 retires that row on successful upload, so no such row exists. Plan 36-15 was written to bound trailProvider's retry policy and add chrome to the detail screen's non-data states, then DROPPED by user decision on 2026-08-03 after review found the residual paths reach only the ordinary WandererError screen. See ROADMAP.md's 2026-08-03 decision for the full argument."
  debug_session: ".planning/debug/resolved/orphaned-local-row-after-post-sync-delete.md"
  residual: "The detail screen's loading state is still a chromeless Container with no back button, and trailProvider still has no retry bound. Judged degraded-connection polish rather than a Phase 36 gap. Not tracked as an open gap here; raise separately if wanted."
