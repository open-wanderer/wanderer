/// The single sanctioned read/write layer for locally-captured [TrailEntity]
/// rows.
///
/// A locally-captured trail lives in the SAME `TrailEntity` box every
/// downloaded trail lives in -- there is no separate pending-queue entity,
/// per the design record. Three signals distinguish a not-yet-uploaded
/// capture from a downloaded/synced row: [TrailEntity.owner] (the capturing
/// account), [TrailEntity.localId] (a permanent local identity minted once
/// at first save, [mintLocalId]), and [TrailEntity.syncState].
///
/// Ownership is expressed EXCLUSIVELY by [TrailEntity.owner] and must never
/// be conflated with [TrailEntity.savedByUserIds] (D-10) -- `owner` is 1:1
/// authorship-on-this-device, `savedByUserIds` is 1:N offline-library
/// membership. Every function in this file that carries `savedByUserIds`
/// forward does so verbatim, never as an ownership check.
///
/// Every function that takes an account id as a parameter requires the
/// CALLER to have re-read it fresh via `currentAccountId(store)` at the
/// point of use, never from a long-lived cached value (D-13,
/// RESEARCH.md Open Question 2) -- a stale cached id is exactly the leak
/// that would show one account's captures to another.
library;

import 'package:flutter/foundation.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/objectbox.g.dart';

// ---------------------------------------------------------------------------
// Pure decisions (no Store, unit-testable)
// ---------------------------------------------------------------------------

/// Which local-write path a save of [Trail] should take.
enum LocalSaveMode {
  /// The trail already lives on the server and is fully synced -- save via
  /// the normal network `PUT`/`POST`, not a local write.
  networkUpdate,

  /// The trail already has a local row -- update it in place.
  updateLocal,

  /// The trail has never been saved locally before -- mint a local id and
  /// create a new [TrailEntity] row.
  createLocal,
}

/// Decides which [LocalSaveMode] a save of [trail] should take.
///
/// `trail.id.isEmpty` alone is NOT a sufficient discriminator after this
/// phase (RESEARCH.md Pitfall 2): an empty id now means both "never saved
/// anywhere" AND "saved locally, still unsynced". Collapsing those two into
/// one branch would route a re-save of an already-local trail back through
/// [createLocal], minting a second local id and creating a second row where
/// only one edit session happened. [Trail.localId] is what actually
/// distinguishes them.
///
/// A trail with a non-empty server id whose [Trail.syncState] is NOT
/// [TrailSyncState.synced] (still `uploading` or `pending`, the mid-drain
/// resume case) is deliberately routed to [LocalSaveMode.updateLocal], not
/// [LocalSaveMode.networkUpdate] -- the server id exists but the row is not
/// yet confirmed synced, so the safe write target is still the local row.
LocalSaveMode resolveLocalSaveMode(Trail trail) {
  if (trail.syncState == TrailSyncState.synced && trail.id.isNotEmpty) {
    return LocalSaveMode.networkUpdate;
  }
  if (trail.localId == null) {
    return LocalSaveMode.createLocal;
  }
  return LocalSaveMode.updateLocal;
}

/// Whether the local row of a trail that has just finished
/// uploading can be deleted outright, rather than demoted.
///
/// A locally-captured row normally has an EMPTY
/// [TrailEntity.savedByUserIds]: [saveNewLocalTrail] never writes
/// it and [updateLocalTrail] only carries it forward, so nothing on
/// the capture path can populate it -- D-10 keeps ownership ("I
/// recorded this") and library membership ("I downloaded this")
/// strictly separate.
///
/// A non-empty list therefore means some account downloaded this
/// trail while its upload was in flight. That is possible from the
/// moment `writeServerTrailId` stamps the server id, because from
/// then on the trail is fetchable and `TrailDownloadService` writes
/// into the SAME row (`TrailEntity.id` is
/// `@Unique(onConflict: replace)`). Deleting it would destroy that
/// account's offline library entry and strand `library/<id>/` on
/// disk with nothing pointing at it.
bool shouldDeleteUploadedRow(List<String> savedByUserIds) =>
    savedByUserIds.isEmpty;

