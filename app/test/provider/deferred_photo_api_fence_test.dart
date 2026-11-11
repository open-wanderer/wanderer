import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/deferred_photo_session_provider.dart';

class _Documents extends PathProviderPlatform {
  _Documents(this.path);
  final String path;
  @override
  Future<String?> getApplicationDocumentsPath() async => path;
}

class _CookieStorage implements Storage {
  final _values = <String, String>{};
  @override
  Future<void> init(bool persistSession, bool ignoreExpires) async {}
  @override
  Future<String?> read(String key) async => _values[key];
  @override
  Future<void> write(String key, String value) async => _values[key] = value;
  @override
  Future<void> delete(String key) async => _values.remove(key);
  @override
  Future<void> deleteAll(List<String> keys) async {
    for (final key in keys) {
      _values.remove(key);
    }
  }
}

class _Server implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    return ResponseBody.fromString(
      '{}',
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('an upload waiting for its file cannot send with a fresh token after the account expires', () async {
    final directory = Directory.systemTemp.createTempSync('photo-api-fence-');
    final previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Documents(directory.path);
    addTearDown(() async {
      PathProviderPlatform.instance = previousPaths;
      await directory.delete(recursive: true);
    });
    ProviderContainer open() {
      final container = ProviderContainer(
        overrides: [
          cookieJarProvider.overrideWithValue(
            PersistCookieJar(storage: _CookieStorage()),
          ),
        ],
      );
      addTearDown(container.dispose);
      return container;
    }

    final container = open();
    final api = container.read(apiProvider);
    final notifier = container.read(apiProvider.notifier);
    notifier.updateBaseUrl('https://a.example');
    final server = _Server();
    api.httpClientAdapter = server;
    final deferred = container.read(deferredPhotoSessionProvider);
    final file = File('${directory.path}/picked.jpg')
      ..writeAsBytesSync([1, 2, 3]);
    final contents = Completer<List<int>>();
    final upload = () async {
      final bytes = await contents.future;
      await api.post(
        '/assets',
        data: FormData.fromMap({
          'trail': 'trail0000000001',
          'files': MultipartFile.fromBytes(bytes, filename: 'picked.jpg'),
        }),
      );
    }();
    final cancelled = isA<DioException>().having(
      (error) => error.type,
      'type',
      DioExceptionType.cancel,
    );
    final stopped = expectLater(upload, throwsA(cancelled));
    await deferred.mark(
      UserEntity(
        id: 'account00000001',
        collectionId: 'users',
        collectionName: 'users',
        actorId: 'actor0000000001',
        username: 'hiker',
        preferredUsername: 'hiker',
        email: 'hiker@a.example',
        iri: 'https://a.example/hiker',
        serverUrl: 'https://a.example',
        created: DateTime(2026),
        updated: DateTime(2026),
      ),
    );
    notifier.cancelPendingRequests();
    contents.complete(await file.readAsBytes());
    await stopped;
    expect(server.requests, isEmpty);

    for (final path in [
      '/assets/asset0000000001',
      '/trail',
      '/trail/form/trail0000000001',
      '/trail/trail0000000001/thumbnail',
      '/waypoint',
      '/waypoint/point0000000001/file',
      '/summit-log/log00000000001',
      '/tag',
    ]) {
      await expectLater(api.post(path, data: {}), throwsA(cancelled));
    }
    expect(server.requests, isEmpty);

    // Account restoration fences a replacement client before its first write.
    final restored = open();
    final restoredApi = restored.read(apiProvider);
    final restoredDeferred = restored.read(deferredPhotoSessionProvider);
    restoredDeferred.selectAccount('account00000001');
    await restoredDeferred.restore();
    restored.read(apiProvider.notifier).updateBaseUrl('https://a.example');
    final restoredServer = _Server();
    restoredApi.httpClientAdapter = restoredServer;
    await expectLater(
      restoredApi.post('/waypoint', data: {}),
      throwsA(cancelled),
    );
    expect(restoredServer.requests, isEmpty);

    await api.post('/auth/login', data: {});
    await api.post('/route', data: {});
    await api.get('/trail/trail0000000001');
    await api.post('https://b.example/api/v1/assets', data: {});
    expect(server.requests, hasLength(4));
    expect(server.requests.last.uri.host, 'b.example');

    deferred.selectAccount('account00000002');
    await api.post('/assets', data: {});
    expect(server.requests, hasLength(5));
    deferred.selectAccount('account00000001');
    await expectLater(api.post('/assets', data: {}), throwsA(cancelled));
    expect(server.requests, hasLength(5));
    await deferred.clear();
    await api.post('/assets', data: {});
    expect(server.requests, hasLength(6));
    expect(server.requests.last.uri.path, '/api/v1/assets');
  });
}
