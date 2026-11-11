import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/store/app_photo_paths.dart';
import 'package:wanderer/util/local/id.dart';

/// Older clients retained picked paths after their upload was confirmed.
/// Clear those receipts once, before a resumed queue can send them again.
Future<bool> upgradePhotoQueue(Store store, {Directory? documents}) async {
  final docs = documents ?? await getApplicationDocumentsDirectory();
  final flag = File(p.join(docs.path, 'photo-queue-receipts-v1'));
  if (await flag.exists()) return false;
  store.runInTransaction(TxMode.write, () {
    final trails = store.box<TrailEntity>();
    for (final trail in trails.getAll()) {
      if (trail.localId != null &&
          !isLocalId(trail.id) &&
          trail.syncState != TrailSyncState.synced &&
          trail.localPhotos.isNotEmpty) {
        trail.localPhotos = [];
        trails.put(trail);
      }
    }
    final waypoints = store.box<WaypointEntity>();
    for (final waypoint in waypoints.getAll()) {
      if (isLocalId(waypoint.id) ||
          waypoint.id.isEmpty ||
          waypoint.photos.isEmpty ||
          waypoint.localPhotos.isEmpty)
        continue;
      final downloaded = waypoint.localPhotos
          .map((path) => resolvePhotoPath(docs.path, path))
          .where((path) => p.isWithin(p.join(docs.path, 'library'), path))
          .map((path) => storePhotoPath(docs.path, path));
      waypoint.downloadedPhotos = {
        ...waypoint.downloadedPhotos,
        ...downloaded,
      }.toList();
      waypoint.localPhotos = [];
      waypoints.put(waypoint);
    }
  });
  await flag.writeAsString('1', flush: true);
  return true;
}
