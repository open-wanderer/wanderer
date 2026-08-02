import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_library_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/util/current_account.dart';
import 'package:wanderer/util/form_data_util.dart';
import 'package:wanderer/util/local_id.dart';
import 'package:wanderer/util/local_photo_store_util.dart';
import 'package:wanderer/util/local_trail_store.dart';
import 'package:wanderer/util/sync_backoff.dart';

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

  @override
  Set<String> build() => {};

  /// The single entry point every trigger calls.
  ///
  /// Always refreshes [onlineStatusProvider] first and bails out
  /// immediately when it reports offline -- mandatory, not defensive:
  /// `OnlineStatus.build()` is optimistic (`true`), settles only from
  /// ordinary request/response traffic, and trusting it at a cold moment
  /// already shipped a bug once (Phase 35 OFFUI-04, RESEARCH.md Pitfall 5).
  Future<void> drainIfOnline() async {
    if (_draining) return;
    _draining = true;
    try {
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
    } finally {
      _draining = false;
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
    state = {...state, localId};

    try {
      final userQuery = store
          .box<UserEntity>()
          .query(UserEntity_.id.equals(accountId))
          .build();
      final userEntity = userQuery.findFirst();
      userQuery.close();
      if (userEntity == null) {
        throw StateError(
          'trail_sync_provider: no UserEntity found for account "$accountId"',
        );
      }
      final authorId = userEntity.actorId;

      entity.syncState = TrailSyncState.uploading;
      store.box<TrailEntity>().put(entity);

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
        final created = Trail.fromJson(response.data);
        serverTrailId = created.id;

        // SYNC-04: commits the instant the server accepted the create,
        // BEFORE any waypoint upload starts below. There is no server-side
        // idempotency key, so a crash between "server accepted" and "id
        // persisted" is exactly what produces a duplicate trail
        // (RESEARCH.md Pitfall 3).
        //
        // The photo list rides along in the same transaction so that a drain
        // RESUMING at step 3 or 4 finds a truthful `photos` column. It used
        // to find `[]` -- the value `TrailEntity.fromModel` leaves behind --
        // and step 4 persisted that emptiness before deleting the on-disk
        // copies.
        writeServerTrailId(
          store,
          localId: localId,
          serverId: serverTrailId,
          serverPhotoFilenames: created.photos,
        );
      }

      // Step 3: each waypoint that has not already been created.
      for (final waypointEntity in entity.waypoints) {
        if (!isLocalId(waypointEntity.id)) continue;

        final waypointLocalKey = waypointEntity.localKey;
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
          if (waypointLocalKey != null) {
            writeServerWaypointId(
              store,
              localId: localId,
              waypointLocalKey: waypointLocalKey,
              serverWaypointId: e.created.id,
              serverPhotoFilenames: e.created.photos,
            );
          }
          rethrow;
        }

        if (waypointLocalKey == null) continue;

        writeServerWaypointId(
          store,
          localId: localId,
          waypointLocalKey: waypointLocalKey,
          serverWaypointId: createdWaypoint.id,
          serverPhotoFilenames: createdWaypoint.photos,
        );
      }

      // Step 4: full success. No photo list is passed: step 2 already
      // committed the server's list onto the row, and on a resumed drain
      // this pass never learned one. `markTrailSynced`'s null default means
      // "keep what the row already holds", so the photos survive either way
      // and it is safe to delete the unsynced copies below.
      markTrailSynced(store, localId: localId);
      await deleteUnsyncedPhotoDir(localId);

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

  /// SYNC-03's manual retry: resets the row's backoff/attempt bookkeeping
  /// then immediately re-drains. Safe to call on a row that is already
  /// draining -- [_drainOne]'s in-flight guard makes the resulting
  /// [drainIfOnline] pass a no-op for that row.
  Future<void> retry(String localId) async {
    final store = ref.read(objectBoxProvider);
    resetDrainBackoff(store, localId);
    await drainIfOnline();
  }

  /// D-14: deletes the local row and its photo copies for [localId].
  ///
  /// Returns `false` without doing anything while [localId] is in [state]
  /// -- deletion is blocked while a drain is in flight, never raced against
  /// it.
  Future<bool> deleteUnsynced(String localId) async {
    if (state.contains(localId)) return false;

    final store = ref.read(objectBoxProvider);
    deleteLocalTrailRow(store, localId);
    await deleteUnsyncedPhotoDir(localId);

    ref.invalidate(trailLibraryProvider);
    final userEntity = store.box<UserEntity>().getAll().firstOrNull;
    if (userEntity != null) {
      ref.invalidate(profileTrailsProvider('@${userEntity.preferredUsername}'));
    }

    return true;
  }
}