/// Which save path a screen should take when it holds
/// [persistedLocalId] and the store's row for it is [persisted]
/// (null when there is no such row).
///
/// [resolveLocalSaveMode] alone cannot answer this. It sees one
/// [Trail], so a caller with no persisted row has to fall back to
/// its own screen snapshot -- and that snapshot is captured while
/// `syncState` is still `pending` and the server id is still blank.
/// Since a successful upload now RETIRES the row
/// ([retireUploadedLocalTrail]), "I saved this locally and the row
/// is gone" is the normal post-upload state, and routing it on the
/// stale snapshot sends the edit into [LocalSaveMode.updateLocal],
/// which writes to a row that no longer exists and reports success.
///
/// A null [persistedLocalId] is the genuinely-never-saved case and
/// is delegated unchanged, so a first save still routes to
/// [LocalSaveMode.createLocal].
LocalSaveMode resolveLocalSaveModeForRow({
  required Trail screenTrail,
  required String? persistedLocalId,
  required Trail? persisted,
}) {
  if (persistedLocalId != null && persisted == null) {
    return LocalSaveMode.networkUpdate;
  }
  return resolveLocalSaveMode(persisted ?? screenTrail);
}

/// Whether [entity]'s upload is due to run now.
///
/// True only for a row whose [TrailEntity.syncState] is [TrailSyncState.pending]
/// or [TrailSyncState.uploading], AND whose [TrailEntity.syncNextAttemptAt]
/// is either unset or not in the future relative to [now].
///
/// A [TrailSyncState.failed] row is deliberately NEVER due -- D-07 parks a
/// row that exhausted its retry budget for manual retry only
/// ([resetDrainBackoff]), not automatic pickup by the next drain pass.
bool isDrainDue(TrailEntity entity, DateTime now) {
  final isUploadable =
      entity.syncState == TrailSyncState.pending ||
      entity.syncState == TrailSyncState.uploading;
  if (!isUploadable) return false;

  final nextAttempt = entity.syncNextAttemptAt;
  return nextAttempt == null || !nextAttempt.isAfter(now);
}

/// Pure decision core of [recordDrainFailure], extracted so its
/// attempt-count boundary is unit-testable without a live ObjectBox [Store]
/// (Phase 31 established there is no ObjectBox test harness for plain
/// `flutter test`).
typedef DrainFailureOutcome = ({
  TrailSyncState syncState,
  int syncAttempts,
  DateTime? syncNextAttemptAt,
});

/// Computes the next [TrailSyncState]/attempt-count/backoff-deadline after
/// one more failed upload attempt.
///
/// When the incremented attempt count reaches [maxAttempts], the row is
/// parked as [TrailSyncState.failed] with no further scheduled retry (D-07).
/// Otherwise it goes back to [TrailSyncState.pending] with its next attempt
/// scheduled via [backoff].
DrainFailureOutcome resolveDrainFailureOutcome({
  required int currentAttempts,
  required DateTime now,
  required int maxAttempts,
  required Duration Function(int attempts) backoff,
}) {
  final newAttempts = currentAttempts + 1;

  if (newAttempts >= maxAttempts) {
    return (
      syncState: TrailSyncState.failed,
      syncAttempts: newAttempts,
      syncNextAttemptAt: null,
    );
  }

  return (
    syncState: TrailSyncState.pending,
    syncAttempts: newAttempts,
    syncNextAttemptAt: now.add(backoff(newAttempts)),
  );
}

// ---------------------------------------------------------------------------
// Write path
// ---------------------------------------------------------------------------

