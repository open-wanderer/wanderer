---
slug: offline-waypoint-photos
status: awaiting_human_verify
trigger: "When accessing a downloaded trail while offline in the library waypoint photos are showing placeholders. If there are more than 3 photos in the waypoint they are all showing the placeholder. If it is 2 the second photo loads. In the picture gallery screen after tapping on the waypoint photos the first 3 photos are placeholders, all subsequent photos show properly. This issue does not occur when online."
created: 2026-09-01
updated: 2026-09-01
---

# Debug Session: offline-waypoint-photos

## Symptoms

- **Expected behavior:** Opening a downloaded (offline-available) trail from the library while the device is offline shows all waypoint photos from the local cache, both in the waypoint list/detail and in the fullscreen photo gallery.
- **Actual behavior:** Waypoint photos render as placeholders while offline.
  - Waypoint view: if a waypoint has >3 photos, *all* of them show the placeholder. With exactly 2 photos, the second one loads.
  - Photo gallery screen (after tapping a waypoint photo): the **first 3** photos are placeholders; every subsequent photo displays correctly.
- **Error messages:** None captured yet. No logcat/app console output collected.
- **Timeline:** Never worked — offline waypoint photos have shown placeholders for as long as this has been checked.
- **Reproduction:** Download a trail in the library → go offline (airplane mode / no network) → open the downloaded trail → view a waypoint with 3+ photos → tap a photo to open the gallery.
- **Does not reproduce when online.**

## Notable pattern

The count-dependent behavior (threshold around 3, and an inverted first-3-vs-rest split in the gallery) strongly suggests an index/offset mismatch or a bounded concurrency/prefetch window in the offline image resolution path — not a simple "file missing" failure. Any hypothesis must explain *both* the waypoint-view and gallery-view halves of the pattern.

## Current Focus

reasoning_checkpoint:
  hypothesis: "WaypointEntityMapping.toModel() (app/lib/entities/waypoint_entity.dart) emits BOTH `photos` (server filenames, unmodified) and `localPhotos` (downloaded local file paths) for a downloaded/library waypoint. waypoint_card.dart and waypoint_sheet.dart build `webPhotos` from `waypoint.photos` unconditionally and pass it to PhotoCollage alongside `waypoint.localPhotos`. PhotoCollage._allPhotos concatenates webPhotos BEFORE localPhotos, duplicating every downloaded photo (once as a network URL, once as a local file). Offline, the leading (web) copies fail to load (no connectivity, no prior HTTP cache) and render the broken-image placeholder via CachedNetworkImageProvider's errorBuilder; the trailing (local) copies, appended after, load fine from disk. Since the collage view caps its preview at 3 tiles and the gallery lists every item, whenever a waypoint's real (server-known) photo count is >=3, ALL visible collage tiles fall inside the leading web-only segment (all placeholders); in the gallery, the first N items (N = webPhotos.length, the real/server photo count) are always placeholders and every item after that (the local copies) succeeds -- exactly the reported 'first 3 placeholders, rest succeed' pattern for a 3-photo waypoint."
  confirming_evidence:
    - "app/lib/entities/waypoint_entity.dart WaypointEntityMapping.toModel() sets `photos: photos` and `localPhotos: localPhotos` unconditionally (both raw entity fields passed straight through)."
    - "app/lib/services/trail_download_service.dart downloadTrail() only ever sets `waypointEntity.localPhotos = paths` after download -- it never touches/clears `waypointEntity.photos`, so server filenames survive alongside the new local paths."
    - "Contrast with the TRAIL level (same file, TrailEntityMapping.toModel(), app/lib/entities/trail_entity.dart line 391): `localPhotos: localPhotos.isNotEmpty ? localPhotos : photos` and the model's `photos:` param is never set at all (always defaults to `[]`) -- trail-level deliberately routes everything through `localPhotos` and leaves `Trail.photos` empty for any DB-backed row, so trail_panel.dart's unconditional `webPhotos = trail.photos.map(...)` is always `[]` for a cached/downloaded trail. This asymmetry (fixed at trail level, never mirrored at waypoint level) is the mechanism gap."
    - "app/lib/components/trail/photo_collage.dart PhotoSource._allPhotos appends webPhotos entries before localPhotos entries -- confirmed order via source read."
    - "app/lib/components/trail/waypoint_card.dart and waypoint_sheet.dart both compute `webPhotos = waypoint.photos.map(getFileUrl...)` unconditionally, with no local-availability or connectivity gate, then pass `localPhotos: waypoint.localPhotos` alongside it to PhotoCollage -- confirmed via source read of both files."
    - "No connectivity/offline-aware branching exists anywhere in waypoint_card.dart, waypoint_sheet.dart, or photo_collage.dart (grep for isLocal/offline/connectivity turned up nothing relevant) -- rules out a race/timing hypothesis in favor of a structural duplication."
  falsification_test: "If a downloaded waypoint's WaypointEntity row is inspected (or logged) and `photos` is found EMPTY (not populated) for library rows, this hypothesis is wrong. Also falsified if PhotoCollage's `_allPhotos` orders localPhotos before webPhotos (would invert which segment shows placeholders)."
  fix_rationale: "Mirror the trail-level pattern at the display layer (not the entity/model layer, to avoid touching WaypointEntity.photos which is still legitimately read elsewhere for sync/upload diffing -- see trail_sync_provider.dart's use of waypointEntity.toModel().photos-adjacent flows and waypoint_provider.dart's updateWaypoint removedFilenames diff). In waypoint_card.dart and waypoint_sheet.dart, only build `webPhotos` from `waypoint.photos` when `waypoint.localPhotos` is empty -- i.e. prefer the local copies exclusively when present, exactly matching how trail.photos is effectively empty for any DB-backed trail. This removes the duplication at its point of use without touching entity/model semantics relied on elsewhere."
  blind_spots: "Have not run the app (offline verification requires a device; user builds/installs per project convention). Have not confirmed on-device that CachedNetworkImageProvider's errorBuilder fires immediately offline rather than after some delay/retry. Have not checked whether a partially-failed download (localPhotos.length < photos.length) is common enough to matter for the 'skip webPhotos entirely when localPhotos non-empty' fix -- accepted as an acceptable minor tradeoff matching the trail-level design's existing philosophy (best-effort, prefer local when any local exists)."

