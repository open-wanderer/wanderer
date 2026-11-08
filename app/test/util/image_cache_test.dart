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
    await jar.saveFromResponse(Uri.parse('https://a.example/api/v1'), [
      Cookie('pb_auth', 'token')..path = '/',
    ]);
    WandererImageCacheManager.cookieJar = jar;
    WandererImageCacheManager.serverHost = 'a.example';
  });

  tearDown(() {
    WandererImageCacheManager.cookieJar = null;
    WandererImageCacheManager.serverHost = null;
  });

  test('sends the session cookie to the instance that set it', () async {
    await service.get(
      'https://a.example/api/v1/files/trails/abc/photo.jpg?thumb=600x0',
    );

    expect(requests.single.headers['cookie'], 'pb_auth=token');
  });

  test('never sends the session cookie to another host', () async {
    await service.get('https://b.example/api/v1/files/u/a/avatar.png');
    await service.get('https://api.dicebear.com/7.x/initials/png?seed=A');

    expect(requests.map((r) => r.headers['cookie']), [null, null]);
  });

  test('limits a domain cookie to the configured host', () async {
    final jar = CookieJar();
    await jar.saveFromResponse(Uri.parse('https://a.example/'), [
      Cookie('pb_auth', 'token')
        ..domain = '.a.example'
        ..path = '/',
    ]);
    WandererImageCacheManager.cookieJar = jar;
    await service.get('https://sub.a.example/photo.jpg');
    expect(requests.single.headers['cookie'], isNull);
  });

  test('checks the host again after an image redirect', () async {
    final jar = CookieJar();
    await jar.saveFromResponse(Uri.parse('https://a.example/'), [
      Cookie('pb_auth', 'token')
        ..domain = '.a.example'
        ..path = '/',
    ]);
    WandererImageCacheManager.cookieJar = jar;
    service = CookieFileService(
      httpClient: MockClient((request) async {
        requests.add(request);
        if (requests.length == 1) {
          return http.Response(
            '',
            302,
            headers: {'location': 'https://sub.a.example/photo.jpg'},
          );
        }
        return http.Response.bytes([0], 200);
      }),
    );
    await service.get(
      'https://a.example/photo.jpg',
      headers: {'Cookie': 'stale'},
    );
    expect(requests.map((request) => request.headers['cookie']), [
      'pb_auth=token',
      null,
    ]);
    expect(requests.every((request) => !request.followRedirects), isTrue);
  });

  test('retains authentication across same-host image redirects', () async {
    service = CookieFileService(
      httpClient: MockClient((request) async {
        requests.add(request);
        return requests.length == 1
            ? http.Response('', 307, headers: {'location': '/photo.jpg'})
            : http.Response.bytes([0], 200);
      }),
    );
    await service.get('https://a.example/api/v1/assets/id/file');
    expect(requests.map((request) => request.headers['cookie']), [
      'pb_auth=token',
      'pb_auth=token',
    ]);
    expect(requests.last.url.toString(), 'https://a.example/photo.jpg');
  });

  group('Secure cookies', () {
    setUp(() async {
      final jar = CookieJar();
      await jar.saveFromResponse(Uri.parse('https://a.example/api/v1'), [
        Cookie('pb_auth', 'token')
          ..path = '/'
          ..secure = true,
      ]);
      WandererImageCacheManager.cookieJar = jar;
      WandererImageCacheManager.serverHost = 'a.example';
    });

    test('are sent over https', () async {
      await service.get('https://a.example/api/v1/files/t/a/p.jpg');

      expect(requests.single.headers['cookie'], 'pb_auth=token');
    });

    // A federated actor chooses its own icon URL, so an image fetch can be
    // pointed at this instance's host over plain http.
    test('are never sent over http', () async {
      await service.get('http://a.example/x.png');

      expect(requests.single.headers['cookie'], isNull);
    });
  });

  test('sends a non-Secure cookie to a plain http instance', () async {
    final jar = CookieJar();
    await jar.saveFromResponse(Uri.parse('http://a.example/api/v1'), [
      Cookie('pb_auth', 'token')..path = '/',
    ]);
    WandererImageCacheManager.cookieJar = jar;
    WandererImageCacheManager.serverHost = 'a.example';

    await service.get('http://a.example/api/v1/files/t/a/p.jpg');

    expect(requests.single.headers['cookie'], 'pb_auth=token');
  });

  test('sends no cookie before a jar is configured', () async {
    WandererImageCacheManager.cookieJar = null;

    await service.get('https://a.example/api/v1/files/t/a/p.jpg');

    expect(requests.single.headers['cookie'], isNull);
  });
}