/// Creates a new locally-captured [TrailEntity] row for [trail] and returns
/// its [localId].
///
/// [trail] is converted via [TrailEntity.fromModel], then [localId] is
/// stamped onto both [TrailEntity.id] (so the row has a collision-free
/// identity) and [TrailEntity.localId] (its permanent local identity),
/// [ownerAccountId] onto [TrailEntity.owner], the sync bookkeeping reset to
/// a fresh [TrailSyncState.pending] row, and [trailLocalPhotos] /
/// [waypointLocalPhotosByKey] onto the trail's and each waypoint's
/// `localPhotos`.
///
/// [authorActorId], when non-null and a matching [ActorEntity] already
/// exists locally, is linked as `entity.author.target` on a best-effort
/// basis ONLY -- so the card shows the hiker's own name instead of
/// "Unknown". A missing actor row is not an error.
String saveNewLocalTrail(
  Store store, {
  required Trail trail,
  required String ownerAccountId,
  String? authorActorId,
  required String localId,
  required List<String> trailLocalPhotos,
  required Map<String, List<String>> waypointLocalPhotosByKey,
}) {
  store.runInTransaction(TxMode.write, () {
    final entity = TrailEntity.fromModel(trail);
    entity.id = localId;
    entity.localId = localId;
    entity.owner = ownerAccountId;
    entity.syncState = TrailSyncState.pending;
    entity.syncAttempts = 0;
    entity.syncNextAttemptAt = null;
    entity.localPhotos = trailLocalPhotos;

    for (final waypointEntity in entity.waypoints) {
      final key = waypointEntity.localKey;
      if (key == null) continue;
      final photos = waypointLocalPhotosByKey[key];
      if (photos != null) waypointEntity.localPhotos = photos;
    }

    if (authorActorId != null) {
      // The scalar is written unconditionally; only the relation is
      // best-effort. The relation is the part that breaks (a replace-on-
      // conflict `ActorEntity` put re-mints the row and orphans the ToOne),
      // so authorship must not depend on it alone.
      entity.authorRecordId = authorActorId;

      final actorQuery = store
          .box<ActorEntity>()
          .query(ActorEntity_.id.equals(authorActorId))
          .build();
      final actor = actorQuery.findFirst();
      actorQuery.close();
      if (actor != null) {
        entity.author.target = actor;
      }
    }

    store.box<TrailEntity>().put(entity);
  });

  return localId;
}

/// What [updateLocalTrail] did, so the caller can tell a completed write
/// apart from a declined one instead of assuming success.
enum LocalUpdateOutcome {
  /// The row was found and rewritten.
  updated,

  /// No row exists for that local id. Nothing was written.
  missing,

  /// The row has already been promoted to [TrailSyncState.synced]. Nothing
  /// was written -- see [updateLocalTrail]'s doc comment for why writing
  /// would silently strand the user's edit on this device.
  alreadySynced,
}

/// Updates the local row for [localId] in place from [trail], preserving its
/// identity, ownership and sync bookkeeping.
///
/// Looks the existing row up by [localId] INSIDE the transaction. If no such
/// row exists, this is a no-op -- there is nothing to update. Otherwise a
/// fresh entity is built from [trail], then the existing row's `obxId`,
/// `id`, `owner`, `localId`, `syncState`, `syncAttempts`,
/// `syncNextAttemptAt`, `savedByUserIds` and `photos` are carried forward
/// onto it before the put, so a metadata re-edit never changes the row's
/// identity or ownership (REC-05, SYNC-05).
///
/// REFUSES to write, returning [LocalUpdateOutcome.alreadySynced], when the
/// existing row is [TrailSyncState.synced]. Writing would carry that `synced`
/// state forward, and `selectDrainCandidates` only picks up rows that are NOT
/// synced -- so the edit would live on this device and nothing would ever
/// upload it. The drain has no update path either: its step 2 is guarded on
/// `isLocalId(entity.id)`, so a re-queued synced row would be marked synced
/// again without the edit ever being sent. A synced trail's only correct write
/// target is the network `PUT`/`POST`, which is the caller's job to route to.
///
/// [WaypointEntity] rows that belonged to the old row but are absent from
/// the new waypoint set are removed, so a deleted waypoint does not linger.
LocalUpdateOutcome updateLocalTrail(
  Store store, {
  required Trail trail,
  required String localId,
  required List<String> trailLocalPhotos,
  required Map<String, List<String>> waypointLocalPhotosByKey,
}) {
  return store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    final query = trailBox.query(TrailEntity_.localId.equals(localId)).build();
    final existing = query.findFirst();
    query.close();
    if (existing == null) return LocalUpdateOutcome.missing;
    if (existing.syncState == TrailSyncState.synced) {
      return LocalUpdateOutcome.alreadySynced;
    }

    final entity = TrailEntity.fromModel(trail);
    entity.obxId = existing.obxId;
    entity.id = existing.id;
    entity.owner = existing.owner;
    entity.localId = existing.localId;
    entity.syncState = existing.syncState;
    entity.syncAttempts = existing.syncAttempts;
    entity.syncNextAttemptAt = existing.syncNextAttemptAt;
    entity.savedByUserIds = existing.savedByUserIds;
    // `TrailEntity.fromModel` always leaves `photos` at `[]` -- the model has
    // no place to put server-side photo FILENAMES on the way in. Without this
    // carry-forward every re-save through this path erased them from the row,
    // and combined with a post-sync edit that left a row with empty `photos`
    // AND empty `localPhotos`: a permanently thumbnail-less card whose photos
    // exist only on the server.
    entity.photos = existing.photos;
    entity.localPhotos = trailLocalPhotos;

    for (final waypointEntity in entity.waypoints) {
      final key = waypointEntity.localKey;
      if (key == null) continue;
      final photos = waypointLocalPhotosByKey[key];
      if (photos != null) waypointEntity.localPhotos = photos;
    }

    // Remove waypoint rows that belonged to the old trail but are absent
    // from the new waypoint set, so a deleted waypoint does not linger.
    final newKeys = entity.waypoints.map((w) => w.localKey ?? w.id).toSet();
    final waypointBox = store.box<WaypointEntity>();
    for (final oldWaypoint in existing.waypoints) {
      final oldKey = oldWaypoint.localKey ?? oldWaypoint.id;
      if (!newKeys.contains(oldKey)) {
        waypointBox.remove(oldWaypoint.obxId);
      }
    }

    trailBox.put(entity);
    return LocalUpdateOutcome.updated;
  });
}

