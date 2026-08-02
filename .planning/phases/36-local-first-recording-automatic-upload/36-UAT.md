---
status: testing
phase: 36-local-first-recording-automatic-upload
source: [36-VERIFICATION.md]
started: 2026-08-02T15:48:59Z
updated: 2026-08-02T15:48:59Z
---

## Current Test

number: 1
name: Offline save and re-save of a captured trail
expected: |
  First save succeeds with the success toast and no offline-error. The trail appears
  once in the own-trails list, badged as not-yet-uploaded. The re-save updates the same
  trail (still one entry, not two).
awaiting: user response

## Tests

### 1. Offline save and re-save of a captured trail
steps: End a recording (or import a GPX) in airplane mode, fill in title and pick two photos, tap Save; re-open the trail from the own-trails list, change the title, save again.
expected: First save succeeds with the success toast and no offline-error. The trail appears once in the own-trails list, badged as not-yet-uploaded. The re-save updates the same trail (still one entry, not two).
why_human: Requires driving ImagePicker, ObjectBox and the full widget tree on a real device; PLAN 36-06 deferred this to end-of-phase human-check (no live Store in `flutter test`).
result: [pending]

### 2. Offline own-trails list
steps: In airplane mode, open your own profile's own-trails list (`/profile/<handle>/trails`).
expected: The list renders (does not error or spin forever), shows the offline banner text plainly stating it is showing only what's on-device, includes every not-yet-uploaded trail plus authored trails you've downloaded. Tapping an unsynced trail opens the offline-capable edit screen with its title/photos populated. With nothing saved, the empty state shows the cloud-up icon and "Nothing saved yet" copy.
why_human: Needs a real connectivity transition and a populated ObjectBox store; PLAN 36-07 deferred this to end-of-phase human-check.
result: [pending]

### 3. Trail dropdown gating for unsynced trails
steps: Open the trail dropdown menu on an unsynced trail: check Download is absent (not just disabled) and Delete shows a "cannot be undone" confirmation. Start (or wait for) that trail's upload and reopen the menu mid-upload. Then check the same menu on an ordinary downloaded trail.
expected: Unsynced trail — no Download entry; Delete confirmation states the deletion is unrecoverable. Mid-upload — Delete is greyed out / disabled. Downloaded trail — menu unchanged from today, Delete still only removes the local download.
why_human: Requires a live drain in progress and real menu interaction; PLAN 36-08 deferred this to end-of-phase human-check.
result: [pending]

### 4. Automatic upload and interrupted-upload resume (no duplicates)
steps: With the app foregrounded and a working connection (or by regaining connectivity while the app stays open), watch an unsynced trail upload without tapping anything. Separately, force-kill the app (or otherwise interrupt) partway through an upload — e.g. after the trail record is created but before all waypoints/photos finish — then relaunch/reconnect and let the drain resume.
expected: The trail's badge transitions Pending → Uploading → (badge disappears) with no user action beyond having the app open and online. After an interrupted-and-resumed upload, exactly one trail (and one row per waypoint) exists on the server — no duplicates — and the local row shows no badge (synced) with its photos intact.
why_human: SYNC-01/SYNC-04/SYNC-05's duplicate-prevention chain is verified by code inspection and unit tests of the pure decision logic, but no automated test in this repo exercises a live PocketBase server or a real ObjectBox `Store` (confirmed untestable in `flutter test` — `libobjectbox.dylib` fails to load). An end-to-end device+server pass is the only way to confirm no duplicate is produced under a genuine mid-drain interruption.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