next_action: awaiting human on-device verification (download a trail, go offline, check a waypoint with 2 photos and one with >3 photos, in both the waypoint card/sheet collage and the fullscreen gallery).

## Evidence

- timestamp: 2026-09-01
  checked: app/lib/components/trail/photo_collage.dart (full file)
  found: PhotoCollage._allPhotos concatenates `webPhotos` (as PhotoSource.network) THEN `localPhotos` (as PhotoSource.local). Collage layout caps preview at 3 tiles (index 0,1,2) with a "+N" badge for the rest when total > 3. PhotoSource.network uses CachedNetworkImageProvider with errorBuilder returning a broken-image placeholder; PhotoSource.local uses Image.file with its own errorBuilder.
  implication: any waypoint whose real/server photo count >= 3 will have its first 3 collage tiles land entirely within the web-sourced segment of the list, regardless of how many local copies also exist further down the list.

- timestamp: 2026-09-01
  checked: app/lib/entities/waypoint_entity.dart (full file)
  found: WaypointEntityMapping.toModel() passes both `photos: photos` and `localPhotos: localPhotos` straight from the entity's raw columns with no fallback/preference logic.
  implication: a WaypointEntity row that has both server filenames (photos) and downloaded local paths (localPhotos) populated (i.e. any waypoint of a downloaded/library trail) produces a Waypoint model with BOTH lists non-empty simultaneously.

- timestamp: 2026-09-01
  checked: app/lib/services/trail_download_service.dart downloadTrail()
  found: "For the TRAIL itself: `entity.photos = localPaths;` (raw server-filename column is overwritten with local download paths). For each WAYPOINT: only `waypointEntity.localPhotos = paths;` is set -- `waypointEntity.photos` (server filenames) is never touched/cleared."
  implication: trail-level download deliberately repurposes/clears the photos column so no duplication happens downstream; waypoint-level download does not apply the same treatment, leaving server filenames and local paths simultaneously present.

- timestamp: 2026-09-01
  checked: app/lib/entities/trail_entity.dart TrailEntityMapping.toModel() (line ~342-409), app/lib/components/trail/trail_panel.dart (webPhotos + PhotoCollage call)
  found: "`Trail.toModel()` never sets the model's `photos:` constructor param (always defaults to `[]`) and instead exposes everything through `localPhotos: localPhotos.isNotEmpty ? localPhotos : photos` (entity-level fallback). trail_panel.dart then unconditionally builds `webPhotos = trail.photos.map(...)` (always empty for any DB-backed trail) and passes `localPhotos: trail.localPhotos` to PhotoCollage -- no duplication occurs at the trail level."
  implication: this confirms the trail-level design intentionally avoids the exact duplication bug seen at waypoint level; the fix was applied once (trail) and never mirrored to the structurally identical waypoint case -- a classic asymmetric-fix pattern.

