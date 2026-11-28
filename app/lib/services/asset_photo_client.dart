import 'dart:io';

import 'package:dio/dio.dart';

/// The relation whose photo links a mutation changes.
class PhotoTarget {
  final String kind;
  final String id;

  const PhotoTarget.trail(this.id) : kind = 'trail';
  const PhotoTarget.waypoint(this.id) : kind = 'waypoint';
  const PhotoTarget.summitLog(this.id) : kind = 'summit_log';
  const PhotoTarget(this.kind, this.id);

  Map<String, String> get fields => {kind: id};
}

String? photoAssetId(String photo) {
  final path = Uri.tryParse(photo)?.path ?? photo;
  return RegExp(r'/api/v1/assets/([a-z0-9]{15})/file(?:/|$)')
          .firstMatch(path)
          ?.group(1) ??
      RegExp(r'/api/v1/files/[^/]+/([a-z0-9]{15})/').firstMatch(path)?.group(1);
}

String assetPhotoUrl(Map<String, dynamic> asset) {
  final id = asset['id'] as String;
  final file = asset['file'];
  if (file is String && file.isNotEmpty) {
    return '/api/v1/files/${asset['collectionId']}/$id/$file';
  }
  return '/api/v1/assets/$id/file';
}

/// Uploads in input order; progress is delivered before a partial error escapes.
class AssetPhotoClient {
  static const batchBytes = 256 * 1024 * 1024;
  static const maxFileBytes = 50 * 1024 * 1024;

  final Dio api;
  final String baseUrl;
  final bool Function()? canSend;

  AssetPhotoClient(this.api, {String? baseUrl, this.canSend})
    : baseUrl = baseUrl ?? api.options.baseUrl;

  Future<List<String>> upload(
    List<String> paths,
    PhotoTarget target, {
    Future<void> Function(int completedFiles, List<String> photos)? onProgress,
    List<String>? filenames,
  }) async {
    final sizes = <int>[];
    for (final path in paths) {
      final size = await File(path).length();
      if (size > maxFileBytes) {
        throw ArgumentError('Photo exceeds the asset request limit');
      }
      sizes.add(size);
    }
    var start = 0;
    final photos = <String>[];
    while (start < paths.length) {
      var end = start;
      var bytes = 0;
      while (end < paths.length &&
          (end == start || bytes + sizes[end] <= batchBytes)) {
        bytes += sizes[end++];
      }
      if (canSend?.call() == false) throw StateError('Photo session changed');
      final form = FormData()..fields.addAll(target.fields.entries);
      for (final path in paths.sublist(start, end)) {
        form.files.add(
          MapEntry(
            'files',
            await MultipartFile.fromFile(
              path,
              filename: filenames?[paths.indexOf(path)],
            ),
          ),
        );
      }
      try {
        if (canSend?.call() == false) throw StateError('Photo session changed');
        final response = await api.put('$baseUrl/assets', data: form);
        final uploaded = (response.data as List)
            .map(
              (asset) => assetPhotoUrl(Map<String, dynamic>.from(asset as Map)),
            )
            .toList();
        photos.addAll(uploaded);
        await onProgress?.call(end, List.of(photos));
      } on DioException catch (error) {
        final body = error.response?.data;
        final detail = body is Map ? body['detail'] : null;
        if (detail is Map) {
          final count = detail['completedFileCount'];
          final assets = detail['uploadedAssets'];
          if (count is int &&
              count > 0 &&
              count <= end - start &&
              assets is List) {
            final uploaded = assets
                .map(
                  (asset) =>
                      assetPhotoUrl(Map<String, dynamic>.from(asset as Map)),
                )
                .toList();
            if (uploaded.length == count) {
              photos.addAll(uploaded);
              await onProgress?.call(start + count, List.of(photos));
            }
          }
        }
        rethrow;
      }
      start = end;
    }
    return photos;
  }

  Future<void> remove(String photo, PhotoTarget target) async {
    final id = photoAssetId(photo);
    if (id == null) throw ArgumentError('Photo has no asset ID');
    if (canSend?.call() == false) throw StateError('Photo session changed');
    await api.delete('$baseUrl/assets/$id', queryParameters: target.fields);
  }

  Future<void> cover(String trailId, String? photo) async {
    if (photo != null && photoAssetId(photo) == null)
      throw ArgumentError('Photo has no asset ID');
    if (canSend?.call() == false) throw StateError('Photo session changed');
    await api.patch(
      '$baseUrl/trail/$trailId/thumbnail',
      data: {'asset': photo == null ? null : photoAssetId(photo)},
    );
  }
}
