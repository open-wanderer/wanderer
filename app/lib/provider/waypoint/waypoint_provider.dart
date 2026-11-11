import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:path/path.dart' as p;
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/server_capabilities_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

part 'waypoint_provider.g.dart';

/// Thrown by [WaypointSave.create] when `PUT /waypoint` SUCCEEDED but the
/// follow-up photo upload did not.
///
/// It exists to carry [created] -- the record the server already holds -- out
/// to the caller. Without it, a photo-upload failure (the largest and most
/// timeout-prone request in the create sequence) was indistinguishable from a
/// failure to create the record at all, so the upload drain retried the whole
/// step and `PUT /waypoint` ran a second time. There is no server-side
/// idempotency key, so that produced a duplicate
/// waypoint on the server -- up to `kMaxSyncAttempts` of them.
///
/// A caller that can persist an id should persist `created.id` before
/// rethrowing; a caller that only counts failures can treat this like any
/// other throw.
class WaypointPhotoUploadException implements Exception {
  /// The waypoint record the server created, with its server-assigned id.
  final Waypoint created;

  /// The underlying photo-upload failure.
  final Object cause;

  const WaypointPhotoUploadException(this.created, this.cause);

  @override
  String toString() =>
      'WaypointPhotoUploadException(waypoint "${created.id}" was created but '
      'its photos failed to upload: $cause)';
}

@Riverpod(keepAlive: true)
class WaypointSave extends _$WaypointSave {
  List<String> _completedPhotoPaths = const [];
  List<String> get completedPhotoPaths => _completedPhotoPaths;
  int rejectedPhotoCount = 0;
  bool photoJobDropped = false;
  bool photoJobPending = false;
  bool photoTargetDeleted = false;
  @override
  FutureOr<void> build() {}

  Future<Waypoint> create(
    Waypoint waypoint, {
    required String authorId,
    required String trailId,
    PhotoApiMode? mode,
    bool retainPhotoReceipts = false,
  }) async {
    final api = ref.read(apiProvider);
    final session = PhotoWriteSession(
      api,
      currentEpoch: () => ref.read(apiProvider.notifier).sessionEpoch,
    );
    rejectedPhotoCount = 0;
    photoJobDropped = false;
    photoJobPending = false;
    photoTargetDeleted = false;
    _completedPhotoPaths = const [];
    mode ??= await ref.read(photoApiModeProvider)();

    final body = waypoint.toJson()
      ..remove('id')
      ..remove('collectionId')
      ..remove('collectionName')
      ..remove('created')
      ..remove('updated')
      ..remove('expand')
      ..['author'] = authorId
      ..['trail'] = trailId;

    if (mode != PhotoApiMode.legacy) body.remove('photos');
    final response = await photoMetadataWrite(
      api.put(session.route('/waypoint'), data: body),
      waypoint.toJson(),
    );
    final created = Waypoint.fromJson(response);

    if (waypoint.localPhotos.isEmpty) return created;

    // Past this line the record EXISTS server-side. Any failure below must
    // therefore surface the created record rather than looking like "the
    // waypoint was never created" -- see [WaypointPhotoUploadException].
    try {
      return await _uploadPhotos(
        created,
        authorId: authorId,
        mode: mode,
        retainPhotoReceipts: retainPhotoReceipts,
        serverBaseUrl: session.baseUrl,
        session: session,
        newPhotos: waypoint.localPhotos,
        removedFilenames: const [],
      );
    } catch (e) {
      throw WaypointPhotoUploadException(created, e);
    }
  }

  Future<Waypoint> updateWaypoint(
    Waypoint oldWaypoint,
    Waypoint newWaypoint, {
    required String authorId,
    PhotoApiMode? mode,
  }) async {
    final api = ref.read(apiProvider);
    final session = PhotoWriteSession(
      api,
      currentEpoch: () => ref.read(apiProvider.notifier).sessionEpoch,
    );
    rejectedPhotoCount = 0;
    photoJobDropped = false;
    photoJobPending = false;
    photoTargetDeleted = false;
    _completedPhotoPaths = const [];
    mode ??= await ref.read(photoApiModeProvider)();

    final body = newWaypoint.toJson()
      ..remove('id')
      ..remove('collectionId')
      ..remove('collectionName')
      ..remove('created')
      ..remove('updated')
      ..remove('expand')
      ..remove('author')
      ..remove('trail');

    if (mode != PhotoApiMode.legacy) body.remove('photos');
    final response = await photoMetadataWrite(
      api.post(session.route('/waypoint/${newWaypoint.id}'), data: body),
      newWaypoint.toJson(),
    );
    var updated = Waypoint.fromJson(response);

    final removedFilenames = oldWaypoint.photos
        .where((p) => !newWaypoint.photos.contains(p))
        .toList();

    final useJournal =
        newWaypoint.removedPendingPhotos.isNotEmpty ||
        newWaypoint.localPhotos.any(
          (path) => p
              .split(path)
              .any((part) => part == 'photo-mutations' || part == 'unsynced'),
        ) ||
        (await ref
                .read(photoMutationServiceProvider)
                .pendingPaths(
                  PhotoTarget.waypoint(newWaypoint.id),
                  authorId,
                  serverBaseUrl: session.baseUrl,
                ))
            .isNotEmpty;
    if (newWaypoint.localPhotos.isNotEmpty ||
        useJournal ||
        removedFilenames.isNotEmpty ||
        newWaypoint.removedPendingPhotos.isNotEmpty) {
      updated = await _uploadPhotos(
        updated,
        useJournal: useJournal,
        authorId: authorId,
        mode: mode,
        serverBaseUrl: session.baseUrl,
        session: session,
        newPhotos: newWaypoint.localPhotos,
        removedFilenames: removedFilenames,
        removedPendingPaths: newWaypoint.removedPendingPhotos,
      );
    }

    return updated;
  }

