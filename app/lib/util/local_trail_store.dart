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

  /// The trail has never been saved locally before -- mint a local id and
  /// create a new [TrailEntity] row.
  updateLocal,

  /// The trail already has a local row -- update it in place.
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

/// Updates the local row for [localId] in place from [trail], preserving its
/// identity, ownership and sync bookkeeping.
///
/// Looks the existing row up by [localId] INSIDE the transaction. If no such
/// row exists, this is a no-op -- there is nothing to update. Otherwise a
/// fresh entity is built from [trail], then the existing row's `obxId`,
/// `id`, `owner`, `localId`, `syncState`, `syncAttempts`,
/// `syncNextAttemptAt` and `savedByUserIds` are carried forward onto it
/// before the put, so a metadata re-edit never changes the row's identity or
/// ownership (REC-05, SYNC-05).
///
/// [WaypointEntity] rows that belonged to the old row but are absent from
/// the new waypoint set are removed, so a deleted waypoint does not linger.
void updateLocalTrail(
  Store store, {
  required Trail trail,
  required String localId,
  required List<String> trailLocalPhotos,
  required Map<String, List<String>> waypointLocalPhotosByKey,
}) {
  store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    final query = trailBox
        .query(TrailEntity_.localId.equals(localId))
        .build();
    final existing = query.findFirst();
    query.close();
    if (existing == null) return;

    final entity = TrailEntity.fromModel(trail);
    entity.obxId = existing.obxId;
    entity.id = existing.id;
    entity.owner = existing.owner;
    entity.localId = existing.localId;
    entity.syncState = existing.syncState;
    entity.syncAttempts = existing.syncAttempts;
    entity.syncNextAttemptAt = existing.syncNextAttemptAt;
    entity.savedByUserIds = existing.savedByUserIds;
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
    final query = trailBox
        .query(TrailEntity_.localId.equals(localId))
        .build();
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
