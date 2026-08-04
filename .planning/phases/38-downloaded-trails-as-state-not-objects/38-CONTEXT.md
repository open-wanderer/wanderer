# Phase 38: Downloaded Trails as State, Not Objects - Context

**Gathered:** 2026-08-04
**Status:** Ready for planning

<domain>
## Phase Boundary

A downloaded trail's detail screen makes it unambiguous what the hiker is looking at and what
each action destroys. Removing a download and deleting the trail become two differently labelled
actions whose availability is derived from **library membership** and **authorship** — never from
`Trail.isLocal`. Editing always operates on the server copy or refuses with a reason. Refreshing a
stored copy is an explicit *Update*.

Two live bugs are fixed as part of this, not deferred: photo duplication on every Library edit,
and the hiker's own edit never reaching the stored copy.

**Not in this phase:** any change to what `/trail/form`, `/waypoint` or the download endpoints
accept server-side; automatic re-download of anything; a staleness/"update available" indicator;
any change to unsynced-capture behaviour (Phase 36 owns that).

</domain>

<decisions>
## Implementation Decisions

### The core coupling to break

- **D-01:** Destructive-action availability is derived from **library membership** (does an
  ObjectBox row exist for this trail, held by the current account) and **authorship** (is the
  signed-in actor the author). **Nothing in the menu may branch on `Trail.isLocal`.** That flag is
  hardcoded `true` by `TrailEntity.toModel()` for every cached row
  (`app/lib/entities/trail_entity.dart:355`), and `TrailNotifier.build()` falls back to the cache
  on *any* fetch exception — so provenance drifts with network conditions and one timeout arms the
  wrong branch. This is the defect the whole phase exists to remove.
- **D-02:** A trail the hiker authored **and** downloaded shows **both** *Remove download* and
  *Delete trail*. They are independent axes; today there is no way to delete your own trail from
  the server once it is in your library.
- **D-03:** Phase 36's **D-10** holds and is relied on: unsynced and downloaded are mutually
  exclusive by construction, so the menu never has to render both states at once. Three trail
  kinds exist — unsynced / downloaded / server-only.

### Remove-download wording and confirm

