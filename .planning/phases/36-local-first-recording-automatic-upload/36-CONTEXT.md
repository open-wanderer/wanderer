# Phase 36: Local-First Recording & Automatic Upload - Context

**Gathered:** 2026-08-02
**Status:** Ready for planning

<domain>
## Phase Boundary

A trail captured on-device — from ending a recording **or** importing a GPX with no
connection — saves immediately without touching the network, appears in the hiker's own-trails
list in a distinguishable state, survives app restart, stays tied to the account that captured
it, and uploads itself when connectivity returns without ever producing a duplicate.

Requirements: REC-01…06, SYNC-01…05.

**Not in this phase:** true OS-level background upload, route re-editing of an unsynced trail,
queuing non-GPX imports for later transcoding (REC-F01), any change to what `/trail/form` or
`/waypoint` accept server-side.

</domain>

<decisions>
## Implementation Decisions

### Photo durability

- **D-01:** The local save **copies** each picked photo into app-owned storage and stores that
  path. `image_picker` returns paths into an OS-purgeable cache and an unsynced trail may sit for
  days on a multi-day hike, so the picker path cannot be trusted to still resolve at drain time.
  Precedent for app-owned directories already exists (`library/`, `avatars/`, `regions/`).
- **D-02:** Copies are deleted **on successful drain**, and deleting an unsynced trail deletes its
  copies. A **startup orphan sweep** catches anything left by a crash between "server accepted"
  and "files deleted" — without it, leaked copies accumulate with nothing ever noticing. Safe
  because after drain the trail renders photos from the server, and a downloaded trail's photos
  are owned separately under `library/<id>/`.
- **D-03:** If the copy itself fails (disk full, permissions): **save the trail, drop that photo,
  tell the user which and why.** The track is irreplaceable — a hiker cannot re-walk the route —
  whereas a photo can be re-added. Matches the app's existing best-effort precedents (reverse
  geocode, `hadWaypointFailures`). Do **not** fall back to the picker path: disk pressure is
  precisely when an OS cache purge is most likely.
- **D-04:** *(user directive)* **Both `Trail` and `Waypoint` must persist `localPhotos`.**
  `WaypointEntity` already has the field (`waypoint_entity.dart:24`); `TrailEntity` does **not**
  and must gain it. The copy rule covers both — waypoints carry photos too, created from GPS EXIF
  at `trail_create_screen.dart:194`.

### Partial-failure semantics

- **D-05:** **Resume from step.** The drain is `PUT /tag` ×N → `PUT /trail/form` → `PUT /waypoint`
  ×N; if it breaks partway, the next attempt continues rather than restarting. Chosen over
  all-or-nothing (its compensating delete can fail on the same bad connection and orphan a
  half-built trail server-side) and over best-effort (a trail silently missing half its waypoints
  is a bad outcome for something unrepeatable). Also satisfies **SYNC-04 by construction**:
  holding the server id is exactly what stops a retry creating a second trail.
- **D-06:** Progress is **derived from the data**, not a separate step counter: **an empty id
  means not-yet-uploaded, everywhere.** Three of the four producers already follow this —
  `_resolveTags` treats null/empty as "create" (`trail_save_provider.dart:37`), the trail uses
  `trail.id.isEmpty` as its create/update discriminator (`trail_create_screen.dart:401`), and
  GPX-derived waypoints are minted with `id: ''` (`gpx_conversion_util.dart:531`).
  **The one exception must be fixed:** photo-EXIF waypoints mint a synthetic
  `'<microseconds>-<index>'` id (`trail_create_screen.dart:194`). Switch to `id: ''` and carry
  list identity in a separate local key.
- **D-07:** A failing drain **backs off over a few attempts, then parks as `Failed`** and surfaces
  for manual retry (SYNC-03 supplies the control). Prevents a permanently-rejected entry burning
  battery and data on every foreground. Deliberately **not** a retryable-vs-permanent failure
  taxonomy — the app has no such classification today and misreading a 401 during token refresh
  would park a perfectly good trail.

### Sync state and how it shows

