---
phase: 38-downloaded-trails-as-state-not-objects
reviewed: 2026-08-04T15:10:20Z
depth: standard
files_reviewed: 18
files_reviewed_list:
  - app/lib/components/trail/like_button.dart
  - app/lib/components/trail/trail_dropdown.dart
  - app/lib/components/trail/trail_panel.dart
  - app/lib/i18n/app_en.arb
  - app/lib/i18n/untranslated_messages.json
  - app/lib/provider/router_provider.dart
  - app/lib/provider/trail/trail_provider.dart
  - app/lib/routes/library_detail_screen.dart
  - app/lib/routes/library_screen.dart
  - app/lib/routes/trail_create_screen.dart
  - app/lib/routes/trail_detail_map_screen.dart
  - app/lib/routes/trail_detail_screen.dart
  - app/lib/store/local_trail_store.dart
  - app/lib/util/trail/route_location.dart
  - app/test/components/trail/trail_dropdown_delete_gate_test.dart
  - app/test/components/trail/trail_dropdown_menu_test.dart
  - app/test/components/trail/trail_panel_sync_badge_test.dart
  - app/test/util/trail/route_location_test.dart
findings:
  critical: 3
  warning: 12
  info: 0
  total: 15
status: issues_found
---

# Phase 38: Code Review Report

**Reviewed:** 2026-08-04T15:10:20Z
**Depth:** standard
**Files Reviewed:** 18
**Status:** issues_found

## Summary

`forceOffline` is genuinely gone (`grep -rn "forceOffline\|offline=1" app/lib app/test` returns nothing), the menu split is real and behaviourally tested, and the new store function carries forward every `TrailEntity` column `fromModel` cannot know. `flutter analyze` is clean on all changed files and all four reviewed test files pass.

