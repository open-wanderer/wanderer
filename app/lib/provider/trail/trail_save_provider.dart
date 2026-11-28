import 'dart:async';
import 'dart:io';

import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/store/local_photo_store.dart';
import 'package:wanderer/provider/server_capabilities_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/provider/trail/tag_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/util/local/id.dart';
import 'package:wanderer/util/trail/form_data.dart';
import 'package:wanderer/util/object_diff.dart';

part 'trail_save_provider.g.dart';

class TrailSaveResult {
  final Trail trail;

  /// Whether `trail.expand.waypointsViaTrail` may be INCOMPLETE.
  ///
  /// Read by callers as "this waypoint set is not authoritative" -- it is
  /// wired straight to `applyServerTrailToLibraryRow`'s
  /// `waypointsAreAuthoritative:` (negated) in `trail_create_screen.dart`,
  /// where a false negative makes the store prune a still-live waypoint out
  /// of the offline copy.
  ///
  /// So EVERY path that omits a waypoint from the returned set must set
  /// this, not just the ones that caught an exception. The name
  /// says "failures" for history; the property callers depend on is
  /// completeness.
  final bool hadWaypointFailures;
  final bool hasPendingPhotos;
  final int rejectedPhotoCount;

  const TrailSaveResult({
    required this.trail,
    this.hadWaypointFailures = false,
    this.hasPendingPhotos = false,
    this.rejectedPhotoCount = 0,
  });
}

@Riverpod(keepAlive: true)
class TrailSave extends _$TrailSave {
  @override
  FutureOr<void> build() {}

  /// Resolves any client-created (unsaved, `id == null`) [tags] into real
  /// PocketBase records, returning the full tag list (existing + newly
  /// created) with real ids.
  ///
  /// Public so the deferred-upload drain (`trail_sync_provider.dart`) can
  /// reuse the exact same tag-reuse-vs-create rule it applies at
  /// interactive save time.
  Future<List<Tag>> resolveTags(
    List<Tag> tags, {
    PhotoWriteSession? session,
  }) async {
    final resolved = <Tag>[];
    for (final tag in tags) {
      session?.check();
      if (tag.id != null && tag.id!.isNotEmpty) {
        resolved.add(tag);
      } else {
        resolved.add(await ref.read(tagProvider.notifier).create(tag.name));
        session?.check();
      }
    }
    return resolved;
  }

  Future<(Map<String, dynamic>, PhotoApiMode)> _writeMetadata(
    Trail trail,
    PhotoWriteSession session,
    PhotoApiMode mode, {
    required bool create,
    required List<File> newPhotos,
    List<String> removedPhotos = const [],
    bool hasPendingPhotos = false,
    bool coverChanged = false,
    String? coverPhoto,
  }) async {
    for (var attempt = 0; ; attempt++) {
      final legacy = mode == PhotoApiMode.legacy && !hasPendingPhotos;
      final form = await trail.toFormData(
        newPhotos: legacy ? newPhotos : const [],
        removedPhotoFilenames: legacy ? removedPhotos : const [],
        isCreate: create,
      );
      if (legacy && coverChanged) {
        final ordered = [
          ...trail.photos.where((photo) => !removedPhotos.contains(photo)),
          ...newPhotos.map((file) => file.path),
        ];
        final index = ordered.indexOf(coverPhoto ?? '');
        form.fields.add(MapEntry('thumbnail', '${index < 0 ? 0 : index}'));
      }
      try {
        final route = session.route(
          create ? '/trail/form' : '/trail/form/${trail.id}',
        );
        final request = create
            ? session.api.put(
                route,
                data: form,
                queryParameters: {'expand': 'category,tags,author'},
              )
            : session.api.post(
                route,
                data: form,
                queryParameters: {'expand': 'category,tags,author'},
              );
        return (await photoMetadataWrite(request, trail.toJson()), mode);
      } catch (error) {
        if (attempt > 0 ||
            mode != PhotoApiMode.legacy ||
            !isLegacyPhotoFieldRemoved(error))
          rethrow;
        ref.read(serverCapabilitiesProvider).markAssetApi();
        mode = PhotoApiMode.assets;
      }
    }
  }

