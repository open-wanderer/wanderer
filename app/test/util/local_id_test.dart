import 'package:flutter_test/flutter_test.dart';
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
}
