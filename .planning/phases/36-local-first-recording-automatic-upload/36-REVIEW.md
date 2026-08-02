---
phase: 36-local-first-recording-automatic-upload
reviewed: 2026-08-02T14:43:17Z
depth: standard
files_reviewed: 41
files_reviewed_list:
  - app/lib/components/trail/sync_status_chip.dart
  - app/lib/components/trail/trail_card.dart
  - app/lib/components/trail/trail_dropdown.dart
  - app/lib/components/trail/trail_list_item.dart
  - app/lib/entities/trail_entity.dart
  - app/lib/entities/waypoint_entity.dart
  - app/lib/i18n/app_en.arb
  - app/lib/main.dart
  - app/lib/models/global_search_models.dart
  - app/lib/models/trail.dart
  - app/lib/models/trail_summary.dart
  - app/lib/models/trail_sync_state.dart
  - app/lib/models/waypoint.dart
  - app/lib/provider/profile/profile_trails_provider.dart
  - app/lib/provider/trail/trail_save_provider.dart
  - app/lib/provider/trail/trail_sync_provider.dart
  - app/lib/routes/profile_trail_screen.dart
  - app/lib/routes/settings_account_screen.dart
  - app/lib/routes/settings_screen.dart
  - app/lib/routes/trail_create_screen.dart
  - app/lib/services/trail_download_service.dart
  - app/lib/util/account_scope_invalidation.dart
  - app/lib/util/local_id.dart
  - app/lib/util/local_photo_store_util.dart
  - app/lib/util/local_trail_store.dart
  - app/lib/util/own_trails_merge.dart
  - app/lib/util/sync_backoff.dart
  - app/lib/util/unsynced_signout_guard.dart
  - app/test/components/trail/sync_status_chip_test.dart
  - app/test/components/trail/trail_dropdown_delete_gate_test.dart
  - app/test/entities/trail_entity_test.dart
  - app/test/routes/settings_screen_signout_gate_test.dart
  - app/test/routes/trail_create_screen_local_save_gate_test.dart
  - app/test/services/trail_download_service_carry_forward_test.dart
  - app/test/util/account_scope_invalidation_test.dart
  - app/test/util/local_id_test.dart
  - app/test/util/local_photo_store_util_test.dart
  - app/test/util/local_trail_store_test.dart
  - app/test/util/own_trails_merge_test.dart
  - app/test/util/sync_backoff_test.dart
  - app/test/util/unsynced_signout_guard_test.dart
findings:
  critical: 4
  warning: 15
  info: 0
  total: 19
status: issues_found
---

# Phase 36: Code Review Report

**Reviewed:** 2026-08-02T14:43:17Z
**Depth:** standard
**Files Reviewed:** 41
**Status:** issues_found

## Summary

The phase adds a local-first capture path (`TrailEntity` + local sentinel id + `TrailSyncState` + app-owned photo copies) and a `keepAlive` drain notifier.

The parts that were designed as pure, testable policy are in good shape: `local_id.dart`'s whitelist regex genuinely blocks traversal, `sync_backoff.dart` is clamped and monotonic, `own_trails_merge.dart`'s non-empty-id dedupe guard is correct, and the Riverpod lifecycle rules called out in the brief are respected (no `late final` in any `build()`, `authProvider` read via `.value`).

The defects concentrate at the **model↔entity boundary** and in **partial-failure handling in the drain**, i.e. exactly the seams the design record spent the most words on:

- `TrailEntity` never persisted `category`, `subcategory`, `tags` or `completed`, and `toModel()` never restored them. Before this phase that was latent (the create screen always POSTed the in-memory `Trail`); now the entity round trip *is* the save path and the drain payload, so four user-entered fields are silently destroyed on every offline capture — including on the eventual server upload (CR-01).
- The drain's "resume from step" is not idempotent at two of its three steps. A waypoint whose record was created but whose photo upload failed is re-created on the next attempt (CR-02); a trail whose photos were uploaded in step 2 has its local photo record and files destroyed by the step-4 bookkeeping when the drain resumes (CR-03).
- Once a trail finishes uploading, the still-open create screen holds a stale `syncState: pending`, so a subsequent edit is routed back into `updateLocalTrail` — which preserves `syncState: synced` — and the edit never reaches the server, under a green "Trail saved successfully" toast (CR-04).