/// Removes the local row for [localId] and its [WaypointEntity] children.
///
/// A no-op for an unknown [localId], mirroring
/// `TrailLibraryNotifier.deleteTrail`'s silent no-op precedent. Does NOT
/// touch the filesystem -- the caller pairs this with
/// `deleteUnsyncedPhotoDir` (from `local_photo_store_util.dart`, 36-02).
void deleteLocalTrailRow(Store store, String localId) {
  store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    final query = trailBox.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    final waypointBox = store.box<WaypointEntity>();
    for (final waypoint in entity.waypoints) {
      waypointBox.remove(waypoint.obxId);
    }
    trailBox.remove(entity.obxId);
  });
}

/// Retires the local capture row for [localId] now that its upload
/// has completed.
///
/// The counterpart to [deleteLocalTrailRow], for the other way a
/// capture's life ends. [shouldDeleteUploadedRow] picks between two
/// shapes:
/// - No library membership (the ordinary case): the row and its
///   [WaypointEntity] children are removed outright. The trail now
///   lives on the server under the id `writeServerTrailId` stamped,
///   and the own-trails list reaches it through the network entry.
///   To have it offline again the hiker downloads it like any
///   other trail, gaining `savedByUserIds` through the normal path.
/// - Some account holds it in its offline library: the row is KEPT
///   and only its capture bookkeeping is cleared, so it becomes an
///   ordinary downloaded row. Never deleted -- that would destroy
///   a library entry this function has no business touching.
///
/// The waypoint children are removed with the row on purpose.
/// `TrailLibraryNotifier.deleteTrail` (`trail_library_provider.dart`)
/// is a bare `box.remove(entity.obxId)` that leaks them; copying
/// that shortcut here would leak a set of waypoints on every
/// successful upload. `author` and `category` are ToOne targets
/// SHARED with other trails and are deliberately left alone.
///
/// Does NOT touch the filesystem -- the caller pairs this with
/// `deleteUnsyncedPhotoDir`, exactly as [deleteLocalTrailRow]'s
/// caller does. Row first, files second: `unsyncedLocalIds` cannot
/// see a retired row, so a directory left behind by a crash between
/// the two is reclaimed by the next startup sweep. The reverse
/// order self-heals into a row with broken thumbnails instead.
void retireUploadedLocalTrail(Store store, String localId) {
  store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    // No account scoping and no `owner` clause: the only caller reached
    // this row through `selectDrainCandidates`, which is already
    // owner-scoped, and the key is a `localId` this device minted.
    final query = trailBox.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    if (!shouldDeleteUploadedRow(entity.savedByUserIds)) {
      // Demote to an ordinary downloaded row -- exactly the shape
      // `TrailDownloadService` produces, so nothing downstream needs to
      // know it was ever a capture.
      entity.owner = null;
      entity.localId = null;
      entity.syncState = TrailSyncState.synced;
      entity.syncAttempts = 0;
      entity.syncNextAttemptAt = null;
      entity.localPhotos = [];
      trailBox.put(entity);
      return;
    }

    final waypointBox = store.box<WaypointEntity>();
    for (final waypoint in entity.waypoints) {
      waypointBox.remove(waypoint.obxId);
    }
    trailBox.remove(entity.obxId);
  });
}

