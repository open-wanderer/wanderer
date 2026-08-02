import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:wanderer/util/local_id.dart';

// ---------------------------------------------------------------------------
// Tests for the collision-free local identity helpers shared by trails and
// waypoints that have not yet been assigned a server id.
// ---------------------------------------------------------------------------

void main() {
  group('mintLocalId', () {
    test('two consecutive calls return different values', () {
      final first = mintLocalId();
      final second = mintLocalId();
      expect(first, isNot(equals(second)));
    });

    test('a minted id satisfies isLocalId', () {
      expect(isLocalId(mintLocalId()), isTrue);
    });
  });

  group('isLocalId', () {
    test('a realistic PocketBase-shaped id is never classified local', () {
      // PocketBase mints ids from [a-z0-9]{15} with no hyphen.
      const serverId = 'abc123xyz456789';
      expect(isLocalId(serverId), isFalse);
    });
  });

  group('localIdDirSegment', () {
    test('accepts a minted id and returns it unchanged', () {
      final id = mintLocalId();
      expect(localIdDirSegment(id), id);
    });

    test('throws ArgumentError for a traversal attempt', () {
      expect(() => localIdDirSegment('../escape'), throwsArgumentError);
    });

    test('throws ArgumentError for an embedded separator', () {
      expect(() => localIdDirSegment('local-/etc'), throwsArgumentError);
    });

    test('throws ArgumentError for the empty string', () {
      expect(() => localIdDirSegment(''), throwsArgumentError);
    });
  });

  // WR-14. These guard the ids that arrive over the network -- a federated or
  // compromised instance returning an id containing `..` used to write
  // outside `library/`, because trail_download_service.dart interpolated them
  // straight into a path.
  group('recordIdDirSegment', () {
    test('accepts a PocketBase-shaped id and returns it unchanged', () {
      const serverId = 'abc123xyz456789';
      expect(recordIdDirSegment(serverId), serverId);
    });

    test('accepts an id that is not exactly PocketBase-shaped', () {
      // Deliberately looser than ^[a-z0-9]{15}$ so an id-format change or a
      // federated peer's convention cannot silently break downloads.
      expect(recordIdDirSegment('Abc_123-XYZ'), 'Abc_123-XYZ');
    });

    for (final hostile in [
      '..',
      '../escape',
      '../../etc/passwd',
      'abc/../../etc',
      'abc/def',
      r'abc\def',
      'abc.def',
      '',
      '.',
    ]) {
      test('throws ArgumentError for ${jsonEncode(hostile)}', () {
        expect(() => recordIdDirSegment(hostile), throwsArgumentError);
      });
    }

    test(
      'a rejected id never becomes a path segment that escapes its parent',
      () {
        // The property that actually matters, stated directly.
        for (final hostile in ['..', '../escape', 'a/../../b']) {
          expect(
            () => p.join('/library', recordIdDirSegment(hostile)),
            throwsArgumentError,
          );
        }
        // ...and an accepted one cannot escape either.
        const safe = 'abc123xyz456789';
        expect(
          p.isWithin('/library', p.join('/library', recordIdDirSegment(safe))),
          isTrue,
        );
      },
    );
  });

  group('fileNameSegment', () {
    test('accepts an ordinary filename with an extension', () {
      expect(fileNameSegment('photo_1.jpg'), 'photo_1.jpg');
    });

    for (final hostile in ['', '.', '..', 'a/b.jpg', r'a\b.jpg', '../x.jpg']) {
      test('throws ArgumentError for ${jsonEncode(hostile)}', () {
        expect(() => fileNameSegment(hostile), throwsArgumentError);
      });
    }

    test(
      'rejects the bare "/" that p.basename yields for a traversal URL, which '
      'p.join would turn into an ABSOLUTE path',
      () {
        // Uri.parse resolves `..` segments away, so this URL's path collapses
        // to "/" -- and p.basename('/') is '/', not ''. Joined naively that is
        // catastrophic: p.join(dir, '/') returns '/', discarding dir entirely,
        // so the download would target the filesystem root rather than
        // library/<id>/photos. p.basename alone is not a control.
        final decoded = Uri.parse('https://x.test/photos/%2e%2e').path;
        expect(p.basename(decoded), '/');
        expect(p.join('/library/abc/photos', p.basename(decoded)), '/');

        expect(() => fileNameSegment(p.basename(decoded)), throwsArgumentError);
      },
    );

    test('a percent-encoded separator is kept as a literal filename character, '
        'not treated as a path break', () {
      // Uri.path leaves %2F encoded, so this is an odd but harmless
      // filename -- no traversal, because %2F is not a separator on disk.
      final decoded = Uri.parse('https://x.test/photos/..%2Fescape.jpg').path;
      final name = fileNameSegment(p.basename(decoded));

      expect(name, '..%2Fescape.jpg');
      expect(
        p.isWithin('/library/abc/photos', p.join('/library/abc/photos', name)),
        isTrue,
      );
    });
  });
}