Account scoping itself holds up: every read in `local_trail_store.dart` is `owner`-filtered, `currentAccountId` is re-read fresh at each drain run, and the `unsyncedLocalIds` cross-account sweep exemption is correctly reasoned. The one soft spot is `_readOwnLocal` reading a cached actor id contrary to its own documented invariant (WR-06).

Several of the phase's tests are source-level greps (`indexOf`/`contains` on the .dart file). They pin structure, not behaviour — none of the four BLOCKERs below would be caught by any test in this phase.

## Critical Issues

### CR-01: Category, subcategory, tags and completed are silently destroyed by every local-first save and by the upload that follows

**File:** `app/lib/entities/trail_entity.dart:11-131` (no fields), `app/lib/entities/trail_entity.dart:167-214` (`fromModel`), `app/lib/entities/trail_entity.dart:218-264` (`toModel`), `app/lib/routes/trail_create_screen.dart:658-665`, `app/lib/provider/trail/trail_sync_provider.dart:125-130`

**Issue:** `TrailEntity` has no `category` (String id), `subcategory`, `tags` or `completed` column — only a `ToOne<CategoryEntity>` relation, which `fromModel` populates *only* from `trail.expand?.category` (always null on a locally-composed draft). `toModel()` correspondingly never sets `Trail.category`, `Trail.subcategory`, `Trail.tags`/`expand.tags`, or `Trail.completed`.

Trace one offline capture:

1. `_onSave` builds `updatedTrail` with `category: categorySelection?.category`, `subcategory:`, `completed:` and `expand.tags` read straight off the form (`trail_create_screen.dart:407-420`).
2. `saveNewLocalTrail` → `TrailEntity.fromModel(updatedTrail)` — all four values are dropped on the floor.
3. `_finishLocalSave` does `trail = readLocalTrail(store, _localId!)` and then `_formKey.currentState?.reset()` post-frame, so the **form visibly reverts**: category picker empty, tags gone, "completed" back to false. The user is shown a success toast at the same time.
4. The drain builds its upload payload from `entity.toModel()` (`trail_sync_provider.dart:126`), so `resolveTags(trailModel.expand?.tags ?? const [])` resolves an empty list and `toFormData` skips `category`/`subcategory` (`form_data_util.dart` emits them only `if (category != null)`) and sends `completed=false`.

So the trail reaches the server permanently stripped. This makes the phase's `resolveTags` reuse in the drain (D-06) dead code in practice — `expand.tags` can never be non-empty on a value that came out of `TrailEntity`.

**Fix:** persist the four scalars on the entity and restore them in `toModel()`.

```dart
// trail_entity.dart
  String? categoryId;      // the Trail.category id string, not the relation
  String? subcategoryId;
  bool completed = false;
  List<String> tagNames = []; // or a ToMany<TagEntity>; names round-trip to
                              // Tag(name:) so resolveTags() can create/reuse

  factory TrailEntity.fromModel(Trail trail) {
    final entity = TrailEntity(/* ... */);
    entity.categoryId = trail.category;
    entity.subcategoryId = trail.subcategory;
    entity.completed = trail.completed;
    entity.tagNames = trail.expand?.tags?.map((t) => t.name).toList() ?? const [];
    // ...
  }

  // toModel()
    category: categoryId,
    subcategory: subcategoryId,
    completed: completed,
    expand: TrailExpand(
      tags: tagNames.map((n) => Tag(name: n)).toList(),
      // ...
    ),
```

Add a round-trip test next to the existing `movingDuration` group in `test/entities/trail_entity_test.dart` asserting all four survive `fromModel -> toModel`.

---

