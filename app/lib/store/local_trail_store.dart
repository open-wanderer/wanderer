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
/// be conflated with [TrailEntity.savedByUserIds] -- `owner` is 1:1
/// authorship-on-this-device, `savedByUserIds` is 1:N offline-library
/// membership. Every function in this file that carries `savedByUserIds`
/// forward does so verbatim, never as an ownership check.
///
/// Every function that takes an account id as a parameter requires the
/// CALLER to have re-read it fresh via `currentAccountId(store)` at the
/// point of use, never from a long-lived cached value -- a stale cached id
/// is exactly the leak that would show one account's captures to another.
library;

import 'package:flutter/foundation.dart';
import 'package:wanderer/entities/actor_entity.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/util/local/id.dart';

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
/// `trail.id.isEmpty` alone is NOT a sufficient discriminator: an empty id
/// means both "never saved anywhere" AND "saved locally, still unsynced".
/// Collapsing those two into one branch would route a re-save of an
/// already-local trail back through [createLocal], minting a second local id
/// and creating a second row where only one edit session happened.
/// [Trail.localId] is what actually distinguishes them.
///
/// This is the COARSE decision: it sees exactly one [Trail] and cannot tell
/// whether that trail is the screen's own in-memory snapshot or the
/// persisted ObjectBox row, so it can be wrong for a row whose upload
/// finished after the caller's snapshot was taken. [resolveLocalSaveModeForRow]
/// is the authority whenever a persisted row is available -- it is what a
/// caller holding a `localId` should call instead of this function directly.
/// And even [resolveLocalSaveModeForRow] is only a routing decision, not a
/// legality check: [updateLocalTrail] is the final authority on whether a
/// local write actually lands, via [LocalUpdateOutcome.alreadyUploaded] and
/// [LocalUpdateOutcome.alreadySynced].
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
/// the capture path can populate it -- ownership ("I recorded this")
/// and library membership ("I downloaded this") are kept strictly
/// separate.
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
///
/// Also routes to [LocalSaveMode.networkUpdate] up front, before
/// [resolveLocalSaveMode] ever runs, when [persisted] is non-null and is
/// EITHER already [TrailSyncState.synced] OR already carries a real server
/// id ([trailHasServerId]) regardless of [TrailSyncState] -- the exact
/// `alreadyUploaded` shape [updateLocalTrail] refuses. Anticipating
/// that refusal here, rather than discovering it only after
/// [updateLocalTrail] returns, is what keeps the caller's
/// `_copyPhotosForLocalSave` from ever running against a row the store is
/// going to refuse: a save doomed to be refused must never reach the
/// filesystem step first.
LocalSaveMode resolveLocalSaveModeForRow({
  required Trail screenTrail,
  required String? persistedLocalId,
  required Trail? persisted,
}) {
  if (persistedLocalId != null && persisted == null) {
    return LocalSaveMode.networkUpdate;
  }
  if (persisted != null &&
      (persisted.syncState == TrailSyncState.synced ||
          trailHasServerId(persisted.id))) {
    return LocalSaveMode.networkUpdate;
  }
  return resolveLocalSaveMode(persisted ?? screenTrail);
}

/// Whether [id] is a real, server-assigned trail id rather than the blank
/// placeholder [Trail.id] carries for a row that has never reached the
/// server.
///
/// `TrailEntity.toModel()` blanks a still-local sentinel id to `''`,
/// so this is the one signal that survives from ObjectBox all the way to a
/// [Trail] model: non-empty here means `writeServerTrailId` already stamped
/// a real id onto the row, independent of [TrailSyncState] entirely -- which
/// is what makes [TrailSyncState.failed] in particular an unreliable stand-in
/// for "the device holds the only copy": a `failed` row can still carry a
/// real server id from a create that succeeded before a later waypoint
/// upload failed. Empty here means there is nothing to route a
/// network write or a network delete to.
bool trailHasServerId(String id) => id.isNotEmpty;

