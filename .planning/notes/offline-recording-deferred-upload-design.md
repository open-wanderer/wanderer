---
title: Offline trail recording & deferred upload — design decisions
date: 2026-07-31
context: /gsd-explore session; feeds v1.8 Phase 33
---

## The problem

A hiker who records or navigates a trail offline cannot save it. `_saveRecordedTrack`
(`app/lib/routes/navigation_screen.dart:732`) calls `buildDraftTrail`
(`app/lib/util/route_planner_handoff_util.dart:187`) → `convertGpxToTrail`
(`app/lib/util/trail_import_util.dart:80`) → `POST /trail/convert`. Offline that throws, the
`error_saving_trail` toast fires, and the recording is stranded in the session.

The framing that started this exploration — "temporarily store the GPX so the user can upload
later" — turned out to be only half the picture. Even with local conversion, persisting still
needs `PUT /trail/form`. So *something* must be queued regardless; the conversion question is
really about what the hiker can do at the trailhead while still offline.

## How Komoot and AllTrails solve it

Both converge on the same architecture, which is **not** a "pending uploads" queue:

- **The local save never touches the network.** Komoot: "Recordings save automatically as soon as
  you finish them." AllTrails: the save "will automatically be queued up for syncing the next time
  you have a stable data connection." Neither has an offline failure path at save time at all.
- **Upload is a separate, invisible projection.** AllTrails is explicit that data "will remain
  intact on your device, and the sync will resume the next time you open the app" — the drain
  trigger is *app foreground + connectivity*, not only a connectivity event.
- **Progress is per-item and inline.** Komoot shows a spinning wheel on the tour itself in the
  normal tour list. The recording appears in the library immediately with a sync badge. There is
  no separate "Pending" inbox.
- **Manual retry exists as an escape hatch** — Komoot: "you can restart the process manually."
- **Photos are the known fragile/slow part** — Komoot warns that "the upload may take longer if you
  took a lot of pictures in the app during your Tour."
- **Account mismatch is their #1 support issue** — Komoot's top troubleshooting item for a missing
  tour is "you're probably logged in to different accounts." Reinforces the existing
  scope-by-account rule.
- **Destructive actions are guarded during sync** — "do not delete Tours while the wheel turns,
  even if you think they are already uploaded."

Sources: [Komoot — Tour is not synchronized automatically](https://support.komoot.com/hc/en-us/articles/360026460792-Tour-is-not-synchronized-automatically),
[Komoot — Record an activity](https://support.komoot.com/hc/en-us/articles/10207928747546-Record-an-activity),
[Komoot — Using komoot offline](https://support.komoot.com/hc/en-us/articles/360023078891-Using-komoot-offline),
[AllTrails — Why won't my activity sync?](https://support.alltrails.com/hc/en-us/articles/360019245031-Why-won-t-my-activity-sync),
[AllTrails — How to track and record an activity](https://support.alltrails.com/hc/en-us/articles/360019244391-How-to-track-and-record-an-activity).

## Decisions taken

### 1. Local-first trail in `TrailEntity`, not a separate pending queue

A recorded trail is written to the existing ObjectBox `TrailEntity` store with no server id and a
sync-state field, and appears in the user's trail library immediately with a sync badge. Upload is
a background reconciler that fills in the server id. This matches both incumbents and reuses the
offline-library plumbing already built in v1.6.

Rejected: a distinct `PendingUploadEntity` with its own "Pending" list — the hiker would see two
places where their hikes live.

### 2. Sync is fully automatic in the background

Drain on app-foreground + connectivity, with a per-item inline indicator and a manual retry
escape hatch. Not a user-initiated "Upload" button as the primary path.

### 3. Port `gpx2trail` to Dart and make it the app's *only* conversion path

Chosen over reusing live recording stats, and over keeping `/trail/convert` for the online case.
`/trail/convert` stops being called from the app entirely — one code path, no drift by
construction. The app and web then diverge in *how* they convert, which is the accepted trade.

**Why not reuse the live recording stats.** `navigation_stats_provider.dart:190-215` already
computes distance and elevation gain/loss during the recording — but with a different algorithm
(raw haversine per fix, no XY smoothing; a noise-floor threshold that only moves the reference
altitude when crossed). It would also only cover recordings, not offline GPX *file* imports.

**Why this is safe.** `PUT /trail/form` (`web/src/routes/api/v1/trail/form/+server.ts`) is a
45-line `uploadCreate` passthrough — it does **not** call `gpx2trail` and does not recompute
anything. `distance`, `elevation_gain`, `duration` are plain client-supplied fields. So whatever
computes them *is* the source of truth, and there is no post-sync reconciliation problem: numbers
will not change under the user after upload.

**The fidelity trap the port must reproduce.** The logic to port is `gpx2trail`
(`web/src/lib/util/gpx_util.ts:21-90`, ~70 lines of field-shuffling), `getTotals`
(`web/src/lib/models/gpx/gpx.ts:100-157`), and `gpx-metrics-computation.ts` (86 lines, haversine
only, no dependencies). Two quirks a naive port will silently get wrong:

- `GpxMetricsComputation(5, 5)`: distance is the **raw** haversine sum, but elevation gain/loss
  uses the **smoothed** accumulators (5 m XY threshold *and* 5 m Z threshold). See
  `gpx.ts:140-142`, which picks `...Smoothed` for elevation and unsmoothed for distance.
- The loop starts at `i = 1` (`gpx.ts:125`), so the very first track point is excluded from
  distance, bounding box, and centroid entirely.

Worth pinning with a shared fixture test (same GPX in, same numbers out of both implementations)
even though the TS path is being retired from the app — the web frontend still uses it.

### 4. `trail_create_screen` runs offline with two targeted fixes

Audited: it is already ~80% offline-capable. Categories fall back to the ObjectBox cache
(`category_provider.dart:44`), subcategories are cache-first, settings are a pure ObjectBox read,
the photo picker and EXIF are local, and the missing-`location` reverse geocode is already
best-effort with a silent catch (`trail_create_screen.dart:121`).

Two things break, both fixed narrowly:

- **The map goes blank.** `trail_create_screen.dart:611` passes `offline: trail.isOffline`, but
  `isOffline` is the *downloaded-trail* flag (`models/trail.dart:100`) and a freshly recorded
  draft has it `false` — so `TrailMap` picks the online style and renders nothing. Must also
  consider `!onlineStatus`.
- **Tag autocomplete throws.** `tag_provider.dart`'s `searchByName` hits `/tag?filter=` with no
  cache and no catch. Decision: **swallow the error** and return an empty list, rather than adding
  an ObjectBox tag cache. The user can still type free-form tags; `_resolveTags`
  (`trail_save_provider.dart:34`) creates them at upload time.

## What's still open

Three questions deliberately left for `discuss-phase` — see `.planning/research/questions.md`:
photo file durability, partial-failure semantics of the `tag → trail → waypoint` sequence, and the
logout-with-undrained-recordings UX.

## Related

- Existing offline-session precedent: `ActiveNavigationEntity` +
  `app/lib/util/active_navigation_store.dart` already persist the live breadcrumb across app
  restarts on a best-effort, single-active-row model.
- The two `trail_create_screen` fixes are live bugs today, independent of this feature — they
  affect anyone editing a downloaded trail offline. Split out to
  `.planning/todos/pending/2026-07-31-trail-create-screen-offline-gaps.md`.
