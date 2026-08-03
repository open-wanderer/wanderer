import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/trail/form_data.dart';
import 'package:wanderer/util/local/id.dart';
import 'package:wanderer/store/local_photo_store.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/util/local/sync_backoff.dart';

part 'trail_sync_provider.g.dart';

/// The local ids of trails whose upload is currently draining, shared
/// across every trigger (app foreground, connectivity regained, cold
/// start, manual retry) exactly like `DownloadingTrailIds` shares download
/// state across its own entry points. `keepAlive` so an in-flight drain
/// survives whichever widget happens to rebuild or unmount mid-upload.
///
/// The KEY is the local id, not the server `id` -- a trail mid-drain may
/// still have no server id at all (pre-create) or one assigned moments ago
/// (post-create, pre-waypoints), so only the local id is stable across the
/// whole resume-from-step sequence (D-05).
@Riverpod(keepAlive: true)
class TrailSync extends _$TrailSync {
  /// Whole-drain re-entrancy guard, separate from the per-trail [state] set.
  /// Both the app-foreground and connectivity-regained triggers
  /// (`main.dart`) can fire within milliseconds of each other on a flaky
  /// reconnect; without this a second call to [drainIfOnline] would start a
  /// second pass over the same candidate list while the first is still
  /// mid-flight.
  bool _draining = false;

  /// Set when [drainIfOnline] is called while a pass is already running, so
  /// the running pass loops once more instead of dropping the request.
  ///
  /// Without it, [retry] reset a row's backoff and then did nothing visible:
  /// `_draining` is a WHOLE-drain guard, not a per-trail one, so tapping
  /// "Upload failed · Tap to retry" while any OTHER trail was uploading
  /// returned immediately, and the reset row sat there until the next
  /// foreground or connectivity trigger happened to fire.
  bool _rerunRequested = false;

  @override
  Set<String> build() => {};

  /// The single entry point every trigger calls.
  ///
  /// Always refreshes [onlineStatusProvider] first and bails out
  /// immediately when it reports offline -- mandatory, not defensive:
  /// `OnlineStatus.build()` is optimistic (`true`), settles only from
  /// ordinary request/response traffic, and trusting it at a cold moment
  /// already shipped a bug once (Phase 35 OFFUI-04, RESEARCH.md Pitfall 5).
  /// A call arriving while a pass is already running is REMEMBERED, not
  /// dropped: it sets [_rerunRequested] so the running pass loops once more
  /// and picks up whatever changed. Returning immediately was what made
  /// [retry] a silent no-op whenever any other trail happened to be
  /// uploading.
  Future<void> drainIfOnline() async {
    if (_draining) {
      _rerunRequested = true;
      return;
    }
    _draining = true;
    try {
      do {
        // Cleared BEFORE the pass, so a request arriving mid-pass is honoured
        // by the next iteration rather than being cleared away by this one.
        _rerunRequested = false;
        await _drainPass();
      } while (_rerunRequested);
    } finally {
      _draining = false;
      _rerunRequested = false;
    }
  }

  /// One sweep over the currently-due candidates. Always re-reads online
  /// status and the account id, so a pass that runs because of a rerun
  /// request is as fresh as the first one.
  Future<void> _drainPass() async {
    // dart format off
    final isOnline = await ref.read(onlineStatusProvider.notifier).refresh();
    // dart format on
    if (!isOnline) return;

    final store = ref.read(objectBoxProvider);
    // D-13: read fresh every run, never a cached field -- a stale id here
    // is exactly the leak account-scoping exists to prevent.
    final accountId = currentAccountId(store);
    if (accountId == null) return;

    final candidates = selectDrainCandidates(
      store,
      accountId: accountId,
      now: DateTime.now(),
    );

    // Sequential, one trail at a time, on purpose: a flaky connection
    // cannot fan out into N concurrent partially-applied uploads, and the
    // in-flight set stays trivially easy to reason about for the delete
    // gate below.
    for (final entity in candidates) {
      await _drainOne(store, entity, accountId);
    }
  }