// ---------------------------------------------------------------------------
// Reads -- every one of them owner-scoped
// ---------------------------------------------------------------------------

/// Single-row lookup of the local trail identified by [localId].
///
/// Guarded by a `try/catch` that logs via [debugPrint] and returns null,
/// matching `TrailLibraryNotifier.build()`'s "losing the one bad row is the
/// correct blast radius" rationale.
Trail? readLocalTrail(Store store, String localId) {
  final query = store
      .box<TrailEntity>()
      .query(TrailEntity_.localId.equals(localId))
      .build();
  final entity = query.findFirst();
  query.close();
  if (entity == null) return null;

  try {
    return entity.toModel();
  } catch (e, st) {
    debugPrint(
      'local_trail_store: readLocalTrail("$localId") failed to parse: '
      '$e\n$st',
    );
    return null;
  }
}

/// [readLocalTrail], scoped to [accountId].
///
/// The unscoped sibling above is safe only because its callers already
/// know the row is theirs. This one exists for the opposite case: its
/// argument comes from a ROUTE PARAMETER (`/trail/local/:localId`),
/// which is attacker-supplied as far as this layer is concerned and
/// survives a logout in the deep-link and back-stack. Without the owner
/// clause, account B could render account A's not-yet-uploaded trail --
/// the exact leak D-13 exists to prevent, arriving through a door
/// `readOwnLocalTrails` had already bolted.
///
/// Deliberately owner-only, with NO `savedByUserIds` clause: D-10 keeps
/// ownership ("I made this") and library membership ("I downloaded
/// this") strictly separate.
Trail? readOwnLocalTrail(
  Store store, {
  required String localId,
  required String accountId,
}) {
  final query = store
      .box<TrailEntity>()
      .query(
        TrailEntity_.localId.equals(localId) &
            TrailEntity_.owner.equals(accountId),
      )
      .build();
  final entity = query.findFirst();
  query.close();
  if (entity == null) return null;

  try {
    return entity.toModel();
  } catch (e, st) {
    debugPrint(
      'local_trail_store: readOwnLocalTrail("$localId") failed to parse: '
      '$e\n$st',
    );
    return null;
  }
}

/// Every trail [accountId] can see in its own-trails list: trails it
/// captured on this device (not yet uploaded, or uploaded already, or
/// downloaded), plus any downloaded trail it happens to have authored
/// itself.
///
/// The query casts a broad net (`owner == accountId` OR `savedByUserIds`
/// contains `accountId`), then keeps a row only when `entity.owner ==
/// accountId` OR (`authorActorId != null && entity.author.target?.id ==
/// authorActorId`). The second clause is REC-06's "downloaded trails the
/// hiker authored themselves"; the `owner` clause is every not-yet-uploaded
/// capture. A row whose `owner` is null can NEVER satisfy the first clause
/// -- that is what keeps a pre-existing downloaded row (owner unset) from
/// leaking into another account's own-trails list (T-36-03-01).
///
/// Converted with the same per-entity `try/catch` guard
/// [TrailLibraryNotifier.build] uses, sorted by `created` descending, and
/// returned as a flat list -- no sectioning and no special sort (D-11).
List<Trail> readOwnLocalTrails(
  Store store, {
  required String accountId,
  String? authorActorId,
}) {
  final box = store.box<TrailEntity>();
  final query = box
      .query(
        TrailEntity_.owner.equals(accountId) |
            TrailEntity_.savedByUserIds.containsElement(accountId),
      )
      .build();

  final trails = <Trail>[];
  for (final entity in query.find()) {
    final isOwn = entity.owner == accountId;
    // Scalar first, relation only as the fallback for rows written before
    // `authorRecordId` existed -- the ToOne is orphaned whenever a sign-in or
    // profile refresh re-mints the `ActorEntity` row.
    final isAuthoredByThisAccount =
        authorActorId != null &&
        (entity.authorRecordId ?? entity.author.target?.id) == authorActorId;
    if (!isOwn && !isAuthoredByThisAccount) continue;

    try {
      trails.add(entity.toModel());
    } catch (e, st) {
      debugPrint(
        'local_trail_store: readOwnLocalTrails skipping "${entity.id}" -- '
        'toModel() failed: $e\n$st',
      );
    }
  }
  query.close();

  trails.sort((a, b) => b.created.compareTo(a.created));
  return trails;
}