- **D-04:** The confirm dialog **mirrors the offline-regions pattern** already shipped at
  `app/lib/routes/settings_offline_regions_screen.dart:1022-1041` — a title naming what is being
  removed, a body modelled on `regions_delete_confirm_body` ("This removes the downloaded map and
  elevation data for this region. You'll need to download it again to use it offline."), and a red
  confirm action. Deliberately **no "cannot be undone" claim**: that is false for an un-download,
  and the existing comment at `app/lib/components/trail/trail_dropdown.dart:239-244` already flags
  the current reuse of `delete_trail_confirm` as wrong on exactly this point.
- **D-05:** **One dialog, no connectivity branching.** The body already states that re-downloading
  is needed, which is the honest cost whether or not there is signal right now. Rejected: an extra
  offline-only warning line (second string, connectivity check inside the dialog) and refusing
  removal while offline (paternalistic — freeing space is legitimate precisely in the field).
- **D-06:** **Reuse the already-translated keys.** `remove` ("Entfernen") and `available_offline`
  ("Offline verfügbar") exist in all 14 locales and are absent from
  `app/lib/i18n/untranslated_messages.json`. Only the confirm body is new copy. This minimises what
  lands English-only under the pending translation todo.
- **D-07:** Fix the hardcoded English literal `'Available offline'` at
  `app/lib/components/trail/trail_dropdown.dart:179` — a translated `available_offline` key already
  exists and is simply not being used.

### Menu shape

- **D-08:** When downloaded, the single inert "Available offline ✓" item becomes **two flat items:
  *Update* and *Remove download*.** When not downloaded, one *Download* item as today. Flat
  `PopupMenuItem` + `ListTile` is the established shape in this file; the app has **no**
  menu-item-opens-a-sub-sheet pattern anywhere, and a sheet would bury both actions two taps deep.
- **D-09:** The **bottom action bar is unchanged**. It already hides its Download button once
  downloaded (`if (!availableOffline)` at `app/lib/routes/trail_detail_screen.dart:219`) and
  Navigate keeps the full width. Navigate is the primary thing a hiker does with a downloaded
  trail; Update is a deliberate maintenance action and should not carry equal visual weight.
- **D-10:** **Re-gate the "Offline" pill** at `app/lib/components/trail/trail_panel.dart:214` from
  `trail.isLocal && !isUnsyncedState(...)` onto the same library-membership signal the menu uses.
  It has the identical defect: today it appears when a fetch fails and vanishes when one succeeds,
  and can contradict the menu by showing "Offline" on a trail whose menu offers *Download*.
  Consistent with Phase 36's **D-09** (one axis per badge) — this badge's axis is "is it stored on
  this device", distinct from the sync badge's "is it on the server yet".

### Update vs. automatic reconciliation

- **D-11:** These are **complementary, not redundant** — the distinction was explicitly raised
  during discussion and must not be collapsed by downstream agents:
  - **Automatic (no user action, no network):** the hiker's own edit *made on this device*. The
    `POST /trail/form/{id}` response is already in hand, so it is written straight into the stored
    row. See D-13.
  - **Explicit *Update* (user-initiated):** changes this device did not make — the trail's author
    corrects the route or adds waypoints, or the hiker edits their own trail on the web app or a
    second device. Nothing on the device can know about these without asking.
- **D-12:** *Update* is implemented as **re-download of the same trail**, not a new code path.
  `TrailDownloadService` already overwrites in place: `TrailEntity.id` is
  `@Unique(onConflict: replace)` and its write transaction deliberately carries `savedByUserIds`,
  `owner`, `localId`, `syncState`, `syncAttempts`, `syncNextAttemptAt` and `localPhotos` forward
  (`app/lib/services/trail_download_service.dart:187-218`). Update is therefore close to
  re-enabling the action that `downloadEnabled = !availableOffline && !isDownloading` currently
  disables. It also removes the neither-copy window of today's only workaround (Remove download →
  Download), where a failed re-download leaves the hiker with nothing.
