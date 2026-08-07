import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';

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
  @override
  FutureOr<void> build() {}

  Future<Waypoint> create(
    Waypoint waypoint, {
    required String authorId,
    required String trailId,
  }) async {
    final api = ref.read(apiProvider);

    final body = waypoint.toJson()
      ..remove('id')
      ..remove('collectionId')
      ..remove('collectionName')
      ..remove('created')
      ..remove('updated')
      ..['author'] = authorId
      ..['trail'] = trailId;

    final response = await api.put('/waypoint', data: body);
    final created = Waypoint.fromJson(response.data);

    if (waypoint.localPhotos.isEmpty) return created;

    // Past this line the record EXISTS server-side. Any failure below must
    // therefore surface the created record rather than looking like "the
    // waypoint was never created" -- see [WaypointPhotoUploadException].
    try {
      return await _uploadPhotos(
        created,
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
  }) async {
    final api = ref.read(apiProvider);

    final body = newWaypoint.toJson()
      ..remove('id')
      ..remove('collectionId')
      ..remove('collectionName')
      ..remove('created')
      ..remove('updated')
      ..remove('author')
      ..remove('trail');

    final response = await api.post('/waypoint/${newWaypoint.id}', data: body);
    var updated = Waypoint.fromJson(response.data);

    final removedFilenames = oldWaypoint.photos
        .where((p) => !newWaypoint.photos.contains(p))
        .toList();

    if (newWaypoint.localPhotos.isNotEmpty || removedFilenames.isNotEmpty) {
      updated = await _uploadPhotos(
        updated,
        newPhotos: newWaypoint.localPhotos,
        removedFilenames: removedFilenames,
      );
    }

    return updated;
  }

  Future<void> delete(Waypoint waypoint) async {
    await ref.read(apiProvider).delete('/waypoint/${waypoint.id}');
  }

  Future<Waypoint> _uploadPhotos(
    Waypoint waypoint, {
    required List<String> newPhotos,
    required List<String> removedFilenames,
  }) async {
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

    final response = await ref
        .read(apiProvider)
        .post('/waypoint/${waypoint.id}/file', data: formData);
    return Waypoint.fromJson(response.data);
  }
}
