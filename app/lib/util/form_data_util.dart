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
      // `uploadUpdate` (web/src/lib/util/api_util.ts) ignores the URL path id
      // and reads it from the form body instead — required on update.
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
          await MultipartFile.fromFile(photo.path, filename: photo.uri.pathSegments.last),
        ),
    ];

    final gpxData = expand?.gpxData;
    if (isCreate && gpxData != null) {
      files.add(
        MapEntry('gpx', MultipartFile.fromString(gpxData, filename: 'track.gpx')),
      );
    }

    return FormData()
      ..fields.addAll(fields)
      ..files.addAll(files);
  }
}

String _basename(String path) => path.split(RegExp(r'[\\/]')).last;