  Future<void> delete(Waypoint waypoint) async {
    final api = ref.read(apiProvider);
    final session = PhotoWriteSession(
      api,
      currentEpoch: () => ref.read(apiProvider.notifier).sessionEpoch,
    );
    final actor = ref.read(authProvider).value?.actorId ?? waypoint.author;
    await api.delete(session.route('/waypoint/${waypoint.id}'));
    await ref
        .read(photoMutationServiceProvider)
        .drop(
          PhotoTarget.waypoint(waypoint.id),
          actor,
          serverBaseUrl: session.baseUrl,
        );
  }

  /// Used by the offline queue for both first creation and photo-only resume.
  Future<Waypoint> saveQueuedWaypoint(
    Waypoint waypoint, {
    required String authorId,
    required String trailId,
    PhotoApiMode? mode,
  }) async {
    _completedPhotoPaths = const [];
    rejectedPhotoCount = 0;
    photoJobDropped = false;
    photoJobPending = false;
    photoTargetDeleted = false;
    if (waypoint.id.isEmpty) {
      return create(
        waypoint,
        authorId: authorId,
        trailId: trailId,
        mode: mode,
        retainPhotoReceipts: true,
      );
    }
    final api = ref.read(apiProvider);
    final base = api.options.baseUrl;
    if (waypoint.localPhotos.isEmpty &&
        !await ref
            .read(photoMutationServiceProvider)
            .hasJob(
              PhotoTarget.waypoint(waypoint.id),
              authorId,
              serverBaseUrl: base,
            ))
      return waypoint;
    if (api.options.baseUrl != base) throw StateError('Photo session changed');
    return uploadPendingPhotos(waypoint, authorId: authorId, mode: mode);
  }

  Future<Waypoint> uploadPendingPhotos(
    Waypoint waypoint, {
    required String authorId,
    PhotoApiMode? mode,
  }) async {
    _completedPhotoPaths = const [];
    final result = await ref
        .read(photoMutationServiceProvider)
        .apply(
          target: PhotoTarget.waypoint(waypoint.id),
          actorId: authorId,
          newPhotos: waypoint.localPhotos,
          existingPhotos: waypoint.photos,
          existingPhotosAreAuthoritative: false,
          retryImmediately: false,
          removedPendingPaths: waypoint.removedPendingPhotos,
          mode: mode,
        );
    rejectedPhotoCount = result.rejectedCount;
    photoJobDropped = result.dropped;
    photoJobPending = result.pending;
    photoTargetDeleted = result.confirmedDeleted;
    _completedPhotoPaths = result.completedPaths;
    return waypoint.copyWith(
      photos: result.photos,
      localPhotos: result.pendingPaths,
      removedPendingPhotos: const [],
    );
  }

  Future<Waypoint> _uploadPhotos(
    Waypoint waypoint, {
    required String authorId,
    required PhotoApiMode mode,
    String? serverBaseUrl,
    PhotoWriteSession? session,
    bool useJournal = false,
    bool retainPhotoReceipts = false,
    required List<String> newPhotos,
    required List<String> removedFilenames,
    List<String> removedPendingPaths = const [],
  }) async {
    if (mode != PhotoApiMode.legacy || useJournal) {
      final result = await ref
          .read(photoMutationServiceProvider)
          .apply(
            target: PhotoTarget.waypoint(waypoint.id),
            actorId: authorId,
            serverBaseUrl: serverBaseUrl,
            mode: mode,
            retryImmediately: !retainPhotoReceipts,
            removedPendingPaths: removedPendingPaths,
            newPhotos: newPhotos,
            removedPhotos: removedFilenames,
            existingPhotos: waypoint.photos,
          );
      rejectedPhotoCount = result.rejectedCount;
      photoJobDropped = result.dropped;
      photoJobPending = result.pending;
      photoTargetDeleted = result.confirmedDeleted;
      _completedPhotoPaths = result.completedPaths;
      final saved = waypoint.copyWith(
        removedPendingPhotos: const [],
        photos: result.photos,
        localPhotos: result.pendingPaths,
      );
      if (!result.pending && !retainPhotoReceipts) {
        await ref
            .read(photoMutationServiceProvider)
            .acknowledge(
              PhotoTarget.waypoint(saved.id),
              authorId,
              serverBaseUrl: serverBaseUrl,
            );
      }
      return saved;
    }
    final formData = FormData();
    for (final path in newPhotos) {
      formData.files.add(
        MapEntry(
          'photos',
          await MultipartFile.fromFile(
            path,
            filename: File(path).uri.pathSegments.last,
          ),
        ),
      );
    }
    for (final filename in removedFilenames) {
      formData.fields.add(
        MapEntry('photos-', filename.split(RegExp(r'[\\/]')).last),
      );
    }

    try {
      final response = await ref
          .read(apiProvider)
          .post(
            session?.route('/waypoint/${waypoint.id}/file') ??
                '/waypoint/${waypoint.id}/file',
            data: formData,
          );
      return Waypoint.fromJson(response.data);
    } catch (error) {
      if (!isLegacyPhotoFieldRemoved(error)) rethrow;
      ref.read(serverCapabilitiesProvider).markAssetApi();
      return _uploadPhotos(
        waypoint,
        authorId: authorId,
        mode: PhotoApiMode.assets,
        retainPhotoReceipts: retainPhotoReceipts,
        serverBaseUrl: serverBaseUrl,
        session: session,
        newPhotos: newPhotos,
        removedFilenames: removedFilenames,
        removedPendingPaths: removedPendingPaths,
      );
    }
  }
}