### CR-02: A waypoint whose photo upload fails is created twice on the server on the next drain attempt

**File:** `app/lib/provider/trail/trail_sync_provider.dart:166-184`, `app/lib/provider/waypoint/waypoint_provider.dart:16-43`

**Issue:** `WaypointSave.create` performs two network calls: `PUT /waypoint` (creates the record) and, only when `localPhotos` is non-empty, `POST /waypoint/{id}/file`. If the second call throws — the common case, since photo uploads are the largest and most timeout-prone request in the sequence — `create` throws, so `_drainOne` never reaches `writeServerWaypointId` for that waypoint and the local `WaypointEntity.id` is still a `local-…` sentinel.

`_drainOne` catches, `recordDrainFailure` puts the trail back to `pending`, and the next drain pass re-enters step 3 with `if (!isLocalId(waypointEntity.id)) continue;` still true for that waypoint — so `PUT /waypoint` runs again and a **second waypoint record** is created on the server. With `kMaxSyncAttempts = 4` this can produce up to four duplicates of the same waypoint. There is no server-side idempotency key (RESEARCH.md Pitfall 3), which is exactly why the trail-level id write was pulled forward; the same reasoning was never applied to the waypoint level.

**Fix:** persist the waypoint's server id the instant the record exists, before its photos are uploaded — mirroring `writeServerTrailId`. Either split the provider call, or have `create` surface the created record on photo failure:

```dart
// waypoint_provider.dart
Future<Waypoint> create(Waypoint waypoint, {required String authorId, required String trailId}) async {
  // ... PUT /waypoint ...
  var created = Waypoint.fromJson(response.data);
  if (waypoint.localPhotos.isEmpty) return created;
  try {
    return await _uploadPhotos(created, newPhotos: waypoint.localPhotos, removedFilenames: const []);
  } catch (e) {
    throw WaypointPhotoUploadException(created, e); // carries the created id
  }
}

// trail_sync_provider.dart, step 3
try {
  createdWaypoint = await ref.read(waypointSaveProvider.notifier).create(...);
} on WaypointPhotoUploadException catch (e) {
  writeServerWaypointId(store, localId: localId,
      waypointLocalKey: waypointLocalKey, serverWaypointId: e.created.id);
  rethrow; // the trail still counts a failed attempt, but never re-creates
}
```

---

### CR-03: A resumed drain wipes the trail's photos from the local row and deletes the on-disk copies

**File:** `app/lib/provider/trail/trail_sync_provider.dart:132`, `app/lib/provider/trail/trail_sync_provider.dart:186-192`, `app/lib/util/local_trail_store.dart:494-513`

**Issue:**

```dart
var serverPhotoFilenames = entity.photos;          // line 132
if (isLocalId(entity.id)) { /* create; serverPhotoFilenames = created.photos */ }
// ...
markTrailSynced(store, localId: localId, serverPhotoFilenames: serverPhotoFilenames);
await deleteUnsyncedPhotoDir(localId);
```

On the **resume path** (`entity.id` is already a server id because a prior attempt got past step 2 and then failed at a waypoint, or the process was killed), the create block is skipped, so `serverPhotoFilenames` stays `entity.photos` — which is `[]` for a locally captured trail, because neither `TrailEntity.fromModel` nor `saveNewLocalTrail` ever writes `photos`. `markTrailSynced` then executes `entity.photos = []` and `entity.localPhotos = []`, and `deleteUnsyncedPhotoDir` removes the actual JPEGs from `<app-docs>/unsynced/<localId>/`.

Result: the trail is marked synced with zero photos on the device. Because `mergeOwnTrails` puts local rows first and drops any network hit sharing the id, the own-trails list shows that photoless local row **permanently** — `TrailCard`'s `localPhotos.isNotEmpty` is false and `summaryThumbnail` (`photos[0]`) is empty, so the placeholder SVG is rendered forever. The photos do exist server-side, but nothing on this device will ever read them back into that row.

This is reachable from a single failed waypoint upload followed by a successful retry, not just from a process kill.