  /// Implements D-05's resume-from-step for a single trail.
  ///
  /// Guards on the entity having a [TrailEntity.localId] (nothing to key
  /// the in-flight set or the photo directory on otherwise), adds it to the
  /// in-flight set, marks the row `uploading`, then replays whichever steps
  /// of `PUT /tag` -> `PUT /trail/form` -> `PUT /waypoint` have not
  /// completed yet.
  Future<void> _drainOne(
    Store store,
    TrailEntity entity,
    String accountId,
  ) async {
    final localId = entity.localId;
    if (localId == null) return;
    if (state.contains(localId)) return;

    // Resolved BEFORE the in-flight set is joined and before the try below,
    // so a missing row costs nothing. A vanished UserEntity is not a network
    // condition at all -- it means the account row went away underneath us (a
    // mid-logout race, `account_data_purge_util` running concurrently). Thrown
    // inside the try it landed in the generic failure handler and consumed one
    // of the four kMaxSyncAttempts; four such passes, which a lifecycle or
    // connectivity flurry can produce within seconds, parked a perfectly good
    // trail as `failed`, after which isDrainDue returns false forever and only
    // a manual tap on the chip revives it. sync_backoff.dart's own doc comment
    // argues the attempt count should escalate only on real upload failures.
    final userQuery = store
        .box<UserEntity>()
        .query(UserEntity_.id.equals(accountId))
        .build();
    final userEntity = userQuery.findFirst();
    userQuery.close();
    if (userEntity == null) {
      debugPrint(
        'trail_sync_provider: no UserEntity for "$accountId"; skipping drain '
        'of "$localId" without recording a failed attempt',
      );
      return;
    }
    final authorId = userEntity.actorId;

    state = {...state, localId};

    try {
      markTrailUploading(store, localId);

      // Step 1: tag reuse/create (D-06) applies whether or not the trail
      // itself still needs to be created.
      final trailModel = entity.toModel();
      final resolvedTags = await ref
          .read(trailSaveProvider.notifier)
          .resolveTags(trailModel.expand?.tags ?? const []);

      var serverTrailId = entity.id;

      // Step 2: create the trail itself, only when it has not already been
      // created by a prior partial attempt.
      if (isLocalId(entity.id)) {
        final trailToSend = trailModel.copyWith(
          author: authorId,
          tags: resolvedTags.map((t) => t.id!).toList(),
        );
        final formData = await trailToSend.toFormData(
          newPhotos: trailModel.localPhotos.map((path) => File(path)).toList(),
          isCreate: true,
        );

        final response = await ref
            .read(apiProvider)
            .put(
              '/trail/form',
              data: formData,
              queryParameters: {'expand': 'category,tags,author'},
            );
        // SYNC-04: commits the instant the server accepted the create,
        // BEFORE any waypoint upload starts below. There is no server-side
        // idempotency key, so a crash between "server accepted" and "id
        // persisted" is exactly what produces a duplicate trail
        // (RESEARCH.md Pitfall 3).
        //
        // The id and photo list are pulled straight off the raw body rather
        // than out of `Trail.fromJson` below, because "the server accepted the
        // create" and "the client could deserialize the response" are
        // different events. An unexpected body shape, a schema change or a
        // null where the freezed model requires non-null throws in
        // `fromJson` -- and the id would never have been persisted, so the
        // next drain pass would `PUT /trail/form` again and create a SECOND
        // trail with no way to reconcile the two.
        //
        // The photo list rides along in the same transaction so that a drain
        // RESUMING at step 3 or 4 finds a truthful `photos` column. It used
        // to find `[]` -- the value `TrailEntity.fromModel` leaves behind --
        // and step 4 persisted that emptiness before deleting the on-disk
        // copies.
        final body = response.data;
        final rawBody = body is Map ? body : const {};
        final rawId = rawBody['id'];
        if (rawId is String && rawId.isNotEmpty) {
          final rawPhotos = rawBody['photos'];
          writeServerTrailId(
            store,
            localId: localId,
            serverId: rawId,
            serverPhotoFilenames: rawPhotos is List
                ? rawPhotos.whereType<String>().toList()
                : null,
          );
        }

        final created = Trail.fromJson(response.data);
        serverTrailId = created.id;
      }

      // Step 3: each waypoint that has not already been created.
      for (final waypointEntity in entity.waypoints) {
        if (!isLocalId(waypointEntity.id)) continue;

        // Hoisted ABOVE the create, and loud rather than a silent `continue`.
        // `WaypointEntity.fromModel` always mints a key for an empty-id
        // waypoint, so this branch should be unreachable -- which is exactly
        // what made skipping it the worst possible handling: the waypoint was
        // created server-side and its returned id then dropped, so a later
        // failure in this same loop re-created it (the CR-02 mechanism), and
        // the broken invariant left no trace anywhere.
        final waypointLocalKey = waypointEntity.localKey;
        if (waypointLocalKey == null) {
          throw StateError(
            'trail_sync_provider: waypoint ${waypointEntity.obxId} of '
            '"$localId" has no localKey',
          );
        }

        final waypointModel = waypointEntity.toModel();

        final Waypoint createdWaypoint;
        try {
          createdWaypoint = await ref
              .read(waypointSaveProvider.notifier)
              .create(
                waypointModel,
                authorId: authorId,
                trailId: serverTrailId,
              );
        } on WaypointPhotoUploadException catch (e) {
          // The record exists server-side; only its photos failed. Persist
          // the id NOW, exactly as SYNC-04 does for the trail itself, so the
          // next attempt resumes at the photo upload instead of re-running
          // `PUT /waypoint` and creating a duplicate. Then rethrow: this
          // trail's upload really did not complete, so it must still count a
          // failed attempt and be retried.
          writeServerWaypointId(
            store,
            localId: localId,
            waypointLocalKey: waypointLocalKey,
            serverWaypointId: e.created.id,
            serverPhotoFilenames: e.created.photos,
          );
          rethrow;
        }

        writeServerWaypointId(
          store,
          localId: localId,
          waypointLocalKey: waypointLocalKey,
          serverWaypointId: createdWaypoint.id,
          serverPhotoFilenames: createdWaypoint.photos,
        );
      }

      // Step 4: full success. The capture row has done its job and is
      // retired here. Per the 2026-08-03 product decision an uploaded
      // trail is reachable through the SERVER entry, not through a
      // retained device row; the hiker downloads it like any other trail
      // to have it offline again.
      //
      // Retiring in the same statement that proves the upload finished is
      // what makes the post-delete orphan structurally impossible. There
      // is no "synced but still local" row for a later delete to leave
      // behind, and no window in which a crash between "marked synced"
      // and "row removed" could strand one -- the row either still exists
      // and is still resumable, or it is gone.
      //
      // This also retires step 2's photo-list reasoning: the row's
      // `photos` column no longer has to survive anything, because the
      // row does not.
      retireUploadedLocalTrail(store, localId);
      // Best-effort, and deliberately NOT inside the failure handler's
      // reach: the row is already retired by this line, so an
      // ArgumentError from a malformed localId would otherwise reach
      // `recordDrainFailure`, which would find nothing to write and log a
      // failure for an upload that actually succeeded.
      //
      // Row first, files second: `unsyncedLocalIds` cannot see a retired
      // row, so a directory left behind by a crash between these two
      // lines is reclaimed by `main.dart`'s startup sweep. The reverse
      // order would leave a live row pointing at deleted files.
      await _deletePhotoDirBestEffort(localId);

      ref.invalidate(trailLibraryProvider);
      ref.invalidate(profileTrailsProvider('@${userEntity.preferredUsername}'));
    } catch (e, st) {
      // Step 5: any thrown failure -- network, parse, or otherwise -- is
      // caught here rather than escaping the notifier. The photo directory
      // is deliberately NOT deleted on failure; the retry still needs it.
      debugPrint('trail_sync_provider: drain failed for "$localId": $e\n$st');
      recordDrainFailure(
        store,
        localId: localId,
        now: DateTime.now(),
        maxAttempts: kMaxSyncAttempts,
        backoff: syncBackoffDelay,
      );
    } finally {
      state = {...state}..remove(localId);
    }
  }

