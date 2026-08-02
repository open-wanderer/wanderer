---
status: diagnosed
trigger: "Tapping on a not synced trail should open this trail like any other trail in the trail detail screen. From here the user can decide to edit it."
created: 2026-08-02T17:05:00Z
updated: 2026-08-02T17:40:00Z
---

## Current Focus

hypothesis: CONFIRMED — `_onTrailSelect` in `profile_trail_screen.dart` unconditionally short-circuits unsynced trails to `/trail/create/edit`; the branch exists because D-06 blanks the model id, making `/trail/<id>` unroutable for a local-only trail.
test: read the tap handler, the go_router route table, `TrailEntity.toModel()`, `TrailNotifier.build()`, `TrailLibraryNotifier.build()`, `saveNewLocalTrail`, and go_router's `canonicalUri`.
expecting: n/a — diagnosis complete
next_action: hand to plan-phase --gaps (goal is find_root_cause_only; no fix applied)

## Symptoms

expected: Tapping an unsynced (not-yet-uploaded) trail opens the normal trail detail screen, exactly like any other trail. From detail the user can choose to edit.
actual: Tapping an unsynced trail navigates straight to the edit screen (`/trail/create/edit`), bypassing detail entirely.
errors: none
reproduction: Test 2 in `.planning/phases/36-local-first-recording-automatic-upload/36-UAT.md` — in the own-trails list (`/profile/<handle>/trails`), tap an unsynced trail.
started: Introduced by Phase 36, PLAN 36-07 (line 193). Discovered during UAT of Phase 36.

## Eliminated

- hypothesis: "The detail route is reached but the detail screen redirects/falls back to edit"
  evidence: `router_provider.dart:359-364` maps `/trail/:id` straight to `TrailDetailScreen`, and `TrailDetailScreen` (`trail_detail_screen.dart`) contains no navigation to `/trail/create/edit`. The redirect at `router_provider.dart:97-129` is auth-only. The divert happens in the list, before any route is pushed.
  timestamp: 2026-08-02T17:20:00Z

- hypothesis: "Removing the branch alone restores detail navigation"
  evidence: `TrailEntity.toModel()` (`entities/trail_entity.dart:292`) sets `id: isLocalId(id) ? '' : id`, so an unsynced trail's `Trail.id` is `''`. `context.push('/trail/')` is normalized by go_router 17.3.0's `canonicalUri` (`path_utils.dart:138-156`) to `/trail`, and the route table has no `/trail` GoRoute (only `/trail/create`, `/trail/create/edit`, `/trail/:id`). Result would be a no-route error page, not the detail screen.
  timestamp: 2026-08-02T17:28:00Z

## Evidence

- timestamp: 2026-08-02T17:05:00Z
  checked: `.planning/phases/36-local-first-recording-automatic-upload/36-UAT.md`
  found: Test 2's own expected text says "Tapping an unsynced trail opens the offline-capable edit screen" — the current behaviour is the phase's own encoded truth (36-07-PLAN line 248: "Tapping an unsynced trail opens the offline-capable edit screen (REC-05)").
  implication: This is a spec/UX revision, not an implementation deviation.

- timestamp: 2026-08-02T17:12:00Z
  checked: `app/lib/routes/profile_trail_screen.dart:53-62`
  found: `_onTrailSelect` — `if (trail is Trail && isUnsyncedState(trail.syncState)) { context.push('/trail/create/edit', extra: trail); return; }` then `context.push('/trail/${trail.id}')`.
  implication: Single, explicit divert point. Exactly one call site.

- timestamp: 2026-08-02T17:16:00Z
  checked: `app/lib/entities/trail_entity.dart:288-292`, `app/lib/util/local_id.dart:16-36`
  found: D-06 blanks the id at the model boundary. ObjectBox row id is `local-<micros>-<seq>`; the model exposes `''` and carries identity in `Trail.localId`.
  implication: A local-only trail has no value that can appear in `/trail/:id`. The divert was a correct workaround for that constraint.

