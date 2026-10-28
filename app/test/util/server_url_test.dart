import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/server_url.dart';

// ---------------------------------------------------------------------------
// The instance address, from both directions it enters the app: typed into the
// picker, and derived from a signed-in user's actor IRI. Pure input/output —
// no container, no widget pump.
// ---------------------------------------------------------------------------

void main() {
  group('normalizeServerUrl', () {
    test(
      'fills in https:// for a bare host — the picker\'s own hint shape',
      () {
        expect(normalizeServerUrl('wanderer.to'), 'https://wanderer.to');
      },
    );

    test('keeps an explicit scheme, http included', () {
      expect(normalizeServerUrl('https://wanderer.to'), 'https://wanderer.to');
      expect(normalizeServerUrl('http://192.168.1.5'), 'http://192.168.1.5');
    });

    test('keeps a port on a bare host and on an explicit scheme', () {
      expect(
        normalizeServerUrl('example.com:8443'),
        'https://example.com:8443',
      );
      expect(
        normalizeServerUrl('http://192.168.1.5:3000'),
        'http://192.168.1.5:3000',
      );
    });

    test('keeps a subpath prefix', () {
      expect(
        normalizeServerUrl('example.com/wanderer'),
        'https://example.com/wanderer',
      );
    });

    test('trims surrounding whitespace', () {
      expect(normalizeServerUrl('  wanderer.to  '), 'https://wanderer.to');
    });

    test('strips trailing slashes so the /api/v1 suffix never doubles up', () {
      expect(normalizeServerUrl('https://wanderer.to/'), 'https://wanderer.to');
      expect(
        normalizeServerUrl('https://wanderer.to///'),
        'https://wanderer.to',
      );
      expect(
        normalizeServerUrl('example.com/wanderer/'),
        'https://example.com/wanderer',
      );
    });

    test('null for empty or whitespace-only input', () {
      expect(normalizeServerUrl(''), isNull);
      expect(normalizeServerUrl('   '), isNull);
    });

    test('null for a value that still has no host — Dio would throw on it', () {
      expect(normalizeServerUrl('https://'), isNull);
    });
  });
}
