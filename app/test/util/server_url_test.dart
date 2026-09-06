import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/server_url.dart';

// ---------------------------------------------------------------------------
// The instance address, from both directions it enters the app: typed into the
// picker, and derived from a signed-in user's actor IRI. Pure input/output —
// no container, no widget pump.
// ---------------------------------------------------------------------------

void main() {
  group('normalizeServerUrl', () {
    test('fills in https:// for a bare host — the picker\'s own hint shape', () {
      expect(normalizeServerUrl('wanderer.to'), 'https://wanderer.to');
    });

    test('keeps an explicit scheme, http included', () {
      expect(normalizeServerUrl('https://wanderer.to'), 'https://wanderer.to');
      expect(normalizeServerUrl('http://192.168.1.5'), 'http://192.168.1.5');
    });

    test('keeps a port on a bare host and on an explicit scheme', () {
      expect(normalizeServerUrl('example.com:8443'), 'https://example.com:8443');
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
      expect(normalizeServerUrl('https://wanderer.to///'), 'https://wanderer.to');
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

  group('serverUrlFromActorIri', () {
    test('strips the local actor path, leaving the origin', () {
      expect(
        serverUrlFromActorIri(
          'https://demo.example/api/v1/activitypub/user/bob',
        ),
        'https://demo.example',
      );
    });

    test('RETAINS a non-default port', () {
      expect(
        serverUrlFromActorIri(
          'https://demo.example:8443/api/v1/activitypub/user/bob',
        ),
        'https://demo.example:8443',
      );
      expect(
        serverUrlFromActorIri(
          'http://192.168.1.5:3000/api/v1/activitypub/user/bob',
        ),
        'http://192.168.1.5:3000',
      );
    });

    test('drops a scheme default port, which carries no information', () {
      expect(
        serverUrlFromActorIri(
          'https://demo.example:443/api/v1/activitypub/user/bob',
        ),
        'https://demo.example',
      );
    });

    test('RETAINS a subpath prefix from the operator\'s ORIGIN', () {
      expect(
        serverUrlFromActorIri(
          'https://example.com/wanderer/api/v1/activitypub/user/bob',
        ),
        'https://example.com/wanderer',
      );
    });

    test('falls back to the origin when the actor path marker is absent', () {
      expect(
        serverUrlFromActorIri('https://remote.example:8443/users/bob'),
        'https://remote.example:8443',
      );
    });

    test('null for a relative or non-http IRI', () {
      expect(serverUrlFromActorIri('/api/v1/activitypub/user/bob'), isNull);
      expect(serverUrlFromActorIri('acct:bob@example.com'), isNull);
      expect(serverUrlFromActorIri(''), isNull);
    });
  });
}
