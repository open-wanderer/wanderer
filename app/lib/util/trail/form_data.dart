import 'dart:io';

import 'package:dio/dio.dart';
import 'package:wanderer/models/trail.dart';

/// Builds the multipart body for `PUT /trail/form` (create) or
/// `POST /trail/form/{id}` (update).
extension TrailFormData on Trail {
  Future<FormData> toFormData({
    List<File> newPhotos = const [],
    List<String> removedPhotoFilenames = const [],
    bool isCreate = true,
  }) async {
    final fields = <MapEntry<String, String>>[
      // uploadUpdate ignores the URL path id and reads it from the form body
      // instead — required on update.
      if (!isCreate) MapEntry('id', id),
      MapEntry('name', name),
      if (location != null) MapEntry('location', location!),
      if (date != null) MapEntry('date', date!.toIso8601String()),
      MapEntry('description', description),
      MapEntry('public', public.toString()),
      MapEntry('completed', completed.toString()),
      MapEntry('difficulty', difficulty.name),
      MapEntry('distance', distance.toString()),
      MapEntry('elevation_gain', elevationGain.toString()),
      MapEntry('elevation_loss', elevationLoss.toString()),
      MapEntry('duration', duration.toString()),
      // moving_duration is only ever written by the capture paths (from a
      // live session's elapsed time), never recomputed from a GPX -- the
      // null guard is required because sending an empty string for an
      // absent value would write 0 into PocketBase and defeat the "no
      // value" state.
      if (movingDuration != null)
        MapEntry('moving_duration', movingDuration!.toString()),
      if (lat != null) MapEntry('lat', lat.toString()),
      if (lon != null) MapEntry('lon', lon.toString()),
      if (category != null) MapEntry('category', category!),
      if (subcategory != null) MapEntry('subcategory', subcategory!),
      MapEntry('author', author),
      for (final tagId in tags) MapEntry('tags', tagId),
      for (final filename in removedPhotoFilenames)
        MapEntry('photos-', _basename(filename)),
    ];

    final files = <MapEntry<String, MultipartFile>>[
      for (final photo in newPhotos)
        MapEntry(
          isCreate ? 'photos' : 'photos+',
          await MultipartFile.fromFile(
            photo.path,
            filename: photo.uri.pathSegments.last,
          ),
        ),
    ];

    // Sent on update too, not just create — otherwise an edited route's
    // track would be silently dropped. Resending unchanged gpxData on a
    // metadata-only update is harmless since it just replaces the file.
    final gpxData = expand?.gpxData;
    if (gpxData != null) {
      files.add(
        MapEntry(
          'gpx',
          MultipartFile.fromString(gpxData, filename: 'track.gpx'),
        ),
      );
    }

    return FormData()
      ..fields.addAll(fields)
      ..files.addAll(files);
  }
}

String _basename(String path) => path.split(RegExp(r'[\\/]')).last;
