import 'dart:io';
import 'package:dio/dio.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:gpx/gpx.dart';
import 'package:objectbox/objectbox.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/map_cell.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailDownloadService {
  final Store _store;
  final Dio _api;

  static const _pollInterval = Duration(seconds: 3);
  static const _pollTimeout = Duration(minutes: 3);

  TrailDownloadService(this._store, this._api);

  Future<void> downloadTrail(Trail trail) async {
    final box = _store.box<TrailEntity>();
    final trailId = trail.id;
    final appDir = await getApplicationDocumentsDirectory();
    final trailDir = Directory('${appDir.path}/library/$trailId');

    if (!await trailDir.exists()) {
      await trailDir.create(recursive: true);
    }

    final List<String> localPaths = await _downloadPhotos(
      trail.photos
          .map((p) => trail.getFileUrl(_api.options.baseUrl, p)!)
          .toList(),
      trailDir,
    );

    final cellPaths = await _downloadMapTiles(trail, trailDir);

    final entity = TrailEntity.fromModel(trail);
    entity.photos = localPaths;
    entity.pmTiles = cellPaths;
    _store.runInTransaction(TxMode.write, () {
      box.put(entity);
    });
  }

  Future<List<String>> _downloadMapTiles(
    Trail trail,
    Directory trailDir,
  ) async {
    final LatLngBounds bounds;

    if (trail.expand?.gpx != null) {
      final b = trail.expand!.gpx!.getBounds();
      if (b == null) {
        throw Exception('GPX for trail ${trail.id} has no track points');
      }
      bounds = b;
    } else if (trail.expand?.gpxData != null) {
      final b = GpxReader().fromString(trail.expand!.gpxData!).getBounds();
      if (b == null) {
        throw Exception('GPX data for trail ${trail.id} has no track points');
      }
      bounds = b;
    } else {
      throw Exception(
        'Trail ${trail.id} has no GPX data in expand. '
        'Make sure gpx_data or gpx is included in the expand when calling downloadTrail.',
      );
    }

    final bbox =
        '${bounds.west},${bounds.south},${bounds.east},${bounds.north}';

    final infoList = await _fetchCellList(bbox);
    if (infoList.cells.isEmpty) return [];

    final tilesDir = Directory('${trailDir.path}/tiles');
    if (!await tilesDir.exists()) {
      await tilesDir.create(recursive: true);
    }

    final savedPaths = <String>[];

    for (final cell in infoList.cells) {
      final key = cell.key;
      final localPath = '${tilesDir.path}/$key.pmtiles';

      if (await File(localPath).exists()) {
        savedPaths.add(localPath);
        continue;
      }

      try {
        final requestUrl = cell.url;
        final requestRes = await _api.get(requestUrl);
        final initialCell = MapCellStatusResponse.fromJson(requestRes.data!);
        final initialStatus = initialCell.status;

        MapCellStatusResponse readyCell;
        if (initialStatus != MapCellStatus.ready) {
          final statusUrl = initialCell.statusUrl!;
          readyCell = await _pollUntilReady(statusUrl, key);
        } else {
          readyCell = initialCell;
        }

        final downloadUrl = readyCell.downloadUrl!;
        await _api.download(downloadUrl, localPath);
        savedPaths.add(localPath);
      } catch (e) {
        print('Failed to download map cell $key: $e');
      }
    }

    return savedPaths;
  }

  Future<MapCellInfoList> _fetchCellList(String bbox) async {
    final res = await _api.get('/map/cells', queryParameters: {'bbox': bbox});
    return MapCellInfoList.fromJson(res.data!);
  }

  Future<MapCellStatusResponse> _pollUntilReady(
    String statusUrl,
    String cellKey,
  ) async {
    final deadline = DateTime.now().add(_pollTimeout);

    while (DateTime.now().isBefore(deadline)) {
      await Future.delayed(_pollInterval);

      final res = await _api.get(statusUrl);
      final data = MapCellStatusResponse.fromJson(res.data!);

      switch (data.status) {
        case MapCellStatus.ready:
          return data;
        case MapCellStatus.error:
          throw Exception(
            'Map cell $cellKey generation failed: ${data.error ?? 'unknown error'}',
          );
        case MapCellStatus.pending:
        case MapCellStatus.isNew:
          continue;
      }
    }
    throw Exception('Timed out waiting for map cell $cellKey');
  }

  Future<List<String>> _downloadPhotos(
    List<String> urls,
    Directory trailDir,
  ) async {
    final photoDir = Directory('${trailDir.path}/photos');
    if (!await photoDir.exists()) {
      await photoDir.create(recursive: true);
    }

    final downloadTasks = urls.map((url) async {
      try {
        final fileName = p.basename(Uri.parse(url).path);
        final savePath = '${photoDir.path}/$fileName';
        await _api.download(
          url,
          savePath,
          onReceiveProgress: (received, total) {
            if (total != -1) {
              // print("Download $fileName: ${(received / total * 100).toStringAsFixed(0)}%");
            }
          },
        );
        return savePath;
      } catch (e) {
        print('Failed to download photo $url: $e');
        return null;
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
    return results.whereType<String>().toList();
  }
}
