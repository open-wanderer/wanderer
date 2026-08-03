import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/store/local_photo_store.dart';

// ---------------------------------------------------------------------------
// Tests for the app-owned unsynced-photo store. This is the load-bearing
// path-safety and copy-failure-tolerance control (D-01/D-02/D-03): every
// directory segment must be validated before any dart:io call, and a
// per-photo copy failure must never abort a save or fall back to an
// OS-purgeable picker path.
// ---------------------------------------------------------------------------

/// Fakes the app-docs directory as a temp dir for [sweepOrphanedUnsyncedPhotos],
/// the only function under test that resolves `getApplicationDocumentsDirectory()`
/// internally rather than taking an `appDocsPath` parameter.
class _FakePathProviderPlatform extends PathProviderPlatform {
  _FakePathProviderPlatform(this.appDocsPath);

  final String appDocsPath;

  @override
  Future<String?> getApplicationDocumentsPath() async => appDocsPath;
}

void main() {
  late Directory tempRoot;

  setUp(() {
    tempRoot = Directory.systemTemp.createTempSync('local_photo_store_test_');
  });

  tearDown(() {
    if (tempRoot.existsSync()) {
      tempRoot.deleteSync(recursive: true);
    }
  });

  group('path builders', () {
    const localId = 'local-1700000000000000-1';
    const waypointKey = 'local-1700000000000001-2';

    test('unsyncedPhotoRoot is p.join-shaped under <root>/unsynced', () {
      final path = unsyncedPhotoRoot(tempRoot.path);
      expect(path, p.join(tempRoot.path, 'unsynced'));
    });

    test('unsyncedTrailPhotoDir is p.join-shaped under the unsynced root', () {
      final path = unsyncedTrailPhotoDir(tempRoot.path, localId);
      expect(path, p.join(tempRoot.path, 'unsynced', localId));
      expect(p.isWithin(unsyncedPhotoRoot(tempRoot.path), path), isTrue);
    });

    test('unsyncedWaypointPhotoDir is p.join-shaped under the trail dir', () {
      final path = unsyncedWaypointPhotoDir(
        tempRoot.path,
        localId,
        waypointKey,
      );
      expect(
        path,
        p.join(tempRoot.path, 'unsynced', localId, 'waypoints', waypointKey),
      );
      expect(
        p.isWithin(unsyncedTrailPhotoDir(tempRoot.path, localId), path),
        isTrue,
      );
    });

    test('unsyncedTrailPhotoDir throws ArgumentError for a traversal id', () {
      expect(
        () => unsyncedTrailPhotoDir(tempRoot.path, '../escape'),
        throwsArgumentError,
      );
    });

    test('unsyncedTrailPhotoDir throws ArgumentError for a malformed id', () {
      expect(
        () => unsyncedTrailPhotoDir(tempRoot.path, 'not-a-local-id'),
        throwsArgumentError,
      );
    });

    test(
      'unsyncedWaypointPhotoDir throws ArgumentError for a traversal trail id',
      () {
        expect(
          () =>
              unsyncedWaypointPhotoDir(tempRoot.path, '../escape', waypointKey),
          throwsArgumentError,
        );
      },
    );

    test(
      'unsyncedWaypointPhotoDir throws ArgumentError for a traversal waypoint key',
      () {
        expect(
          () => unsyncedWaypointPhotoDir(tempRoot.path, localId, '../escape'),
          throwsArgumentError,
        );
      },
    );

    test(
      'unsyncedWaypointPhotoDir throws ArgumentError for a malformed waypoint key',
      () {
        expect(
          () => unsyncedWaypointPhotoDir(
            tempRoot.path,
            localId,
            'not-a-local-id',
          ),
          throwsArgumentError,
        );
      },
    );
  });

  group('reconcileLocalPhotos', () {
    test(
      'copies a real temp file in and returns its new path inside dir',
      () async {
        final dir = p.join(tempRoot.path, 'dest');
        final source = File(p.join(tempRoot.path, 'photo.jpg'))
          ..writeAsStringSync('fake-photo-bytes');

        final result = await reconcileLocalPhotos(
          dir: dir,
          desiredPaths: [source.path],
        );

        expect(result.failedCount, 0);
        expect(result.paths, hasLength(1));
        expect(p.isWithin(dir, result.paths.single), isTrue);
        expect(File(result.paths.single).existsSync(), isTrue);
        expect(
          File(result.paths.single).readAsStringSync(),
          'fake-photo-bytes',
        );
      },
    );

    test(
      'a path already inside dir is kept verbatim, not duplicated',
      () async {
        final dir = p.join(tempRoot.path, 'dest');
        await Directory(dir).create(recursive: true);
        final existing = File(p.join(dir, 'already-here.jpg'))
          ..writeAsStringSync('bytes');

        final result = await reconcileLocalPhotos(
          dir: dir,
          desiredPaths: [existing.path],
        );

        expect(result.failedCount, 0);
        expect(result.paths, [existing.path]);
        expect(Directory(dir).listSync().length, 1);
      },
    );

    test(
      'a non-existent source path increments failedCount by 1, is absent from paths, and does not throw',
      () async {
        final dir = p.join(tempRoot.path, 'dest');
        final missing = p.join(tempRoot.path, 'does-not-exist.jpg');

        final result = await reconcileLocalPhotos(
          dir: dir,
          desiredPaths: [missing],
        );

        expect(result.failedCount, 1);
        expect(result.paths, isEmpty);
      },
    );

    test(
      'a previously-copied file absent from desiredPaths is removed from dir',
      () async {
        final dir = p.join(tempRoot.path, 'dest');
        final source = File(p.join(tempRoot.path, 'photo.jpg'))
          ..writeAsStringSync('bytes');

        final first = await reconcileLocalPhotos(
          dir: dir,
          desiredPaths: [source.path],
        );
        expect(first.paths, hasLength(1));
        expect(File(first.paths.single).existsSync(), isTrue);

        final second = await reconcileLocalPhotos(dir: dir, desiredPaths: []);
        expect(second.paths, isEmpty);
        expect(File(first.paths.single).existsSync(), isFalse);
      },
    );

    // WR-13. The keep decision uses `p.isWithin`, which normalizes; the
    // delete pass used to compare `listSync().path` by raw string equality,
    // which does not. So a non-canonical spelling of an in-dir path was
    // reported as KEPT and deleted from disk in the same call -- the entity
    // ended up pointing at a file that no longer existed, silently, with no
    // failedCount increment. Every spelling below is the same file.
    for (final entry in {
      'a "./" segment': (String dir, String name) => p.join(dir, '.', name),
      'a doubled separator': (String dir, String name) =>
          '$dir${p.separator}${p.separator}$name',
      'a redundant parent hop': (String dir, String name) =>
          p.join(dir, '..', p.basename(dir), name),
    }.entries) {
      test(
        'a kept in-dir path spelled with ${entry.key} survives the delete pass',
        () async {
          final dir = p.join(tempRoot.path, 'dest');
          await Directory(dir).create(recursive: true);
          final canonical = File(p.join(dir, 'already-here.jpg'))
            ..writeAsStringSync('bytes');

          final spelled = entry.value(dir, 'already-here.jpg');
          // Guard the premise: this really is a different STRING that really
          // does denote the same file.
          expect(spelled, isNot(canonical.path));
          expect(p.canonicalize(spelled), p.canonicalize(canonical.path));

          final result = await reconcileLocalPhotos(
            dir: dir,
            desiredPaths: [spelled],
          );

          expect(result.failedCount, 0);
          expect(result.paths, [spelled]);
          expect(
            File(spelled).existsSync(),
            isTrue,
            reason:
                'reconcileLocalPhotos returned this path as kept and then '
                'deleted the file it names.',
          );
          expect(Directory(dir).listSync(), hasLength(1));
        },
      );
    }

    test(
      'a trailing separator on dir does not make a kept file get deleted',
      () async {
        final dir = p.join(tempRoot.path, 'dest');
        await Directory(dir).create(recursive: true);
        final existing = File(p.join(dir, 'already-here.jpg'))
          ..writeAsStringSync('bytes');

        final result = await reconcileLocalPhotos(
          dir: '$dir${p.separator}',
          desiredPaths: [existing.path],
        );

        expect(result.paths, [existing.path]);
        expect(existing.existsSync(), isTrue);
      },
    );
  });

  group('photosNotYetOnServer', () {
    test('a path directly inside unsyncedDir is excluded', () {
      final result = photosNotYetOnServer(
        unsyncedDir: '/docs/unsynced/local-1-0',
        pickedPaths: ['/docs/unsynced/local-1-0/a.jpg'],
      );

      expect(result, isEmpty);
    });

    test(
      'a path inside a waypoints/<key>/ subdirectory of unsyncedDir is '
      'excluded',
      () {
        final result = photosNotYetOnServer(
          unsyncedDir: '/docs/unsynced/local-1-0',
          pickedPaths: [
            '/docs/unsynced/local-1-0/waypoints/local-2-0/b.jpg',
          ],
        );

        expect(result, isEmpty);
      },
    );

    test('an image_picker-style cache path outside unsyncedDir is kept', () {
      final result = photosNotYetOnServer(
        unsyncedDir: '/docs/unsynced/local-1-0',
        pickedPaths: ['/cache/picker/c.jpg'],
      );

      expect(result, ['/cache/picker/c.jpg']);
    });

    test(
      'a non-canonical spelling of an in-dir path is excluded -- '
      'p.isWithin normalizes, mirroring reconcileLocalPhotos\' own '
      'canonicalization fix',
      () {
        final result = photosNotYetOnServer(
          unsyncedDir: '/docs/unsynced/local-1-0',
          pickedPaths: ['/docs/unsynced/local-1-0/./a.jpg'],
        );

        expect(result, isEmpty);
      },
    );

    test('input order is preserved for the kept entries in a mixed list', () {
      final result = photosNotYetOnServer(
        unsyncedDir: '/docs/unsynced/local-1-0',
        pickedPaths: [
          '/cache/picker/first.jpg',
          '/docs/unsynced/local-1-0/already-uploaded.jpg',
          '/cache/picker/second.jpg',
        ],
      );

      expect(result, ['/cache/picker/first.jpg', '/cache/picker/second.jpg']);
    });

    test('an empty pickedPaths returns an empty list', () {
      final result = photosNotYetOnServer(
        unsyncedDir: '/docs/unsynced/local-1-0',
        pickedPaths: const [],
      );

      expect(result, isEmpty);
    });
  });

  group('sweepOrphanedUnsyncedPhotos', () {
    setUp(() {
      PathProviderPlatform.instance = _FakePathProviderPlatform(tempRoot.path);
    });

    test(
      'deletes a directory not in keepLocalIds and leaves a kept one intact',
      () async {
        final root = unsyncedPhotoRoot(tempRoot.path);
        const keptId = 'local-1700000000000000-1';
        const orphanId = 'local-1700000000000000-2';
        await Directory(p.join(root, keptId)).create(recursive: true);
        await Directory(p.join(root, orphanId)).create(recursive: true);
        File(p.join(root, keptId, 'photo.jpg')).writeAsStringSync('bytes');

        final deleted = await sweepOrphanedUnsyncedPhotos(
          keepLocalIds: {keptId},
        );

        expect(deleted, 1);
        expect(Directory(p.join(root, keptId)).existsSync(), isTrue);
        expect(File(p.join(root, keptId, 'photo.jpg')).existsSync(), isTrue);
        expect(Directory(p.join(root, orphanId)).existsSync(), isFalse);
      },
    );
  });
}
