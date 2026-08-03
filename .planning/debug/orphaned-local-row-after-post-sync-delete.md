---
status: diagnosed
trigger: "1. create trail while offline 2. go online 3. trail is uploaded correctly 4. delete trail 5. trail is now deleted server side 6. an orphaned trail now permanently exists in the 'own trail' list. Clicking on it leads to an idefinite loading spinner."
created: 2026-08-03T00:00:00Z
updated: 2026-08-03T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED (two independent defects)
  A) `deleteLocalTrailRow` has exactly ONE call site — `TrailSync.deleteUnsynced`
     (`trail_sync_provider.dart:365`) — which `_deleteTrail` only reaches on its
     `isUnsyncedState(trail.syncState)` branch. A trail that HAS synced takes the
     third (server-DELETE) branch, which touches no local storage. The post-sync
     ObjectBox row is deliberately KEPT by `markTrailSynced` (SYNC-05) and still
     carries `owner == accountId`, so `readOwnLocalTrails` re-emits it forever.
  B) `trailProvider`'s ObjectBox fallback is gated on `savedByUserIds`, which a
     local capture never gains (D-10). So the 404 rethrows, and Riverpod 3's
     `defaultRetry` renders 10 loading emissions (~38 s) before the error screen.
test: traced saveNewLocalTrail -> drain success write-back -> `_deleteTrail` branch
  selection -> all three delete implementations -> `readOwnLocalTrails` -> merge ->
  `_onTrailSelect` -> `trailProvider` 404 -> riverpod retry -> `AsyncValue.when`
expecting: n/a — diagnosis complete
next_action: hand to plan-phase --gaps (goal: find_root_cause_only; no fix applied)

## Symptoms

expected: Deleting a trail that was created offline and has since uploaded successfully removes both the server record and the local ObjectBox row, so it disappears from the own-trails list.
actual: The server record is deleted; the local ObjectBox row survives permanently (across restarts) and keeps rendering in the own-trails list. Tapping it shows a loading spinner that never appears to resolve.
errors: none reported beyond the indefinite spinner
reproduction: UAT Test 5 in `.planning/phases/36-local-first-recording-automatic-upload/36-UAT.md` — create a trail offline, go online, let it upload, open it, delete it from the trail dropdown, return to `/profile/<handle>/trails`.
started: Discovered after the initial 4-test UAT pass of Phase 36 (local-first recording + automatic upload)

## Eliminated

- hypothesis: "The orphan is a stale Meilisearch hit from the network half of the own-trails list, not a local ObjectBox row."
  evidence: `db/main.go:117` registers `app.OnRecordAfterDeleteSuccess("trails")` -> `hooks.DeleteTrailHandler`, whose body calls `client.Index("trails").DeleteDocument(record.Id, nil)` (`db/hooks/trails.go:125`). The search document is removed with the record, so `POST /profile/<handle>/trails` cannot keep returning it. Independently, the local row is provably retained and provably rendered (see Evidence), so the local half alone is sufficient to produce the orphan. `mergeOwnTrails` (`own_trails_merge.dart:40-44`) puts local rows FIRST and dedupes network hits against non-empty local ids, so even if a stale hit existed it would be the one suppressed — exactly one orphan entry, as reported.
  timestamp: 2026-08-03

- hypothesis: "The delete took `_deleteTrail`'s un-download branch (`if (trail.isLocal)`), which would explain the local row surviving."
  evidence: That branch cannot have run, for two reasons. (1) It never issues a server DELETE, yet the user observed the record removed server-side. (2) `TrailDropdown` is instantiated at exactly one place — `trail_detail_screen.dart:98` — and is handed the model from `trailProvider(widget.id)`, i.e. the model parsed from `GET /trail/<id>`. `Trail.isLocal` carries `@Default(false)` (`models/trail.dart:125`) and is set true ONLY by `TrailEntity.toModel()` (`trail_entity.dart:323`), so a network-loaded trail has `isLocal == false`. Also note the branch would actually have DELETED the row had it run: `TrailLibraryNotifier.deleteTrail` (`trail_library_provider.dart:65-84`) computes `libraryMembersAfterDelete(entity.savedByUserIds, userId)`, which is empty for a local capture, and therefore calls `box.remove(entity.obxId)`.
  timestamp: 2026-08-03

