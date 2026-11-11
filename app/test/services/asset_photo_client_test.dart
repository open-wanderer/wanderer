import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/services/asset_photo_client.dart';

void main() {
  late Directory directory;
  late Dio api;
  setUp(() {
    directory = Directory.systemTemp.createTempSync('asset-client');
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
  });
  tearDown(() => directory.deleteSync(recursive: true));

  Future<String> photo(String name, int bytes) async {
    final file = File('${directory.path}/$name');
    final handle = await file.open(mode: FileMode.write);
    await handle.truncate(bytes);
    await handle.close();
    return file.path;
  }

  test('batches above the byte budget and waits for each request', () async {
    final paths = [
      for (var i = 0; i < 7; i++) await photo('photo$i.jpg', 40 * 1024 * 1024),
    ];
    final requests = <RequestOptions>[];
    final releaseFirst = Completer<void>();
    var nextAsset = 0;
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          requests.add(options);
          if (requests.length == 1) await releaseFirst.future;
          final files = (options.data as FormData).files;
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: [
                for (final file in files)
                  {
                    'id': '${nextAsset++}'.padLeft(15, '0'),
                    'collectionId': 'assets',
                    'file': file.value.filename,
                  },
              ],
            ),
          );
        },
      ),
    );
    final operation = AssetPhotoClient(api)
        .upload(paths, const PhotoTarget.trail('trail0000000001'));
    await Future<void>.delayed(const Duration(milliseconds: 30));
    expect(requests, hasLength(1));
    releaseFirst.complete();
    final photos = await operation;
    expect(photos, hasLength(7));
    expect(requests.map((r) => (r.data as FormData).files.length), [6, 1]);
    for (final request in requests) {
      expect(request.uri.path, '/api/v1/assets');
      expect(request.method, 'PUT');
      expect(
        Map<String, String>.fromEntries((request.data as FormData).fields),
        {'trail': 'trail0000000001'},
      );
      expect(
        (request.data as FormData).files.every((file) => file.key == 'files'),
        isTrue,
      );
    }
  });

  test('reports the attached prefix before a partial error escapes', () async {
    final first = await photo('first.jpg', 1);
    final second = await photo('second.jpg', 1);
    final progress = <int>[];
    final photos = <String>[];
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          final stored = {
            'id': 'asset0000000001',
            'collectionId': 'assets',
            'file': 'first.jpg',
          };
          handler.reject(
            DioException(
              requestOptions: options,
              response: Response(
                requestOptions: options,
                statusCode: 504,
                data: {
                  'detail': {
                    'completedFileCount': 1,
                    'uploadedAssets': [stored],
                    'attachedAssets': [stored],
                  },
                },
              ),
            ),
          );
        },
      ),
    );
    await expectLater(
      AssetPhotoClient(api).upload(
        [first, second],
        const PhotoTarget.waypoint('waypoint0000001'),
        onProgress: (completed, urls) async {
          progress.add(completed);
          photos.addAll(urls);
        },
      ),
      throwsA(isA<DioException>()),
    );
    expect(progress, [1]);
    expect(photos, ['/api/v1/files/assets/asset0000000001/first.jpg']);
  });

  test(
    'rejects an individual file above the route limit without sending',
    () async {
      final path = await photo('large.jpg', AssetPhotoClient.maxFileBytes + 1);
      var sent = false;
      api.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            sent = true;
            handler.resolve(
              Response(requestOptions: options, statusCode: 200, data: []),
            );
          },
        ),
      );
      await expectLater(
        AssetPhotoClient(api)
            .upload([path], const PhotoTarget.summitLog('summitlog000001')),
        throwsArgumentError,
      );
      expect(sent, isFalse);
    },
  );
  test(
    'unresolved legacy removals and covers cannot become asset requests',
    () async {
      var sent = 0;
      api.interceptors.add(
        InterceptorsWrapper(
          onRequest: (options, handler) {
            sent++;
            handler.resolve(Response(requestOptions: options, statusCode: 200));
          },
        ),
      );
      final client = AssetPhotoClient(api);
      await expectLater(
        client.remove('old.jpg', const PhotoTarget.trail('trail0000000001')),
        throwsArgumentError,
      );
      await expectLater(
        client.cover('trail0000000001', 'old.jpg'),
        throwsArgumentError,
      );
      expect(sent, 0);
    },
  );
}
