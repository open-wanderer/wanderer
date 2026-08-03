---
status: testing
phase: 36-local-first-recording-automatic-upload
source: [36-VERIFICATION.md]
started: 2026-08-02T15:48:59Z
updated: 2026-08-03T16:45:00Z
round: 2
---

## Current Test

number: 1
name: Trail dropdown gating for unsynced trails (round 1's blocked test, now unblocked)
expected: |
  Tapping an unsynced trail opens the DETAIL screen (not the edit screen), with no Like
  button and an Edit button in place of Download/Navigate. The dropdown's Download entry
  is absent (not merely disabled) and Delete's confirmation states the deletion is
  unrecoverable. Mid-upload, Delete is disabled. An ordinary downloaded trail's menu is
  unchanged.
awaiting: user response

## Tests

### 1. Trail dropdown gating for unsynced trails
steps: Open an unsynced trail from the own-trails list (it should now route to the detail screen). Open its dropdown menu — check Download is absent and Delete says "cannot be undone". Start (or wait for) that trail's upload and reopen the menu mid-upload. Then check an ordinary downloaded trail's menu.
expected: Detail screen opens (not edit); no Download entry; Delete confirmation states the deletion is unrecoverable; mid-upload Delete is disabled; a downloaded trail's menu is unaffected.
why_human: Round 1 recorded this as `blocked` — it was never re-run after 36-11/36-12 (routing fix) and 36-13 (behavioural widget coverage) landed. Source and a real widget test are both confirmed correct, but no device pass exists since the fix.
result: pending

### 2. Round 1 Test 2's three defects, re-checked
steps: (a) Sit on the offline own-trails list for a minute. (b) Edit an unsynced trail's title offline, save, pop back. (c) Tap an unsynced trail.
expected: (a) No repeated full-screen spinner. (b) The list shows the new title with no manual pull-to-refresh. (c) It opens the detail screen, not the edit screen.
why_human: All three fixes (36-09, 36-10, 36-11/12) are confirmed present in source, but the only round of live device testing predates all three.
result: pending

### 3. Delete after sync, plus the failed-with-server-id case
steps: Create a trail offline, go online, let it fully upload, then delete it. Separately — create a trail offline, let its create (`PUT /trail/form`) succeed but force a waypoint/photo upload to keep failing until the row parks as Failed (airplane mode after the first waypoint, or a bad photo), then delete that Failed trail and confirm via the web UI or a second device that it is actually gone server-side.
expected: Case 1 — no local row survives a full upload; deleting removes it cleanly with no orphan. Case 2 — the Delete confirmation copy and the actual behaviour both agree with whether the trail already has a server id; if it does, deleting must remove the server record too, not just the local row.
why_human: This is code review finding CR-04. The fix is confirmed correct in source, but nothing in this repo can open a live ObjectBox `Store` or reach a real PocketBase server, so the server-side DELETE succeeding — and the local row surviving a failed one — has never been exercised end to end.
result: pending

### 4. Offline edit in the create-succeeded-but-not-fully-synced window
steps: Edit an unsynced trail's title while its create has already reached the server but a waypoint upload is still retrying (chip reads Pending/Uploading/Failed but the create genuinely succeeded), while offline. Then do the same edit once the trail has fully synced and the row has been retired, from a stale screen still open on it.
expected: Both cases tell the hiker clearly that the edit did not land (error toast) — never a false success toast, never a crash or corruption.
why_human: This is code review finding CR-03, fixed with the review's stated *minimum acceptable* fix rather than its preferred one, to avoid an ObjectBox schema change. It prevents the silent data loss, but it also means editing offline in this narrow window now fails outright instead of succeeding — a real narrowing of REC-05's "edit an unsynced trail while still offline" promise for this sub-state. Needs a human judgment call on whether that UX is acceptable.
result: pending

### 5. Automatic upload and interrupted-upload resume, against the new retirement flow
steps: Foreground the app with a working connection and watch an unsynced trail upload with no user action. Separately, kill the app mid-drain (after the trail record is created, before all waypoints finish) and relaunch/reconnect to confirm the drain resumes.
expected: Badge transitions Pending → Uploading → disappears with no tap. After an interrupted-and-resumed upload exactly one trail (and one row per waypoint) exists on the server, and the local row is gone (the new retire-on-success behaviour) with photos intact server-side.
why_human: Round 1 passed this, but against the OLD design where the local row survived as `synced`. 36-14 replaced that with retirement, so the previous pass does not cover the current code. Still the only way to confirm SYNC-01/SYNC-04/SYNC-05's duplicate-prevention chain against a live server.
result: pending

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps

_No open gaps. Round 1's five gaps are all closed — see the resolved record below._

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