**Fix:** re-read the server's photo list on the resume path instead of trusting the stale local `photos` column.

```dart
var serverPhotoFilenames = entity.photos;

if (isLocalId(entity.id)) {
  // ... create, serverPhotoFilenames = created.photos ...
} else {
  // Resume: the trail already exists server-side; fetch its authoritative
  // photo list rather than persisting this row's (empty) local column.
  final existing = await ref.read(apiProvider).get('/trail/${entity.id}');
  serverPhotoFilenames = Trail.fromJson(existing.data).photos;
}
```

Alternatively, persist `created.photos` into the row inside `writeServerTrailId` at step 2 so the resume path has a truthful `entity.photos` to carry forward.

---

### CR-04: An edit made after the upload finishes is written local-only and never reaches the server

**File:** `app/lib/routes/trail_create_screen.dart:425`, `app/lib/routes/trail_create_screen.dart:547-568`, `app/lib/util/local_trail_store.dart:65-73`, `app/lib/util/local_trail_store.dart:226-233`

**Issue:** `TrailCreateScreen` snapshots `trail` from `readLocalTrail` at the end of `_finishLocalSave`, at which point `syncState` is `pending`. The screen never watches the drain, so when the upload completes moments later the in-memory `trail` still says `pending`.

If the user then edits and saves again (a normal flow — fix a typo right after recording):

1. `resolveLocalSaveMode(updatedTrail)` sees `syncState != synced` → `LocalSaveMode.updateLocal`.
2. The branch is documented "Fully local-first: never touches the network, online or offline" and calls `updateLocalTrail`.
3. `updateLocalTrail` carries `existing.syncState` forward — which is now `synced` — so the row stays `synced`.
4. `selectDrainCandidates` filters on `dbSyncState != synced`, so the drain will never pick it up.

The edit lives on this device only, forever, and the user was shown `trail_saved_successfully`. Same failure occurs for an edit made *during* the drain: `_drainOne` built its payload from a snapshot taken before the edit (`trail_sync_provider.dart:126`) and marks the row synced afterwards.

**Fix:** decide the save mode from the persisted row, not the screen's snapshot, and re-queue on a post-sync edit.

```dart
// _onSave, before resolveLocalSaveMode
final localId = _localId;
final persisted = localId == null ? null : readLocalTrail(store, localId);
final saveMode = resolveLocalSaveMode(persisted ?? updatedTrail);
```

and, in `updateLocalTrail`, when the existing row is already `synced`, either route the caller to the network update path or reset the row to `pending`/`syncAttempts = 0` so the drain re-uploads the change. Whichever is chosen, `_finishLocalSave` should also `ref.watch`/`listen` the sync provider so the screen's `syncState` does not go stale.

## Warnings

### WR-01: A duplicate server trail is still possible — `writeServerTrailId` runs after `Trail.fromJson`, not after the response lands

**File:** `app/lib/provider/trail/trail_sync_provider.dart:146-162`

**Issue:** The comment claims the id write "commits the instant the server accepted the create, BEFORE any waypoint upload starts". It actually commits after `Trail.fromJson(response.data)` (line 153). Any parse failure — an unexpected body shape, a schema change, a `null` field where the freezed model requires non-null — throws before the id is persisted, and the next drain pass re-runs `PUT /trail/form`, creating a second trail with no way to reconcile them.

**Fix:** extract and persist the id before full deserialization.

```dart
final response = await ref.read(apiProvider).put('/trail/form', ...);
final rawId = (response.data as Map<String, dynamic>?)?['id'] as String?;
if (rawId != null && rawId.isNotEmpty) {
  writeServerTrailId(store, localId: localId, serverId: rawId);
}
final created = Trail.fromJson(response.data);
serverTrailId = created.id;
serverPhotoFilenames = created.photos;
```

---

### WR-02: A refused delete is swallowed, and the route is popped before the refusal is known

**File:** `app/lib/components/trail/trail_dropdown.dart:253-257`, `app/lib/provider/trail/trail_sync_provider.dart:228-233`