- timestamp: 2026-09-01
  checked: app/lib/components/trail/waypoint_card.dart, app/lib/components/trail/waypoint_sheet.dart (full files)
  found: "Both widgets compute `webPhotos = waypoint.photos.map((p) => waypoint.getFileUrl(user.serverUrl, p, thumb: '200x0') ?? '').toList()` unconditionally (no gate on localPhotos or connectivity), then call `PhotoCollage(localPhotos: waypoint.localPhotos, webPhotos: webPhotos)`."
  implication: confirms the duplication happens at both of the two waypoint-photo display call sites (card and sheet), consistent with the bug being reported in both waypoint-list/detail view and (via _openGallery) the photo gallery.

- timestamp: 2026-09-01
  checked: app/lib/provider/waypoint/waypoint_provider.dart (WaypointSave.updateWaypoint), app/lib/provider/trail/trail_sync_provider.dart (waypoint upload loop, lines ~356-406)
  found: "`waypoint.photos` is read elsewhere for legitimate non-display purposes: updateWaypoint() diffs `oldWaypoint.photos` vs `newWaypoint.photos` to compute `removedFilenames` for the server PATCH; trail_sync_provider's upload loop only calls `waypointEntity.toModel()` for waypoints with `isLocalId(waypointEntity.id) == true` (never-yet-synced waypoints, whose `photos` is empty by construction since they have no server id yet)."
  implication: a fix must NOT blank/mutate `WaypointEntity.photos` or `Waypoint.photos` at the entity/model layer -- that field is still legitimately consumed by the sync/edit-diff code paths. The fix belongs at the display (widget) layer only, gating `webPhotos` construction on `waypoint.localPhotos` being empty, not by altering what `toModel()` returns.

## Eliminated

- hypothesis: "Bounded concurrency/prefetch window during photo download (trail_download_service.dart) causes the first N photo downloads to fail while later ones succeed, explaining the index-dependent placeholder pattern."
  evidence: "_downloadPhotos in trail_download_service.dart uses Future.wait over all URLs with no concurrency limiter, and any download failure for a given URL simply drops that entry from the returned localPaths list (filtered via whereType<String>()) rather than leaving a null placeholder at a fixed index -- so a download-time failure could not by itself explain a threshold consistently anchored at index 3 or account for the pattern reproducing 'every time', independent of network conditions during the original download. The duplication hypothesis (both webPhotos and localPhotos populated and concatenated) explains the exact index behavior without requiring any download failure at all."
  timestamp: 2026-09-01

## Resolution

- root_cause: "WaypointEntityMapping.toModel() (app/lib/entities/waypoint_entity.dart) exposes both `photos` (server filenames) and `localPhotos` (downloaded local file paths) simultaneously for a downloaded/library waypoint, unlike TrailEntityMapping.toModel() which deliberately routes everything through `localPhotos` and leaves `Trail.photos` empty for any DB-backed row. waypoint_card.dart and waypoint_sheet.dart unconditionally build `webPhotos` from `waypoint.photos` and pass it to PhotoCollage alongside `waypoint.localPhotos`; PhotoCollage concatenates webPhotos before localPhotos, duplicating every downloaded photo. Offline, the leading (network) copies fail and show the broken-image placeholder while the trailing (local) copies load fine -- producing the exact 'first N placeholders, rest succeed' / 'waypoints with >=3 photos show all placeholders' pattern."
- fix: "Gate `webPhotos` construction in waypoint_card.dart and waypoint_sheet.dart on `waypoint.localPhotos.isEmpty`, mirroring the trail-level pattern (prefer local copies exclusively when present, never build/display redundant network URLs for photos already available locally)."
- verification: "Self-verified: `flutter analyze` on the two changed files and on the whole app reports no errors (only 13 pre-existing infos unrelated to this change). `flutter test` runs the full suite: 1093 passed, 0 failed. No unit/widget test exists specifically for waypoint_card.dart/waypoint_sheet.dart's photo logic, so this fix has not yet been exercised by an automated test -- on-device confirmation (offline library trail, waypoints with 2 and >3 photos, both collage and gallery views) is required from the user per project convention (user builds/installs; agent hands off after analyze+test)."
- files_changed:
    - app/lib/components/trail/waypoint_card.dart
    - app/lib/components/trail/waypoint_sheet.dart
