---
title: "Correction: unsynced and downloaded are NOT mutually exclusive by construction"
date: 2026-08-04
context: Phase 38 code review (38-REVIEW.md, CR-01/CR-02/CR-03); corrects Phase 36 D-10
---

## The claim that turned out to be false

Phase 36's **D-10** states that unsynced and downloaded are *"mutually exclusive by
construction"*, reasoning that `savedByUserIds` is written only by `TrailDownloadService`, that
downloading pulls from the server, and therefore that a trail with no server copy cannot be
downloaded.

Phase 38's **D-03** leaned on this directly, and it is why both the plan-checker and I cleared
`_allowDelete`'s `isUnsyncedState(...)` escape hatch as safe: if the two states cannot co-occur,
an unsynced check can never fire on a downloaded row.

**They can co-occur.** The premise only holds for a trail that has *never* reached the server.

## What the codebase already said

Two places document the overlap explicitly — this was never hidden, just not carried into D-10:

- `app/lib/services/trail_download_service.dart:211-212` — a re-download into an existing row
  deliberately carries `owner`, `localId`, `syncState`, `syncAttempts` and `syncNextAttemptAt`
  **forward**. A downloaded row can therefore hold local-capture bookkeeping.
- `app/lib/store/local_trail_store.dart` (`shouldDeleteUploadedRow` doc) — *"A non-empty list
  therefore means some account downloaded this trail while its upload was in flight. That is
  possible from the moment `writeServerTrailId` stamps the server id, because from then on the
  trail is fetchable and `TrailDownloadService` writes into the SAME row."*

`TrailEntity.id` is `@Unique(onConflict: replace)`, so both writers target one row.

## Why the window is unbounded

Phase 36's **D-07** parks a repeatedly-failing upload in `failed` state rather than retrying
forever. A `failed` row keeps its `localId` and its non-synced `syncState` indefinitely — so the
overlap is not a brief race during an in-flight upload. It persists until the hiker resolves it.

## What this invalidates

- **Phase 36 D-10** — the "by construction" guarantee. The *badges* it justified are still fine
  (they are cosmetic); the **destructive-action gating** it was later used to justify is not.
- **Phase 38 D-03** — inherited the same error.
- Any future reasoning of the form "an `isUnsyncedState` check cannot fire on a downloaded row."

## The rule that replaces it

**Treat unsynced-ness and library membership as independent axes.** A row may be neither, either,
or both. Any destructive action must be scoped by the identity it actually destroys:

- Deleting **local capture state** (rows, `unsynced/<localId>/` photo dirs) must be scoped by
  `owner`/account — never reached via a flag read off a shared cache row.
- Removing a **download** must be scoped by `savedByUserIds` membership and must not remove a row
  that still holds another account's capture state.

Note that `unsyncedTrailPhotoDir(appDocsPath, localId)` resolves to
`unsyncedPhotoRoot(appDocsPath)/localIdDirSegment(localId)` — **no account component at all** —
which is what turns the reasoning error into real cross-account data loss (CR-01).

Full findings with file:line evidence:
`.planning/phases/38-downloaded-trails-as-state-not-objects/38-REVIEW.md`.
Remediation: Phase 38.1.