  /// Deletes [localId]'s unsynced photo directory, swallowing and logging any
  /// failure.
  ///
  /// `deleteUnsyncedPhotoDir` swallows filesystem errors but evaluates
  /// `unsyncedTrailPhotoDir` -- and therefore the `^local-\d+-\d+$`
  /// validation -- outside its own try, on purpose, so a malformed id is a
  /// loud caller bug. Neither of this notifier's two call sites can act on
  /// that ArgumentError, and both run AFTER the row has already been deleted
  /// or marked synced, so both need it to degrade rather than escape.
  Future<void> _deletePhotoDirBestEffort(String localId) async {
    try {
      await deleteUnsyncedPhotoDir(localId);
    } catch (e, st) {
      debugPrint(
        'trail_sync_provider: photo dir cleanup failed for "$localId": '
        '$e\n$st',
      );
    }
  }

  /// SYNC-03's manual retry: resets the row's backoff/attempt bookkeeping
  /// then immediately re-drains.
  ///
  /// Safe to call on a row that is already draining -- [_drainOne]'s
  /// in-flight guard makes the resulting pass a no-op for THAT row. Safe to
  /// call while some OTHER trail is draining too: [drainIfOnline] records the
  /// request and the running pass loops again, so the reset row is picked up
  /// as soon as the current one finishes rather than waiting for the next
  /// foreground or connectivity trigger.
  Future<void> retry(String localId) async {
    final store = ref.read(objectBoxProvider);
    resetDrainBackoff(store, localId);
    await drainIfOnline();
  }

