import 'dart:io';
import 'dart:math' as math;

import 'package:dio/dio.dart';
import 'package:objectbox/objectbox.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailDownloadService {
  final Store _store;
  final Dio _dio;
  final String _serverUrl;

  TrailDownloadService(this._store, this._dio, this._serverUrl);

  Future<void> downloadTrail(Trail trail) async {
    final box = _store.box<TrailEntity>();
    final trailId = trail.id;

    final appDir = await getApplicationDocumentsDirectory();
    final trailDir = Directory('${appDir.path}/library/$trailId');

    if (!await trailDir.exists()) {
      await trailDir.create(recursive: true);
    }

    final List<String> localPaths = await _downloadPhotos(
      trail.photos.map((p) => trail.getFileUrl(_serverUrl, p)!).toList(),
      trailDir,
    );

    await _downloadMapTiles(trail, trailDir);

    final entity = TrailEntity.fromModel(trail);
    entity.photos = localPaths;

    _store.runInTransaction(TxMode.write, () {
      box.put(entity);
    });
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

        await _dio.download(
          url,
          savePath,
          onReceiveProgress: (received, total) {
            if (total != -1) {
              // You could pipe this to a progress notifier/stream
              // print("Download $fileName: ${(received / total * 100).toStringAsFixed(0)}%");
            }
          },
        );

        return savePath;
      } catch (e) {
        // Log error but don't stop the entire process
        print("Failed to download photo $url: $e");
        return null;
      }
    }).toList();

    final results = await Future.wait(downloadTasks);
    return results.whereType<String>().toList();
  }

  int _getTileX(double lon, int zoom) {
    return ((lon + 180) / 360 * math.pow(2, zoom)).floor();
  }

  int _getTileY(double lat, int zoom) {
    var latRad = lat * math.pi / 180;
    return ((1 - math.log(math.tan(latRad) + 1 / math.cos(latRad)) / math.pi) /
            2 *
            math.pow(2, zoom))
        .floor();
  }

  Future<void> _downloadMapTiles(Trail trail, Directory trailDir) async {
    const minZoom = 0;
    const maxZoom = 14;

    final bounds = trail.expand?.gpx?.getBounds();
    if (bounds == null) return;

    final mapDir = Directory('${trailDir.path}/map');
    if (!await mapDir.exists()) await mapDir.create(recursive: true);

    const baseUrl = "https://tiles.openfreemap.org/planet/latest";

    for (int z = minZoom; z <= maxZoom; z++) {
      final xMin = _getTileX(bounds.west, z);
      final xMax = _getTileX(bounds.east, z);
      final yMin = _getTileY(bounds.north, z);
      final yMax = _getTileY(bounds.south, z);

      for (int x = xMin; x <= xMax; x++) {
        for (int y = yMin; y <= yMax; y++) {
          final url = "$baseUrl/$z/$x/$y.pbf";
          final savePath = '${mapDir.path}/$z/$x/$y.pbf';
          final file = File(savePath);

          if (!await file.exists()) {
            await file.create(recursive: true);
            try {
              await _dio.download(url, savePath);
            } catch (e) {
              print("Skipping tile $z/$x/$y: $e");
            }
          }
        }
      }
    }
  }
}