**Issue:** `TrailSync.deleteUnsynced` returns `false` when the local id is in the in-flight set, and its doc comment says deletion is "Refused (not silently ignored)". The caller discards the boolean:

```dart
Navigator.of(context).pop();
await ref.read(trailSyncProvider.notifier).deleteUnsynced(trail.localId!);
return;
```

The pop happens *first*, so the user is navigated away and then nothing is deleted, with no toast and no explanation — the exact "silently ignored" outcome the comment says is avoided. The `isDraining` menu disable does not cover this: the drain can start between the menu build and the confirm-dialog tap.

**Fix:**

```dart
final deleted = await ref.read(trailSyncProvider.notifier).deleteUnsynced(trail.localId!);
if (!mounted) return;
if (!deleted) {
  ref.read(toastProvider.notifier).add(ToastMessage(
    type: ToastType.warning,
    icon: FontAwesomeIcons.circleExclamation,
    text: l18n.delete_blocked_while_uploading, // new key
  ));
  return;
}
if (context.mounted && Navigator.of(context).canPop()) Navigator.of(context).pop();
```

---

### WR-03: Manual retry is a no-op whenever any drain is in flight

**File:** `app/lib/provider/trail/trail_sync_provider.dart:57-59`, `app/lib/provider/trail/trail_sync_provider.dart:217-221`

**Issue:** `retry()` resets the row then calls `drainIfOnline()`, which returns immediately if `_draining` is true. `_draining` is a *whole-drain* guard, not a per-trail one, so tapping "Upload failed · Tap to retry" while any other trail is uploading resets the backoff and then does nothing visible until the next foreground/connectivity trigger. The doc comment claims only "that row" no-ops, which is not what the guard does.

**Fix:** re-run the drain once the current pass finishes, e.g. set a `_drainRequestedAgain` flag when `_draining` and loop in the `finally`:

```dart
Future<void> drainIfOnline() async {
  if (_draining) { _rerunRequested = true; return; }
  _draining = true;
  try {
    do {
      _rerunRequested = false;
      await _drainPass();
    } while (_rerunRequested);
  } finally {
    _draining = false;
  }
}
```

---

### WR-04: `updateLocalTrail` does not carry `existing.photos` forward

**File:** `app/lib/util/local_trail_store.dart:226-235`

**Issue:** The function carefully carries `obxId`, `id`, `owner`, `localId`, `syncState`, `syncAttempts`, `syncNextAttemptAt` and `savedByUserIds` forward from the existing row, but not `photos`. `TrailEntity.fromModel` always leaves `photos` at `[]`, so any re-save through this path erases the server-side photo filenames from the local row. Combined with CR-04 (a post-sync edit lands here) this leaves a synced row with empty `photos` *and* empty `localPhotos` — a permanently thumbnail-less card.

**Fix:** add `entity.photos = existing.photos;` alongside the other carry-forwards, and extend the source-level carry-forward gate in `test/services/trail_download_service_carry_forward_test.dart`'s style to cover this function too.

---

### WR-05: `LocalSaveMode.updateLocal` and `createLocal` doc comments are swapped

**File:** `app/lib/util/local_trail_store.dart:37-48`

**Issue:**

```dart
  /// The trail has never been saved locally before -- mint a local id and
  /// create a new [TrailEntity] row.
  updateLocal,

  /// The trail already has a local row -- update it in place.
  createLocal,
```

Both doc comments describe the other value. `resolveLocalSaveMode` behaves per the *names* (`localId == null → createLocal`), so the code is right and the comments are wrong — but this is the single routing decision that guards against minting a duplicate local row (RESEARCH.md Pitfall 2), and it is exactly where a reader most needs the comment to be trustworthy.

**Fix:** swap the two doc comments.

---

### WR-06: `_readOwnLocal` uses the cached `_isOwnHandle`/`_authorActorId` despite documenting the opposite