- **D-08:** Visible states are **Pending → Uploading → Failed**. Synced shows nothing at all.
  Rejected a single unsynced/synced boolean because it cannot express the parked-failure state
  that D-07 produces — a trail that will never upload on its own would look identical to one
  simply waiting, with no cue to retry.
- **D-09:** The badge answers **only** "is this on the server yet". It does **not** encode how the
  trail was captured. Recorded and imported trails are indistinguishable in the list — one axis
  per badge, the same discipline the `isOffline` → `isLocal` rename restored.
- **D-10 (RETRACTED 2026-08-04):** ~~Unsynced and downloaded are mutually exclusive by
  construction, so the two badges can never collide. `savedByUserIds` is written only by
  `TrailDownloadService` and downloading pulls from the server, so a trail with no server copy
  cannot be downloaded.~~ **This premise is false.** `TrailDownloadService.downloadTrail`
  (`app/lib/services/trail_download_service.dart:210-215`) carries `owner`, `localId`,
  `syncState`, `syncAttempts` and `syncNextAttemptAt` forward when it writes into an existing row,
  and `TrailEntity.id` is `@Unique(onConflict: replace)`, so both writers target one row — a row
  can be both unsynced and a library member at once. This phase's own **D-07** (a repeatedly-
  failing upload parks in `failed` with no scheduled retry) is what makes the overlap window
  unbounded rather than a brief race: a `failed` row keeps its `localId` and non-synced
  `syncState` indefinitely.

  **What survives:** the two **badges** this decision justified are still fine — they are
  cosmetic and are correctly re-derived from library membership alone (see
  `app/lib/components/trail/trail_panel.dart`), not from this premise. The consequence clause
  also still holds: **unsynced trails must NOT be expressed by adding the user to
  `savedByUserIds`** — doing so would make them indistinguishable from downloads and show the
  wrong badge.

  **What does not survive:** any reasoning of the form "an `isUnsyncedState` check cannot fire on
  a downloaded row," and any use of this decision to justify **destructive-action gating or
  scoping** — that is precisely the class of error Phase 38's CR-01/CR-02/CR-03 found (three
  destructive code paths trusted this premise).

  See `.planning/notes/unsynced-and-downloaded-are-not-mutually-exclusive.md` for the full
  derivation and `.planning/phases/38.1-downloaded-trail-blocker-closure/` for the remediation.
  The replacement rule (38.1 **D-02**): any destructive action must be scoped by the identity it
  actually destroys — local capture state by `owner`/account, download removal by
  `savedByUserIds` membership, never by a flag read off the shared row.
- **D-11:** REC-06's offline list mixes unsynced trails and downloaded-authored-by-me trails. That
  is **fine as a flat list** — the badges distinguish them and can never appear on the same row.
  No sectioning, no special sort.

### Account scoping and destructive actions

- **D-12:** Signing out with unsynced trails shows a **warning dialog naming the count**, but
  sign-out proceeds. Nothing is lost either way, but the hiker has no way to know that, and
  Komoot's single most common "my tour is missing" report is being signed into a different
  account.
- **D-13:** Account B signing in over account A's unsynced trails sees **nothing at all** — A's
  entries are filtered out of both the list and the drain, never deleted. Matches the existing
  rule that trail rows survive logout and are hidden by filtering (`savedByUserIds` today).
  No "claim these" affordance: that is a data-ownership decision made by whoever holds the phone,
  and the trails would upload under the wrong author.