/// Decides which id a network save should target once
/// [resolveLocalSaveModeForRow] has already routed the save to
/// [LocalSaveMode.networkUpdate].
///
/// Returns [screenTrailId] when it is already a real server id -- the
/// ordinary case, where the screen's own snapshot is trustworthy. Otherwise
/// falls back to [retiredServerId]: the id [retireUploadedLocalTrail]
/// returned when it retired this trail's row, memoized by `TrailSync` for
/// exactly this situation -- a still-mounted create/edit screen
/// whose `trail.id` reads `''` (`toModel()` blanks a local-sentinel id)
/// because the
/// upload finished after this screen's last snapshot and there are no
/// ObjectBox `Query.watch()` streams to tell it. Returns null when neither
/// is a real id: there is no target at all, and the caller must refuse the
/// write rather than issue `POST /trail/form/` with a blank id, which can
/// never be routed (SvelteKit normalizes an empty `[id]` segment away). The
/// guard was always correct; what was missing was a fallback target and a
/// message the hiker can act on.
String? resolveNetworkSaveTarget({
  required String screenTrailId,
  required String? retiredServerId,
}) {
  if (trailHasServerId(screenTrailId)) return screenTrailId;
  if (retiredServerId != null && trailHasServerId(retiredServerId)) {
    return retiredServerId;
  }
  return null;
}

/// What a delete decision should do once the server DELETE for a row's
/// server copy has been attempted (or was never attempted at all).
enum ServerDeleteOutcome {
  /// Proceed with the local delete -- either the server DELETE succeeded,
  /// or the server has already confirmed there is nothing left to delete
  /// (404).
  proceedWithLocalDelete,

  /// Refuse the delete and tell the hiker a connection is needed. Nothing
  /// local is touched.
  abortNeedsConnection,

  /// Refuse the delete and report a generic failure. Nothing local is
  /// touched.
  abortAndReport,
}

/// Decides what a failed (or 404'd) server DELETE means for the local
/// delete that would otherwise follow it.
///
/// - `statusCode == 404`: the server copy is already gone -- proceeding
///   with the local delete is strictly better than blocking it forever
///   over a trail the server itself no longer has.
/// - `statusCode == null && connectionFailed`: the request never reached a
///   server at all. [abortNeedsConnection], not a local-only delete --
///   there is deliberately NO "delete on this device only" escape hatch
///   for the offline case here. That would destroy the device's only
///   pointer to a trail that may still be live (and possibly public) on
///   the server, which is precisely the failure this decision exists to
///   prevent. An offline hiker is told to reconnect instead, and
///   nothing is destroyed.
/// - Every other case (401, 403, 500, or a null status with
///   `connectionFailed == false`) is [abortAndReport]: the failure is real
///   and reported, rather than silently treated as success or retried
///   forever.
ServerDeleteOutcome resolveServerDeleteOutcome({
  required int? statusCode,
  required bool connectionFailed,
}) {
  if (statusCode == 404) {
    return ServerDeleteOutcome.proceedWithLocalDelete;
  }
  if (statusCode == null && connectionFailed) {
    return ServerDeleteOutcome.abortNeedsConnection;
  }
  return ServerDeleteOutcome.abortAndReport;
}

/// Whether [entity]'s upload is due to run now.
///
/// True only for a row whose [TrailEntity.syncState] is [TrailSyncState.pending]
/// or [TrailSyncState.uploading], AND whose [TrailEntity.syncNextAttemptAt]
/// is either unset or not in the future relative to [now].
///
/// A [TrailSyncState.failed] row is deliberately NEVER due: a row that
/// exhausted its retry budget is parked for manual retry only
/// ([resetDrainBackoff]), not automatic pickup by the next drain pass.
bool isDrainDue(TrailEntity entity, DateTime now) {
  final isUploadable =
      entity.syncState == TrailSyncState.pending ||
      entity.syncState == TrailSyncState.uploading;
  if (!isUploadable) return false;

  final nextAttempt = entity.syncNextAttemptAt;
  return nextAttempt == null || !nextAttempt.isAfter(now);
}

/// Whether [waypoints] contains a waypoint that still needs creating (its
/// `id` is a local sentinel) but has no `localKey` to record the server's
/// returned id against.
///
/// An INVARIANT BREAK, not a network condition:
/// `WaypointEntity.fromModel` always mints a `localKey` for an empty-id
/// waypoint, so a true result means the row is corrupt in a way no number of
/// drain retries can fix. `_drainOne` must bail on this BEFORE joining the
/// in-flight set and before its `try`, so a corrupt waypoint costs the trail
/// nothing from [kMaxSyncAttempts] -- the identical reasoning already applied
/// to a missing [UserEntity] one guard up.
///
/// Takes a plain record list rather than [WaypointEntity] so the decision is
/// testable without a live [Store], matching this file's existing
/// extract-the-pure-half discipline (see [resolveDrainFailureOutcome],
/// [isDrainDue]).
bool hasKeylessPendingWaypoint(
  List<({String id, String? localKey})> waypoints,
) {
  return waypoints.any((w) => isLocalId(w.id) && w.localKey == null);
}