- timestamp: 2026-08-02T17:24:00Z
  checked: `app/lib/provider/trail/trail_provider.dart:17-98`
  found: `TrailNotifier.build(id)` does `GET /trail/$id`, and on any exception falls back to ObjectBox with `TrailEntity_.id.equals(id) & TrailEntity_.savedByUserIds.containsElement(userId)`.
  implication: Both halves fail for a local capture — the network call has no id to use, and the cache query filters on `savedByUserIds`.

- timestamp: 2026-08-02T17:30:00Z
  checked: `app/lib/util/local_trail_store.dart:153-195` (`saveNewLocalTrail`), `:323-341` (`readLocalTrail`), `app/lib/provider/trail/trail_library_provider.dart:26-28`
  found: `saveNewLocalTrail` sets `owner` and never `savedByUserIds` (D-10 keeps the two strictly separate). `trailLibraryProvider` queries only `savedByUserIds`. `readLocalTrail(store, localId)` already exists as a single-row lookup by `localId`.
  implication: `trailProvider`'s cache fallback and `trailLibraryProvider` can never see an unsynced trail. `readLocalTrail` is the ready-made data source a local detail path needs. Also: the drain never adds `savedByUserIds` either, so a trail captured on this device stays outside the offline library even after it syncs.

- timestamp: 2026-08-02T17:34:00Z
  checked: `app/lib/components/trail/trail_panel.dart`
  found: `TrailPanel` already degrades for local trails — `trail.isLocal` hides the summit-log/comments TabBar (line 240) and shows the offline chip (81-116); `PhotoCollage` takes `localPhotos` directly (60-66); map/elevation/waypoints come from `expand.gpx` and `expand.waypointsViaTrail`, both populated by `toModel()` from `gpxData`. But lines 300, 324 and 332 push `/trail/${trail.id}/map`.
  implication: The About tab would render a local-only trail as-is. The map/expand affordances would push `/trail//map` and fail.

- timestamp: 2026-08-02T17:36:00Z
  checked: `app/lib/routes/trail_detail_screen.dart:62-98,123-159`, `app/lib/components/trail/like_button.dart:28`, `app/lib/util/navigation_launch_util.dart:212,226`
  found: The detail chrome assumes a server id — `availableOffline`/`isDownloading` compare `t.id == trail.id`; the bottom bar renders a **Download** button whenever `!availableOffline` (always true for an unsynced trail); `LikeButton` reads `trailProvider(trail.id)` and PUTs `/trail-like`; navigation pushes `/trail/${trail.id}/navigate`.
  implication: D-17 ("an unsynced trail must not be downloadable") is enforced in the dropdown (`trail_dropdown.dart:122`) but NOT in the detail screen's bottom bar. Four id-dependent affordances need gating.

- timestamp: 2026-08-02T17:38:00Z
  checked: grep for `TrailDropdown` across `app/lib` and `app/test`; `app/lib/components/trail/trail_list_item.dart`
  found: `TrailDropdown` is instantiated in exactly one place — `trail_detail_screen.dart:98`. `TrailListItem` has only `InkWell(onTap: onTrailSelect)`; no menu. `test/components/trail/trail_dropdown_delete_gate_test.dart` is a source-inspection test (greps `trail_dropdown.dart` text), so it passes regardless of reachability.
  implication: UAT Test 3 is GENUINELY blocked. The whole of D-14/D-17 dropdown gating is unreachable in the running app for unsynced trails.

## Resolution

root_cause: |
  `_onTrailSelect` in `app/lib/routes/profile_trail_screen.dart:53-62` short-circuits every
  unsynced trail to `/trail/create/edit` before the `/trail/${trail.id}` push can run. This was a
  deliberate PLAN 36-07 decision (line 193, truth line 248) forced by D-06: `TrailEntity.toModel()`
  (`entities/trail_entity.dart:292`) blanks a local-sentinel id to `''`, and go_router canonicalizes
  `/trail/` to `/trail`, which matches no route. So the detail screen was never merely bypassed — it
  is currently unaddressable for a trail with no server id, and its data provider
  (`trailProvider`) has no path that can load one.
fix: (not applied — goal was find_root_cause_only)
verification: (n/a)
files_changed: []