/// Counts [accountId]'s not-yet-synced local trails. Used by the sign-out
/// warning (D-12).
int countUnsyncedTrails(Store store, String accountId) {
  final query = store
      .box<TrailEntity>()
      .query(
        TrailEntity_.owner.equals(accountId) &
            TrailEntity_.dbSyncState.notEquals(TrailSyncState.synced.index),
      )
      .build();
  final count = query.count();
  query.close();
  return count;
}

/// Every non-null [TrailEntity.localId] across ALL accounts whose row is
/// not synced.
///
/// Deliberately NOT account-scoped: its only consumer is the startup photo
/// orphan sweep, which must not delete a signed-out account's still-pending
/// photos just because that account is not the currently signed-in one
/// (D-13 hides another account's content, it never deletes it).
Set<String> unsyncedLocalIds(Store store) {
  final query = store
      .box<TrailEntity>()
      .query(
        TrailEntity_.localId.notNull() &
            TrailEntity_.dbSyncState.notEquals(TrailSyncState.synced.index),
      )
      .build();

  final ids = <String>{};
  for (final entity in query.find()) {
    final localId = entity.localId;
    if (localId != null) ids.add(localId);
  }
  query.close();
  return ids;
}

/// [accountId]'s local trails whose upload is due right now, oldest
/// [TrailEntity.created] first so a hike's trails upload in capture order.
///
/// Returns entities, not models, because the drain needs the live rows to
/// pass into the bookkeeping writes below.
List<TrailEntity> selectDrainCandidates(
  Store store, {
  required String accountId,
  required DateTime now,
}) {
  final query = store
      .box<TrailEntity>()
      .query(
        TrailEntity_.owner.equals(accountId) &
            TrailEntity_.dbSyncState.notEquals(TrailSyncState.synced.index),
      )
      .build();
  final candidates = query.find().where((e) => isDrainDue(e, now)).toList();
  query.close();

  candidates.sort((a, b) => a.created.compareTo(b.created));
  return candidates;
}

// ---------------------------------------------------------------------------
// Drain bookkeeping -- each one its own small write transaction
// ---------------------------------------------------------------------------

/// SYNC-04's load-bearing write: stamps the server-assigned [serverId] and
/// [serverPhotoFilenames] onto the local row identified by [localId], leaving
/// `localId`, `owner` and `syncState` untouched.
///
/// Must be callable, and must commit, the instant `PUT /trail/form`
/// returns, before any waypoint upload starts (RESEARCH.md Pitfall 3 --
/// there is no server-side idempotency key, so a crash between "server
/// accepted" and "id persisted" is what produces a duplicate trail).
///
/// [serverPhotoFilenames] commits in the SAME transaction as [serverId] on
/// purpose. The two facts are learned from one response and are only
/// meaningful together: a row that has the server id but not the photo list
/// looks, to a resumed drain, exactly like a trail with no photos at all --
/// which is how [markTrailSynced] came to persist an empty `photos` column
/// and `deleteUnsyncedPhotoDir` came to delete the only copies left on the
/// device. It is nullable, and null means "leave `photos` alone", so a
/// response that carries an id but no usable photo list cannot erase one.
void writeServerTrailId(
  Store store, {
  required String localId,
  required String serverId,
  List<String>? serverPhotoFilenames,
}) {
  store.runInTransaction(TxMode.write, () {
    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    entity.id = serverId;
    if (serverPhotoFilenames != null) {
      entity.photos = serverPhotoFilenames;
    }
    box.put(entity);
  });
}

