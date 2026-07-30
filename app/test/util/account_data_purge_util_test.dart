import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:wanderer/util/account_data_purge_util.dart';

// ---------------------------------------------------------------------------
// Tests for account_data_purge_util's pure switch predicate and the on-disk
// purge, which must delete ONLY the account-scoped directories (`library`,
// `avatars`) and leave `objectbox`, `regions`, `map_cache` and `.cookies`
// fully intact (T-h2p-04).
// ---------------------------------------------------------------------------

void main() {
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
      'deletes library and avatars including nested content, leaves everything else intact',
      () async {
        seedRoot(tempDir);

        await purgeAccountScopedDirectories(tempDir.path);

        expect(Directory(p.join(tempDir.path, 'library')).existsSync(), isFalse);
        expect(Directory(p.join(tempDir.path, 'avatars')).existsSync(), isFalse);

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
      await expectLater(
        purgeAccountScopedDirectories(tempDir.path),
        completes,
      );
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
