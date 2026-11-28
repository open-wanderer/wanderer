import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/store/app_photo_paths.dart';

void main() {
  setUp(() => configurePhotoDocumentsDirectory('/docs'));
  test(
    'downloaded copies remain separate from pending uploads and wire fields',
    () {
      final waypoint = Waypoint(
        id: 'point0000000001',
        lat: 1,
        lon: 2,
        created: DateTime(2026),
        updated: DateTime(2026),
        photos: const ['/api/v1/assets/asset0000000001/file'],
        localPhotos: const ['/docs/unsynced/pending.jpg'],
        downloadedPhotos: const ['/docs/library/cached.jpg'],
        removedPendingPhotos: const ['/docs/unsynced/removed.jpg'],
      );
      final stored = WaypointEntity.fromModel(waypoint);
      expect(stored.photos, waypoint.photos);
      expect(stored.localPhotos, ['unsynced/pending.jpg']);
      expect(stored.downloadedPhotos, ['library/cached.jpg']);
      final restored = stored.toModel();
      expect(restored.photos, waypoint.photos);
      expect(restored.localPhotos, waypoint.localPhotos);
      expect(restored.downloadedPhotos, waypoint.downloadedPhotos);
      expect(restored.toJson().containsKey('downloadedPhotos'), isFalse);
      expect(restored.toJson().containsKey('removedPendingPhotos'), isFalse);
    },
  );
  test(
    'stored and earlier absolute waypoint paths follow a moved app container',
    () {
      configurePhotoDocumentsDirectory('/container/old/Documents');
      final waypoint = Waypoint(
        id: 'point0000000001',
        lat: 1,
        lon: 2,
        created: DateTime(2026),
        updated: DateTime(2026),
        localPhotos: const [
          '/container/old/Documents/unsynced/account/local-1-1/pending.jpg',
        ],
        downloadedPhotos: const [
          '/container/old/Documents/library/trail/cached.jpg',
        ],
      );
      final stored = WaypointEntity.fromModel(waypoint);
      expect(stored.localPhotos, ['unsynced/account/local-1-1/pending.jpg']);
      expect(stored.downloadedPhotos, ['library/trail/cached.jpg']);
      configurePhotoDocumentsDirectory('/container/new/Documents');
      final model = stored.toModel();
      expect(model.localPhotos, [
        '/container/new/Documents/unsynced/account/local-1-1/pending.jpg',
      ]);
      expect(model.downloadedPhotos, [
        '/container/new/Documents/library/trail/cached.jpg',
      ]);
      stored.localPhotos = waypoint.localPhotos;
      stored.downloadedPhotos = waypoint.downloadedPhotos;
      expect(stored.toModel().localPhotos, model.localPhotos);
      expect(stored.toModel().downloadedPhotos, model.downloadedPhotos);
    },
  );
}