**File:** `app/lib/provider/profile/profile_trails_provider.dart:54-60`, `app/lib/provider/profile/profile_trails_provider.dart:154-169`

**Issue:** The fields are documented as "Recomputed at the top of every `build()`, then re-derived fresh (never read from these cached copies) inside `search`/`loadNextPage` via `_readOwnLocal` -- D-13's 'always fresh, never cached' invariant." `_readOwnLocal` re-reads only `accountId`; it reads `_isOwnHandle` and `_authorActorId` from the cached fields. The actor id is a matching key in `readOwnLocalTrails`' second clause (`entity.author.target?.id == authorActorId`), so a stale value pairs the *new* account's id with the *previous* account's actor — the shape of leak D-13 exists to prevent. In practice `build()` watches `authProvider`, so the window is narrow; but the comment asserts a guarantee the code does not provide.

**Fix:** derive both inside `_readOwnLocal`:

```dart
List<Trail> _readOwnLocal(String q) {
  final store = ref.read(objectBoxProvider);
  final accountId = currentAccountId(store);
  final user = ref.read(authProvider).value;
  if (accountId == null || user == null) return const <Trail>[];
  if (_handle != '@${user.preferredUsername}') return const <Trail>[];
  return filterOwnTrailsByQuery(
    readOwnLocalTrails(store, accountId: accountId, authorActorId: user.actorId),
    q,
  );
}
```

---

### WR-07: The `updateLocal` branch dereferences `_localId!` rather than the value the router actually decided on

**File:** `app/lib/routes/trail_create_screen.dart:552`

**Issue:** `resolveLocalSaveMode` routes on `updatedTrail.localId`, but the branch then uses the separately-tracked `_localId!`. The two are kept in sync only by `initState` and the `createLocal` branch; any future path that produces a `trail` with a `localId` without going through those (a `context.push('/trail/create/edit', extra: …)` from a new call site, a `copyWith` chain) throws a bare null-check error that surfaces to the user as the generic `error_saving_trail` toast with no diagnosis.

**Fix:** `final localId = updatedTrail.localId ?? _localId!;` — or better, have `resolveLocalSaveMode` return the id it routed on so the two can never diverge.

---

### WR-08: Waypoints with a null `localKey` lose their photo protection and their server id

**File:** `app/lib/routes/trail_create_screen.dart:613-622`, `app/lib/provider/trail/trail_sync_provider.dart:166-184`

**Issue:** Two `continue`s treat a null `localKey` as "skip silently":

- `_copyPhotosForLocalSave` skips the copy entirely, so that waypoint's `localPhotos` remain OS-purgeable `image_picker` cache paths — precisely the D-01 failure the whole `local_photo_store_util.dart` module exists to prevent — and no failure is reported to the user.
- `_drainOne` creates the waypoint server-side and *then* checks `localKey`, `continue`ing without persisting the returned id. If a later waypoint in the same loop fails, the retry re-creates this one (same mechanism as CR-02).

Today `WaypointEntity.fromModel` always mints a key for an empty-id waypoint, so the null branch should be unreachable — which makes silent `continue` the worst possible handling: an invariant break becomes invisible.

**Fix:** hoist the guard and make a violation loud.

```dart
final waypointLocalKey = waypointEntity.localKey;
if (waypointLocalKey == null) {
  throw StateError('waypoint ${waypointEntity.obxId} of "$localId" has no localKey');
}
final createdWaypoint = await ref.read(waypointSaveProvider.notifier).create(...);
writeServerWaypointId(store, localId: localId, waypointLocalKey: waypointLocalKey, ...);
```

and in `_copyPhotosForLocalSave`, count a keyless waypoint with photos into `failedCount` rather than dropping it.

---

### WR-09: Any exception, including a parse error, is reported to the user as "you're offline"

**File:** `app/lib/provider/profile/profile_trails_provider.dart:191-204`

