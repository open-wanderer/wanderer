import 'dart:async';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

const target = PhotoTarget.trail('trail0000000001');
const actor = 'actor0000000001';

void main() {
  late Directory documents;
  late Dio api;
  late PhotoMutationService service;
  late DateTime clock;
  late PhotoApiMode mode;
  late List<RequestOptions> requests;
  late FutureOr<void> Function(RequestOptions, RequestInterceptorHandler)
  respond;
  var nextAsset = 0;

  void success(
    RequestOptions request,
    RequestInterceptorHandler handler,
    dynamic body,
  ) {
    handler.resolve(
      Response(requestOptions: request, statusCode: 200, data: body),
    );
  }

  void failure(
    RequestOptions request,
    RequestInterceptorHandler handler,
    int status,
    dynamic body, {
    bool json = false,
  }) {
    handler.reject(
      DioException(
        requestOptions: request,
        response: Response(
          requestOptions: request,
          statusCode: status,
          data: body,
          headers: Headers.fromMap({
            Headers.contentTypeHeader: [
              json ? 'application/json' : 'text/plain',
            ],
          }),
        ),
      ),
    );
  }

  List<Map<String, String>> uploaded(RequestOptions request) => [
    for (final file in (request.data as FormData).files)
      {
        'id': '${nextAsset++}'.padLeft(15, '0'),
        'collectionId': 'assets',
        'file': file.value.filename!,
      },
  ];

  Future<String> pick(String name, {int size = 1}) async {
    final file = File('${documents.path}/$name');
    final handle = await file.open(mode: FileMode.write);
    await handle.truncate(size);
    await handle.close();
    return file.path;
  }

  setUp(() {
    documents = Directory.systemTemp.createTempSync('journal-recovery-');
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    mode = PhotoApiMode.assets;
    clock = DateTime.utc(2026);
    requests = [];
    nextAsset = 0;
    respond = (request, handler) =>
        success(request, handler, uploaded(request));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) async {
          requests.add(request);
          await respond(request, handler);
        },
      ),
    );
    service = PhotoMutationService(
      api: api,
      resolveMode: () async => mode,
      documentsDirectory: () async => documents,
      now: () => clock,
      currentAccountId: () => 'account00000001',
    );
  });
  tearDown(() => documents.deleteSync(recursive: true));

  for (final status in [403, 404]) {
    test('a proxy $status preserves picks until the server recovers', () async {
      var unavailable = true;
      respond = (request, handler) => unavailable
          ? failure(request, handler, status, 'proxy is restarting')
          : success(request, handler, uploaded(request));
      final first = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [await pick('first.jpg'), await pick('second.jpg')],
      );
      expect(first.pending, isTrue);
      expect(first.dropped, isFalse);
      expect(first.completedPaths, isEmpty);
      expect(
        first.pendingPaths.every((path) => File(path).existsSync()),
        isTrue,
      );
      expect(first.nextAttemptAt, clock.add(const Duration(seconds: 30)));
      unavailable = false;
      clock = clock.add(const Duration(seconds: 31));
      final recovered = await service.drain(actor);
      expect(recovered.pending, isFalse);
      expect(
        requests.where((request) => request.method == 'PUT'),
        hasLength(2),
      );
      expect(recovered.rejectedCount, 0);
    });
  }

  for (final status in [403, 404]) {
    test('an API $status confirms its target before dropping files', () async {
      respond = (request, handler) => request.uri.path.contains('/user/')
          ? success(request, handler, {'id': 'account00000001'})
          : failure(request, handler, status, {
              'message': status == 404 ? 'not_found' : 'forbidden',
            }, json: true);
      final result = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [await pick('first.jpg'), await pick('second.jpg')],
      );
      expect(requests.map((request) => request.method), ['PUT', 'GET', 'GET']);
      expect(requests[1].uri.path, '/api/v1/trail/${target.id}');
      expect(requests.last.uri.path, '/api/v1/user/account00000001');
      expect(result.pending, isFalse);
      expect(result.dropped, isTrue);
      expect(result.confirmedDeleted, status == 404);
      expect(result.completedPaths, isEmpty);
      expect(result.rejectedCount, 2);
    });
  }

  for (final proof in [401, 403, 404, 503, 'other-account', 'proxy']) {
    test(
      'a $proof session proof preserves a supposedly deleted target',
      () async {
        respond = (request, handler) {
          if (request.uri.path.contains('/user/')) {
            if (proof is int) {
              failure(request, handler, proof, {
                'message': 'Unavailable',
              }, json: true);
            } else if (proof == 'proxy') {
              failure(request, handler, 404, '<html>Not found</html>');
            } else {
              success(request, handler, {'id': 'account00000002'});
            }
          } else {
            failure(request, handler, 404, {
              'message': 'Not found',
            }, json: true);
          }
        };
        final saved = await service.apply(
          target: target,
          actorId: actor,
          newPhotos: [await pick('only-copy.jpg')],
        );
        expect(saved.pending, isTrue);
        expect(saved.dropped, isFalse);
        expect(saved.confirmedDeleted, isFalse);
        expect(saved.rejectedCount, 0);
        expect(File(saved.pendingPaths.single).existsSync(), isTrue);
      },
    );
  }

  test(
    'a changed session cannot confirm deletion with an old account response',
    () async {
      var epoch = 0;
      service = PhotoMutationService(
        api: api,
        resolveMode: () async => mode,
        documentsDirectory: () async => documents,
        currentAccountId: () => 'account00000001',
        sessionEpoch: () => epoch,
      );
      respond = (request, handler) {
        if (request.uri.path.contains('/user/')) {
          epoch++;
          success(request, handler, {'id': 'account00000001'});
        } else {
          failure(request, handler, 404, {'message': 'Not found'}, json: true);
        }
      };
      final saved = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [await pick('only-copy.jpg')],
      );
      expect(saved.pending, isTrue);
      expect(saved.confirmedDeleted, isFalse);
      expect(File(saved.pendingPaths.single).existsSync(), isTrue);
    },
  );

  test('an asset API error cannot drop a target that still exists', () async {
    respond = (request, handler) => request.method == 'GET'
        ? success(request, handler, {'id': target.id, 'photos': []})
        : failure(request, handler, 404, {'message': 'not_found'}, json: true);
    final result = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [await pick('first.jpg')],
    );
    expect(result.pending, isTrue);
    expect(result.dropped, isFalse);
    expect(result.rejectedCount, 0);
    expect(File(result.pendingPaths.single).existsSync(), isTrue);
  });

  test(
    'a proxy batch limit uploads every good photo without rejecting one',
    () async {
      final sent = <String>[];
      respond = (request, handler) {
        final files = (request.data as FormData).files;
        if (files.length > 2) {
          failure(request, handler, 413, 'request body too large');
        } else {
          sent.addAll(files.map((file) => file.value.filename!));
          success(request, handler, uploaded(request));
        }
      };
      final result = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [for (var i = 0; i < 6; i++) await pick('photo$i.jpg')],
      );
      expect(result.pending, isFalse);
      expect(result.rejectedCount, 0);
      expect(sent, [for (var i = 0; i < 6; i++) 'photo$i.jpg']);
      expect(result.photos, hasLength(6));
    },
  );

  test('an empty record-save 400 keeps each file for a later save', () async {
    var unavailable = true;
    final sent = <String>[];
    respond = (request, handler) {
      if (unavailable) {
        failure(request, handler, 400, {
          'message': 'Failed to create record.',
          'data': <String, dynamic>{},
        }, json: true);
      } else {
        sent.addAll(
          (request.data as FormData).files.map((file) => file.value.filename!),
        );
        success(request, handler, uploaded(request));
      }
    };
    final first = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [await pick('first.jpg'), await pick('second.jpg')],
    );
    expect(first.pending, isTrue);
    expect(first.pendingPaths, hasLength(2));
    expect(first.rejectedCount, 0);
    unavailable = false;
    final saved = await service.apply(target: target, actorId: actor);
    expect(saved.pending, isFalse);
    expect(saved.rejectedCount, 0);
    expect(sent, ['first.jpg', 'second.jpg']);
  });

  test('legacy validation isolates an oversized photo without rejecting good files', () async {
    mode = PhotoApiMode.legacy;
    final sent = <String>[];
    respond = (request, handler) {
      final files = (request.data as FormData).files;
      if (files.any((file) => file.value.length > 20 * 1024 * 1024)) {
        failure(request, handler, 400, {
          'message': 'Failed to create record.',
          'data': {
            'photos': {
              'code': 'validation_file_size_limit',
              'message': 'File exceeds 20 MiB.',
            },
          },
        }, json: true);
      } else {
        sent.addAll(files.map((file) => file.value.filename!));
        success(request, handler, {'photos': List.of(sent)});
      }
    };
    final result = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [
        await pick('first.jpg'),
        await pick('large.jpg', size: 25 * 1024 * 1024),
        await pick('last.jpg'),
      ],
    );
    expect(result.pending, isFalse);
    expect(result.rejectedCount, 1);
    expect(result.photos, ['first.jpg', 'last.jpg']);
    expect(sent, ['first.jpg', 'last.jpg']);
  });

  test(
    'an interactive retry ignores old backoff and reports only new rejections',
    () async {
      var unavailable = true;
      respond = (request, handler) => unavailable
          ? failure(request, handler, 503, 'temporarily unavailable')
          : success(request, handler, uploaded(request));
      final first = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [
          await pick('good.jpg'),
          await pick('too-large.jpg', size: 50 * 1024 * 1024 + 1),
        ],
      );
      expect(first.rejectedCount, 1);
      expect(first.pending, isTrue);
      final second = await service.apply(target: target, actorId: actor);
      expect(second.rejectedCount, 0);
      expect(second.pending, isTrue);
      expect(requests, hasLength(2));
      unavailable = false;
      final third = await service.apply(target: target, actorId: actor);
      expect(third.pending, isFalse);
      expect(third.rejectedCount, 0);
      expect(third.photos, hasLength(1));
    },
  );
}