/// Pure decision core of [recordDrainFailure], extracted so its
/// attempt-count boundary is unit-testable without a live ObjectBox [Store]
/// (there is no ObjectBox test harness for plain `flutter test`).
typedef DrainFailureOutcome = ({
  TrailSyncState syncState,
  int syncAttempts,
  DateTime? syncNextAttemptAt,
});

/// Computes the next [TrailSyncState]/attempt-count/backoff-deadline after
/// one more failed upload attempt.
///
/// When the incremented attempt count reaches [maxAttempts], the row is
/// parked as [TrailSyncState.failed] with no further scheduled retry.
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

  /// The row's id is already a real server id, even though its
  /// [TrailSyncState] is NOT [TrailSyncState.synced] (still
  /// `pending`/`uploading`/`failed`). Nothing was written -- see
  /// [updateLocalTrail]'s doc comment for why.
  alreadyUploaded,
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
/// identity or ownership.
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
///
/// ALSO refuses, returning [LocalUpdateOutcome.alreadyUploaded], when the
/// existing row's `id` is already a real server id (`!isLocalId(existing.id)`)
/// even though its [TrailEntity.syncState] has not (yet) reached
/// [TrailSyncState.synced]. `writeServerTrailId` stamps that id the instant
/// `PUT /trail/form` is accepted -- well before the drain's waypoint loop
/// finishes or [retireUploadedLocalTrail] runs -- so a row parked as
/// `pending`/`uploading`/`failed` can carry a real id for the rest of its
/// (unbounded) time in that state. The drain's own create step is guarded on
/// `isLocalId(entity.id)`, so it has no update path for a row in this
/// window either: writing the edit here would sit on the row until a later
/// retry succeeds and [retireUploadedLocalTrail] destroys it, silently
/// discarding the edit under a green "trail saved successfully" toast.
/// The caller must route this case to the network write instead,
/// exactly as it already does for [LocalUpdateOutcome.alreadySynced] and
/// [LocalUpdateOutcome.missing].
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
    if (!isLocalId(existing.id)) {
      return LocalUpdateOutcome.alreadyUploaded;
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

/// Reconciles the local row for [localId], owned by [accountId], onto the
/// metadata [trail] carries -- called ONLY after a successful network save
/// for a row [resolveLocalSaveModeForRow] routed to [LocalSaveMode.networkUpdate]
///
/// This is the missing half of the `alreadyUploaded` fix:
/// [resolveLocalSaveModeForRow] correctly sends the edit to the server, but
/// nothing wrote it back onto the local row -- so the row kept its pre-edit
/// values under a real server id, and `mergeOwnTrails` (which dedupes a
/// network hit against any local row carrying the same non-empty id) hid the
/// server's up-to-date copy behind the stale local one, indefinitely, under
/// a green "trail saved successfully" toast. Calling this right after the
/// network save succeeds -- and BEFORE the own-trails list is invalidated
/// and re-read -- closes that window.
///
/// No dirty flag, no ObjectBox schema change: a dirty flag is only needed
/// when a local write happens BEFORE the edit reaches the server, so the
/// device has to remember it owes the server a push. That is not this
/// shape -- this function runs AFTER `POST /trail/form/{id}` already
/// succeeded, reconciling the row to a known-good server state. There is
/// nothing new to track and nothing left to push. The tradeoff accepted: if
/// THIS write throws, the row stays stale until `retireUploadedLocalTrail`
/// removes it -- which is exactly today's behaviour (nothing currently
/// reconciles the row at all), not a regression, and it is bounded by
/// retirement, so the caller should catch and log rather than surface a
/// local-write failure as a failed save when the server already accepted it.
///
/// Query is scoped by BOTH [localId] AND [accountId]:
/// [accountId] must be read fresh via `currentAccountId(store)` at the call
/// site, never a cached value, or account B's edit could be written onto
/// account A's row. Silently returns when no row matches -- the row may
/// have been retired (by the drain finally succeeding) between the network
/// save and this call landing; that is an ordinary race, not an error.
///
/// Writes ONLY the metadata columns the hiker can edit through this form:
/// `name`, `location`, `date`, `public`, `completed`, `difficulty`,
/// `description`, `categoryRecordId`, `subcategoryRecordId`, `tagsJson` (via
/// [encodeTrailTags], the exact same encoding [TrailEntity.fromModel] uses),
/// and `updated`. Mutates the FOUND entity in place -- deliberately never
/// via `TrailEntity.fromModel`, which would erase `photos`/`localPhotos` and
/// drop the sync bookkeeping. Never touches `id`, `owner`, `localId`,
/// `syncState`, `syncAttempts`, `syncNextAttemptAt`, `savedByUserIds`,
/// `photos`, `localPhotos`, `gpxData` or `waypoints` -- each omission is
/// deliberate: those are the row's identity, ownership, sync
/// resume position, and the fields with no source of truth in a plain
/// [Trail] model, and clobbering any of them here would either corrupt the
/// drain's resume state or silently discard data this function was never
/// given.
void applyNetworkEditToLocalRow(
  Store store, {
  required String localId,
  required String accountId,
  required Trail trail,
}) {
  store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    final query = trailBox
        .query(
          TrailEntity_.localId.equals(localId) &
              TrailEntity_.owner.equals(accountId),
        )
        .build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    entity.name = trail.name;
    entity.location = trail.location;
    entity.date = trail.date;
    entity.public = trail.public;
    entity.completed = trail.completed;
    entity.difficulty = trail.difficulty;
    entity.description = trail.description;
    entity.categoryRecordId = trail.category;
    entity.subcategoryRecordId = trail.subcategory;
    entity.tagsJson = encodeTrailTags(trail.expand?.tags);
    entity.updated = trail.updated;

    trailBox.put(entity);
  });
}

