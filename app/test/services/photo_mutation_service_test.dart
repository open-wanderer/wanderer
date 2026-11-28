import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

void main() {
  late Directory directory;
  late Dio api;
  late PhotoMutationService service;
  late PhotoApiMode mode;
  late DateTime clock;
  late List<RequestOptions> requests;
  const target = PhotoTarget.trail('trail0000000001');
  const actor = 'actor0000000001';
  const url = '/api/v1/files/assets/asset0000000001/photo.jpg';

  setUp(() {
    directory = Directory.systemTemp.createTempSync('photo-mutations');
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    mode = PhotoApiMode.unknown;
    clock = DateTime.utc(2026);
    requests = [];
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: options.method == 'PUT'
                  ? [
                      {
                        'id': 'asset0000000001',
                        'collectionId': 'assets',
                        'file': 'photo.jpg',
                      },
                    ]
                  : {'acknowledged': true},
            ),
          );
        },
      ),
    );
    service = PhotoMutationService(
      api: api,
      resolveMode: () async => mode,
      documentsDirectory: () async => directory,
      now: () => clock,
    );
  });
  tearDown(() => directory.deleteSync(recursive: true));

  test(
    'pending model resave deduplicates copies and resolves their cover',
    () async {
      final original = File('${directory.path}/picked.jpg')
        ..writeAsBytesSync([1, 2, 3]);
      final first = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [original.path],
        replacePendingPhotos: true,
      );
      expect(first.pending, isTrue);
      expect(first.pendingPaths, hasLength(1));
      await original.delete();
      final resaved = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: first.pendingPaths,
        replacePendingPhotos: true,
        setCover: true,
        coverPhoto: first.pendingPaths.single,
      );
      expect(resaved.pendingPaths, first.pendingPaths);
      expect(requests, isEmpty);

      mode = PhotoApiMode.assets;
      await service.drain(actor);
      expect(requests.map((request) => request.method), ['PUT', 'PATCH']);
      expect((requests.first.data as FormData).files, hasLength(1));
      expect(requests.last.data, {'asset': 'asset0000000001'});
      final completed = await service.apply(
        target: target,
        actorId: actor,
        existingPhotos: [url],
      );
      expect(completed.photos, [url]);
      expect(completed.pending, isFalse);
    },
  );

  test('removing an unknown pending pick cancels its durable upload', () async {
    final original = File('${directory.path}/picked.jpg')
      ..writeAsBytesSync([1]);
    final first = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [original.path],
      replacePendingPhotos: true,
    );
    final cancelled = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: const [],
      removedPendingPaths: first.pendingPaths,
    );
    expect(cancelled.pending, isFalse);
    expect(await File(first.pendingPaths.single).exists(), isFalse);
    mode = PhotoApiMode.assets;
    await service.drain(actor);
    expect(requests, isEmpty);
  });

  test('pending removals survive stale metadata; completed jobs accept server truth', () async {
    final queued = await service.apply(
      target: target,
      actorId: actor,
      removedPhotos: [url],
      existingPhotos: [url],
    );
    expect(queued.photos, isEmpty);
    final resaved = await service.apply(
      target: target,
      actorId: actor,
      existingPhotos: [url],
    );
    expect(resaved.photos, isEmpty);
    expect(requests, isEmpty);
    mode = PhotoApiMode.assets;
    await service.drain(actor);
    expect(requests.single.method, 'DELETE');
    final completed = await service.apply(
      target: target,
      actorId: actor,
      existingPhotos: [url],
    );
    expect(completed.photos, [url]);
    expect(completed.pending, isFalse);
  });
  test(
    'cancelling one pending basename cannot overwrite another copy',
    () async {
      final sources = <String>[];
      for (var i = 1; i <= 3; i++) {
        final folder = Directory('${directory.path}/$i')..createSync();
        final source = File('${folder.path}/photo.jpg')..writeAsBytesSync([i]);
        sources.add(source.path);
      }
      final first = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: sources.take(2).toList(),
        replacePendingPhotos: true,
      );
      final replaced = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [first.pendingPaths[1], sources[2]],
        removedPendingPaths: [first.pendingPaths[0]],
      );
      expect(replaced.pendingPaths, hasLength(2));
      expect(await File(replaced.pendingPaths[0]).readAsBytes(), [2]);
      expect(await File(replaced.pendingPaths[1]).readAsBytes(), [3]);
    },
  );

  test('legacy cover retries reuse the completed upload receipt', () async {
    final source = File('${directory.path}/picked.jpg')..writeAsBytesSync([1]);
    final thumbnails = <int>[];
    api.interceptors.clear();
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requests.add(options);
          if (options.data is FormData) {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'photos': ['old.jpg', 'new.jpg'],
                },
              ),
            );
          } else {
            thumbnails.add((options.data as Map)['thumbnail'] as int);
            if (thumbnails.length == 1) {
              handler.reject(
                DioException(
                  requestOptions: options,
                  response: Response(requestOptions: options, statusCode: 500),
                ),
              );
            } else {
              handler.resolve(
                Response(
                  requestOptions: options,
                  statusCode: 200,
                  data: {'acknowledged': true},
                ),
              );
            }
          }
        },
      ),
    );
    await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [source.path],
      existingPhotos: ['old.jpg'],
      setCover: true,
      coverPhoto: source.path,
    );
    mode = PhotoApiMode.legacy;
    await service.drain(actor);
    clock = clock.add(const Duration(seconds: 31));
    await service.drain(actor);
    expect(requests.where((request) => request.data is FormData), hasLength(1));
    expect(thumbnails, [1, 1]);
  });
}
