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

  test('sends no cookie before a jar is configured', () async {
    WandererImageCacheManager.cookieJar = null;

    await service.get('https://wanderer.example/api/v1/files/t/a/p.jpg');

    expect(requests.single.headers['cookie'], isNull);
  });
}