/// Refreshes the downloaded LIBRARY row for [trail], keyed on library
/// membership rather than local-capture ownership -- the counterpart to
/// [applyNetworkEditToLocalRow] for a row [applyNetworkEditToLocalRow] can
/// never reach.
///
/// [applyNetworkEditToLocalRow] keys on `TrailEntity.localId` +
/// `TrailEntity.owner` -- the unsynced-capture identity. A downloaded row has
/// `localId == null`, so that function's query can never match it, which is
/// exactly bug 2: the hiker's own edit of a downloaded trail never
/// reached the stored copy. This function is a SEPARATE write path, keyed on
/// `TrailEntity.id` + `TrailEntity.savedByUserIds.containsElement(accountId)`
/// -- the same predicate `TrailNotifier._readCached` already builds to read
/// library membership.
///
/// Two triggers call this, both best-effort:
/// - The save trigger: right after the hiker's own successful
///   `POST /trail/form/{id}`, so their edit lands on the downloaded copy
///   without a re-download.
/// - The fetch trigger: any successful fetch of a trail this account has
///   downloaded, opportunistically refreshing its metadata and track.
///
/// Both triggers cost ZERO additional network traffic -- the server record
/// and the GPX are already in hand by the time either call site runs.
/// Photos are deliberately never touched here: they are the one
/// asset not already in the response, so an automatic refresh must never
/// spend bytes on them -- that stays the explicit *Update* action's job.
///
/// [accountId] MUST be read fresh via `currentAccountId(store)` at the call
/// site, never a cached value -- a stale id would let account B's fetch
/// write onto account A's row.
///
/// A silent no-op, never an error, when [trail]'s `id` is empty (a
/// local-sentinel id must never match a row) or when no row in this
/// account's library matches -- either means there is nothing to refresh.
///
/// Rebuilds the row via [TrailEntity.fromModel] (the [updateLocalTrail]
/// precedent), then carries forward every column `fromModel` cannot know or
/// would wrongly reset: `obxId`, `id`, `owner`, `localId`, `syncState`,
/// `syncAttempts`, `syncNextAttemptAt`, `savedByUserIds`, `photos`,
/// `localPhotos`, `navCacheJson`. `savedByUserIds`, `photos` and
/// `localPhotos` are the load-bearing three -- dropping `savedByUserIds`
/// would erase every account's claim on the trail, and `photos` on a
/// downloaded row holds LOCAL FILE PATHS written by `TrailDownloadService`,
/// not server filenames, so overwriting it would strand the downloaded
/// images: nothing automatic ever spends bytes, and nothing automatic ever
/// destroys already-spent ones.
///
/// `gpxData` is guarded: when the incoming `trail.expand?.gpxData` is null or
/// empty, the existing row's `gpxData` is kept rather than blanking the
/// stored track. When it is present, it is taken -- the track refresh, a
/// single ObjectBox string write with no file I/O.
///
/// The author/category relations and `authorRecordId`/`categoryRecordId`
/// fall back to the existing row's values whenever the incoming model
/// carries no expanded author/category (or no scalar id), so a partial
/// response cannot drop them.
///
/// Waypoints are only touched when `trail.expand?.waypointsViaTrail` is
/// non-null. When it is, each existing child's `localPhotos` is carried onto
/// its refreshed counterpart by matching `WaypointEntity.id` -- unconditionally,
/// since it only matches by id and is harmless either way. What happens next
/// depends on [waypointsAreAuthoritative]:
/// - `true` (the default): any `WaypointEntity` row absent from the
///   refreshed set is removed, following [updateLocalTrail]'s orphan-pruning
///   loop. The fetch trigger (`trail_provider.dart`) keeps this default
///   -- a full server fetch IS authoritative, and must stay so, or
///   server-side waypoint deletes stop propagating.
/// - `false`: nothing is pruned. Any existing waypoint absent from the
///   incoming set is re-added to `entity.waypoints` as-is instead, because
///   the incoming list is known-incomplete. The save trigger
///   (`trail_create_screen.dart`) passes `!result.hadWaypointFailures` --
///   `TrailSaveResult.finalWaypoints` drops any waypoint whose create/update
///   threw, so treating it as a complete set here used to delete a
///   still-live waypoint from the hiker's offline copy and orphan its photo
///   files, under a warning toast that said nothing about local deletion.
/// The row still reflects the last state the server confirmed,
///   which is truthful -- the failed PATCH did not happen server-side
///   either.
/// When `trail.expand?.waypointsViaTrail` is null, the existing children are
/// carried forward untouched regardless of [waypointsAreAuthoritative].
void applyServerTrailToLibraryRow(
  Store store, {
  required String accountId,
  required Trail trail,
  bool waypointsAreAuthoritative = true,
}) {
  if (trail.id.isEmpty) return;

  store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    final query = trailBox
        .query(
          TrailEntity_.id.equals(trail.id) &
              TrailEntity_.savedByUserIds.containsElement(accountId),
        )
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
    entity.photos = existing.photos;
    entity.localPhotos = existing.localPhotos;
    entity.navCacheJson = existing.navCacheJson;

    final incomingGpxData = trail.expand?.gpxData;
    entity.gpxData = (incomingGpxData != null && incomingGpxData.isNotEmpty)
        ? incomingGpxData
        : existing.gpxData;

    if (trail.expand?.author == null) {
      entity.author.target = existing.author.target;
    }
    entity.authorRecordId ??= existing.authorRecordId;

    if (trail.expand?.category == null) {
      entity.category.target = existing.category.target;
    }
    entity.categoryRecordId ??= existing.categoryRecordId;

    if (trail.expand?.waypointsViaTrail != null) {
      // Carry each existing child's downloaded photos onto its refreshed
      // counterpart by id -- never touched over the network here.
      final existingLocalPhotosById = {
        for (final w in existing.waypoints) w.id: w.localPhotos,
      };
      for (final waypointEntity in entity.waypoints) {
        final photos = existingLocalPhotosById[waypointEntity.id];
        if (photos != null) waypointEntity.localPhotos = photos;
      }

      final newIds = entity.waypoints.map((w) => w.id).toSet();
      if (waypointsAreAuthoritative) {
        // Remove waypoint rows absent from the refreshed set, mirroring
        // updateLocalTrail's orphan-pruning loop. Only safe when the
        // caller has told us the incoming set is complete.
        final waypointBox = store.box<WaypointEntity>();
        for (final oldWaypoint in existing.waypoints) {
          if (!newIds.contains(oldWaypoint.id)) {
            waypointBox.remove(oldWaypoint.obxId);
          }
        }
      } else {
        // The incoming set is known-incomplete: re-add any existing
        // waypoint it omits rather than pruning it. Not marked, no entity
        // field added -- its photo files under
        // `library/<id>/waypoints/<key>/` stay valid.
        for (final oldWaypoint in existing.waypoints) {
          if (!newIds.contains(oldWaypoint.id)) {
            entity.waypoints.add(oldWaypoint);
          }
        }
      }
    } else {
      // No waypoint expand in this response -- fromModel left `entity.waypoints`
      // empty, so carry the existing children forward untouched rather than
      // silently dropping them.
      entity.waypoints.addAll(existing.waypoints);
    }

    trailBox.put(entity);
  });
}