- **D-12a:** Given D-14, *Update*'s remaining job is precise: **it is the same refresh, without
  having to open the trail** (user's framing) — and **photos are the only thing it actually pulls
  over the network**. Its two real uses are refreshing several trails deliberately before heading
  out of signal, and recovering when an earlier refresh failed partway. Keep it in the menu:
  metadata and track staying fresh via D-14 does *not* make it redundant, because photos never
  refresh on their own and a trail the hiker has not opened online refreshes nothing at all.

### Bug 2 — the stored copy never receives the hiker's own edit

- **D-13:** After a successful network update, write the returned model into the existing
  ObjectBox row when one exists for that server id and the current account holds it.
  `applyNetworkEditToLocalRow` today runs only when `reconcileLocalId != null`
  (`app/lib/routes/trail_create_screen.dart:756`), which is the unsynced-capture path; a downloaded
  row has `localId == null`, so nothing writes the edit into ObjectBox. **No new network traffic** —
  the response is already in hand.
- **D-14:** Generalised beyond the edit case: **any successful fetch of a trail that is also
  downloaded opportunistically refreshes the stored row's metadata *and* its `gpxData`**, since
  both arrive in the same response and writing them costs **zero additional network traffic**
  (see D-23). **Photos are explicitly excluded** — they are the one asset not already in hand.
  Effect: a downloaded trail's name, description, waypoints and track stay current merely by being
  viewed online, with no bytes spent beyond what opening any trail already costs.
- **D-14a:** The line is drawn at *free versus costly*, deliberately and not arbitrarily: nothing
  automatic ever spends bytes. A trail whose author swapped its photos shows the old ones offline
  until *Update* is tapped, while its name and track are already current. Accepted trade.

### Bug 1 — editing a Library trail duplicates its photos on the server

- **D-15:** **Edit stays enabled and fetches the server copy on tap** — the editor opens only on
  success, and shows a toast on failure. Chosen over disabling or hiding because it is the option
  that *structurally* fixes the bug rather than merely hiding the path to it: the editor can then
  never receive a cached model. Online it is invisible to the user.
- **D-16:** Rejected alternatives and why, so they are not revisited: **disable + grey** matches
  what "Show on map" and the drain-in-flight Delete already do in this file, but both are silent —
  exactly what the D-17 comment at `trail_dropdown.dart:150-153` calls out as reading like a broken
  app. **Hide entirely** matches D-17's own choice, but D-17's reasoning was specific to an action
  that would have been *meaningless* (fetching with an empty id), not merely temporarily
  unavailable; Edit vanishing from your own trail reads as a permissions problem.
- **D-17:** Mint **one** new refusal string, modelled on the existing `delete_needs_connection`
  ("This trail is already on the server. Connect to the internet to delete it."). It ships
  English-only in 13 locales exactly as its sibling already does — accepted, and to be added to
  `app/lib/i18n/untranslated_messages.json` and the pending translation todo.
- **D-18:** The root cause for the researcher/planner to verify, not re-derive:
  `TrailDownloadService` overwrites `entity.photos` with **local file paths**
  (`trail_download_service.dart:148`), and `toModel()` surfaces those as `localPhotos` while
  leaving the model's `photos` at its `[]` default (`trail_entity.dart:360`). The edit form
  therefore seeds `initialValue: trail.localPhotos` (real files, so they render normally and look
  correct) and `initialWebPhotos: trail.photos` — empty
  (`trail_create_screen.dart:1469`). On save those paths survive the `existsSync()` filter and are
  sent under the **append-only `photos+`** key (`app/lib/util/trail/form_data.dart:47-51`), so the
  server photo set doubles per edit. `_removedServerPhotos` is computed from the same empty
  `trail.photos` (`trail_create_screen.dart:383`), so removing a photo silently no-ops.

### forceOffline — retire it

- **D-19:** **Hard constraint from the user: a downloaded trail must never be stale while online.**
  This rules out disk-first-while-online in every form.
- **D-20:** **The `forceOffline` flag is retired entirely**, along with everything added for it on
  `feature/app` on 2026-08-04: the `?offline=1` query parameter and its two route builders in
  `app/lib/provider/router_provider.dart`, the `forceOffline` options on `trailDetailLocation` /
  `trailMapLocation` (`app/lib/util/trail/route_location.dart`) and their tests, the Library call
  sites, and the forwarding through `LikeButton`, `TrailDropdown`, `TrailPanel` and
  `TrailDetailMapScreen`. **This is a deliberate revert, not a regression** — the flag's only job
  was making the Library's Delete deterministic, which D-01/D-02 now handle properly.
- **D-21:** Rationale to preserve: because the flag was part of the provider's **family key**,
  `trailProvider(id)` and `trailProvider(id, forceOffline: true)` are different instances, so every
  consumer had to forward it or silently mutate an invisible second copy. Three call sites needed
  threading in a single sitting. Any future call site that forgets breaks quietly. Do not
  reintroduce a family-key flag for display source.
- **D-22:** Resulting read model: **online always fetches; disk is the offline fallback** (the
  pre-existing `catch` behaviour). A download exists for **offline availability, not online
  data-saving** — that is the reframing that reconciles this with the "bandwidth is an active
  decision" principle: nothing re-downloads automatically, and both *Download* and *Update* remain
  explicit.
- **D-23:** **The actual network-cost model**, verified 2026-08-04 — planning should rely on this
  rather than re-deriving it, and an earlier draft of this document got it wrong:
  - `TrailNotifier.build()` fetches the trail record **and then unconditionally fetches the GPX
    file** `/files/trails/$id/${trail.gpx}` (`app/lib/provider/trail/trail_provider.dart:35-58`).
    This happens on **every online open of every trail**, downloaded or not. The GPX is therefore
    *already paid for* by the time a downloaded trail renders — which is what makes D-14 free.
  - `TrailDownloadService.download(trail)` takes an **already-fetched model** and pulls **only
    photos** over the network — trail photos plus per-waypoint photos
    (`app/lib/services/trail_download_service.dart:70-97`). It never fetches the GPX.
  - The track is persisted straight from memory: `TrailEntity.gpxData` is a plain `String?`
    populated from `trail.expand?.gpxData` (`app/lib/entities/trail_entity.dart:48`, `:241`), not
    a file on disk. Refreshing it is a single ObjectBox string write, no file I/O.
  - Consequence: **photos are the only expensive asset in the entire download path**, which is
    exactly where the "active decision" principle is enforced (D-14a, D-12a).

### Claude's Discretion

- Exact wording of the new confirm body and the new edit-refusal string, within D-04's shape
  (name what happens, state the recovery, no false permanence claims) and D-17's model.
- Whether *Update* and *Remove download* sit under one `PopupMenuDivider` or two, and their order.
- How library membership is surfaced to the menu — `availableOffline` is already computed at
  `app/lib/routes/trail_detail_screen.dart:121-123` and forwarded to both `TrailDropdown` and
  `TrailPanel`; reusing that versus reading membership directly is a planning decision.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### This phase
- `.planning/ROADMAP.md` § "Phase 38: Downloaded Trails as State, Not Objects" — goal, success
  criteria, the two live bugs with verified call chains, prior-art findings, and the explicitly
  rejected options ("Go to source", staleness badge).

### Prior decisions that constrain this phase
- `.planning/phases/36-local-first-recording-automatic-upload/36-CONTEXT.md` — **D-06** (empty id
  means not-yet-uploaded; `trailHasServerId` is the sanctioned signal), **D-08/D-09** (sync badge
  states; one axis per badge), **D-10** (unsynced and downloaded are mutually exclusive by
  construction — load-bearing for D-03 above).
- `.planning/todos/pending/2026-08-03-destructive-action-strings-untranslated.md` — destructive
  copy is English-only in 13 of 14 locales; **machine translation is explicitly rejected** for
  irreversible actions. Constrains D-06 and D-17. Any new destructive string must be added to this
  todo's backlog.

### Implementation precedents named during discussion
- `app/lib/routes/settings_offline_regions_screen.dart:1022-1041` — the un-download confirm dialog
  this phase mirrors (D-04). `regions_delete_confirm_title` / `_body` / `_action` in
  `app/lib/i18n/app_en.arb:447-456`, translated in all locales.
- `app/lib/i18n/untranslated_messages.json` — the authoritative list of English-only keys. Check
  before minting any string; `remove` and `available_offline` are absent from it, i.e. translated.

### Bug sites (verified 2026-08-04 by reading the call chain)
- `app/lib/services/trail_download_service.dart:148` and `:187-218` — `entity.photos` overwritten
  with local paths; in-place overwrite semantics that make *Update* nearly free.
- `app/lib/entities/trail_entity.dart:355` (`isLocal: true` hardcoded) and `:360`
  (`localPhotos` fallback).
- `app/lib/routes/trail_create_screen.dart:383`, `:756`, `:1469` — removed-photo computation,
  reconciliation gate, photo-picker seeding.
- `app/lib/util/trail/form_data.dart:47-51` — the append-only `photos+` key.
- `app/lib/components/trail/trail_dropdown.dart:150-153`, `:179`, `:239-244`, `:406` — D-17 hide
  precedent, hardcoded English literal, the wrong-confirm-string comment, the un-download branch.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`availableOffline`** (`app/lib/routes/trail_detail_screen.dart:121-123`) — already computed
  from `trailLibraryProvider` (library membership, *not* the trail model) and already forwarded to
  `TrailDropdown` and `TrailPanel`. This is the signal D-01 wants; the wiring largely exists.
- **`TrailDownloadService`** — already overwrites in place and carries local bookkeeping forward,
  so *Update* needs no new download path (D-12).
- **`applyNetworkEditToLocalRow`** — exists and works; only its call gate is too narrow (D-13).
- **`toastProvider` + `ToastMessage`** — the established way this file reports a refused action;
  `trail_uploaded_reopen_to_edit` is the precedent for an edit-specific refusal toast.
- **The offline-regions confirm dialog** — a complete, translated, structurally identical
  un-download confirm to model D-04 on.

### Established Patterns
- Menu items are flat `PopupMenuItem` wrapping a `ListTile`, separated by `PopupMenuDivider`.
  No sub-sheet pattern exists anywhere in the app (constrains D-08).
- Riverpod codegen: `@riverpod` class notifiers with generated `.g.dart`. Retiring the
  `forceOffline` parameter requires regenerating `trail_provider.g.dart` via
  `dart run build_runner build --delete-conflicting-outputs`.
- Destructive actions use an `AlertDialog` with Cancel + a red-styled confirm `TextButton`.

### Integration Points
- `app/lib/components/trail/trail_dropdown.dart` — the menu itself; all of D-04…D-09, D-15…D-17.
- `app/lib/components/trail/trail_panel.dart:214` — the "Offline" pill re-gate (D-10).
- `app/lib/provider/trail/trail_provider.dart` (+ `.g.dart`) — retire `forceOffline` (D-20),
  add opportunistic metadata refresh (D-14).
- `app/lib/routes/trail_create_screen.dart` — widen the reconciliation gate (D-13).
- `app/lib/util/trail/route_location.dart` + `app/test/util/trail/route_location_test.dart` —
  remove the `forceOffline` options and the two tests added for them (D-20).
- `app/lib/provider/router_provider.dart`, `app/lib/routes/library_screen.dart` — remove
  `?offline=1` plumbing (D-20).

### Testing note
`app/test/` cannot open a real ObjectBox `Store` — the host test environment has no
`libobjectbox.dylib`, which is why no existing unit test touches one. Anything asserting on
library membership or stored-row reconciliation must be tested through a seam that does not need a
live Store, or verified on device.

</code_context>

<specifics>
## Specific Ideas

- **komoot's model was the reference** for the whole phase: "Store for offline use" is a *toggle on
  the Tour*, not a separate library object — switching it off "removes the tour data from your
  device while keeping the tour in your komoot account". There is no "source" to navigate to
  because the user never left it.
- **AllTrails' vocabulary discipline** was the reference for D-04/D-06: "Remove" is reserved
  exclusively for downloads, behind a two-step confirm, and unsaving a list item is a separate flow.
- The user's framing that produced D-19: *"We cannot allow a downloaded trail to be stale when we
  are online."*
- The user's originating principle, preserved in D-22: re-downloading consumes bandwidth and should
  therefore be an **active decision**, never automatic.

</specifics>

<deferred>
## Deferred Ideas

- **"Go to source" menu item** — the user's own initial proposal, dropped during exploration once
  the single-object model removed the gap it was meant to bridge. Recorded so it is not
  re-proposed without knowing why it went.
- **Staleness / "update available" indicator** — comparing the server `updated` timestamp against
  the stored row's to surface that a download is out of date. No researched app does this. Not in
  this phase.
- **Automatic GPX re-fetch when the track itself changed** — considered under the safety argument
  that a wrong route is not a convenience issue. Rejected for this phase as the only option that
  spends bytes without being asked; revisit if stale tracks prove to be a real problem.
- **Photo refresh by filename diff** — considered as a way to make photos auto-refresh nearly free:
  the trail record lists the server's photo filenames and PocketBase mints a new filename whenever
  a file is replaced, so diffing would fetch nothing in the common unchanged case and only the
  genuinely new files otherwise. **Not taken** for this phase: it requires retaining the server
  filenames on the entity, which means untangling the `photos` / `localPhotos` overloading that
  Phase 36's D-10 deliberately established (`TrailDownloadService` currently overwrites
  `entity.photos` with local paths at `trail_download_service.dart:148`). Revisit if stale photos
  prove to be a real problem — the decision was to keep every automatic action free of bytes.
- **Server-side cleanup of already-duplicated photos** — D-18 stops the duplication, but any
  trail already edited from the Library carries duplicate photos server-side and nothing prunes
  them. Out of scope; needs its own decision about touching existing user data.
- **Interim mitigation before this phase lands** — the photo duplication is live on `feature/app`
  today, widened by the 2026-08-04 `forceOffline` change. Pulling D-15 forward as a standalone fix
  was offered and not taken up during this session; it remains available.

</deferred>

---

*Phase: 38-downloaded-trails-as-state-not-objects*
*Context gathered: 2026-08-04*
