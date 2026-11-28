import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/store/app_photo_paths.dart';

void main() {
  setUp(() => configurePhotoDocumentsDirectory('/container/new/Documents'));

  TrailEntity row() =>
      TrailEntity.fromModel(Trail.empty().copyWith(id: 'trail0000000001'));

  test(
    'a resumed capture exposes server photos without creating local picks',
    () {
      final entity = row()
        ..localId = 'local-1-1'
        ..syncState = TrailSyncState.pending
        ..photos = [
          '/api/v1/files/assets/asset0000000001/photo.jpg',
          'https://b.example/photo.jpg',
          '//b.example/other.jpg',
          'legacy.jpg',
          'legacy-query.jpg?share=reader/part',
        ];
      final model = entity.toModel();
      expect(model.photos, entity.photos);
      expect(model.localPhotos, isEmpty);
    },
  );

  test('mixed downloaded paths and server references remain separate after a container move', () {
    final entity = row()
      ..photos = [
        '/api/v1/assets/asset0000000001/file',
        'legacy.jpg',
        'library/trail0000000001/photos/cached.jpg',
        '/container/old/Documents/library/trail0000000001/photos/older.jpg',
      ];
    final model = entity.toModel();
    expect(model.photos, ['/api/v1/assets/asset0000000001/file', 'legacy.jpg']);
    expect(model.localPhotos, [
      '/container/new/Documents/library/trail0000000001/photos/cached.jpg',
      '/container/new/Documents/library/trail0000000001/photos/older.jpg',
    ]);
  });

  test('pending picks keep precedence over downloaded copies', () {
    final entity = row()
      ..photos = ['legacy.jpg', 'library/trail0000000001/photos/cached.jpg']
      ..localPhotos = ['unsynced/account/local-1-1/pending.jpg'];
    final model = entity.toModel();
    expect(model.photos, ['legacy.jpg']);
    expect(model.localPhotos, [
      '/container/new/Documents/unsynced/account/local-1-1/pending.jpg',
    ]);
  });
}