/// Stamps the server-assigned [serverWaypointId] and [serverPhotoFilenames]
/// onto the child [WaypointEntity] identified by [waypointLocalKey] under
/// the trail [localId], clearing its `localPhotos`.
void writeServerWaypointId(
  Store store, {
  required String localId,
  required String waypointLocalKey,
  required String serverWaypointId,
  List<String> serverPhotoFilenames = const [],
}) {
  store.runInTransaction(TxMode.write, () {
    final trailQuery = store
        .box<TrailEntity>()
        .query(TrailEntity_.localId.equals(localId))
        .build();
    final trailEntity = trailQuery.findFirst();
    trailQuery.close();
    if (trailEntity == null) return;

    WaypointEntity? target;
    for (final waypoint in trailEntity.waypoints) {
      if (waypoint.localKey == waypointLocalKey) {
        target = waypoint;
        break;
      }
    }
    if (target == null) return;

    target.id = serverWaypointId;
    target.photos = serverPhotoFilenames;
    target.localPhotos = [];
    store.box<WaypointEntity>().put(target);
  });
}

/// Marks the local row for [localId] as [TrailSyncState.uploading], touching
/// nothing else.
///
/// Exists because the drain used to do this by mutating the entity it got
/// from [selectDrainCandidates] and calling `box.put(entity)` directly. That
/// snapshot was taken before the pass's `refresh()`/tag-resolution awaits and
/// before every preceding trail in the same pass finished uploading, and
/// putting the whole object back wrote EVERY column from it -- so a user who
/// edited a queued trail in the meantime had their edit silently reverted.
/// Only *delete* is gated on the in-flight set, not edit, so nothing else
/// prevented it. Re-querying inside the transaction and writing one field is
/// what every other bookkeeping write in this file already does.
void markTrailUploading(Store store, String localId) {
  store.runInTransaction(TxMode.write, () {
    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    entity.syncState = TrailSyncState.uploading;
    box.put(entity);
  });
}

/// Marks the local row for [localId] as fully synced.
///
/// Sets [TrailSyncState.synced], resets `syncAttempts` to 0 and
/// `syncNextAttemptAt` to null, and clears `localPhotos`. The row is kept,
/// not deleted -- its `obxId` never changes across the transition, which is
/// exactly what SYNC-05's "keeps its identity in place" means concretely.
///
/// [serverPhotoFilenames] is NULLABLE and defaults to null, meaning "leave
/// the row's existing `photos` alone". That default is the safe one and the
/// one the drain uses: [writeServerTrailId] already persisted the server's
/// photo list in the same transaction as the server id, so by the time this
/// runs the row is authoritative. Passing a non-null list REPLACES `photos`,
/// and passing an empty one therefore erases it -- which is precisely the bug
/// this signature change exists to make impossible to write by accident. On a
/// resumed drain (the trail was created by an earlier attempt that then failed
/// at a waypoint) the caller has no fresh photo list to offer, and the old
/// `const []` default silently wiped the column and left
/// `deleteUnsyncedPhotoDir` to delete the last copies off the device.
void markTrailSynced(
  Store store, {
  required String localId,
  List<String>? serverPhotoFilenames,
}) {
  store.runInTransaction(TxMode.write, () {
    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    entity.syncState = TrailSyncState.synced;
    entity.syncAttempts = 0;
    entity.syncNextAttemptAt = null;
    if (serverPhotoFilenames != null) {
      entity.photos = serverPhotoFilenames;
    }
    entity.localPhotos = [];
    box.put(entity);
  });
}

/// Records one more failed upload attempt for the local row [localId].
///
/// Delegates the attempt-count/backoff decision to
/// [resolveDrainFailureOutcome] -- see that function's doc comment for the
/// D-07 parking behaviour.
void recordDrainFailure(
  Store store, {
  required String localId,
  required DateTime now,
  required int maxAttempts,
  required Duration Function(int attempts) backoff,
}) {
  store.runInTransaction(TxMode.write, () {
    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    final outcome = resolveDrainFailureOutcome(
      currentAttempts: entity.syncAttempts,
      now: now,
      maxAttempts: maxAttempts,
      backoff: backoff,
    );

    entity.syncState = outcome.syncState;
    entity.syncAttempts = outcome.syncAttempts;
    entity.syncNextAttemptAt = outcome.syncNextAttemptAt;
    box.put(entity);
  });
}

/// SYNC-03's manual-retry primitive: resets the local row for [localId]
/// back to a fresh [TrailSyncState.pending] state with zero attempts and no
/// scheduled backoff.
void resetDrainBackoff(Store store, String localId) {
  store.runInTransaction(TxMode.write, () {
    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    entity.syncState = TrailSyncState.pending;
    entity.syncAttempts = 0;
    entity.syncNextAttemptAt = null;
    box.put(entity);
  });
}