- hypothesis: "A startup/reconciliation sweep should have cleaned the row up and is failing."
  evidence: No such sweep exists. `grep -rn "unsyncedLocalIds" app/lib` returns only `main.dart:122`, which passes it to `sweepOrphanedUnsyncedPhotos(keepLocalIds: ...)` — a PHOTO-DIRECTORY sweep, not a row sweep. There is no code anywhere in `app/lib` that reconciles a synced local row against server state. Nothing was supposed to clean it up; the cleanup was never written.
  timestamp: 2026-08-03

- hypothesis: "The spinner is genuinely infinite (an unresolved future or a swallowed error)."
  evidence: It is bounded, but only just. `TrailNotifier.build` (`trail_provider.dart:96`) `rethrow`s the `DioException`. `ProviderElement.triggerRetry` (`riverpod-3.2.1/lib/src/core/element.dart:699-731`) consults `ProviderContainer.defaultRetry` (`provider_container.dart:831-845`): 10 attempts, and `DioException` is neither an `Error` nor a `ProviderException`, so every attempt is taken. Each scheduled retry emits `AsyncLoading._(..., error: (..., retrying: true))`, which `AsyncValue.when` (`async_value.dart:250-260`) treats as `isReloading` with the default `skipLoadingOnReload: false` -> `loading()`. Delays are 200/400/800/1600/3200/6400 x5 = 38.2 s of pure spinner (plus 10 fast 404 round trips; `connectTimeout` is 8 s but a 404 returns immediately). After the 10th, `AsyncError(retrying: false)` renders `WandererError` (`trail_detail_screen.dart:200-202`). So "indefinite" is a ~38 s spinner with no cancel affordance — a genuine second defect, but not an unresolved future.
  timestamp: 2026-08-03

## Evidence

- timestamp: 2026-08-03
  checked: `app/lib/util/local_trail_store.dart:579-600` (`markTrailSynced`) and `:480-499` (`writeServerTrailId`), i.e. what the row looks like after a successful drain
  found: The row is KEPT, not deleted — its doc comment says so explicitly ("The row is kept, not deleted -- its `obxId` never changes across the transition, which is exactly what SYNC-05's 'keeps its identity in place' means concretely"). After a successful drain the row holds: `id` = the SERVER id (stamped by `writeServerTrailId` at `trail_sync_provider.dart:220`), `localId` = `local-<micros>-<seq>` (untouched), `owner` = the capturing account id (untouched since `saveNewLocalTrail`), `syncState` = `TrailSyncState.synced`, `syncAttempts` = 0, `localPhotos` = `[]`, `photos` = the server filename list, `savedByUserIds` = `[]`.
  implication: A synced-from-local trail is a full ObjectBox row keyed by the SERVER id but with EMPTY library membership. That combination is what every downstream check gets wrong.

- timestamp: 2026-08-03
  checked: `grep -rn "deleteLocalTrailRow" app/lib app/test`
  found: Exactly two hits — the definition at `local_trail_store.dart:298`, and one call site at `trail_sync_provider.dart:365`, inside `TrailSync.deleteUnsynced`.
  implication: The ONLY way a local trail row is ever removed is `deleteUnsynced`. If a delete flow does not reach `deleteUnsynced`, the row is immortal.

- timestamp: 2026-08-03
  checked: `app/lib/components/trail/trail_dropdown.dart:246-333` (`_deleteTrail`), branch by branch, against a network-loaded synced trail
  found: Branch 1 `if (isUnsyncedState(trail.syncState) && trail.localId != null)` — FALSE. The model came from `GET /trail/<id>`, and `Trail.syncState` carries `@Default(TrailSyncState.synced)` with `includeFromJson: false` (`models/trail.dart:137-139`), while `localId` carries `includeFromJson: false` and is therefore `null` (`:133`). Branch 2 `if (trail.isLocal)` — FALSE (`@Default(false)`, `models/trail.dart:125`). Branch 3 (lines 310-332) runs: `trailSaveProvider.deleteTrail(trail)` -> `apiProvider.delete('/trail/${trail.id}')` (`trail_save_provider.dart:202-204`), then `ref.invalidate(trailLibraryProvider)`, `ref.invalidate(trailSearchProvider)`, `ref.invalidate(profileTrailsProvider('@$handle'))`, then `router.pop()`.
  implication: THE ROOT CAUSE. Branch 3 is a pure network DELETE plus four invalidations. It never touches ObjectBox. `trailSaveProvider.deleteTrail` is a two-line method with no local-storage awareness at all. The local row is orphaned by construction — and the invalidation on line 318 actively re-renders the list that then shows it.

