import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:wanderer/util/image_cache.dart';

void main() {
  late List<http.BaseRequest> requests;
  late CookieFileService service;

  setUp(() async {
    requests = [];
    service = CookieFileService(
      httpClient: MockClient((request) async {
        requests.add(request);
        return http.Response.bytes([0], 200);
      }),
    );

    // Mirrors the Set-Cookie `exportToCookie` sends from hooks.server.ts.
    final jar = CookieJar();
    await jar.saveFromResponse(Uri.parse('https://wanderer.example/api/v1'), [
      Cookie('pb_auth', 'token')..path = '/',
    ]);
    WandererImageCacheManager.cookieJar = jar;
  });

  tearDown(() => WandererImageCacheManager.cookieJar = null);

  test('sends the session cookie to the instance that set it', () async {
    await service.get(
      'https://wanderer.example/api/v1/files/trails/abc/photo.jpg?thumb=600x0',
    );

    expect(requests.single.headers['cookie'], 'pb_auth=token');
  });

  test('never sends the session cookie to another host', () async {
    await service.get('https://remote.example/api/v1/files/u/a/avatar.png');
    await service.get('https://api.dicebear.com/7.x/initials/png?seed=A');

    expect(requests.map((r) => r.headers['cookie']), [null, null]);
  });

  group('Secure cookies', () {
    setUp(() async {
      final jar = CookieJar();
      await jar.saveFromResponse(Uri.parse('https://wanderer.example/api/v1'), [
        Cookie('pb_auth', 'token')
          ..path = '/'
          ..secure = true,
      ]);
      WandererImageCacheManager.cookieJar = jar;
    });

    test('are sent over https', () async {
      await service.get('https://wanderer.example/api/v1/files/t/a/p.jpg');

      expect(requests.single.headers['cookie'], 'pb_auth=token');
    });

    // A federated actor chooses its own icon URL, so an image fetch can be
    // pointed at this instance's host over plain http.
    test('are never sent over http', () async {
      await service.get('http://wanderer.example/x.png');

      expect(requests.single.headers['cookie'], isNull);
    });
  });

  test('sends a non-Secure cookie to a plain http instance', () async {
    final jar = CookieJar();
    await jar.saveFromResponse(Uri.parse('http://wanderer.lan/api/v1'), [
      Cookie('pb_auth', 'token')..path = '/',
    ]);
    WandererImageCacheManager.cookieJar = jar;

    await service.get('http://wanderer.lan/api/v1/files/t/a/p.jpg');

    expect(requests.single.headers['cookie'], 'pb_auth=token');
  });

  test('sends no cookie before a jar is configured', () async {
    WandererImageCacheManager.cookieJar = null;

    await service.get('https://wanderer.example/api/v1/files/t/a/p.jpg');

    expect(requests.single.headers['cookie'], isNull);
  });
}