/// What [deleteLocalTrailRow] actually did.
///
/// A plain `bool` could only say "I matched a row"; it could not tell the
/// caller whether the ROW is gone, and that is the fact the caller needs
/// in order to know whether `library/<serverId>/` still has a referent
enum LocalRowDeleteOutcome {
  /// The row and its [WaypointEntity] children were removed outright. No
  /// account held it in `savedByUserIds`, so nothing on this device points
  /// at `library/<serverId>/` any more and the caller owns reclaiming it.
  removed,

  /// A row owned by [accountId] matched and its capture bookkeeping was
  /// cleared, but the ROW WAS KEPT because some account still holds it in
  /// its offline library. This account's capture state is gone (which is
  /// what it asked for); the other account's library entry survives.
  demotedStillHeld,

  /// No row carrying this `localId` is owned by this account. NOTHING was
  /// touched, and the caller must not delete files or report success.
  noMatch,
}

/// Ends the local capture for [localId], owned by [accountId].
///
/// A no-op for an unknown [localId] (or one owned by a different account),
/// mirroring `TrailLibraryNotifier.deleteTrail`'s silent no-op precedent.
/// Owner-scoped and the project rule "scope, don't delete user
/// data": the `localId` this is called with may arrive from a [Trail] model
/// constructed before an account switch, and without the clause account B
/// could delete account A's device-only row. Does NOT touch the filesystem
/// -- the caller pairs this with `deleteUnsyncedPhotoDir` (from
/// `local_photo_store.dart`).
///
/// [shouldDeleteUploadedRow] picks between two shapes, exactly as
/// [retireUploadedLocalTrail] does for the other way a capture's life ends:
/// - No library membership (the ordinary case): the row and its
///   [WaypointEntity] children are removed outright.
/// - Some account holds it in its offline library: the row is KEPT and only
///   the capture bookkeeping is cleared, so it becomes an ordinary
///   downloaded row. `TrailEntity.id` is `@Unique(onConflict: replace)`, so
///   once `writeServerTrailId` has stamped a server id another account's
///   download lands in THIS row. Removing it outright destroyed that
///   account's library entry and orphaned `library/<serverId>/` -- whose
///   only referent was the `entity.photos` column on the row just removed
///   -- with no `library/` sweep anywhere in the app to reclaim it: a
///   destructive action scoped by one identity (`owner`) destroying state
///   belonging to another (`savedByUserIds`).
///
/// Returns which of those happened, or [LocalRowDeleteOutcome.noMatch]
/// [LocalRowDeleteOutcome.noMatch]. `noMatch` means no row owned by
/// [accountId] carried this [localId], so the caller deleted NOTHING and
/// must not go on to delete files or report success. Without it, account B
/// tapping delete with account A's `localId` matched no row -- correctly --
/// yet the caller went on to delete the photo directory and report
/// `deleted`, because a `void` signature made no-match indistinguishable
/// from success.
LocalRowDeleteOutcome deleteLocalTrailRow(
  Store store,
  String localId, {
  required String accountId,
}) {
  return store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    final query = trailBox
        .query(
          TrailEntity_.localId.equals(localId) &
              TrailEntity_.owner.equals(accountId),
        )
        .build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return LocalRowDeleteOutcome.noMatch;

    if (!shouldDeleteUploadedRow(entity.savedByUserIds)) {
      // Demote to an ordinary downloaded row -- the same shape
      // `retireUploadedLocalTrail` produces, so nothing downstream needs to
      // know it was ever a capture. The waypoint children stay: they are
      // part of the downloaded copy the other account still holds.
      entity.owner = null;
      entity.localId = null;
      entity.syncState = TrailSyncState.synced;
      entity.syncAttempts = 0;
      entity.syncNextAttemptAt = null;
      entity.localPhotos = [];
      trailBox.put(entity);
      return LocalRowDeleteOutcome.demotedStillHeld;
    }

    final waypointBox = store.box<WaypointEntity>();
    for (final waypoint in entity.waypoints) {
      waypointBox.remove(waypoint.obxId);
    }
    trailBox.remove(entity.obxId);
    return LocalRowDeleteOutcome.removed;
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
///
/// Returns `serverId` -- the server id the row carried at the moment it was
/// retired, or null when no row matched -- alongside `rowRemoved`, which
/// distinguishes the two branches above.
///
/// `rowRemoved` exists because the remove branch can strand
/// `library/<serverId>/`. That branch runs precisely when `savedByUserIds`
/// is empty, so no account holds the trail offline any more, yet the
/// directory (populated by `TrailDownloadService` back when someone did)
/// outlives the only row that referenced it -- `entity.photos` held the
/// absolute paths into it and there is no `library/` sweep anywhere in the
/// app. `TrailLibraryNotifier.deleteTrail`'s keep-the-row path reaches this
/// state legitimately: it writes `savedByUserIds = []` onto a row it keeps
/// because the row is a live capture, and this function later removes that
/// row. The caller pairs `rowRemoved` with a best-effort directory reclaim,
/// exactly as `deleteUnsynced` does on its own `removed` branch.
///
/// The id is the trail's identity after this
/// row is gone -- once retirement runs (either branch), it is the ONLY
/// handle a still-mounted create/edit or detail screen has left on the
/// trail: there are no ObjectBox `Query.watch()` streams anywhere
/// in this app, so a screen's own `trail.id` snapshot, taken while the row
/// was still local, keeps reading the blank local-sentinel value forever
/// unless the caller captures and memoizes this return value.
({String? serverId, bool rowRemoved}) retireUploadedLocalTrail(
  Store store,
  String localId,
) {
  return store.runInTransaction(TxMode.write, () {
    final trailBox = store.box<TrailEntity>();
    // No account scoping and no `owner` clause: the only caller reached
    // this row through `selectDrainCandidates`, which is already
    // owner-scoped, and the key is a `localId` this device minted.
    final query = trailBox.query(TrailEntity_.localId.equals(localId)).build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return (serverId: null, rowRemoved: false);

    // Captured before the row is mutated or removed by either branch below
    // -- see this function's doc comment.
    final serverId = isLocalId(entity.id) ? null : entity.id;

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
      return (serverId: serverId, rowRemoved: false);
    }

    final waypointBox = store.box<WaypointEntity>();
    for (final waypoint in entity.waypoints) {
      waypointBox.remove(waypoint.obxId);
    }
    trailBox.remove(entity.obxId);
    return (serverId: serverId, rowRemoved: true);
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
/// the exact leak account scoping exists to prevent, arriving through a door
/// `readOwnLocalTrails` had already bolted.
///
/// Deliberately owner-only, with NO `savedByUserIds` clause: ownership
/// ("I made this") and library membership ("I downloaded this") stay
/// strictly separate.
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

/// The raw, persisted [TrailEntity.id] for the row identified by [localId]
/// and owned by [accountId], or null when no such row exists or the row's
/// id is still a local sentinel.
///
/// Deliberately NOT via [TrailEntity.toModel()]: a delete decision
/// must not depend on the row's cached GPX still parsing.
/// `TrailEntity.toModel()` runs `parseGpxSafely` and returns null on ANY
/// conversion failure -- including one caused only by unparseable GPX --
/// which previously made `deleteUnsynced` treat a row with a real,
/// server-stamped id as if it had none, skipping the server DELETE and
/// stranding a live (possibly public) server trail with no device left
/// pointing at it. Reading `entity.id` off the column sidesteps `toModel()`
/// entirely.
///
/// Owner-scoped for the same reason [readOwnLocalTrail] is: the
/// `localId` this is called with arrives from a [Trail] model that may have
/// been constructed before an account switch, or retained by a
/// backgrounded widget -- the same class of value `readOwnLocalTrail`
/// already guards against leaking across accounts.
String? readLocalTrailServerId(
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

  return isLocalId(entity.id) ? null : entity.id;
}

/// The single owner-scoped live-capture predicate.
///
/// True when [entity] is some account's not-yet-uploaded capture. This is
/// the ONE rule; everything around it ([isOwnLiveCapture],
/// `ownLiveCaptureProvider`) is a scoping wrapper -- there must be exactly
/// one predicate, never two. A row satisfying this
/// predicate is some account's not-yet-uploaded capture, and its
/// [TrailEntity] row is the only handle `selectDrainCandidates` has on it,
/// so removing the row destroys the recording permanently.
///
/// `entity.owner != null` is load-bearing: [TrailEntity.fromModel] (the
/// download path) never sets `owner`, so a plain downloaded row has
/// `owner == null` and must never satisfy this predicate.
bool isLiveCaptureRow(TrailEntity entity) {
  return entity.localId != null &&
      entity.owner != null &&
      isUnsyncedState(entity.syncState);
}

/// [isLiveCaptureRow], scoped to the row identified by [localId] and owned
/// by [accountId].
///
/// Returns `false` immediately when [localId] is null or fails
/// [isLocalId]. Otherwise builds the same owner-scoped query
/// [readOwnLocalTrail] builds and evaluates [isLiveCaptureRow] against the
/// raw row -- deliberately NOT via `entity.toModel()`, for the same reason
/// [readLocalTrailServerId] avoids it: a
/// destructive-action gate must not silently flip because the row's cached
/// GPX stopped parsing. This function consults the ROW's own `syncState`
/// and `owner`, never the caller's `Trail` model, because the shared cache
/// row can carry another account's `localId` and non-synced `syncState`
/// indefinitely -- unsynced and downloaded are not mutually exclusive.
bool isOwnLiveCapture(
  Store store, {
  required String? localId,
  required String accountId,
}) {
  if (localId == null || !isLocalId(localId)) return false;

  final query = store
      .box<TrailEntity>()
      .query(
        TrailEntity_.localId.equals(localId) &
            TrailEntity_.owner.equals(accountId),
      )
      .build();
  final entity = query.findFirst();
  query.close();
  return entity != null && isLiveCaptureRow(entity);
}

/// Every trail [accountId] can see in its own-trails list: trails it
/// captured on this device (not yet uploaded, or uploaded already, or
/// downloaded), plus any downloaded trail it happens to have authored
/// itself.
///
/// The query casts a broad net (`owner == accountId` OR `savedByUserIds`
/// contains `accountId`), then keeps a row only when `entity.owner ==
/// accountId` OR (`authorActorId != null && entity.author.target?.id ==
/// authorActorId`). The second clause is the "downloaded trails the
/// hiker authored themselves"; the `owner` clause is every not-yet-uploaded
/// capture. A row whose `owner` is null can NEVER satisfy the first clause
/// -- that is what keeps a pre-existing downloaded row (owner unset) from
/// leaking into another account's own-trails list.
///
/// Converted with the same per-entity `try/catch` guard
/// [TrailLibraryNotifier.build] uses, sorted by `created` descending, and
/// returned as a flat list -- no sectioning and no special sort.
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
      // includeGpx: false — same list-surface rationale as
      // trailLibraryProvider: these rows feed the profile own-trails cards,
      // which render scalar columns only; a tap re-reads the full model via
      // readOwnLocalTrail/trailProvider.
      trails.add(entity.toModel(includeGpx: false));
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
/// warning.
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
/// (account scoping hides another account's content, it never deletes it).
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

/// The load-bearing write: stamps the server-assigned [serverId] and
/// [serverPhotoFilenames] onto the local row identified by [localId], leaving
/// `localId`, `owner` and `syncState` untouched.
///
/// Must be callable, and must commit, the instant `PUT /trail/form`
/// returns, before any waypoint upload starts: there is no server-side
/// idempotency key, so a crash between "server accepted" and "id persisted"
/// is what produces a duplicate trail.
///
/// [serverPhotoFilenames] commits in the SAME transaction as [serverId] on
/// purpose. The two facts are learned from one response and are only
/// meaningful together: a row that has the server id but not the photo list
/// looks, to a resumed drain, exactly like a trail with no photos at all --
/// which is how the since-removed `markTrailSynced` came to persist an
/// empty `photos` column and `deleteUnsyncedPhotoDir` came to delete the
/// only copies left on the device. It is nullable, and null means "leave
/// `photos` alone", so a
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
/// the trail [localId].
///
/// Deliberately does NOT clear `localPhotos`: the trail's upload as
/// a whole is not complete the instant one waypoint's create succeeds -- a
/// LATER waypoint in the same drain loop can still fail and park the row as
/// `failed`, and a waypoint whose `localPhotos` was already wiped would then
/// be unreachable through the model while its JPEGs still sit on disk under
/// `unsynced/<localId>/waypoints/<key>/`. `retireUploadedLocalTrail` (full
/// success) and `deleteUnsyncedPhotoDir` (delete) own reclaiming those files
/// once the trail's fate is actually decided.
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
    // localPhotos deliberately retained -- see this function's doc comment
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

/// Records one more failed upload attempt for the local row [localId].
///
/// Delegates the attempt-count/backoff decision to
/// [resolveDrainFailureOutcome] -- see that function's doc comment for the
/// parking behaviour.
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

/// The manual-retry primitive: resets the local row for [localId],
/// owned by [accountId], back to a fresh [TrailSyncState.pending] state
/// with zero attempts and no scheduled backoff.
///
/// Owner-scoped and the project rule "scope, don't delete user
/// data": `SyncStatusChip`'s retry tap reaches this with the same kind of
/// value [deleteLocalTrailRow] guards against -- a `localId` that may have
/// survived an account switch.
void resetDrainBackoff(
  Store store,
  String localId, {
  required String accountId,
}) {
  store.runInTransaction(TxMode.write, () {
    final box = store.box<TrailEntity>();
    final query = box
        .query(
          TrailEntity_.localId.equals(localId) &
              TrailEntity_.owner.equals(accountId),
        )
        .build();
    final entity = query.findFirst();
    query.close();
    if (entity == null) return;

    entity.syncState = TrailSyncState.pending;
    entity.syncAttempts = 0;
    entity.syncNextAttemptAt = null;
    box.put(entity);
  });
}