**Issue:** `_fetchAndMerge`'s `catch (_) { if (!_isOwnHandle) rethrow; offline = true; }` swallows every throwable — including the `FormatException` thrown deliberately at line 244 for an unexpected response shape, and the implicit `TypeError` from `data['totalPages'] ?? 1` when the server returns a non-int. The screen then renders `own_trails_offline_banner` ("Showing what's saved on this device — connect to see everything") for a fully-online, server-side or client-side bug, and `loadNextPage` refuses to page because `state.offline` is true. The doc comment on `ProfileTrailsState.offline` says it is "Decided from the fetch outcome itself"; it is decided from *any* outcome.

**Fix:** narrow the catch to transport errors and let the rest surface.

```dart
} on DioException catch (_) {
  if (!_isOwnHandle) rethrow;
  offline = true;
}
```

---

### WR-10: `deleteUnsyncedPhotoDir` can throw out of `deleteUnsynced` as an unhandled async error

**File:** `app/lib/util/local_photo_store_util.dart:172-182`, `app/lib/provider/trail/trail_sync_provider.dart:228-242`

**Issue:** `unsyncedTrailPhotoDir` (and therefore `localIdDirSegment`) is evaluated *outside* `deleteUnsyncedPhotoDir`'s `try`, which the doc comment states is deliberate. But `TrailSync.deleteUnsynced` has no `try/catch` around it, and the call site in `trail_dropdown.dart` `await`s it without one either. A row whose `localId` does not match `^local-\d+-\d+$` (a pre-`mintLocalId` row, a hand-edited store, a future id-format change) makes `deleteUnsynced` throw an `ArgumentError` into the button's async callback — the local row has already been deleted at that point (line 232 runs first), so the user is left with an orphaned photo directory and an unexplained red screen / silent failure.

**Fix:** wrap the call and degrade to best-effort, matching the sweep's discipline:

```dart
deleteLocalTrailRow(store, localId);
try {
  await deleteUnsyncedPhotoDir(localId);
} catch (e, st) {
  debugPrint('trail_sync_provider: photo dir cleanup failed for "$localId": $e\n$st');
}
```

---

### WR-11: The drain writes a stale, fully-materialised entity outside a transaction, clobbering concurrent local edits

**File:** `app/lib/provider/trail/trail_sync_provider.dart:121-122`

**Issue:**

```dart
entity.syncState = TrailSyncState.uploading;
store.box<TrailEntity>().put(entity);
```

`entity` came from `selectDrainCandidates` at the top of `drainIfOnline`, before the (potentially long) `refresh()`/tag-resolution awaits and before every preceding trail in the same pass finished uploading. Putting the whole object back writes *every* column from that snapshot, not just `syncState`. Nothing prevents `_onSave`'s `updateLocalTrail` from committing an edit in the meantime (only *delete* is gated on the in-flight set), so a user editing a queued trail can have their edit reverted by this write. Every other bookkeeping write in `local_trail_store.dart` correctly re-queries inside `runInTransaction`; this one does not.

**Fix:** add a `markTrailUploading(store, localId: localId)` to `local_trail_store.dart` following the shape of `markTrailSynced`/`resetDrainBackoff`, and call that instead of the direct `box.put`.

---

### WR-12: A missing `UserEntity` burns a retry attempt and can park a perfectly good trail as `failed`

**File:** `app/lib/provider/trail/trail_sync_provider.dart:108-118`

**Issue:** The `StateError` for a missing `UserEntity` is thrown *inside* `_drainOne`'s `try`, so it lands in the generic failure handler and consumes one of the four attempts from `kMaxSyncAttempts`. This is not a network condition at all — it means the account row vanished (mid-logout race, `account_data_purge_util` running concurrently). Four such passes, which can occur within seconds on a lifecycle/connectivity flurry, park the trail as `TrailSyncState.failed`, at which point `isDrainDue` returns false forever and only a manual tap on the chip revives it. `sync_backoff.dart`'s own doc comment argues the attempt count should escalate only on real upload failures.

**Fix:** resolve the user before entering the try, and bail without recording a failure:

