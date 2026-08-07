import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/store/local_photo_store.dart';

// ---------------------------------------------------------------------------
// Tests for the app-owned unsynced-photo store. This is the load-bearing
// path-safety and copy-failure-tolerance control: every
// directory segment must be validated before any dart:io call, and a
// per-photo copy failure must never abort a save or fall back to an
// OS-purgeable picker path.
//
// The account segment (`unsynced/<accountId>/<localId>/`) exists so
// one account's unsynced photos are structurally unreachable from another
// account's delete path -- the defect this file also proves closed.
// ---------------------------------------------------------------------------

/// Fakes the app-docs directory as a temp dir for [sweepOrphanedUnsyncedPhotos]
/// and [deleteUnsyncedPhotoDir], the two functions under test that resolve
/// `getApplicationDocumentsDirectory()` internally rather than taking an
/// `appDocsPath` parameter.
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
    const accountId = 'acct000000000a1';
    const localId = 'local-1700000000000000-1';
    const waypointKey = 'local-1700000000000001-2';

    test('unsyncedPhotoRoot is p.join-shaped under <root>/unsynced', () {
      final path = unsyncedPhotoRoot(tempRoot.path);
      expect(path, p.join(tempRoot.path, 'unsynced'));
    });

    test(
      'unsyncedAccountPhotoDir is p.join-shaped under the unsynced root',
      () {
        final path = unsyncedAccountPhotoDir(tempRoot.path, accountId);
        expect(path, p.join(tempRoot.path, 'unsynced', accountId));
        expect(p.isWithin(unsyncedPhotoRoot(tempRoot.path), path), isTrue);
      },
    );

    test(
      'unsyncedTrailPhotoDir is p.join-shaped under the account dir, in '
      'appDocsPath/accountId/localId order',
      () {
        final path = unsyncedTrailPhotoDir(tempRoot.path, accountId, localId);
        expect(
          path,
          p.join(tempRoot.path, 'unsynced', accountId, localId),
        );
        expect(
          p.isWithin(
            unsyncedAccountPhotoDir(tempRoot.path, accountId),
            path,
          ),
          isTrue,
        );
      },
    );

    test('unsyncedWaypointPhotoDir is p.join-shaped under the trail dir', () {
      final path = unsyncedWaypointPhotoDir(
        tempRoot.path,
        accountId,
        localId,
        waypointKey,
      );
      expect(
        path,
        p.join(
          tempRoot.path,
          'unsynced',
          accountId,
          localId,
          'waypoints',
          waypointKey,
        ),
      );
      expect(
        p.isWithin(
          unsyncedTrailPhotoDir(tempRoot.path, accountId, localId),
          path,
        ),
        isTrue,
      );
    });

    test(
      'unsyncedAccountPhotoDir throws ArgumentError for a traversal id',
      () {
        expect(
          () => unsyncedAccountPhotoDir(tempRoot.path, '../escape'),
          throwsArgumentError,
        );
      },
    );

    test(
      'unsyncedAccountPhotoDir throws ArgumentError for an id containing a '
      'path separator',
      () {
        expect(
          () => unsyncedAccountPhotoDir(tempRoot.path, 'a/b'),
          throwsArgumentError,
        );
      },
    );

    test('unsyncedAccountPhotoDir throws ArgumentError for an empty id', () {
      expect(
        () => unsyncedAccountPhotoDir(tempRoot.path, ''),
        throwsArgumentError,
      );
    });

    test(
      'unsyncedTrailPhotoDir throws ArgumentError for a traversal accountId '
      'even when localId is valid',
      () {
        expect(
          () => unsyncedTrailPhotoDir(tempRoot.path, '../escape', localId),
          throwsArgumentError,
        );
      },
    );

    test('unsyncedTrailPhotoDir throws ArgumentError for a traversal localId', () {
      expect(
        () => unsyncedTrailPhotoDir(tempRoot.path, accountId, '../escape'),
        throwsArgumentError,
      );
    });

    test('unsyncedTrailPhotoDir throws ArgumentError for a malformed localId', () {
      expect(
        () =>
            unsyncedTrailPhotoDir(tempRoot.path, accountId, 'not-a-local-id'),
        throwsArgumentError,
      );
    });

    test(
      'unsyncedWaypointPhotoDir throws ArgumentError for a traversal '
      'accountId',
      () {
        expect(
          () => unsyncedWaypointPhotoDir(
            tempRoot.path,
            '../escape',
            localId,
            waypointKey,
          ),
          throwsArgumentError,
        );
      },
    );

    test(
      'unsyncedWaypointPhotoDir throws ArgumentError for a traversal trail id',
      () {
        expect(
          () => unsyncedWaypointPhotoDir(
            tempRoot.path,
            accountId,
            '../escape',
            waypointKey,
          ),
          throwsArgumentError,
        );
      },
    );

    test(
      'unsyncedWaypointPhotoDir throws ArgumentError for a traversal waypoint key',
      () {
        expect(
          () => unsyncedWaypointPhotoDir(
            tempRoot.path,
            accountId,
            localId,
            '../escape',
          ),
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
            accountId,
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
      'An undirectory-able dir is reported as failedCount, not '
      'raised -- the batch contract covers the batch\'s own setup',
      () async {
        // `dir`'s parent is a FILE, so `create(recursive: true)` throws.
        // That exception used to escape `_copyPhotosForLocalSave` into
        // `_onSave`'s generic catch, and the hiker saw `error_saving_trail`
        // (the whole save failed) instead of `photo_copy_failed_toast(n)`.
        final blocker = File(p.join(tempRoot.path, 'not-a-dir'))
          ..writeAsStringSync('bytes');
        final dir = p.join(blocker.path, 'dest');
        final source = File(p.join(tempRoot.path, 'photo.jpg'))
          ..writeAsStringSync('fake-photo-bytes');

        // Awaited bare, not wrapped in expect(..., throwsA/returnsNormally):
        // reconcileLocalPhotos raises nothing at all, so any throw fails
        // this test directly with the real stack. failedCount is the only
        // channel by which its caller learns of a failure.
        final result = await reconcileLocalPhotos(
          dir: dir,
          desiredPaths: [source.path, source.path],
        );

        expect(result.paths, isEmpty);
        expect(
          result.failedCount,
          2,
          reason:
              'No destination directory means no photo could be copied, so '
              'every desired path is a failure -- an honest count is what '
              'the caller\'s toast is built on.',
        );
        expect(source.existsSync(), isTrue);
      },
    );

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

    // The keep decision uses `p.isWithin`, which normalizes; the
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
        unsyncedDir: '/docs/unsynced/acct-a/local-1-0',
        pickedPaths: ['/docs/unsynced/acct-a/local-1-0/a.jpg'],
      );

      expect(result, isEmpty);
    });

    test(
      'a path inside a waypoints/<key>/ subdirectory of unsyncedDir is '
      'excluded',
      () {
        final result = photosNotYetOnServer(
          unsyncedDir: '/docs/unsynced/acct-a/local-1-0',
          pickedPaths: [
            '/docs/unsynced/acct-a/local-1-0/waypoints/local-2-0/b.jpg',
          ],
        );

        expect(result, isEmpty);
      },
    );

    test('an image_picker-style cache path outside unsyncedDir is kept', () {
      final result = photosNotYetOnServer(
        unsyncedDir: '/docs/unsynced/acct-a/local-1-0',
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
          unsyncedDir: '/docs/unsynced/acct-a/local-1-0',
          pickedPaths: ['/docs/unsynced/acct-a/local-1-0/./a.jpg'],
        );

        expect(result, isEmpty);
      },
    );

    test('input order is preserved for the kept entries in a mixed list', () {
      final result = photosNotYetOnServer(
        unsyncedDir: '/docs/unsynced/acct-a/local-1-0',
        pickedPaths: [
          '/cache/picker/first.jpg',
          '/docs/unsynced/acct-a/local-1-0/already-uploaded.jpg',
          '/cache/picker/second.jpg',
        ],
      );

      expect(result, ['/cache/picker/first.jpg', '/cache/picker/second.jpg']);
    });

    test('an empty pickedPaths returns an empty list', () {
      final result = photosNotYetOnServer(
        unsyncedDir: '/docs/unsynced/acct-a/local-1-0',
        pickedPaths: const [],
      );

      expect(result, isEmpty);
    });
  });

  group('deleteUnsyncedPhotoDir -- cross-account isolation', () {
    setUp(() {
      PathProviderPlatform.instance = _FakePathProviderPlatform(tempRoot.path);
    });

    // Before the account segment, `unsyncedTrailPhotoDir` resolved
    // `<appDocs>/unsynced/<localId>/` with no account component at all --
    // so two accounts that happened to share the same `localId` (the
    // overlap `TrailDownloadService`'s carry-forward produces) resolved to
    // the SAME directory. Account B tapping Delete on a row that actually
    // carried account A's `localId` therefore recursively deleted account
    // A's un-uploaded photos and reported success. This test proves the
    // account segment makes that directory structurally unreachable.
    test(
      'Deleting account B\'s photo dir for a localId also held by '
      'account A leaves account A\'s identically-named directory intact',
      () async {
        const accountA = 'acctA0000000001';
        const accountB = 'acctB0000000002';
        const localId = 'local-1700000000000000-1';

        final root = unsyncedPhotoRoot(tempRoot.path);
        final dirA = p.join(root, accountA, localId);
        final dirB = p.join(root, accountB, localId);
        await Directory(dirA).create(recursive: true);
        await Directory(dirB).create(recursive: true);
        final fileA = File(p.join(dirA, 'photo.jpg'))
          ..writeAsStringSync('account-a-bytes');
        File(p.join(dirB, 'photo.jpg')).writeAsStringSync('account-b-bytes');

        await deleteUnsyncedPhotoDir(accountB, localId);

        expect(
          fileA.existsSync(),
          isTrue,
          reason:
              'account B\'s delete must never reach account A\'s directory, '
              'even though both share the same localId.',
        );
        expect(Directory(dirA).existsSync(), isTrue);
        expect(Directory(dirB).existsSync(), isFalse);
      },
    );
  });

  group('sweepOrphanedUnsyncedPhotos', () {
    setUp(() {
      PathProviderPlatform.instance = _FakePathProviderPlatform(tempRoot.path);
    });

    test(
      'deletes exactly the orphaned second-level localId directories across '
      'two account directories, leaving both account dirs and both kept '
      'dirs intact',
      () async {
        final root = unsyncedPhotoRoot(tempRoot.path);
        const accountA = 'acctA0000000001';
        const accountB = 'acctB0000000002';
        const keptA = 'local-1700000000000000-1';
        const orphanA = 'local-1700000000000000-2';
        const keptB = 'local-1700000000000000-3';
        const orphanB = 'local-1700000000000000-4';

        await Directory(p.join(root, accountA, keptA)).create(recursive: true);
        await Directory(
          p.join(root, accountA, orphanA),
        ).create(recursive: true);
        await Directory(p.join(root, accountB, keptB)).create(recursive: true);
        await Directory(
          p.join(root, accountB, orphanB),
        ).create(recursive: true);
        File(
          p.join(root, accountA, keptA, 'photo.jpg'),
        ).writeAsStringSync('bytes');

        final deleted = await sweepOrphanedUnsyncedPhotos(
          keepLocalIds: {keptA, keptB},
        );

        expect(deleted, 2);
        expect(Directory(p.join(root, accountA)).existsSync(), isTrue);
        expect(Directory(p.join(root, accountB)).existsSync(), isTrue);
        expect(Directory(p.join(root, accountA, keptA)).existsSync(), isTrue);
        expect(
          File(p.join(root, accountA, keptA, 'photo.jpg')).existsSync(),
          isTrue,
        );
        expect(Directory(p.join(root, accountB, keptB)).existsSync(), isTrue);
        expect(
          Directory(p.join(root, accountA, orphanA)).existsSync(),
          isFalse,
        );
        expect(
          Directory(p.join(root, accountB, orphanB)).existsSync(),
          isFalse,
        );
      },
    );

    test(
      'a "waypoints" directory sitting at the second level is not deleted -- '
      'its basename fails isLocalId, proving the sweep cannot reach a '
      'waypoint subtree by accident',
      () async {
        final root = unsyncedPhotoRoot(tempRoot.path);
        const accountA = 'acctA0000000001';
        const localId = 'local-1700000000000000-1';

        final waypointsDir = Directory(p.join(root, accountA, 'waypoints'));
        await waypointsDir.create(recursive: true);
        final strayFile = File(p.join(waypointsDir.path, 'local-2-0.jpg'))
          ..writeAsStringSync('bytes');

        final deleted = await sweepOrphanedUnsyncedPhotos(
          keepLocalIds: {localId},
        );

        expect(deleted, 0);
        expect(waypointsDir.existsSync(), isTrue);
        expect(strayFile.existsSync(), isTrue);
      },
    );

    test(
      'a legacy first-level unsynced/<localId>/ directory (the pre-account '
      'layout) and its contents survive the sweep untouched (no '
      'migration, no cleanup)',
      () async {
        final root = unsyncedPhotoRoot(tempRoot.path);
        const legacyLocalId = 'local-1700000000000000-9';

        final legacyDir = Directory(p.join(root, legacyLocalId));
        await legacyDir.create(recursive: true);
        final legacyFile = File(p.join(legacyDir.path, 'photo.jpg'))
          ..writeAsStringSync('legacy-bytes');

        final deleted = await sweepOrphanedUnsyncedPhotos(
          keepLocalIds: const {},
        );

        expect(deleted, 0);
        expect(legacyDir.existsSync(), isTrue);
        expect(legacyFile.existsSync(), isTrue);
      },
    );

    test(
      'An account directory left empty by the sweep is '
      'reclaimed, while one that still holds a kept localId dir survives',
      () async {
        final root = unsyncedPhotoRoot(tempRoot.path);
        const emptiedAccount = 'acctA0000000001';
        const keepingAccount = 'acctB0000000002';
        const orphan = 'local-1700000000000000-1';
        const kept = 'local-1700000000000000-2';

        await Directory(
          p.join(root, emptiedAccount, orphan),
        ).create(recursive: true);
        await Directory(
          p.join(root, keepingAccount, kept),
        ).create(recursive: true);

        final deleted = await sweepOrphanedUnsyncedPhotos(
          keepLocalIds: const {kept},
        );

        // The account directory is NOT counted -- the return value stays
        // "orphaned localId directories deleted", which is what the caller
        // logs.
        expect(deleted, 1);
        expect(
          Directory(p.join(root, emptiedAccount)).existsSync(),
          isFalse,
          reason:
              'A signed-out account whose UserEntity row is long gone '
              'otherwise leaves a permanent inode behind -- the same '
              'unbounded-growth class this sweep exists to close.',
        );
        expect(Directory(p.join(root, keepingAccount)).existsSync(), isTrue);
        expect(
          Directory(p.join(root, keepingAccount, kept)).existsSync(),
          isTrue,
        );
      },
    );

    test(
      'An account directory holding a non-localId entry is left '
      'alone -- the empty-dir reclamation never widens what is deletable',
      () async {
        final root = unsyncedPhotoRoot(tempRoot.path);
        const accountA = 'acctA0000000001';

        final waypointsDir = Directory(p.join(root, accountA, 'waypoints'));
        await waypointsDir.create(recursive: true);

        final deleted = await sweepOrphanedUnsyncedPhotos(
          keepLocalIds: const {},
        );

        expect(deleted, 0);
        expect(
          Directory(p.join(root, accountA)).existsSync(),
          isTrue,
          reason:
              'The account dir is not empty -- `waypoints/` survived the '
              'isLocalId filter, so the dir it lives in must survive too.',
        );
        expect(waypointsDir.existsSync(), isTrue);
      },
    );
  });
}