  Future<TrailSaveResult> createTrail(
    Trail trail, {
    required String authorId,
    required List<File> newPhotos,
    PhotoApiMode? mode,
  }) async {
    final api = ref.read(apiProvider);
    final session = PhotoWriteSession(
      api,
      currentEpoch: () => ref.read(apiProvider.notifier).sessionEpoch,
    );
    final resolvedTags = await resolveTags(
      trail.expand?.tags ?? const [],
      session: session,
    );

    final trailToSend = trail.copyWith(
      author: authorId,
      tags: resolvedTags.map((t) => t.id!).toList(),
    );

    final metadata = await _writeMetadata(
      trailToSend,
      session,
      mode ?? await ref.read(photoApiModeProvider)(),
      create: true,
      newPhotos: newPhotos,
    );
    final actualMode = metadata.$2;
    final body = metadata.$1;

    var model = Trail.fromJson(body);
    var pendingPhotos = false;
    var rejected = 0;
    var completedPhotoJob = false;
    if (actualMode != PhotoApiMode.legacy && newPhotos.isNotEmpty) {
      final result = await ref
          .read(photoMutationServiceProvider)
          .apply(
            target: PhotoTarget.trail(model.id),
            actorId: authorId,
            serverBaseUrl: session.baseUrl,
            mode: actualMode,
            newPhotos: newPhotos.map((file) => file.path).toList(),
            existingPhotos: model.photos,
          );
      pendingPhotos = result.pending;
      rejected += result.rejectedCount;
      model = _withPhotoResult(model, result);
      completedPhotoJob = !result.pending;
    }

    session.check();
    final waypointNotifier = ref.read(waypointSaveProvider.notifier);
    final createdWaypoints = <Waypoint>[];
    var hadFailures = false;

    for (final waypoint in trail.expand?.waypointsViaTrail ?? const []) {
      session.check();
      try {
        createdWaypoints.add(
          await waypointNotifier.create(
            waypoint,
            authorId: authorId,
            trailId: model.id,
            mode: ref.read(serverCapabilitiesProvider).assetApi == true
                ? PhotoApiMode.assets
                : actualMode,
          ),
        );
        rejected += waypointNotifier.rejectedPhotoCount;
      } catch (e) {
        hadFailures = true;
      }
    }

    model = model.copyWith(
      expand: (model.expand ?? const TrailExpand()).copyWith(
        waypointsViaTrail: createdWaypoints,
        tags: resolvedTags,
        // Trail.fromJson never populates expand.gpx/gpxData (gpx is
        // includeFromJson: false; gpxData only comes from the server's file
        // endpoint). Carry the just-submitted track forward so TrailMap
        // (which gates the route line and markers on expand.gpx != null)
        // doesn't go blank the instant the caller adopts this result.
        gpx: trailToSend.expand?.gpx,
        gpxData: trailToSend.expand?.gpxData,
      ),
    );

    pendingPhotos |= (model.expand?.waypointsViaTrail ?? const []).any(
      (waypoint) => waypoint.localPhotos.isNotEmpty,
    );
    if (completedPhotoJob) {
      await ref
          .read(photoMutationServiceProvider)
          .acknowledge(
            PhotoTarget.trail(model.id),
            authorId,
            serverBaseUrl: session.baseUrl,
          );
    }
    return TrailSaveResult(
      trail: model,
      hadWaypointFailures: hadFailures,
      hasPendingPhotos: pendingPhotos,
      rejectedPhotoCount: rejected,
    );
  }