- timestamp: 2026-08-03
  checked: `app/lib/util/local_trail_store.dart:360-393` (`readOwnLocalTrails`) against the surviving row
  found: The query is `TrailEntity_.owner.equals(accountId) | TrailEntity_.savedByUserIds.containsElement(accountId)`, kept when `entity.owner == accountId`. There is NO `syncState` predicate and no server-existence check. The orphan's `owner` still equals the signed-in account, so it is returned on every read, forever, including after an app restart (ObjectBox is on disk).
  implication: This is why the orphan is PERMANENT rather than transient. The invalidation at `trail_dropdown.dart:318` faithfully rebuilds `ProfileTrailsNotifier`, whose `build()` (`profile_trails_provider.dart:85-96`) re-runs `readOwnLocalTrails` and re-emits the orphan. The user's own successful delete is what makes the orphan visible.

- timestamp: 2026-08-03
  checked: `app/lib/util/own_trails_merge.dart:36-45` (`mergeOwnTrails`)
  found: `[...local, ...dedupedNetwork]` — local rows are never filtered, only network hits are deduped against non-empty local ids.
  implication: The orphan renders as the FIRST card in the own-trails list. Because `syncState == synced` it wears no sync badge, so it is visually indistinguishable from a real trail (its thumbnail falls through to `NetworkImage` of a now-404 file, since `markTrailSynced` cleared `localPhotos` and `toModel()`'s fallback hands `trail_list_item.dart:37-42` server filenames that fail `File(...).existsSync()`).

- timestamp: 2026-08-03
  checked: `app/lib/routes/profile_trail_screen.dart:53-62` (`_onTrailSelect`) with the orphan
  found: `trail is Trail && isUnsyncedState(trail.syncState)` is FALSE (the row is `synced`), so the unsynced divert is skipped and `context.push('/trail/${trail.id}')` runs with the SERVER id of a record that no longer exists.
  implication: The orphan is "reachable-but-broken", not unaddressable. It routes correctly to `/trail/<serverId>`; the destination just 404s.

- timestamp: 2026-08-03
  checked: `app/lib/provider/trail/trail_provider.dart:61-97` (the `catch (_)` fallback)
  found: The ObjectBox fallback query is `TrailEntity_.id.equals(id) & TrailEntity_.savedByUserIds.containsElement(userId)`. `grep -rn "savedByUserIds =" app/lib` shows the field is written in exactly three places: `trail_download_service.dart:206` (download), `trail_library_provider.dart:81` (un-download), and `local_trail_store.dart:259` (carry-forward in `updateLocalTrail`). Neither `saveNewLocalTrail` nor any drain bookkeeping write ever sets it.
  implication: The fallback cannot find the very row that is causing the orphan — the row IS in the box, keyed by that exact `id`, but its `savedByUserIds` is empty, so the `containsElement` clause excludes it. `rethrow` on line 96 surfaces the 404. This is the same D-10 ownership/library-membership split flagged in `.planning/debug/unsynced-trail-skips-detail-screen.md`, biting a second time — this time for an ALREADY-SYNCED row.

- timestamp: 2026-08-03
  checked: `riverpod-3.2.1/lib/src/core/element.dart:699-731`, `provider_container.dart:831-845`, `async_value.dart:250-267`, against `trail_detail_screen.dart:62-203`
  found: `defaultRetry` = 10 attempts, min 200 ms doubling, capped at 6400 ms, skipped only for `Error` / `ProviderException`. `DioException` is a plain `Exception`, so all 10 fire. Each pending retry emits `AsyncLoading(..., retrying: true)`, which `when()` classifies as `isReloading` and — with the default `skipLoadingOnReload: false` — renders `loading()`. `trail_detail_screen.dart:196-199` is a full-screen `CircularProgressIndicator` with no cancel affordance and no back button (the AppBar lives inside the `data:` branch only).
  implication: SECOND, INDEPENDENT DEFECT. A 404 is a permanent condition, not a transient one, yet it is retried 10 times over ~38 s. The user is trapped on a bare spinner with no chrome for the whole duration. This defect exists for ANY deleted-elsewhere trail (e.g. deleted from the web UI), not only for orphans — the orphan just makes it trivially reproducible. It is the same class of finding as UAT gap 1 (`trailFilterProvider` retry storm, 36-09-PLAN), and 36-09 does not touch `trailProvider`.

- timestamp: 2026-08-03
  checked: `grep -rn "deleteTrail(" app/lib` — every delete entry point in the app
  found: Four call sites: `trail_dropdown.dart:306` (un-download), `trail_dropdown.dart:313` (server delete), `library_screen.dart:192` (un-download), and the two definitions. `TrailDropdown` is mounted only at `trail_detail_screen.dart:98`. `library_screen.dart` lists only `trailLibraryProvider` rows, i.e. rows with non-empty `savedByUserIds` — which the orphan is not.
  implication: The orphan is UNDELETABLE through every existing affordance. It is not in the library screen; its detail screen never reaches the `data:` branch so its dropdown never mounts; and the own-trails list item has only `InkWell(onTap: onTrailSelect)` with no menu (`trail_list_item.dart:71-73`). Only clearing app data or signing out removes it. This is what makes the gap a blocker rather than a cosmetic defect.

## Resolution

root_cause: |
  Two independent defects compound.

  (A) THE ORPHAN. After a successful drain, `markTrailSynced`
  (`app/lib/util/local_trail_store.dart:579-600`) deliberately KEEPS the local
  ObjectBox row (SYNC-05), leaving `owner == accountId`, `id == <server id>`,
  `syncState == synced` and `savedByUserIds == []`. Deleting that trail from the
  trail dropdown takes `_deleteTrail`'s THIRD branch
  (`app/lib/components/trail/trail_dropdown.dart:310-332`) — the model came from
  `GET /trail/<id>`, so `syncState` is the `synced` default and `isLocal` is the
  `false` default (`app/lib/models/trail.dart:125,137-139`), skipping both the
  unsynced branch and the un-download branch. That third branch is a pure network
  `DELETE /trail/{id}` (`app/lib/provider/trail/trail_save_provider.dart:202-204`)
  plus four `ref.invalidate` calls; it touches no local storage.
  `deleteLocalTrailRow` has exactly ONE call site in the whole app —
  `TrailSync.deleteUnsynced` at `app/lib/provider/trail/trail_sync_provider.dart:365`
  — which that branch never reaches. So the row survives, and because
  `readOwnLocalTrails` (`local_trail_store.dart:360-393`) filters only on
  `owner`/`savedByUserIds` with no `syncState` and no server-existence predicate,
  the invalidation at `trail_dropdown.dart:318` re-renders the list and re-emits
  the orphan. It persists across restarts and cannot be removed through any
  existing affordance: it is absent from `library_screen`, its detail screen never
  reaches the `data:` branch so `TrailDropdown` never mounts, and `TrailListItem`
  has no menu.

  (B) THE SPINNER. Tapping the orphan pushes `/trail/<serverId>` (correct routing;
  `syncState == synced` so `_onTrailSelect`'s unsynced divert is skipped).
  `TrailNotifier.build` 404s, and its ObjectBox fallback
  (`app/lib/provider/trail/trail_provider.dart:70-96`) is gated on
  `savedByUserIds.containsElement(userId)` — which a local capture never gains
  (D-10; `savedByUserIds` is written only by the download service and the library
  provider). The row it needs is in the box under that exact `id`, but the
  membership clause excludes it, so line 96 `rethrow`s. Riverpod 3's
  `defaultRetry` then retries the `DioException` 10 times
  (200/400/800/1600/3200/6400x5 = ~38 s), each pending retry emitting
  `AsyncLoading(retrying: true)`, which `AsyncValue.when`'s default
  `skipLoadingOnReload: false` renders as `loading()` —
  `trail_detail_screen.dart:196-199`, a bare full-screen spinner with no AppBar
  and no cancel. After the 10th it finally shows `WandererError`. So the spinner
  is bounded at ~38 s, not infinite, but it is a real defect in its own right and
  it fires for ANY trail deleted elsewhere (e.g. from the web UI), not just for
  orphans.
fix: NOT APPLIED (goal: find_root_cause_only)
verification: n/a
files_changed: []