  /// D-14: deletes the local row and its photo copies for [localId] --
  /// and, when the row already carries a real server id, the server's copy
  /// too (CR-04).
  ///
  /// Returns `false` without doing anything while [localId] is in [state]
  /// -- deletion is blocked while a drain is in flight, never raced against
  /// it. THROWS, also without touching the local row or its photos, when a
  /// server id exists but the network `DELETE` fails: D-14's premise --
  /// "unsynced means the device holds the only copy" -- stopped holding the
  /// moment `writeServerTrailId` (SYNC-04) started stamping that id the
  /// instant the create is accepted, well before the row is retired. A
  /// `pending`/`uploading`/`failed` row can carry a real, live server id for
  /// as long as a later step keeps failing, so destroying the local row
  /// unconditionally would strand a possibly-public trail on the server
  /// with no device left pointing at it. Thrown, not folded into a `false`
  /// return, so the caller cannot mistake a failed server delete for the
  /// in-flight refusal above.
  Future<bool> deleteUnsynced(String localId) async {
    if (state.contains(localId)) return false;

    final store = ref.read(objectBoxProvider);

    // `readLocalTrail` -> `TrailEntity.toModel()` blanks a still-local
    // sentinel id to `''` (D-06) but passes a real server id through
    // unchanged -- the same signal `trail_dropdown.dart`'s `_confirmDelete`
    // uses to choose its copy, checked again here because the confirm
    // dialog and this call are not atomic with each other.
    final row = readLocalTrail(store, localId);
    final serverId = (row != null && trailHasServerId(row.id)) ? row.id : null;

    if (serverId != null) {
      // Let a network failure escape uncaught -- see this method's doc
      // comment for why the local row must survive it.
      await ref.read(apiProvider).delete('/trail/$serverId');
    }

    deleteLocalTrailRow(store, localId);
    // `unsyncedTrailPhotoDir` validates the id OUTSIDE
    // `deleteUnsyncedPhotoDir`'s own try, deliberately, so a malformed
    // localId throws an ArgumentError. The row is already gone by this line,
    // so letting that escape into the button's async callback would leave the
    // user with an orphaned photo directory and an unexplained failure. Match
    // the sweep's best-effort discipline instead.
    await _deletePhotoDirBestEffort(localId);

    ref.invalidate(trailLibraryProvider);
    final userEntity = store.box<UserEntity>().getAll().firstOrNull;
    if (userEntity != null) {
      ref.invalidate(profileTrailsProvider('@${userEntity.preferredUsername}'));
    }

    return true;
  }
}
