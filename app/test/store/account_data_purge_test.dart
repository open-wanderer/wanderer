import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:wanderer/store/account_data_purge.dart';

// ---------------------------------------------------------------------------
// Tests for account_data_purge_util's pure switch predicate and the on-disk
// purge, which must delete ONLY the `avatars` cache and leave `library`,
// `objectbox`, `regions`, `map_cache` and `.cookies` fully intact.
//
// `library` is the load-bearing one: it used to be purged, which meant
// signing out of an account and back in destroyed every trail it had
// downloaded. Downloaded trails now survive and are hidden from other
// accounts by `TrailEntity.savedByUserIds` membership instead.
// ---------------------------------------------------------------------------

void main() {
  group('accountScopedDirNames', () {
    test('never includes library — an account switch must not destroy it', () {
      // Structural guard against re-adding it: `library` holds downloaded
      // trails, and purging it meant logging out of account A and back in
      // wiped every trail A had downloaded.
      expect(accountScopedDirNames, isNot(contains('library')));
    });

    test('leaves shared device data alone', () {
      expect(accountScopedDirNames, isNot(contains('regions')));
      expect(accountScopedDirNames, isNot(contains('map_cache')));
      expect(accountScopedDirNames, isNot(contains('objectbox')));
    });
  });

  group('shouldPurgeForIncomingUser', () {
    test('null cached id (fresh install / first login) is false', () {
      expect(shouldPurgeForIncomingUser(null, 'b'), isFalse);
    });

    test('same account refreshing its own session is false', () {
      expect(shouldPurgeForIncomingUser('a', 'a'), isFalse);
    });

    test('different account (a switch) is true', () {
      expect(shouldPurgeForIncomingUser('a', 'b'), isTrue);
    });
  });

  group('purgeAccountScopedDirectories', () {
    late Directory tempDir;

    setUp(() {
      tempDir = Directory.systemTemp.createTempSync('account_purge_test');
    });

    tearDown(() {
      if (tempDir.existsSync()) tempDir.deleteSync(recursive: true);
    });

    void seedRoot(Directory root) {
      File(
        p.join(root.path, 'library', 't1', 'photos', 'a.jpg'),
      ).createSync(recursive: true);
      File(
        p.join(root.path, 'library', 't1', 'waypoints', 'w1', 'b.jpg'),
      ).createSync(recursive: true);
      File(
        p.join(root.path, 'avatars', 'u1_avatar.png'),
      ).createSync(recursive: true);
      File(
        p.join(root.path, 'regions', 'de-nrw', 'vector.pmtiles'),
      ).createSync(recursive: true);
      File(
        p.join(root.path, 'objectbox', 'data.mdb'),
      ).createSync(recursive: true);
      File(
        p.join(root.path, 'map_cache', 'glyphs', 'x.pbf'),
      ).createSync(recursive: true);
    }

    test(
      'deletes the avatars cache and leaves everything else intact',
      () async {
        seedRoot(tempDir);

        await purgeAccountScopedDirectories(tempDir.path);

        expect(
          Directory(p.join(tempDir.path, 'avatars')).existsSync(),
          isFalse,
        );

        // The regression this pins: an account switch must NOT destroy the
        // offline library. Downloaded GPX and photos survive; other accounts
        // simply cannot see them (savedByUserIds membership).
        expect(
          File(
            p.join(tempDir.path, 'library', 't1', 'photos', 'a.jpg'),
          ).existsSync(),
          isTrue,
        );
        expect(
          File(
            p.join(tempDir.path, 'library', 't1', 'waypoints', 'w1', 'b.jpg'),
          ).existsSync(),
          isTrue,
        );

        expect(
          File(
            p.join(tempDir.path, 'regions', 'de-nrw', 'vector.pmtiles'),
          ).existsSync(),
          isTrue,
        );
        expect(
          File(p.join(tempDir.path, 'objectbox', 'data.mdb')).existsSync(),
          isTrue,
        );
        expect(
          File(
            p.join(tempDir.path, 'map_cache', 'glyphs', 'x.pbf'),
          ).existsSync(),
          isTrue,
        );
      },
    );

    test('a root with none of those dirs completes without throwing', () async {
      await expectLater(purgeAccountScopedDirectories(tempDir.path), completes);
    });

    test(
      'a root containing an unwritable/absent path never throws (best-effort)',
      () async {
        final missingRoot = p.join(tempDir.path, 'does-not-exist');
        await expectLater(
          purgeAccountScopedDirectories(missingRoot),
          completes,
        );
      },
    );
  });
}