  Future<TrailSaveResult> updateTrail(
    Trail oldTrail,
    Trail newTrail, {
    required String authorId,
    required List<File> newPhotos,
    required List<String> removedPhotoFilenames,
    String? coverPhoto,
    bool setCover = false,
    List<String> removedPendingPaths = const [],
    PhotoApiMode? mode,
  }) async {
    final api = ref.read(apiProvider);
    final session = PhotoWriteSession(
      api,
      currentEpoch: () => ref.read(apiProvider.notifier).sessionEpoch,
    );
    final resolvedTags = await resolveTags(
      newTrail.expand?.tags ?? const [],
      session: session,
    );

    final trailToSend = newTrail.copyWith(
      author: oldTrail.author,
      tags: resolvedTags.map((t) => t.id!).toList(),
    );

    final selectedCover = setCover
        ? coverPhoto
        : newTrail.photos.elementAtOrNull(newTrail.thumbnail);
    final coverChanged = setCover
        ? selectedCover != oldTrail.photos.elementAtOrNull(oldTrail.thumbnail)
        : oldTrail.thumbnail != newTrail.thumbnail;
    final metadata = await _writeMetadata(
      trailToSend,
      session,
      mode ?? await ref.read(photoApiModeProvider)(),
      create: false,
      newPhotos: newPhotos,
      removedPhotos: removedPhotoFilenames,
      hasPendingPhotos: oldTrail.localPhotos.isNotEmpty,
      coverChanged: coverChanged,
      coverPhoto: selectedCover,
    );
    final actualMode = metadata.$2;
    final usesJournal =
        actualMode != PhotoApiMode.legacy || oldTrail.localPhotos.isNotEmpty;
    final body = metadata.$1;

    var model = Trail.fromJson(body);
    var pendingPhotos = false;
    var rejected = 0;
    var completedPhotoJob = false;
    if (usesJournal &&
        (newPhotos.isNotEmpty ||
            oldTrail.localPhotos.isNotEmpty ||
            removedPhotoFilenames.isNotEmpty ||
            coverChanged ||
            removedPendingPaths.isNotEmpty)) {
      final result = await ref
          .read(photoMutationServiceProvider)
          .apply(
            target: PhotoTarget.trail(model.id),
            actorId: authorId,
            serverBaseUrl: session.baseUrl,
            mode: actualMode,
            removedPendingPaths: removedPendingPaths,
            newPhotos: newPhotos.map((file) => file.path).toList(),
            removedPhotos: removedPhotoFilenames,
            existingPhotos: model.photos,
            setCover: coverChanged,
            coverPhoto: selectedCover,
          );
      pendingPhotos = result.pending;
      rejected += result.rejectedCount;
      model = _withPhotoResult(model, result);
      completedPhotoJob = !result.pending;
    }

    final diff = compareObjectArrays<Waypoint>(
      oldTrail.expand?.waypointsViaTrail ?? const [],
      newTrail.expand?.waypointsViaTrail ?? const [],
      idOf: (w) => w.listKey,
    );

    session.check();
    final waypointNotifier = ref.read(waypointSaveProvider.notifier);
    var hadFailures = false;

    final finalWaypoints = <Waypoint>[
      for (final wp in newTrail.expand?.waypointsViaTrail ?? const [])
        if (!diff.added.contains(wp) && !diff.updated.contains(wp)) wp,
    ];

    for (final wp in diff.added) {
      session.check();
      try {
        finalWaypoints.add(
          await waypointNotifier.create(
            wp,
            authorId: authorId,
            trailId: newTrail.id,
            mode: ref.read(serverCapabilitiesProvider).assetApi == true
                ? PhotoApiMode.assets
                : actualMode,
          ),
        );
        rejected += waypointNotifier.rejectedPhotoCount;
      } catch (_) {
        hadFailures = true;
      }
    }

    for (final wp in diff.updated) {
      session.check();
      final old = (oldTrail.expand?.waypointsViaTrail ?? const [])
          .where((o) => o.id == wp.id)
          .firstOrNull;
      if (old == null) {
        // Dropping `wp` here without marking makes
        // `finalWaypoints` incomplete while `hadWaypointFailures` stays
        // false, and the caller passes `waypointsAreAuthoritative:
        // !hadWaypointFailures` -- so `applyServerTrailToLibraryRow` prunes
        // a still-live waypoint out of the offline copy. That is exactly
        // the failure `waypointsAreAuthoritative` was added to close.
        // Reachable because `diff` is keyed on `listKey` (`id` when
        // non-empty, else `localKey`) while this re-lookup is on `id`: an
        // old waypoint with id `X` and no localKey against a new one with
        // `localKey == 'X'` and an empty id matches on listKey and misses
        // here. `hadWaypointFailures` means "this waypoint set is not
        // authoritative", not "an exception was thrown".
        hadFailures = true;
        continue;
      }
      try {
        finalWaypoints.add(
          await waypointNotifier.updateWaypoint(
            old,
            wp,
            authorId: authorId,
            mode: ref.read(serverCapabilitiesProvider).assetApi == true
                ? PhotoApiMode.assets
                : actualMode,
          ),
        );
        rejected += waypointNotifier.rejectedPhotoCount;
      } catch (_) {
        hadFailures = true;
      }
    }

    for (final wp in diff.deleted) {
      session.check();
      try {
        await waypointNotifier.delete(wp);
        session.check();
        final store = ref.read(objectBoxProvider);
        final accountId = currentAccountId(store);
        if (accountId != null) {
          final removed = removeQueuedWaypoint(
            store,
            accountId: accountId,
            trailId: newTrail.id,
            waypointId: wp.id,
          );
          for (final sources in removed) {
            await deleteUnsyncedPhotoFiles(
              accountId,
              sources.localId,
              sources.paths,
            );
          }
        }
      } catch (_) {
        hadFailures = true;
      }
    }

    model = model.copyWith(
      expand: (model.expand ?? const TrailExpand()).copyWith(
        waypointsViaTrail: finalWaypoints,
        tags: resolvedTags,
        gpx: trailToSend.expand?.gpx,
        gpxData: trailToSend.expand?.gpxData,
      ),
    );

    pendingPhotos |= (model.expand?.waypointsViaTrail ?? const []).any(
      (waypoint) => waypoint.localPhotos.isNotEmpty,
    );
    if (completedPhotoJob) {
      await ref
          .read(photoMutationServiceProvider)
          .acknowledge(
            PhotoTarget.trail(model.id),
            authorId,
            serverBaseUrl: session.baseUrl,
          );
    }
    return TrailSaveResult(
      trail: model,
      hadWaypointFailures: hadFailures,
      hasPendingPhotos: pendingPhotos,
      rejectedPhotoCount: rejected,
    );
  }