But the phase rests on a load-bearing premise that this codebase contradicts in writing. Six new comments (`trail_dropdown.dart:63-66`, `:382-384`, `trail_panel.dart:203-207`, and the plan's D-10 references) assert that "unsynced and downloaded are mutually exclusive by construction". They are not: `TrailDownloadService.downloadTrail` (`app/lib/services/trail_download_service.dart:210-215`) explicitly carries `owner`, `localId`, `syncState`, `syncAttempts` and `syncNextAttemptAt` forward when it writes into an existing row, and `shouldDeleteUploadedRow`'s own doc comment (`app/lib/store/local_trail_store.dart:81-98`) spells out exactly when that overlap arises — "some account downloaded this trail while its upload was in flight ... possible from the moment `writeServerTrailId` stamps the server id". Because a `failed` row parks indefinitely (D-07), the overlap window is unbounded, not momentary.

Three destructive paths in this phase's new code are gated on that false premise, one of which reaches across accounts. `_allowDelete`'s unconditional `isUnsyncedState` escape hatch is, structurally, still a destructive gate decided by a field read off a shared cache row rather than by authorship — the exact shape the phase set out to eliminate.

Separately, `applyServerTrailToLibraryRow` prunes waypoints from a source (`TrailSaveResult.trail`) that is known to omit waypoints whose save failed, and it re-mints `@Unique(onConflict: replace)` `ActorEntity`/`CategoryEntity` rows on *every* trail detail open now that D-14 fires it opportunistically.

---

## Critical Issues

### CR-01: A non-authoring library member can trigger another account's unsynced-delete path, wiping that account's un-uploaded photos

**File:** `app/lib/components/trail/trail_dropdown.dart:385-387`, `:461-491`, `:511-526`
**Issue:**

`_allowDelete` bypasses the authorship check entirely for any trail whose `syncState` is not `synced`:

```dart
if (isUnsyncedState(widget.trail.syncState)) {
  return true;
}
```

The justification given ("Phase 36's D-10 guarantees unsynced and downloaded are mutually exclusive by construction, so this branch can never swallow a downloaded trail") is false. Trace the reachable state:

1. Account A captures a trail. The drain's `writeServerTrailId` stamps a real server id onto the row (`local_trail_store.dart:1094-1113`), leaving `owner = A`, `localId = L`, `syncState` still `pending`/`uploading`.
2. A later waypoint upload fails, `recordDrainFailure` parks the row as `failed` with no scheduled retry (D-07). The row now sits in this state indefinitely.
3. The trail is fetchable and indexed, so account B downloads it. `TrailDownloadService.downloadTrail` writes into the **same** row (`id` is `@Unique(onConflict: replace)`) and deliberately preserves `owner`, `localId` and `syncState` (`trail_download_service.dart:210-215`), adding `B` to `savedByUserIds`.

Now `TrailNotifier._readCached(id)` — scoped only by `savedByUserIds` — hands account B a `Trail` model carrying **account A's `localId` and `pending`/`failed` `syncState`** (`trail_provider.dart:120-150`, via `TrailEntity.toModel()` which copies `localId` and `syncState` with no owner filter). On the detail screen, `availableOffline` is true, so `isUnsynced` is true and:

- `_allowDelete` returns `true` for B with **no authorship check at all**.
- `_confirmDelete` shows `delete_trail_confirm` (the row has a server id).
- `_deleteTrail` takes the unsynced branch, `localId` is non-null, and calls `deleteUnsynced(L)`.

`TrailSync.deleteUnsynced` (`app/lib/provider/trail/trail_sync_provider.dart:522-595`) is only *partially* owner-scoped. `readLocalTrailServerId` and `deleteLocalTrailRow` are owner-filtered and correctly no-op for B — but the line after them is not:

```dart
deleteLocalTrailRow(store, localId, accountId: accountId); // no-op for B
await _deletePhotoDirBestEffort(localId);                  // NOT account-scoped
```

`deleteUnsyncedPhotoDir(localId)` (`app/lib/store/local_photo_store.dart:206`) resolves `<appDocs>/unsynced/<localId>/` — a path with no account component — and deletes it recursively. **Account A's un-uploaded trail photos are destroyed by account B's tap.** `deleteUnsynced` then returns `UnsyncedDeleteResult.deleted`, so `_deleteTrail` announces the deletion to the map providers and pops the route, reporting success for an operation that deleted nothing it was allowed to delete and everything it was not.

Note also that in this same state the dropdown's whole download family sits behind `if (!isUnsynced)` (`trail_dropdown.dart:206`), so *Remove download* is not offered for a row the account genuinely holds in its library — the destructive action is available while the safe one is hidden.

**Fix:** Stop deriving the escape hatch from `syncState` on a shared cache row. Gate it on ownership of the local row, and never on a value that another account's capture bookkeeping can supply:

```dart
bool _allowDelete(WidgetRef ref) {
  final user = ref.watch(authProvider).value;
  if (user == null) return false;

  // Device-only capture: allow deletion only when THIS account owns the
  // local row. `readOwnLocalTrail`/`readLocalTrailServerId` are already
  // owner-scoped; expose the same predicate here instead of trusting
  // `syncState`, which `TrailDownloadService` carries forward onto a row
  // another account has downloaded (see shouldDeleteUploadedRow's doc).
  final localId = widget.trail.localId;
  if (isUnsyncedState(widget.trail.syncState) && localId != null) {
    final store = ref.read(objectBoxProvider);
    final accountId = currentAccountId(store);
    return accountId != null &&
        readOwnLocalTrail(store, localId: localId, accountId: accountId) != null;
  }

  return widget.trail.author == user.actorId;
}
```

Independently, `deleteUnsynced` must not delete the photo directory when `deleteLocalTrailRow` matched no row. Have `deleteLocalTrailRow` return a `bool` and gate `_deletePhotoDirBestEffort`/`UnsyncedDeleteResult.deleted` on it.

---

### CR-02: `applyServerTrailToLibraryRow` prunes waypoints from a list that is known to omit failed saves — silently destroying waypoints from the downloaded copy

**File:** `app/lib/store/local_trail_store.dart:700-719`; trigger at `app/lib/routes/trail_create_screen.dart:800-812`
**Issue:**

The D-13 trigger passes `result.trail` straight into `applyServerTrailToLibraryRow`. `TrailSave.updateTrail` builds that model's `expand.waypointsViaTrail` as `finalWaypoints` (`app/lib/provider/trail/trail_save_provider.dart:150-191`), which **omits** any waypoint whose create or update threw, and any `diff.updated` entry whose `old` lookup returned null:

```dart
for (final wp in diff.updated) {
  final old = ...firstOrNull;
  if (old == null) continue;                       // dropped, not even counted
  try { finalWaypoints.add(await updateWaypoint(...)); }
  catch (_) { hadFailures = true; }                // dropped
}
```

`finalWaypoints` is non-null (an empty or partial `List`), so the store function takes the pruning branch and deletes every `WaypointEntity` whose `id` is absent from it:

```dart
final newIds = entity.waypoints.map((w) => w.id).toSet();
for (final oldWaypoint in existing.waypoints) {
  if (!newIds.contains(oldWaypoint.id)) waypointBox.remove(oldWaypoint.obxId);
}
```

A single failed waypoint `PATCH` therefore removes a waypoint that still exists on the server from the hiker's offline copy, orphans its `library/<id>/waypoints/<key>/` photo files on disk, and does so under a warning toast (`some_waypoints_failed_to_save`) that says nothing about local deletion. The hiker discovers the missing waypoint offline, in the field, with no way to restore it.

This is precisely why the sibling `applyNetworkEditToLocalRow` deliberately never touches `waypoints` (`local_trail_store.dart:548-556`, "clobbering any of them here would ... silently discard data this function was never given"). The new function does not follow that precedent even though it consumes the same `result.trail`.

**Fix:** Only prune when the incoming waypoint set is authoritative. Either (a) plumb a `prune` flag that the D-14 fetch trigger sets and the D-13 save trigger clears, or (b) refuse to prune when the caller reports partial failure:

```dart
void applyServerTrailToLibraryRow(
  Store store, {
  required String accountId,
  required Trail trail,
  /// False when the caller's waypoint list may be incomplete (a partial
  /// save). Absent waypoints are then left alone rather than deleted.
  bool waypointsAreAuthoritative = true,
}) {
  ...
  if (waypointsAreAuthoritative) {
    final newIds = entity.waypoints.map((w) => w.id).toSet();
    for (final oldWaypoint in existing.waypoints) {
      if (!newIds.contains(oldWaypoint.id)) waypointBox.remove(oldWaypoint.obxId);
    }
  } else {
    for (final oldWaypoint in existing.waypoints) {
      if (!entity.waypoints.any((w) => w.id == oldWaypoint.id)) {
        entity.waypoints.add(oldWaypoint);
      }
    }
  }
```

and at the `trail_create_screen.dart` call site pass `waypointsAreAuthoritative: !result.hadWaypointFailures`.

---

### CR-03: "Remove download" can destroy a not-yet-uploaded capture, contradicting its own confirm copy

**File:** `app/lib/routes/library_screen.dart:162-215`, `app/lib/components/trail/trail_dropdown.dart:318-356`
**Issue:**

Both new confirm dialogs promise `remove_download_confirm_body`: *"This removes the downloaded copy from this device. The trail itself is not deleted."* Both then call `ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id)`.

`TrailLibraryNotifier.deleteTrail` (`app/lib/provider/trail/trail_library_provider.dart:58-94`) queries the row by `id` **unscoped** and, when this account is the last member, removes the whole row:

```dart
if (remaining.isEmpty) {
  box.remove(entity.obxId);   // whole TrailEntity, no owner check
  return false;
}
```

In the CR-01 overlap state (owner set, `localId` set, `syncState == failed`, `savedByUserIds == [A]`), that row *is* the pending capture. Removing it:

- destroys the queued/failed upload — `selectDrainCandidates` can never find it again, so the recording is permanently lost;
- leaks its `WaypointEntity` children (this is a documented shortcut: see `retireUploadedLocalTrail`'s doc, `local_trail_store.dart:784-790`, "copying that shortcut here would leak a set of waypoints");
- leaves `<appDocs>/unsynced/<localId>/` orphaned, since `deleteTrail` only cleans `library/<id>/`.

The dropdown hides *Remove download* for an unsynced trail (`trail_dropdown.dart:206`), but `library_screen.dart`'s long-press sheet has **no such guard** — its `Remove` `ListTile` (`:162-174`) fires unconditionally for any row `trailLibraryProvider` returns, and that provider filters only on `savedByUserIds`.

**Fix:** Make `deleteTrail` refuse to remove a row that is still a live capture, and drop only the membership:

```dart
final remaining = libraryMembersAfterDelete(entity.savedByUserIds, userId);
final isLiveCapture =
    entity.localId != null && entity.syncState != TrailSyncState.synced;

if (remaining.isEmpty && !isLiveCapture) {
  box.remove(entity.obxId);
  return false;
}
entity.savedByUserIds = remaining;
box.put(entity);
return remaining.isNotEmpty;
```

(and delete `library/<id>/` only when `remaining.isEmpty && !isLiveCapture`). Add the same `if (!isUnsyncedState(trail.syncState))` guard to `library_screen.dart`'s `Remove` tile that `trail_dropdown.dart:206` already applies.

---

## Warnings

### WR-01: A failed *refresh* now renders a bare, un-Scaffolded error page — the opposite of what the new comment claims

**File:** `app/lib/routes/trail_detail_screen.dart:107-126`
**Issue:** The new guard only catches `hasError && !hasValue`. `AsyncLoader.build` (`app/lib/components/async_loader.dart:25-27`) checks `asyncValue.hasError` **first**, with no `hasValue` check, and returns `errorBuilder(...)` — a bare `WandererError` with no `Scaffold`, no `AppBar`, no back button and no stack trace. So the case the comment describes ("On a refresh AsyncLoader prefers the retained `.value`") is exactly the case that regresses: `ref.invalidate(trailProvider(trail.id))` after an edit (`trail_dropdown.dart:193`) followed by a dropped connection leaves the user on a dead-end screen. The previous `trailAsync.when(...)` at least wrapped the error in a `Scaffold`.

**Fix:** Widen the guard so AsyncLoader never sees an error state:

```dart
if (trailAsync.hasError) {
  return Scaffold(
    appBar: AppBar(leading: IconButton(
      icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
      onPressed: () => context.pop(),
    )),
    body: WandererError(err: trailAsync.error!, stack: trailAsync.stackTrace),
  );
}
```

### WR-02: `ref.invalidate` after `await context.push(...)` with no `mounted` guard

**File:** `app/lib/components/trail/trail_dropdown.dart:147-152`, `:192-193`
**Issue:** Both Edit branches `await context.push(...)` and then call `ref.invalidate(...)` with no `if (!mounted) return;`. `context.push`'s future completes when the pushed route pops; if this dropdown was disposed in the meantime (the editor did a `pushReplacement`, an account switch tore the screen down, the detail route was popped by a concurrent delete) the `ref` access throws `StateError: Cannot use "ref" after the widget was disposed`. The analyzer does not catch it because `ref` is not a `BuildContext`. The codebase's own convention is the opposite — see `trail_detail_screen.dart:198` (`if (!mounted) return;` before the identical `ref.invalidate(localTrailProvider(lid))`).

**Fix:**
```dart
await context.push('/trail/create/edit', extra: fetched);
if (!mounted) return;
ref.invalidate(trailProvider(trail.id));
```
Apply to both branches.

### WR-03: `navCacheJson` is carried forward while `gpxData` is replaced — the cached Valhalla route can outlive the track it was computed for

**File:** `app/lib/store/local_trail_store.dart:683-688`
**Issue:** D-14's whole point is to refresh a downloaded row's `gpxData` from the server ("D-14's track refresh"). But two lines earlier the function preserves `navCacheJson`, which `TrailDownloadService` computed by POSTing `buildNavShape(gpx.allPoints)` to `/valhalla/navigate` for the *old* track (`trail_download_service.dart:155-180`). After a route edit on the server, the offline navigation cache describes maneuvers for a path the hiker is no longer on — and offline is exactly when that cache is the only source. The doc comment lists `navCacheJson` among the "carry forward" set with no discussion of this coupling.

**Fix:** Invalidate the cache whenever the track actually changes:

```dart
final incomingGpxData = trail.expand?.gpxData;
final gpxChanged = incomingGpxData != null &&
    incomingGpxData.isNotEmpty &&
    incomingGpxData != existing.gpxData;
entity.gpxData = gpxChanged ? incomingGpxData : (incomingGpxData?.isNotEmpty == true ? incomingGpxData : existing.gpxData);
// A nav cache computed for the previous shape is worse than none: drop it
// and let the next explicit Update recompute it (D-14a: no bytes here).
entity.navCacheJson = gpxChanged ? null : existing.navCacheJson;
```

### WR-04: Every downloaded-trail detail open now re-mints the author/category rows, orphaning other trails' `ToOne` targets

**File:** `app/lib/store/local_trail_store.dart:672`, `:690-698`; trigger at `app/lib/provider/trail/trail_provider.dart:83`
**Issue:** `applyServerTrailToLibraryRow` rebuilds the row via `TrailEntity.fromModel(trail)`. When the fetched model has an expanded author (it always does — `fetchServerTrail` requests `expand=...,author`), `fromModel` sets `entity.author.target = ActorEntity.fromModel(...)` — a **fresh** entity with `obxId == 0`. `Box.put` cascades new `ToOne` targets, and `ActorEntity.id` is `@Unique(onConflict: replace)` (`app/lib/entities/actor_entity.dart:10-12`), so the existing actor row is deleted and re-inserted under a new ObjectBox id. `TrailEntity.authorRecordId`'s own doc comment (`trail_entity.dart:73-84`) documents exactly this failure: "The `author` ToOne stores its target by that ObjectBox id, so it silently resolves to null afterwards — the trail's author line degraded to 'Unknown', the avatar to a bare grey circle."

`TrailEntity.toModel()` reads `expand.author` from `author.target?.toModel()` (`trail_entity.dart:367`), with no `authorRecordId` fallback, so `TrailSummary.summaryAuthorName` degrades to "Unknown" for **every other cached trail by that author**. `CategoryEntity` has the same `@Unique(onConflict: replace)` and the same cascade. Before this phase `TrailNotifier.build` never wrote to ObjectBox; D-14 now makes this fire on every detail open of a downloaded trail.

**Fix:** Do not let `fromModel`'s fresh relation targets reach the put. Resolve the existing rows by id inside the transaction:

```dart
final incomingAuthorId = trail.expand?.author?.id;
if (incomingAuthorId != null) {
  final q = store.box<ActorEntity>().query(ActorEntity_.id.equals(incomingAuthorId)).build();
  final existingActor = q.findFirst();
  q.close();
  // Reuse the persisted row (obxId != 0) so the put updates rather than
  // replaces it -- a replace re-mints obxId and orphans every other
  // TrailEntity.author ToOne (see TrailEntity.authorRecordId's doc).
  if (existingActor != null) entity.author.target = existingActor;
} else {
  entity.author.target = existing.author.target;
}
```
Same treatment for `category`.

### WR-05: A trail whose waypoints were all deleted server-side keeps them offline forever

**File:** `app/lib/store/local_trail_store.dart:720-725`
**Issue:** PocketBase omits an expand key entirely when the relation resolves to nothing, so a trail with zero waypoints comes back with `expand.waypointsViaTrail == null`, not `[]`. The `else` branch then re-adds every existing child untouched. Deleting a trail's last waypoint on the server therefore never propagates to any downloaded copy — the phantom waypoints render on the map and in `TrailTimeline` indefinitely. The doc comment frames this branch purely as "no waypoint expand in this response", which conflates "the server did not tell us" with "the server told us there are none".

**Fix:** Distinguish the two. `fetchServerTrail` already knows it requested the expand, so have it normalize an absent expand to `const []` for that field before returning, and keep the `else` branch only for callers that genuinely did not request waypoints.

### WR-06: Both new remove-download handlers drop an unawaited `Future`

**File:** `app/lib/components/trail/trail_dropdown.dart:355`, `app/lib/routes/library_screen.dart:214`
**Issue:** `TrailLibraryNotifier.deleteTrail` is `Future<void>` and performs real filesystem work (`Directory.delete(recursive: true)`, `trail_library_provider.dart:85-91`), which can throw (`FileSystemException` on a locked/absent path). Both call sites fire it and return immediately, so a failure surfaces only as an unhandled async error in the zone — the user sees the dialog close, no toast, and (because `state = state.where(...)` never runs on the throwing path) the trail still listed. Every other destructive action in this file is awaited and classified (`_deleteOnServer`, `_deleteTrail`).

**Fix:**
```dart
try {
  await ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id);
} catch (e) {
  debugPrint('trail_dropdown: removeDownload("${trail.id}") failed: $e');
  if (!mounted) return;
  ref.read(toastProvider.notifier).add(ToastMessage(
    type: ToastType.error,
    icon: FontAwesomeIcons.xmark,
    text: l18n.error_deleting_trail,
  ));
}
```

### WR-07: The `availableOffline: true` change lands on a screen nothing can reach, and its comment asserts otherwise

**File:** `app/lib/routes/library_detail_screen.dart:13`, `:36-45`
**Issue:** The new comment claims "without this the stored-on-device badge (D-10) would silently disappear from the Library detail sheet." No such sheet is reachable: `LibraryDetailScreen` is mounted only at `/library/:id` (`router_provider.dart:181-187`), and `grep -rn "'/library" app/lib` shows the only navigations are `router.go('/library')` and a prefix test in `wanderer_layout.dart`. `library_screen.dart` pushes `trailDetailLocation(trail)` → `/trail/<id>`, and did so before this phase too (the removed `forceOffline: true` only appended a query parameter). So this is untested dead code carrying a comment that will mislead the next reader into believing it is a live surface.

Compounding it, line 13 is `firstWhere((t) => t.id == id)` with **no `orElse`** — a `StateError` the moment the id is not in the library, with no error boundary above it.

**Fix:** Delete `LibraryDetailScreen` and the `/library/:id` route, or wire the library list to it and add `firstWhereOrNull` + a not-found state. Do not leave a third trail-detail surface that no test and no user exercises.

### WR-08: `Trail.isLocal`'s doc comment still names the three consumers this phase deleted as "load-bearing"

**File:** `app/lib/models/trail.dart:108-125`
**Issue:** The field doc still reads: *"Live consumers ... the 'downloaded' badge and the mutually-exclusive 'available offline' badge (`trail_panel.dart`), hiding the comments/summit-log TabBar for a local-only trail (`trail_panel.dart`), and — load-bearing — routing delete to `trailLibraryProvider.deleteTrail` (un-download) instead of a server delete (`trail_dropdown.dart`)."* All three were removed by this phase. `grep -rn isLocal app/lib` shows the only remaining consumers are thumbnail-path selection in `trail_card.dart:48` and `trail_list_item.dart:38`. A future reader restoring "load-bearing" behaviour from this doc reintroduces the exact bug the phase fixed.

**Fix:** Rewrite the "Live consumers" paragraph to list only the two thumbnail call sites, and add an explicit prohibition: *"Never gate a destructive action, a badge, or tab visibility on this flag — `TrailEntity.toModel()` hardcodes it `true` for every cached row (phase 38, D-01)."*

### WR-09: The new *Update* action re-enters a download path that deletes the existing offline copy on any failure

**File:** `app/lib/components/trail/trail_dropdown.dart:213-244`
**Issue:** *Update* calls `downloadingTrailIdsProvider.notifier.download(trail)` → `TrailDownloadService.downloadTrail`, whose failure handler is:

```dart
try { await Future.wait(futures); }
catch (e) {
  if (await trailDir.exists()) await trailDir.delete(recursive: true);
  rethrow;
}
```
(`app/lib/services/trail_download_service.dart:137-144`). `trailDir` is the **existing** `library/<id>/` directory. There is no connectivity precondition anywhere in `DownloadingTrailIds.download`. Two ways this bites:

- Offline tap: photo downloads fail → the whole `library/<id>/` tree is deleted while the `TrailEntity` row still points at those paths → a downloaded trail with broken images.
- Online tap after a failed fetch: `TrailNotifier.build`'s `catch` fallback means `trail` is the **cached** model, whose `photos` hold local absolute file paths (`trail_download_service.dart:148`). `trail.getFileUrl(baseUrl, p)` then builds nonsense URLs, every fetch 404s, and the same recursive delete runs.

Previously this menu offered only an inert-looking "Available offline" label; promoting it to a labelled *Update* action makes the destructive path an inviting affordance.

**Fix:** Make `downloadTrail`'s cleanup non-destructive for an existing library row (download into a temp dir and swap on success), and gate the *Update* item on `ref.watch(onlineStatusProvider)` plus `!isLocal`-style provenance of the model it is handed — at minimum refuse when `trail.photos.any((p) => p.startsWith('/'))`.

### WR-10: `_showContextMenu`'s `router` parameter is implicitly `dynamic`

**File:** `app/lib/routes/library_screen.dart:138`
**Issue:** `void _showContextMenu(BuildContext context, Trail trail, router)` — the third parameter has no type, so `router.push(location)` is an unchecked dynamic dispatch. A signature change on `GoRouter` (or passing the wrong object) becomes a runtime `NoSuchMethodError` inside a bottom sheet instead of a compile error. The method body was rewritten in this phase, so the type was in scope for the change.

**Fix:** `void _showContextMenu(BuildContext context, Trail trail, GoRouter router)` and import `package:go_router/go_router.dart`.

### WR-11: New query handles are not closed on the exception path

**File:** `app/lib/store/local_trail_store.dart:661-670`
**Issue:** `applyServerTrailToLibraryRow` builds a `Query`, calls `findFirst()`, then `close()`. If `findFirst()` throws, `close()` never runs and the native handle leaks; the enclosing `runInTransaction` will propagate the throw out of the function. This matches the rest of the file, so it is a consistency issue rather than a new defect — but this function is the one that now runs on *every* trail fetch (`trail_provider.dart:83`), so it is the highest-frequency instance of the pattern.

**Fix:** Wrap in `try/finally` here and, ideally, introduce a small `T withQuery<T>(Query q, T Function(Query) body)` helper so the whole file converges:

```dart
final query = trailBox.query(...).build();
final TrailEntity? existing;
try {
  existing = query.findFirst();
} finally {
  query.close();
}
```

### WR-12: The new tests never exercise the state the new comments claim is impossible

**File:** `app/test/components/trail/trail_dropdown_menu_test.dart:186-228`, `app/test/components/trail/trail_panel_sync_badge_test.dart:234-298`
**Issue:** Both suites are genuinely behavioural and neither was weakened to pass — but their fixture matrix is exactly the one that cannot surface CR-01. Every `availableOffline: true` fixture is `TrailSyncState.synced`, and every unsynced fixture is `availableOffline: false` (the harness default). The one state that matters — `availableOffline: true` **and** `syncState: pending/failed`, which `TrailDownloadService`'s carry-forward produces — is never constructed, so the suites cannot fail when the "mutually exclusive by construction" premise breaks. `trail_panel_sync_badge_test.dart`'s header even asserts the premise in prose (lines 21-27) without a fixture behind it.

`trail_dropdown_delete_gate_test.dart` is a source-text grep that asserts the absence of the string `isLocal` in `_deleteTrail`. That pins the *old* provenance flag by name while the surviving provenance-derived gate (`isUnsyncedState(widget.trail.syncState)` in `_allowDelete`) is not covered by any assertion at all.

**Fix:** Add the missing fixture and let it fail until CR-01 is fixed:

```dart
// A row that is BOTH a live capture and a library member -- the shape
// TrailDownloadService.downloadTrail produces when it writes into an
// already-existing capture row (trail_download_service.dart:210-215).
final unsyncedAndDownloaded = Trail.empty().copyWith(
  id: 'server-4',
  localId: 'local-9-0',
  author: 'someone-elses-actor-id',
  syncState: TrailSyncState.failed,
  lat: 1, lon: 1,
);

testWidgets('a downloaded row still carrying another account\'s capture '
    'bookkeeping offers Remove, and never Delete', (tester) async {
  await tester.pumpWidget(_harness(unsyncedAndDownloaded, availableOffline: true));
  await _openMenu(tester);
  expect(find.text('Remove'), findsOneWidget);
  expect(find.text('Delete'), findsNothing);
});
```


---

_Reviewed: 2026-08-04T15:10:20Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
