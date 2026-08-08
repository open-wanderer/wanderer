import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
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
import 'package:wanderer/provider/trail/local_trail_provider.dart';
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

/// What [TrailSync.deleteUnsynced] did, so the caller cannot confuse a
/// refusal with a completion.
enum UnsyncedDeleteResult {
  /// The local row (and, if it carried one, the server copy) were deleted.
  deleted,

  /// Refused: [localId] is currently mid-drain. Nothing was touched.
  blockedInFlight,

  /// Refused: the row already carries a server id, but the DELETE request
  /// never reached a server. A connection is needed before this trail can
  /// be deleted. Nothing was touched.
  needsConnection,

  /// Refused: either the signed-in account could not be resolved, the
  /// server id was rejected as an unsafe path segment, or the server
  /// DELETE failed for a reason other than "already gone" or "offline".
  /// Nothing was touched.
  failed,
}

/// The local ids of trails whose upload is currently draining, shared
/// across every trigger (app foreground, connectivity regained, cold
/// start, manual retry) exactly like `DownloadingTrailIds` shares download
/// state across its own entry points. `keepAlive` so an in-flight drain
/// survives whichever widget happens to rebuild or unmount mid-upload.
///
/// The KEY is the local id, not the server `id` -- a trail mid-drain may
/// still have no server id at all (pre-create) or one assigned moments ago
/// (post-create, pre-waypoints), so only the local id is stable across the
/// whole resume-from-step sequence.
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

  /// Memoizes the server id a completed upload retired a local row into,
  /// keyed on `localId`, so a still-mounted create/edit or detail screen for
  /// that trail can still reach it after [retireUploadedLocalTrail] removes
  /// the row -- there are no ObjectBox `Query.watch()` streams
  /// anywhere in this app, so this memo is the only surviving handle.
  ///
  /// Each entry stores the owning `accountId` alongside the server id.
  /// `trailSyncProvider` is deliberately excluded from
  /// `accountScopedProviders` (`account_scope_invalidation.dart`) so an
  /// in-flight drain survives an account switch -- which also means this
  /// memo survives one, and without the account guard [serverIdForRetired]
  /// would let account B resolve account A's server id and post an edit to
  /// A's trail.
  ///
  /// Bounded to 64 entries, oldest evicted first (Dart `Map` preserves
  /// insertion order) -- a long session that uploads many trails must not
  /// grow this without bound on a `keepAlive` notifier.
  final Map<String, ({String accountId, String serverId})> _retiredServerIds =
      {};

  /// Records that [localId]'s upload retired into [serverId], owned by
  /// [accountId]. Evicts the oldest entry once the memo exceeds 64 rows.
  void _rememberRetiredServerId(
    String localId,
    String serverId,
    String accountId,
  ) {
    _retiredServerIds[localId] = (accountId: accountId, serverId: serverId);
    while (_retiredServerIds.length > 64) {
      _retiredServerIds.remove(_retiredServerIds.keys.first);
    }
  }

  /// The server id [localId]'s row was retired into, or null when there is
  /// no such entry, no account is currently signed in, or the signed-in
  /// account does not match the one that captured it.
  ///
  /// Re-reads `currentAccountId(objectBoxProvider)` fresh at the point of
  /// use rather than a cached field -- a stale cached id here is
  /// exactly the cross-account leak the account check exists to prevent. A
  /// mismatch is refused and logged rather than silently returning null
  /// indistinguishably from "nothing memoized".
  String? serverIdForRetired(String localId) {
    final store = ref.read(objectBoxProvider);
    final accountId = currentAccountId(store);
    if (accountId == null) return null;

    final entry = _retiredServerIds[localId];
    if (entry == null) return null;
    if (entry.accountId != accountId) {
      debugPrint(
        'trail_sync_provider: serverIdForRetired("$localId") refused -- '
        'memoized for a different account',
      );
      return null;
    }
    return entry.serverId;
  }

  @override
  Set<String> build() => {};

  /// The single entry point every trigger calls.
  ///
  /// When (and only when) there are due candidates, refreshes
  /// [onlineStatusProvider] and bails out when it reports offline --
  /// mandatory, not defensive: `OnlineStatus.build()` is optimistic
  /// (`true`), settles only from ordinary request/response traffic, and
  /// trusting it at a cold moment already shipped a bug once. With an empty
  /// queue no probe fires at all (this runs on every app resume).
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
    final store = ref.read(objectBoxProvider);
    // Read fresh every run, never a cached field -- a stale id here
    // is exactly the leak account-scoping exists to prevent.
    final accountId = currentAccountId(store);
    if (accountId == null) return;

    final candidates = selectDrainCandidates(
      store,
      accountId: accountId,
      now: DateTime.now(),
    );

    // Candidates BEFORE the connectivity probe: this drain runs on every app
    // resume, and probing first meant a `/health` network round-trip per
    // foreground for an almost-always-empty queue. With nothing to upload
    // there is nothing to be online FOR — the dio interceptor keeps
    // onlineStatusProvider fresh from ordinary traffic anyway.
    if (candidates.isEmpty) return;

    final isOnline = await ref.read(onlineStatusProvider.notifier).refresh();
    if (!isOnline) return;

    // Sequential, one trail at a time, on purpose: a flaky connection
    // cannot fan out into N concurrent partially-applied uploads, and the
    // in-flight set stays trivially easy to reason about for the delete
    // gate below.
    for (final entity in candidates) {
      await _drainOne(store, entity, accountId);
    }
  }

  /// Implements the resume-from-step for a single trail.
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

    // Resolved BEFORE the in-flight set is joined and before the try below,
    // same placement and reasoning as the missing-UserEntity bail above:
    // a waypoint whose id is still a local sentinel but has no
    // localKey to record the server's returned id against is an invariant
    // break, not a network condition -- `WaypointEntity.fromModel` always
    // mints a localKey for an empty-id waypoint, so no retry will ever fix
    // this row. Recording a failed attempt for it would let four fast
    // passes (a lifecycle or connectivity flurry produces these within
    // seconds, since syncBackoffDelay only starts at 30s after the FIRST
    // failure) park an otherwise-healthy trail as `failed`, after which
    // isDrainDue never picks it up again.
    if (hasKeylessPendingWaypoint(
      entity.waypoints.map((w) => (id: w.id, localKey: w.localKey)).toList(),
    )) {
      debugPrint(
        'trail_sync_provider: "$localId" has a keyless pending waypoint '
        '(invariant break); skipping drain without recording a failed '
        'attempt',
      );
      return;
    }

    state = {...state, localId};

    try {
      markTrailUploading(store, localId);

      // Step 1: tag reuse/create applies whether or not the trail
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
        // Commits the instant the server accepted the create,
        // BEFORE any waypoint upload starts below. There is no server-side
        // idempotency key, so a crash between "server accepted" and "id
        // persisted" is exactly what produces a duplicate trail.
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
          // `rawId` is deliberately NOT validated through
          // `recordIdDirSegment` here, before it is ever persisted: a throw
          // at this point means the id is never persisted, so the NEXT drain
          // pass sees `isLocalId(entity.id) == true` again and issues a
          // second `PUT /trail/form` -- creating exactly the duplicate trail
          // this file's step 2 exists to prevent. Validation happens at
          // every REQUEST site instead
          // (`deleteUnsynced`'s `recordIdDirSegment(serverId)`,
          // `trail_save_provider.dart`'s `deleteTrail`), where a rejected id
          // only aborts that one request rather than corrupting the row's
          // resume state.
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

        // `WaypointEntity.fromModel` always mints a key for an empty-id
        // waypoint, and the hasKeylessPendingWaypoint guard above this
        // method's try already bailed the whole drain out before this loop
        // could ever run against a corrupt row -- so a null here is
        // unreachable by construction, not merely unlikely. Previously this
        // was a `StateError` thrown from inside the try, which consumed one
        // of the four kMaxSyncAttempts for an invariant break no retry could
        // ever fix; see the guard's doc comment for the failure mode that
        // produced.
        final waypointLocalKey = waypointEntity.localKey!;

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
          // the id NOW, exactly as the trail itself does, so the
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
      // retired here: an uploaded trail is reachable through the SERVER
      // entry, not through a
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
      final retirement = retireUploadedLocalTrail(store, localId);
      final retiredServerId = retirement.serverId;
      if (retiredServerId != null) {
        _rememberRetiredServerId(localId, retiredServerId, accountId);
      }
      // The remove branch runs only when `savedByUserIds` is
      // empty, so nothing holds this trail offline any more -- but
      // `library/<serverId>/` may still exist from when something did, and
      // the row that was removed held the only paths into it. Reclaim it
      // here, mirroring `deleteUnsynced`'s `removed` branch. On the demote
      // branch the row is kept as an ordinary downloaded row and those files
      // are still its photos, so it is deliberately left alone.
      if (retirement.rowRemoved && retiredServerId != null) {
        await _deleteLibraryDirBestEffort(retiredServerId);
      }
      // A hiker sitting on `/trail/local/<localId>` while the upload
      // completes otherwise keeps rendering a row that no longer exists,
      // with a live Edit button that walks straight into the dead end.
      ref.invalidate(localTrailProvider(localId));
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
      await _deletePhotoDirBestEffort(accountId, localId);

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

  /// Deletes [accountId]'s [localId] unsynced photo directory, swallowing
  /// and logging any failure.
  ///
  /// `deleteUnsyncedPhotoDir` swallows filesystem errors but evaluates
  /// `unsyncedTrailPhotoDir` -- and therefore the `^local-\d+-\d+$`/
  /// `recordIdDirSegment` validation on BOTH segments -- outside its
  /// own try, on purpose, so a malformed id is a loud caller bug. Neither of
  /// this notifier's two call sites can act on that ArgumentError, and both
  /// run AFTER the row has already been deleted or marked synced, so both
  /// need it to degrade rather than escape into a button callback.
  Future<void> _deletePhotoDirBestEffort(
    String accountId,
    String localId,
  ) async {
    try {
      await deleteUnsyncedPhotoDir(accountId, localId);
    } catch (e, st) {
      debugPrint(
        'trail_sync_provider: photo dir cleanup failed for "$localId": '
        '$e\n$st',
      );
    }
  }

  /// Deletes `library/<serverId>/` -- the downloaded-photo directory keyed
  /// on the trail's SERVER id -- swallowing and logging any failure.
  ///
  /// Only ever called after [deleteLocalTrailRow] reported
  /// [LocalRowDeleteOutcome.removed], i.e. no account held the row in
  /// `savedByUserIds`. Nothing used to reclaim this directory on the
  /// capture-delete path: `deleteUnsynced` cleaned only
  /// `unsynced/<accountId>/<localId>/`, and the row whose `photos` column
  /// held the absolute paths into `library/<serverId>/` had just been
  /// removed, so the whole downloaded photo set (tens of MB) leaked
  /// permanently.
  ///
  /// Best-effort by the same reasoning as [_deletePhotoDirBestEffort]: the
  /// row is already gone by this line, so an I/O error or a
  /// [recordIdDirSegment] rejection must degrade into a leaked directory,
  /// never escape into the delete button's async callback.
  Future<void> _deleteLibraryDirBestEffort(String serverId) async {
    try {
      final appDir = await getApplicationDocumentsDirectory();
      // p.join + a whitelisted segment, never interpolation: `serverId`
      // arrives from a response body that a federated or compromised
      // instance controls (`trail_download_service.dart` builds the same
      // path the same way).
      final dir = Directory(
        p.join(appDir.path, 'library', recordIdDirSegment(serverId)),
      );
      if (await dir.exists()) {
        await dir.delete(recursive: true);
      }
    } catch (e, st) {
      debugPrint(
        'trail_sync_provider: library dir cleanup failed for "$serverId": '
        '$e\n$st',
      );
    }
  }

  /// The manual retry: resets the row's backoff/attempt bookkeeping
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
    // Read fresh, never cached -- a null account means nothing to
    // scope the reset to, so the retry is a no-op rather than an unscoped
    // write.
    final accountId = currentAccountId(store);
    if (accountId == null) return;
    resetDrainBackoff(store, localId, accountId: accountId);
    await drainIfOnline();
  }

  /// Deletes the local row and its photo copies for [localId] --
  /// and, when the row already carries a real server id, the server's copy
  /// too -- classifying every outcome instead of throwing.
  ///
  /// Returns [UnsyncedDeleteResult.blockedInFlight] without doing anything
  /// while [localId] is in [state] -- deletion is blocked while a drain is
  /// in flight, never raced against it.
  ///
  /// The server id is read via [readLocalTrailServerId], never
  /// `readLocalTrail`/`toModel()`: a delete decision must not
  /// depend on the row's cached GPX still parsing. When a server id exists,
  /// it is validated through [recordIdDirSegment] before it reaches the
  /// Dio path -- a rejected id returns
  /// [UnsyncedDeleteResult.failed] without touching the local row.
  ///
  /// A DELETE failure is classified via [resolveServerDeleteOutcome]:
  /// a 404 (the server copy is already gone) still proceeds with the local
  /// delete; an unreachable server returns [UnsyncedDeleteResult.needsConnection]
  /// so the hiker is told to reconnect rather than being offered a
  /// "delete on this device only" escape hatch that would strand a
  /// possibly-public server trail with no device left pointing at it; every
  /// other failure (401, 403, 500, ...) returns
  /// [UnsyncedDeleteResult.failed]. Every branch logs the underlying
  /// exception via [debugPrint].
  ///
  /// [deleteLocalTrailRow]'s owner-scoped write can match no row -- the row
  /// this caller intended to delete may actually be owned by another account
  /// (`savedByUserIds` scoping can hand a non-owning account a
  /// `localId`/non-synced `syncState` that were really someone else's). When it matches nothing, this method skips the photo
  /// directory delete and the provider invalidations entirely -- it never
  /// touches data it was not allowed to touch.
  ///
  /// The no-match branch still reports [UnsyncedDeleteResult.deleted]
  /// in exactly one case -- when the owner-scoped [readLocalTrailServerId]
  /// yielded an id for THIS account and the server DELETE then succeeded (or
  /// 404'd), proving we owned the row and the server copy is gone. That is a
  /// race with the drain, not a scoping failure, and reporting `failed` for it
  /// suppressed the map announcement and route pop. In the cross-account
  /// case the
  /// owner-scoped read returns null, no DELETE is issued, and this still
  /// returns [UnsyncedDeleteResult.failed] having deleted nothing.
  Future<UnsyncedDeleteResult> deleteUnsynced(String localId) async {
    if (state.contains(localId)) return UnsyncedDeleteResult.blockedInFlight;

    final store = ref.read(objectBoxProvider);
    // Read fresh, never cached -- a stale id here is exactly the
    // leak account-scoping exists to prevent.
    final accountId = currentAccountId(store);
    if (accountId == null) {
      debugPrint(
        'trail_sync_provider: deleteUnsynced("$localId") found no signed-in '
        'account; refusing',
      );
      return UnsyncedDeleteResult.failed;
    }

    final serverId = readLocalTrailServerId(
      store,
      localId: localId,
      accountId: accountId,
    );

    // Tracks whether the server copy is provably gone by the
    // time the local row is touched. Only meaningful when `serverId != null`.
    var serverCopyDeleted = false;

    if (serverId != null) {
      final String path;
      try {
        path = '/trail/${recordIdDirSegment(serverId)}';
      } on ArgumentError catch (e) {
        debugPrint(
          'trail_sync_provider: deleteUnsynced("$localId") rejected server '
          'id "$serverId" as an unsafe path segment: $e',
        );
        return UnsyncedDeleteResult.failed;
      }

      try {
        await ref.read(apiProvider).delete(path);
        serverCopyDeleted = true;
      } on DioException catch (e) {
        final outcome = resolveServerDeleteOutcome(
          statusCode: e.response?.statusCode,
          connectionFailed: isConnectionFailure(e),
        );
        debugPrint(
          'trail_sync_provider: deleteUnsynced("$localId") server DELETE '
          'failed: $e (resolved: $outcome)',
        );
        switch (outcome) {
          case ServerDeleteOutcome.abortNeedsConnection:
            return UnsyncedDeleteResult.needsConnection;
          case ServerDeleteOutcome.abortAndReport:
            return UnsyncedDeleteResult.failed;
          case ServerDeleteOutcome.proceedWithLocalDelete:
            // 404: the server already agrees there is nothing left to
            // delete. Fall through to the local delete below.
            serverCopyDeleted = true;
            break;
        }
      }
    }

    final rowOutcome = deleteLocalTrailRow(
      store,
      localId,
      accountId: accountId,
    );
    if (rowOutcome == LocalRowDeleteOutcome.noMatch) {
      // `deleteLocalTrailRow` is owner-scoped and matched no row for this
      // account -- either the row belongs to another account or it is
      // already gone. Reporting
      // `deleted` here, or running the unscoped recursive photo-dir delete
      // below, is how a no-op row delete turned into destroying another
      // account's un-uploaded photos and a success toast for an operation
      // that touched nothing. Refuse instead: no photo delete, no provider
      // invalidation, no `deleted` result.
      debugPrint(
        'trail_sync_provider: deleteUnsynced("$localId") matched no row '
        'owned by account "$accountId"; nothing was deleted '
        '(serverCopyDeleted: $serverCopyDeleted)',
      );
      // Distinguish "raced" from "not ours". `serverCopyDeleted`
      // can only be true when `readLocalTrailServerId` -- itself owner-scoped
      // -- returned an id for THIS account, so we provably owned the row when
      // the method started and the server copy is now gone. The row vanishing
      // between the DELETE and here is a drain pass or a concurrent tap, not a
      // scoping failure; reporting `failed` toasted an error for a trail the
      // server had already deleted and skipped both the map announcement and
      // the route pop, leaving the map rendering a trail that exists nowhere.
      //
      // This does NOT weaken account scoping: in the cross-account case the
      // localId belongs to another account, the owner-scoped
      // `readLocalTrailServerId` returns
      // null, no DELETE is issued, `serverCopyDeleted` stays false, and this
      // still returns `failed` having touched nothing.
      return serverCopyDeleted
          ? UnsyncedDeleteResult.deleted
          : UnsyncedDeleteResult.failed;
    }

    // `unsyncedTrailPhotoDir` validates the id OUTSIDE
    // `deleteUnsyncedPhotoDir`'s own try, deliberately, so a malformed
    // localId throws an ArgumentError. The row is already gone by this line,
    // so letting that escape into the button's async callback would leave the
    // user with an orphaned photo directory and an unexplained failure. Match
    // the sweep's best-effort discipline instead.
    await _deletePhotoDirBestEffort(accountId, localId);

    // The row was the ONLY handle on `library/<serverId>/` --
    // `entity.photos` held the absolute paths into it and there is no
    // `library/` sweep anywhere in the app. Reclaim it here, and ONLY on the
    // `removed` branch: on `demotedStillHeld` another account still holds
    // the trail in its offline library and those files are its photos.
    if (rowOutcome == LocalRowDeleteOutcome.removed && serverId != null) {
      await _deleteLibraryDirBestEffort(serverId);
    }

    ref.invalidate(trailLibraryProvider);
    final userEntity = store.box<UserEntity>().getAll().firstOrNull;
    if (userEntity != null) {
      ref.invalidate(profileTrailsProvider('@${userEntity.preferredUsername}'));
    }

    return UnsyncedDeleteResult.deleted;
  }
}