```dart
final userEntity = _findUser(store, accountId);
if (userEntity == null) {
  debugPrint('trail_sync_provider: no UserEntity for "$accountId"; skipping drain');
  return;   // before state = {...state, localId}
}
```

---

### WR-13: `reconcileLocalPhotos`' delete pass matches kept files by raw string equality

**File:** `app/lib/util/local_photo_store_util.dart:113-145`

**Issue:** A desired path is kept verbatim when `p.isWithin(dir, sourcePath)` is true, but the subsequent delete pass tests `keptSet.contains(entry.path)` — a plain string compare against `Directory(dir).listSync()` output. `p.isWithin` normalizes (`dir/./photo.jpg`, `dir//photo.jpg`, a trailing-separator `dir`), `listSync().path` does not. Any non-canonical spelling of an in-dir path is therefore "kept" in the returned list and *deleted from disk in the same call*, leaving the entity pointing at a file that no longer exists — silently, since both loops swallow errors.

Reachability is narrow today (paths originate from a previous `p.join`), but the failure mode is silent photo loss with no `failedCount` increment.

**Fix:** compare canonical forms.

```dart
final keptSet = keptPaths.map(p.canonicalize).toSet();
for (final entry in directory.listSync()) {
  if (entry is! File) continue;
  if (keptSet.contains(p.canonicalize(entry.path))) continue;
  // ...
}
```

---

### WR-14: `trail_download_service` still builds filesystem paths by interpolating unvalidated server ids

**File:** `app/lib/services/trail_download_service.dart:56`, `app/lib/services/trail_download_service.dart:80`, `app/lib/services/trail_download_service.dart:258`, `app/lib/services/trail_download_service.dart:273-274`

**Issue:** `Directory('${appDir.path}/library/$trailId')`, `Directory('${trailDir.path}/waypoints/${waypoint.id}')` and `'${photoDir.path}/$fileName'` (where `fileName = p.basename(Uri.parse(url).path)`) are all raw interpolation of values that arrive over the network. `local_photo_store_util.dart`'s header explicitly names this file's style as the one it refuses to reuse, and `local_id.dart` documents `p.join` + whitelist as the required pattern — but the service was edited in this phase (the six-field carry-forward block) without bringing it onto that standard. A federated/compromised instance returning an id or photo filename containing `..` writes outside `library/`.

This is pre-existing rather than newly introduced, but it is now the last path-construction site in the offline-storage surface that lacks the control, and it sits in a file this phase touched.

**Fix:** route these through `p.join` and validate the id segment the same way `localIdDirSegment` does:

```dart
String recordIdDirSegment(String id) {
  if (!RegExp(r'^[a-z0-9]{15}$').hasMatch(id)) {
    throw ArgumentError.value(id, 'id', 'must be a PocketBase record id');
  }
  return id;
}
final trailDir = Directory(p.join(appDir.path, 'library', recordIdDirSegment(trailId)));
```

---

### WR-15: Nine new user-facing strings exist only in `app_en.arb`

**File:** `app/lib/i18n/app_en.arb:69`, `:181-183`, `:189`, `:272`, `:290-292`

**Issue:** `delete_unsynced_trail_confirm`, `own_trails_empty_body`, `own_trails_empty_title`, `own_trails_offline_banner`, `photo_copy_failed_toast`, `photos_skipped_no_gps`, `signout_unsynced_warning`, `sync_failed`, `sync_pending` and `sync_uploading` are absent from all twelve other `app_*.arb` files (verified for `de`/`fr`). Every non-English user sees raw English for the whole sync-status surface, including the sign-out data-safety warning (D-12) and the "cannot be undone" delete confirmation (D-14) — the two strings whose entire purpose is preventing a destructive misunderstanding.

**Fix:** at minimum add the two destructive-action strings (`signout_unsynced_warning`, `delete_unsynced_trail_confirm`) to every locale file, and confirm `flutter gen-l10n`'s untranslated-message report is being read rather than suppressed.

---

_Reviewed: 2026-08-02T14:43:17Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
