import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/util/recent_servers.dart';

// ---------------------------------------------------------------------------
// Pure recording logic for the "Last used" instance list: dedupe by
// normalized URL, move-to-top on re-login, cap at kMaxRecentServers, and a
// tolerant JSON codec for on-device persistence. No container, no widget
// pump — just input/output.
// ---------------------------------------------------------------------------

void main() {
  group('pushRecentServer', () {
    test('pushing into an empty list stores the normalized URL', () {
      final result = pushRecentServer(
        const [],
        const ServerInstance(url: 'wanderer.to'),
      );

      expect(result, hasLength(1));
      expect(result.single.url, 'https://wanderer.to');
    });

    test('pushing A then B yields [B, A] — most recent first', () {
      final afterA = pushRecentServer(
        const [],
        const ServerInstance(url: 'a.example'),
      );
      final afterB = pushRecentServer(
        afterA,
        const ServerInstance(url: 'b.example'),
      );

      expect(afterB.map((s) => s.url), [
        'https://b.example',
        'https://a.example',
      ]);
    });

    test('pushing A, B, C caps at kMaxRecentServers, oldest evicted', () {
      var list = const <ServerInstance>[];
      for (final url in ['a.example', 'b.example', 'c.example']) {
        list = pushRecentServer(list, ServerInstance(url: url));
      }

      expect(list.map((s) => s.url), [
        'https://c.example',
        'https://b.example',
      ]);
    });

    test('re-pushing A onto [B, A] moves it to top without duplicating', () {
      var list = pushRecentServer(
        const [],
        const ServerInstance(url: 'a.example'),
      );
      list = pushRecentServer(list, const ServerInstance(url: 'b.example'));

      final result = pushRecentServer(
        list,
        const ServerInstance(url: 'a.example'),
      );

      expect(result, hasLength(2));
      expect(result.map((s) => s.url), [
        'https://a.example',
        'https://b.example',
      ]);
    });

    test('URL variants of one server dedupe to a single entry', () {
      var list = pushRecentServer(
        const [],
        const ServerInstance(url: 'https://wanderer.to/'),
      );
      list = pushRecentServer(
        list,
        const ServerInstance(url: 'HTTPS://Wanderer.to'),
      );
      list = pushRecentServer(list, const ServerInstance(url: 'wanderer.to'));

      expect(list, hasLength(1));
    });

    test('a distinct port stays a distinct entry', () {
      var list = pushRecentServer(
        const [],
        const ServerInstance(url: 'example.com:8443'),
      );
      list = pushRecentServer(list, const ServerInstance(url: 'example.com'));

      expect(list, hasLength(2));
    });

    test('a distinct subpath stays a distinct entry', () {
      var list = pushRecentServer(
        const [],
        const ServerInstance(url: 'example.com/wanderer'),
      );
      list = pushRecentServer(list, const ServerInstance(url: 'example.com'));

      expect(list, hasLength(2));
    });

    test(
      'a nameless typed match keeps the stored metadata (name and image)',
      () {
        final existing = [
          const ServerInstance(
            name: 'Wanderer',
            url: 'https://wanderer.to',
            image: 'w.png',
          ),
        ];

        final result = pushRecentServer(
          existing,
          const ServerInstance(url: 'wanderer.to'),
        );

        expect(result, hasLength(1));
        expect(result.single.name, 'Wanderer');
        expect(result.single.image, 'w.png');
      },
    );

    test('a pushed NAMED entry replaces the stored metadata of its match', () {
      final existing = [
        const ServerInstance(
          name: 'Old Name',
          url: 'https://wanderer.to',
          image: 'old.png',
        ),
      ];

      final result = pushRecentServer(
        existing,
        const ServerInstance(
          name: 'New Name',
          url: 'wanderer.to',
          image: 'new.png',
        ),
      );

      expect(result, hasLength(1));
      expect(result.single.name, 'New Name');
      expect(result.single.image, 'new.png');
    });

    test('an unnormalizable url leaves the list unchanged', () {
      final existing = [const ServerInstance(url: 'https://wanderer.to')];

      expect(pushRecentServer(existing, const ServerInstance(url: '')), same(
        existing,
      ));
      expect(
        pushRecentServer(existing, const ServerInstance(url: '   ')),
        same(existing),
      );
    });

    test('never mutates the input list', () {
      final existing = [const ServerInstance(url: 'https://wanderer.to')];
      final originalLength = existing.length;

      pushRecentServer(existing, const ServerInstance(url: 'b.example'));

      expect(existing.length, originalLength);
    });
  });

  group('encodeRecentServers / decodeRecentServers round trip', () {
    test('round-trips name, url, image, and a null name', () {
      const servers = [
        ServerInstance(name: 'Wanderer', url: 'https://wanderer.to', image: 'w.png'),
        ServerInstance(url: 'https://self.hosted.example'),
      ];

      final decoded = decodeRecentServers(encodeRecentServers(servers));

      expect(decoded, hasLength(2));
      expect(decoded[0].name, 'Wanderer');
      expect(decoded[0].url, 'https://wanderer.to');
      expect(decoded[0].image, 'w.png');
      expect(decoded[1].name, isNull);
      expect(decoded[1].url, 'https://self.hosted.example');
    });
  });

  group('decodeRecentServers tolerance', () {
    test('empty string decodes to an empty list', () {
      expect(decodeRecentServers(''), isEmpty);
    });

    test('non-JSON input decodes to an empty list', () {
      expect(decodeRecentServers('not json'), isEmpty);
    });

    test('a JSON object (not a list) decodes to an empty list', () {
      expect(decodeRecentServers('{}'), isEmpty);
    });

    test('an element with no url is dropped while valid siblings survive', () {
      final decoded = decodeRecentServers(
        '[{"name":"No URL"},{"url":"https://wanderer.to"}]',
      );

      expect(decoded, hasLength(1));
      expect(decoded.single.url, 'https://wanderer.to');
    });

    test('an element whose url does not normalize is dropped', () {
      final decoded = decodeRecentServers(
        '[{"url":""},{"url":"https://wanderer.to"}]',
      );

      expect(decoded, hasLength(1));
      expect(decoded.single.url, 'https://wanderer.to');
    });

    test('duplicate-by-key elements keep only the first', () {
      final decoded = decodeRecentServers(
        '[{"url":"https://wanderer.to"},{"url":"HTTPS://Wanderer.to/"}]',
      );

      expect(decoded, hasLength(1));
    });

    test('more than 2 valid elements are capped at 2', () {
      final decoded = decodeRecentServers(
        '[{"url":"a.example"},{"url":"b.example"},{"url":"c.example"}]',
      );

      expect(decoded, hasLength(2));
    });

    test('never throws on garbage input', () {
      expect(() => decodeRecentServers('[1, 2, "x", null, {}]'), returnsNormally);
      expect(decodeRecentServers('[1, 2, "x", null, {}]'), isEmpty);
    });
  });

  group('removeRecentServer', () {
    const list = [
      ServerInstance(url: 'https://b.example'),
      ServerInstance(url: 'https://a.example'),
    ];

    test('removes the matching entry and keeps the rest in order', () {
      final result = removeRecentServer(
        list,
        const ServerInstance(url: 'https://b.example'),
      );

      expect(result.map((s) => s.url), ['https://a.example']);
    });

    test('matches by normalized URL, ignoring scheme and case', () {
      final result = removeRecentServer(
        list,
        const ServerInstance(url: 'A.Example'),
      );

      expect(result.map((s) => s.url), ['https://b.example']);
    });

    test('returns the same list when nothing matches', () {
      expect(
        removeRecentServer(list, const ServerInstance(url: 'c.example')),
        same(list),
      );
      expect(
        removeRecentServer(list, const ServerInstance(url: '')),
        same(list),
      );
    });
  });
}
