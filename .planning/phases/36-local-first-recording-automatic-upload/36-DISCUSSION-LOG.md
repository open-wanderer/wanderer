# Phase 36: Local-First Recording & Automatic Upload - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-02
**Phase:** 36-local-first-recording-automatic-upload
**Areas discussed:** Photo durability, Partial-failure semantics, Sync state vocabulary, Logout with undrained trails, Drain triggers, Offline editing scope

---

## Photo durability

**Where do picked photos live until the drain?**

| Option | Description | Selected |
|--------|-------------|----------|
| Copy into app-owned storage on save | Survives OS cache purge; costs disk + cleanup obligation | ✓ |
| Keep the picker path, verify at drain | No duplication, but photos genuinely vanish and the user finds out days later | |
| Copy only once a drain has failed | Avoids duplication in the common online case; loses the race if the OS purges first | |

**Who removes the copies, and when?**

| Option | Description | Selected |
|--------|-------------|----------|
| Delete on successful drain + startup orphan sweep | Sweep catches crash-between-steps leaks | ✓ |
| Delete on successful drain only | Simplest; a crash leaks copies permanently with nothing noticing | |
| Keep until the trail leaves the device | Renders photos offline post-upload; two competing owners of local photos | |

**What if the copy itself fails (disk full, permissions)?**

| Option | Description | Selected |
|--------|-------------|----------|
| Save the trail, drop the photo, tell the user | The track is irreplaceable, a photo is not | ✓ |
| Fail the whole save | Honest but discards an unrepeatable recording over a disk problem | |
| Save, keep the picker path as fallback | Reintroduces the fragile state, in the case where purge is most likely | |

**Notes:** User added a directive rather than picking from a menu — *"Make sure to also persist
waypoint photos. In the end, both entities (trail & waypoint) should have localPhotos."* Scouting
had found `WaypointEntity.localPhotos` exists (`waypoint_entity.dart:24`) while `TrailEntity` has
no such field, so the asymmetry runs the opposite way from what the question implied.

---

## Partial-failure semantics

**Connection drops after the trail is created but before waypoints land.**

| Option | Description | Selected |
|--------|-------------|----------|
| Resume from step | Most state; satisfies SYNC-04 by construction | ✓ |
| Best-effort and surface it | Least machinery, has precedent in `hadWaypointFailures` | |
| All-or-nothing | Simplest to reason about; compensating delete can itself fail | |

**Notes:** User asked *"Can we track progress by checking what has a server ID and what doesn't?"*
Investigation showed the convention already holds for tags, the trail, and GPX-derived waypoints —
the sole exception being photo-EXIF waypoints, which mint a synthetic id. That finding drove the
next question.

**How should the drain tell a server id from a local one?**

| Option | Description | Selected |
|--------|-------------|----------|
| Empty id means not-yet-uploaded, everywhere | No new fields; matches 3 of 4 existing producers | ✓ |
| Explicit `serverId` field alongside local id | Unambiguous; adds dual-id reasoning to every read path | |
| Record uploaded ids on the queue entry | Keeps sync bookkeeping out of domain models; can drift from the data | |

**What stops a doomed drain retrying forever?**

| Option | Description | Selected |
|--------|-------------|----------|
| Back off, then park as failed for manual retry | SYNC-03 already supplies the retry control | ✓ |
| Retry on every foreground, indefinitely | Simplest; wastes data on permanently-rejected entries | |
| Distinguish retryable from permanent failures | Most precise; needs a taxonomy the app lacks, and a 401 misread would park a good trail | |

---

## Sync state vocabulary

**What states does an unsynced trail move through?**

| Option | Description | Selected |
|--------|-------------|----------|
| Pending → Uploading → Failed | Synced shows nothing; Failed is what manual retry acts on | ✓ |
| Just unsynced vs synced | Cannot express the parked-failure state, so no cue to retry | |
| …plus Uploaded-with-problems | Most honest about resume-from-step; needs its own remedy | |

**Should the badge also show recorded vs imported?**

| Option | Description | Selected |
|--------|-------------|----------|
| No — unsynced is unsynced | One axis per badge | ✓ |
| Yes — show capture source | Second axis on the same row | |
| Only in the detail view | Middle ground; would anyone look? | |

**How do the sync badge and downloaded badge coexist?**

**Question withdrawn — premise was wrong.** User challenged it: *"how can it be unsynced and
downloaded? A downloaded trail can only come from the server, not the user's device."* Correct.
`savedByUserIds` is written only by `TrailDownloadService` and downloading pulls from the server,
so the two states are mutually exclusive by construction. Recorded in CONTEXT.md as D-10, with the
consequence that unsynced trails must not be expressed via `savedByUserIds`.

**REC-06's offline list mixes unsynced and downloaded-authored-by-me. Problem?**

| Option | Description | Selected |
|--------|-------------|----------|
| Fine — badges already tell them apart | Flat list; badges can never appear on the same row | ✓ |
| Group into sections | Clearer, but a different shape from the online list | |
| Sort unsynced to the top | Same shape; a sort rule that applies in one state only | |

---

## Logout with undrained trails

**Do we warn on sign-out?**

| Option | Description | Selected |
|--------|-------------|----------|
| Warn with a count, but let them proceed | Komoot's #1 "missing tour" cause is account mismatch | ✓ |
| Say nothing | Nothing is lost, but the hiker cannot know that | |
| Non-blocking notice near sign-out | Discoverable but easily missed in the moment that matters | |

**What does account B see over account A's unsynced trails?**

| Option | Description | Selected |
|--------|-------------|----------|
| Nothing at all | Filtered from list and drain, never deleted | ✓ |
| Nothing, but tell B the device holds other work | Transparent; leaks another account's activity | |
| Offer to claim them | Rescues a mistyped login; wrong author, decided by whoever holds the phone | |

**Can an unsynced trail be deleted?**

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, with an unrecoverable confirm; blocked mid-drain | No re-download safety net applies | ✓ |
| Yes, same confirm as any other trail | Wording was written for recoverable server trails | |
| No — must upload first | Traps a hiker who recorded a junk track | |

---

## Drain triggers *(follow-up area)*

| Option | Description | Selected |
|--------|-------------|----------|
| Foreground + connectivity-regained while open | Both signals already available via `onlineStatusProvider` | ✓ |
| Foreground only | What AllTrails documents; forces a needless background round trip | |
| Add true OS background upload | Best for the hiker; platform-specific, arguably its own phase | |

---

## Offline editing scope *(follow-up area)*

| Option | Description | Selected |
|--------|-------------|----------|
| No — details only, as REC-05 says | Matches the requirement wording and Phase 35's offline gate | ✓ |
| Yes — the planner works offline now | Possible since Phase 34; widens REC-05 | |
| Yes, but only when online | No new rule; answer changes if the gate is revisited | |

**Notes:** User added *"Make sure a un-synced trail is not downloadable"* — captured as D-17, and
it surfaced a second `trail_dropdown` landmine alongside the delete one.

---

## Claude's Discretion

- Shape of the sync-state field and the account-owner field on `TrailEntity`
- Whether the drain uploads one trail at a time or several concurrently
- Backoff curve and attempt count before parking as Failed
- Location and naming of the app-owned photo directory
- Wording of the warning and confirm strings (must be l10n keys)

## Deferred Ideas

- True OS background upload — platform work, own phase
- Queueing non-GPX imports for transcoding on reconnect — already REC-F01
- Retryable-vs-permanent failure taxonomy — rejected for D-07, revisitable
- Route re-editing of an unsynced trail — possible since Phase 34, out of REC-05
- Concurrent drains — left to the planner unless it becomes user-visible