  Trail _withPhotoResult(Trail model, PhotoMutationResult result) {
    final previews = (model.expand?.photoAssets ?? const [])
        .where((link) => link.photoAsset?.isGeneratedRoutePreview == true)
        .map((link) => link.asset)
        .toSet();
    final realPhotos = result.photos
        .where((photo) => !previews.contains(photoAssetId(photo)))
        .toList();
    final requested =
        result.coverPhoto ?? model.photos.elementAtOrNull(model.thumbnail);
    final chosen =
        requested != null &&
            result.photos.contains(requested) &&
            (realPhotos.isEmpty || !previews.contains(photoAssetId(requested)))
        ? requested
        : realPhotos.firstOrNull ?? result.photos.firstOrNull;
    final index = chosen == null ? 0 : result.photos.indexOf(chosen);
    return model.copyWith(
      photos: result.photos,
      localPhotos: result.pendingPaths,
      thumbnail: index < 0 ? 0 : index,
    );
  }

  Future<void> deleteTrail(Trail trail) async {
    final api = ref.read(apiProvider);
    final session = PhotoWriteSession(
      api,
      currentEpoch: () => ref.read(apiProvider.notifier).sessionEpoch,
    );
    final actor = ref.read(authProvider).value?.actorId ?? trail.author;
    final route = session.route('/trail/${recordIdDirSegment(trail.id)}');
    final children = <String>{
      ...?trail.expand?.waypointsViaTrail?.map((waypoint) => waypoint.id),
    };
    final response = await api.get(
      route,
      queryParameters: {'expand': 'waypoints_via_trail'},
    );
    final expanded = (response.data as Map)['expand'];
    if (expanded is Map && expanded['waypoints_via_trail'] is List) {
      children.addAll(
        (expanded['waypoints_via_trail'] as List)
            .whereType<Map>()
            .map((waypoint) => waypoint['id'])
            .whereType<String>(),
      );
    }
    session.check();
    await api.delete(route);
    for (final id in children.where((id) => id.isNotEmpty)) {
      await ref
          .read(photoMutationServiceProvider)
          .drop(
            PhotoTarget.waypoint(id),
            actor,
            serverBaseUrl: session.baseUrl,
          );
    }
    await ref
        .read(photoMutationServiceProvider)
        .drop(
          PhotoTarget.trail(trail.id),
          actor,
          serverBaseUrl: session.baseUrl,
        );
  }
}