- **D-14:** An unsynced trail **can** be deleted, behind a confirm that states it is
  **unrecoverable** — the usual "you can re-download it" safety net does not apply. Deletion is
  **blocked while a drain is in flight** rather than racing it (Komoot: "do not delete Tours while
  the wheel turns").

### Drain triggers and offline editing

- **D-15:** Drain fires on **app foreground AND on connectivity being regained while the app is
  open**. Foreground alone (what AllTrails documents) would make a hiker watching their signal
  return background and re-foreground the app to make anything happen. Both signals are already
  available — `onlineStatusProvider` is kept live by the api interceptor. **No OS background
  execution** in this phase.
- **D-16:** REC-05's offline editing is **metadata only** — title, description, category, photos.
  **No route re-editing** of an unsynced trail. Phase 35 already hides the edit-route button
  offline because auto-routing needs Valhalla; this keeps the boundary where the requirement drew
  it.
- **D-17:** *(user directive)* **An unsynced trail must not be downloadable.** The download action
  has to be hidden or disabled for it — see the landmine below.

### Claude's Discretion

- The concrete shape of the sync-state field on `TrailEntity` (enum vs int, indexed or not) and
  the owner field that scopes entries to an account.
- Whether the drain uploads one trail at a time or several concurrently — not discussed, no
  user preference expressed.
- Backoff curve and attempt count for D-07.
- Where the app-owned photo directory lives and how it is named, following the existing
  `library/` / `regions/` conventions.
- Copy of the warning and confirm strings (must be l10n keys in `app_en.arb`, per project
  convention).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase design record
- `.planning/notes/offline-recording-deferred-upload-design.md` — the architecture this phase
  implements: local-first in `TrailEntity` rather than a separate pending queue (§Decisions 1),
  background drain with inline per-item state and manual retry (§2), and the Komoot/AllTrails
  research the choices rest on. **Note two stale passages:** §3's claim that "distance is the raw
  haversine sum but elevation uses smoothed" describes pre-Phase-33 code, and §4's two
  `trail_create_screen` fixes were delivered by Phase 35.
- `.planning/research/questions.md` §"Offline recording & deferred upload" — the four open
  questions this discussion answers. Now resolved: Q1 → D-01…04, Q2 → D-05…07, Q3 → D-12…14,
  Q4 → D-08…11.

### Requirements and roadmap
- `.planning/REQUIREMENTS.md` §"Local-First Unsynced Trails" + §"Background Upload" — REC-01…06,
  SYNC-01…05. Note the source-agnostic wording: an *unsynced trail* covers both a recording and
  an offline GPX import.
- `.planning/ROADMAP.md` §"Phase 36" — the **IA decision block** is binding: unsynced trails live
  in the own-trails list (`/profile/<handle>/trails`), **not** the Library, with the three code
  constraints it names.

### Prior phase artifacts that constrain this one
- `.planning/phases/35-offline-trail-creation/35-SECURITY.md` — two pre-existing findings, both
  since fixed (commit `079ba889`); the delete fall-through it describes is the direct ancestor of
  the landmine below.
- `.planning/phases/34-dart-conversion-port/34-SUMMARY.md` — the on-device Dart conversion path
  REC-01 depends on.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable assets
- **`TrailEntity` + ObjectBox** — the store to write into (D-05's premise). Already survives
  logout by design (`account_data_purge_util.dart:76-98` deliberately does not purge it), which
  gives REC-04's "signing out never deletes it" for free.
- **`trail_save_provider.dart`** — `createTrail` already is the upload sequence. Its create loop
  (`:72-88`) accumulates only successes into `createdWaypoints` and swallows failures into
  `hadFailures`, so a partial run already leaves the local list intact to diff against — which is
  most of what D-05 needs.
- **`onlineStatusProvider`** — kept live by the api interceptor; supplies both of D-15's triggers.
  Note it is **optimistic** (`build() => true`) and settles from ordinary traffic, so a drain
  decision made at a cold moment should `refresh()` first — the same trap Phase 35's OFFUI-04
  guard hit.
- **`isConnectionFailure`** (`online_status_provider.dart:28`) — already distinguishes a genuine
  connectivity failure from a server fault; useful if D-07's backoff is ever refined.
- **`ActiveNavigationEntity` + `active_navigation_store.dart`** — the existing precedent for
  persisting in-progress work across restarts, on a best-effort single-active-row model.

### Established patterns
- **Provenance vs connectivity are now separate.** `Trail.isLocal` means "read from local
  storage"; connectivity is `onlineStatusProvider`. See the field's doc comment in
  `models/trail.dart` — it exists because conflating them shipped a real bug.
- **Escaping for PocketBase filters** — `util/pb_filter_util.dart`. Any new filtered query must
  use `escapePbFilterValue` and pass the expression via `queryParameters`, never path
  concatenation.
- **l10n** — new user-facing strings go in `app/lib/i18n/app_en.arb` (template), then
  `flutter gen-l10n`; the other 13 locales fall back to English.
- **Source-level gates** are an accepted test form here for structural invariants that are
  expensive to test behaviourally (see the PORT-03 gate and
  `test/components/trail/trail_dropdown_delete_gate_test.dart`).

### Integration points
- **`/profile/<handle>/trails`** (`profile_trails_provider.dart`) — the surface unsynced trails
  appear on. Currently **network-only with no ObjectBox fallback**, so REC-06 requires making it
  local-first for the hiker's own handle. This is a meaningful chunk of the phase, not a detail.
- **`trail_card.dart` / `trail_list_item.dart`** — where the sync badge renders. Both already
  branch on `trail.isLocal` for local thumbnails.
- **`trail_dropdown.dart`** — hosts both the delete and download actions. See landmine.

### ⚠ Landmine: `trail_dropdown` branches on `trail.isLocal`, which is about to become ambiguous

After this phase, `trail.isLocal` will be true for **both** downloaded trails and unsynced ones —
`TrailEntity.toModel()` hardcodes it. Two menu actions branch on exactly that and will do the
wrong thing:

1. **Delete** (`_deleteTrail`, `trail_dropdown.dart:203`) — `if (trail.isLocal)` currently means
   "un-download", calling `trailLibraryProvider.deleteTrail(trail.id)`. For an unsynced trail that
   id is **empty**, and un-downloading is the wrong operation entirely (D-14 wants a delete with
   an unrecoverable-confirm, blocked mid-drain).
2. **Download** (`downloadEnabled`, `trail_dropdown.dart:106-111`) — would offer Download on an
   unsynced trail, fetching from the server with an empty id. D-17 requires it hidden or disabled.

Both need unsynced separated from downloaded before either action ships. The delete path is
already guarded by `test/components/trail/trail_dropdown_delete_gate_test.dart`, which will need
updating rather than deleting when that branch is split.

</code_context>

<specifics>
## Specific Ideas

- **Komoot and AllTrails are the reference implementations**, per the design note: local save never
  touches the network, upload is an invisible projection, progress is per-item and inline in the
  normal list, manual retry exists as an escape hatch, and destructive actions are guarded while
  the wheel turns. D-08, D-12 and D-14 all trace to their documented behaviour or their documented
  support burden.
- The user's own framing for D-06: *"Can we track progress by checking what has a server ID and
  what doesn't?"* — which turned out to be the convention three of four producers already follow.

</specifics>

<deferred>
## Deferred Ideas

- **True OS background upload** — registering a background task so trails upload without the app
  being opened. Best outcome for the hiker, but platform-specific work on both iOS and Android
  with its own permissions and testing story. D-15 scopes this phase to foreground + regained
  connectivity.
- **Queueing non-GPX imports for transcoding on reconnect** — already tracked as REC-F01 in
  `.planning/REQUIREMENTS.md` §Future.
- **A retryable-vs-permanent failure taxonomy** — considered for D-07 and rejected for now; would
  reduce wasted traffic but needs classification the app does not have.
- **Route re-editing of an unsynced trail** — D-16 keeps REC-05 to metadata. Phase 34 did make the
  planner offline-survivable, so this is possible later if wanted.
- **Concurrent drains of several trails** — not discussed; left to the planner, but if it becomes
  a user-visible choice it belongs in a later phase.

### Reviewed Todos (not folded)
- `2026-07-24-comaps-poly-region-extraction-spike.md` — keyword false-positive (matched on
  "region/polygon/planning"). Unrelated to this phase.
- `2026-07-18-way-types-and-surfaces-breakdown.md` — keyword false-positive. Unrelated.
- `2026-07-31-trail-create-screen-offline-gaps.md` — **already delivered by Phase 35** but still
  filed under `todos/pending/`. Should be moved to `todos/completed/`; not folded because there is
  nothing left to do.

</deferred>

---

*Phase: 36-local-first-recording-automatic-upload*
*Context gathered: 2026-08-02*
