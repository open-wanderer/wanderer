import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'waypoint_provider.g.dart';

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
    var created = Waypoint.fromJson(response.data);

    if (waypoint.localPhotos.isNotEmpty) {
      created = await _uploadPhotos(
        created,
        newPhotos: waypoint.localPhotos,
        removedFilenames: const [],
      );
    }

    return created;
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
        MapEntry('photos', await MultipartFile.fromFile(path, filename: File(path).uri.pathSegments.last)),
      );
    }
    for (final filename in removedFilenames) {
      formData.fields.add(MapEntry('photos-', filename.split(RegExp(r'[\\/]')).last));
    }

    final response = await ref
        .read(apiProvider)
        .post('/waypoint/${waypoint.id}/file', data: formData);
    return Waypoint.fromJson(response.data);
  }
}
