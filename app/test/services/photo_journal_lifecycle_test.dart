import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/store/app_photo_paths.dart';

const _target = PhotoTarget.trail('trail0000000001');
const _actor = 'actor0000000001';
String _url(int number, String name) =>
    '/api/v1/files/assets/${'$number'.padLeft(15, '0')}/$name';

void main() {
  late Directory docs;
  late Dio api;
  late PhotoMutationService service;
  late PhotoApiMode mode;
  late DateTime clock;
  late List<RequestOptions> requests;
  late FutureOr<void> Function(RequestOptions, RequestInterceptorHandler)
  respond;
  var nextAsset = 1;
  List<Map<String, String>> assets(FormData form) => [
    for (final file in form.files)
      {
        'id': '${nextAsset++}'.padLeft(15, '0'),
        'collectionId': 'assets',
        'file': file.value.filename!,
      },
  ];
  void success(
    RequestOptions options,
    RequestInterceptorHandler handler,
    dynamic data,
  ) => handler.resolve(
    Response(requestOptions: options, statusCode: 200, data: data),
  );
  void failure(
    RequestOptions options,
    RequestInterceptorHandler handler,
    int status, [
    dynamic data,
  ]) => handler.reject(
    DioException(
      requestOptions: options,
      response: Response(
        requestOptions: options,
        statusCode: status,
        data: data,
        headers: data is Map && data['message'] is String
            ? Headers.fromMap({
                Headers.contentTypeHeader: ['application/json'],
              })
            : null,
      ),
    ),
  );
  Future<String> picked(String name, {int size = 1}) async {
    final file = File('${docs.path}/$name');
    final handle = await file.open(mode: FileMode.write);
    await handle.truncate(size);
    await handle.close();
    return file.path;
  }

  Future<List<File>> jobs() async => [
    await for (final entry in docs.list(recursive: true))
      if (entry is File && entry.path.endsWith('.json')) entry,
  ];
  PhotoMutationService build() => PhotoMutationService(
    api: api,
    resolveMode: () async => mode,
    now: () => clock,
    documentsDirectory: () async => docs,
    currentAccountId: () => 'account00000001',
  );

  setUp(() {
    docs = Directory.systemTemp.createTempSync('photo-journal');
    clock = DateTime.utc(2026);
    mode = PhotoApiMode.assets;
    requests = [];
    nextAsset = 1;
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    respond = (options, handler) => success(
      options,
      handler,
      options.method == 'PUT'
          ? assets(options.data as FormData)
          : {'acknowledged': true},
    );
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          requests.add(options);
          await respond(options, handler);
        },
      ),
    );
    service = build();
  });
  tearDown(() async {
    if (await docs.exists()) await docs.delete(recursive: true);
  });

  test(
    'a reopened editor preserves earlier failed picks by omission',
    () async {
      var fail = true;
      respond = (options, handler) => fail
          ? failure(options, handler, 503)
          : success(options, handler, assets(options.data as FormData));
      final first = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('first.jpg')],
      );
      expect(first.pending, isTrue);
      expect(await service.pendingPaths(_target, _actor), first.pendingPaths);
      final second = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('second.jpg')],
        replacePendingPhotos: true,
      );
      expect(await service.pendingPaths(_target, _actor), hasLength(2));
      fail = false;
      clock = second.nextAttemptAt!.add(const Duration(seconds: 1));
      await service.drain(_actor);
      final form = requests.last.data as FormData;
      expect(form.files.map((entry) => entry.value.filename), [
        'first.jpg',
        'second.jpg',
      ]);
      expect(await service.pendingPaths(_target, _actor), isEmpty);
    },
  );

  test('a file above 50 MiB is rejected between two good originals', () async {
    final result = await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [
        await picked('a.jpg'),
        await picked('large.jpg', size: 50 * 1024 * 1024 + 1),
        await picked('c.jpg'),
      ],
    );
    expect(result.pending, isFalse);
    expect(result.rejectedCount, 1);
    expect(result.madeProgress, isTrue);
    expect(
      (requests.single.data as FormData).files.map(
        (entry) => entry.value.filename,
      ),
      ['a.jpg', 'c.jpg'],
    );
    expect(result.photos, [_url(1, 'a.jpg'), _url(2, 'c.jpg')]);
  });

  test(
    'a partial rejected file does not block later files, removal or cover',
    () async {
      var first = true;
      respond = (options, handler) {
        if (options.method == 'PUT') {
          if (first) {
            first = false;
            final stored = assets(
              FormData()..files.add((options.data as FormData).files.first),
            );
            failure(options, handler, 415, {
              'detail': {
                'completedFileCount': 1,
                'uploadedAssets': stored,
                'attachedAssets': stored,
              },
            });
          } else {
            success(options, handler, assets(options.data as FormData));
          }
        } else {
          success(options, handler, {'acknowledged': true});
        }
      };
      final keep = _url(8, 'keep.jpg'), remove = _url(9, 'remove.jpg');
      final result = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [
          await picked('a.jpg'),
          await picked('bad.jpg'),
          await picked('c.jpg'),
        ],
        existingPhotos: [keep, remove],
        removedPhotos: [remove],
        setCover: true,
        coverPhoto: keep,
      );
      expect(result.pending, isFalse);
      expect(result.rejectedCount, 1);
      expect(requests.map((request) => request.method), [
        'PUT',
        'PUT',
        'DELETE',
        'PATCH',
      ]);
      expect(
        (requests[1].data as FormData).files.single.value.filename,
        'c.jpg',
      );
      expect(result.photos, [keep, _url(1, 'a.jpg'), _url(2, 'c.jpg')]);
    },
  );

  test(
    'transient uploads have persisted backoff while removals still run',
    () async {
      respond = (options, handler) => options.method == 'PUT'
          ? failure(options, handler, 503)
          : success(options, handler, {'acknowledged': true});
      final result = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('a.jpg')],
        removedPhotos: [_url(9, 'old.jpg')],
      );
      expect(result.pending, isTrue);
      expect(result.madeProgress, isTrue);
      expect(result.nextAttemptAt, clock.add(const Duration(seconds: 30)));
      expect(requests.map((request) => request.method), ['PUT', 'DELETE']);
      service = build();
      await service.drain(_actor);
      expect(requests, hasLength(2));
      clock = clock.add(const Duration(seconds: 31));
      final retry = await service.drain(_actor);
      expect(requests, hasLength(3));
      expect(retry.madeProgress, isFalse);
      final job = jsonDecode(await (await jobs()).single.readAsString()) as Map;
      expect(job['attempts'], 2);
    },
  );

  for (final status in [403, 404]) {
    test('target status $status drops its journal and copies', () async {
      respond = (options, handler) => options.uri.path.contains('/user/')
          ? success(options, handler, {'id': 'account00000001'})
          : failure(options, handler, status, {
              'message': status == 404 ? 'not_found' : 'forbidden',
            });
      final result = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('a.jpg')],
      );
      expect(result.pending, isFalse);
      expect(result.dropped, isTrue);
      expect(await jobs(), isEmpty);
      expect(await service.pendingCount(_actor), 0);
      await service.drain(_actor);
      expect(requests.map((request) => request.method), ['PUT', 'GET', 'GET']);
      expect(requests[1].uri.path, '/api/v1/trail/${_target.id}');
      expect(requests.last.uri.path, '/api/v1/user/account00000001');
    });
  }

  test(
    'legacy removals and covers resolve current asset URLs after upgrade',
    () async {
      mode = PhotoApiMode.unknown;
      await service.apply(
        target: _target,
        actorId: _actor,
        existingPhotos: ['remove.jpg', 'cover.jpg'],
        removedPhotos: ['remove.jpg'],
        setCover: true,
        coverPhoto: 'cover.jpg',
      );
      mode = PhotoApiMode.assets;
      respond = (options, handler) => success(
        options,
        handler,
        options.method == 'GET'
            ? {
                'photos': [_url(1, 'remove.jpg'), _url(2, 'cover.jpg')],
              }
            : {'acknowledged': true},
      );
      final result = await service.drain(_actor);
      expect(result.pending, isFalse);
      final deletion = requests.singleWhere(
        (request) => request.method == 'DELETE',
      );
      expect(deletion.uri.path, '/api/v1/assets/000000000000001');
      expect(deletion.queryParameters, {'trail': _target.id});
      expect(
        requests.singleWhere((request) => request.method == 'PATCH').data,
        {'asset': '000000000000002'},
      );
      final job = jsonDecode(await (await jobs()).single.readAsString()) as Map;
      expect(job['photos'], [_url(2, 'cover.jpg')]);
    },
  );

  test('a failed legacy lookup stays pending and never clears cover', () async {
    mode = PhotoApiMode.unknown;
    await service.apply(
      target: _target,
      actorId: _actor,
      removedPhotos: ['a.jpg'],
      setCover: true,
      coverPhoto: 'b.jpg',
    );
    mode = PhotoApiMode.assets;
    respond = (options, handler) => failure(options, handler, 400);
    final result = await service.drain(_actor);
    expect(result.pending, isTrue);
    expect(requests.every((request) => request.method == 'GET'), isTrue);
  });

  test(
    'legacy rejection switches sibling jobs without another legacy send',
    () async {
      mode = PhotoApiMode.unknown;
      await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('a.jpg')],
      );
      await service.apply(
        target: const PhotoTarget.trail('trail0000000002'),
        actorId: _actor,
        newPhotos: [await picked('b.jpg')],
      );
      var marked = 0;
      service = PhotoMutationService(
        api: api,
        resolveMode: () async => mode,
        now: () => clock,
        documentsDirectory: () async => docs,
        markAssetApi: () => marked++,
      );
      mode = PhotoApiMode.legacy;
      respond = (options, handler) => options.method == 'POST'
          ? failure(options, handler, 400, {
              'message': 'trail_photos_form_field_removed',
            })
          : success(options, handler, assets(options.data as FormData));
      final result = await service.drain(_actor, mode: PhotoApiMode.legacy);
      expect(result.pending, isFalse);
      expect(marked, 1);
      expect(
        requests.where((request) => request.method == 'POST'),
        hasLength(1),
      );
      expect(
        requests.where((request) => request.method == 'PUT'),
        hasLength(2),
      );
    },
  );

  test(
    'a removal-only legacy rejection resolves and retries through assets',
    () async {
      mode = PhotoApiMode.unknown;
      await service.apply(
        target: _target,
        actorId: _actor,
        removedPhotos: ['old.jpg'],
      );
      mode = PhotoApiMode.legacy;
      respond = (options, handler) {
        if (options.method == 'POST') {
          failure(options, handler, 400, {
            'message': 'trail_photos_form_field_removed',
          });
        } else {
          success(
            options,
            handler,
            options.method == 'GET'
                ? {
                    'photos': [_url(9, 'old.jpg')],
                  }
                : {'acknowledged': true},
          );
        }
      };
      final result = await service.drain(_actor);
      expect(result.pending, isFalse);
      expect(requests.map((request) => request.method), [
        'POST',
        'GET',
        'DELETE',
      ]);
    },
  );

  test(
    'missing legacy references settle without clearing an unrelated cover',
    () async {
      mode = PhotoApiMode.unknown;
      await service.apply(
        target: _target,
        actorId: _actor,
        removedPhotos: ['missing.jpg'],
        setCover: true,
        coverPhoto: 'missing-cover.jpg',
      );
      mode = PhotoApiMode.assets;
      respond = (options, handler) =>
          success(options, handler, {'photos': <String>[]});
      final result = await service.drain(_actor);
      expect(result.pending, isFalse);
      expect(requests.every((request) => request.method == 'GET'), isTrue);
    },
  );

  test(
    'an unrelated save does not wait behind another target upload',
    () async {
      final entered = Completer<void>(), release = Completer<void>();
      respond = (options, handler) async {
        if ((options.data as FormData).fields.any(
          (entry) => entry.value == _target.id,
        )) {
          entered.complete();
          await release.future;
        }
        success(options, handler, assets(options.data as FormData));
      };
      final first = service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('a.jpg')],
      );
      await entered.future;
      final other = await service
          .apply(
            target: const PhotoTarget.trail('trail0000000002'),
            actorId: _actor,
            newPhotos: [await picked('b.jpg')],
          )
          .timeout(const Duration(seconds: 2));
      expect(other.pending, isFalse);
      release.complete();
      await first;
    },
  );

  test(
    'completed jobs accept server truth and prune after acknowledgement',
    () async {
      final saved = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('a.jpg')],
        existingPhotos: [_url(9, 'old.jpg')],
      );
      expect(saved.photos, hasLength(2));
      final truth = await service.apply(
        target: _target,
        actorId: _actor,
        existingPhotos: [_url(7, 'replacement.jpg')],
      );
      expect(truth.photos, [_url(7, 'replacement.jpg')]);
      expect(truth.coverPhoto, isNull);
      await service.pruneCompleted(_target, _actor);
      expect(await jobs(), isEmpty);
    },
  );

  test('relative journal paths survive a moved documents container', () async {
    mode = PhotoApiMode.unknown;
    final first = await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [await picked('a.jpg')],
    );
    final raw = await (await jobs()).single.readAsString();
    expect(raw.contains(docs.path), isFalse);
    final moved = Directory('${docs.path}-moved');
    await docs.rename(moved.path);
    docs = moved;
    final pending = await service.pendingPaths(_target, _actor);
    expect(pending.single, startsWith(docs.path));
    expect(await File(pending.single).exists(), isTrue);
    expect(pending.single, isNot(first.pendingPaths.single));
    mode = PhotoApiMode.assets;
    await service.drain(_actor);
    expect(
      (requests.single.data as FormData).files.single.value.filename,
      'a.jpg',
    );
  });

  test('reservations recover copies and sweep orphaned file windows', () async {
    mode = PhotoApiMode.unknown;
    final sourceDir = Directory('${docs.path}/unsynced/account/local-1-1')
      ..createSync(recursive: true);
    final source = File('${sourceDir.path}/source.jpg')..writeAsBytesSync([42]);
    final first = await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [source.path],
    );
    final file = (await jobs()).single;
    final job = jsonDecode(await file.readAsString()) as Map<String, dynamic>;
    (job['files'] as List).single['copying'] = true;
    await File(first.pendingPaths.single).delete();
    await file.writeAsString(jsonEncode(job));
    expect(
      await File((await service.pendingPaths(_target, _actor)).single)
          .readAsBytes(),
      [42],
    );
    final orphan = File('${file.parent.path}/orphan.jpg')
      ..writeAsBytesSync([99]);
    await service.sweepOrphanCopies(_actor);
    expect(await orphan.exists(), isFalse);
    final ready = jsonDecode(await file.readAsString()) as Map<String, dynamic>;
    (ready['files'] as List).single['photo'] = _url(1, 'source.jpg');
    await file.writeAsString(jsonEncode(ready));
    await service.pendingPaths(_target, _actor);
    expect(await File(first.pendingPaths.single).exists(), isFalse);
  });

  test('explicit target and logout scope drops remove pending work', () async {
    mode = PhotoApiMode.unknown;
    await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [await picked('a.jpg')],
    );
    await service.drop(_target, _actor);
    expect(await service.pendingCount(_actor), 0);
    await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [await picked('b.jpg')],
    );
    await service.dropScope(_actor);
    expect(await jobs(), isEmpty);
  });

  test(
    'managed helpers rebase earlier paths and leave server references alone',
    () {
      configurePhotoDocumentsDirectory('/new/Documents');
      expect(
        resolveStoredPhotoPath('/old/Documents/unsynced/a/local-1-1/a.jpg'),
        '/new/Documents/unsynced/a/local-1-1/a.jpg',
      );
      expect(
        resolveStoredPhotoPath('/api/v1/files/assets/a/file.jpg'),
        '/api/v1/files/assets/a/file.jpg',
      );
      expect(resolveStoredPhotoPath('file.jpg'), 'file.jpg');
      expect(
        storePhotoPath('/new/Documents', '/old/Documents/library/trail/a.jpg'),
        'library/trail/a.jpg',
      );
      expect(
        () => storePhotoPath('/new/Documents', '../outside.jpg'),
        throwsArgumentError,
      );
    },
  );
  test('partial progress identifies owned originals across sign-out', () async {
    final folder = Directory('${docs.path}/unsynced/account/local-1-1')
      ..createSync(recursive: true);
    final a = File('${folder.path}/a.jpg')..writeAsBytesSync([1]);
    final b = File('${folder.path}/b.jpg')..writeAsBytesSync([2]);
    var first = true;
    respond = (options, handler) {
      if (first) {
        first = false;
        final stored = {
          'id': '000000000000001',
          'collectionId': 'assets',
          'file': 'a.jpg',
        };
        failure(options, handler, 504, {
          'detail': {
            'completedFileCount': 1,
            'uploadedAssets': [stored],
            'attachedAssets': [stored],
          },
        });
      } else {
        success(options, handler, assets(options.data as FormData));
      }
    };
    final partial = await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [a.path, b.path],
    );
    expect(partial.pending, isTrue);
    expect(partial.completedPaths, [a.path]);
    await service.dropScope(_actor);
    service = build();
    final resumed = await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [b.path],
    );
    expect(resumed.pending, isFalse);
    expect(
      (requests.last.data as FormData).files.single.value.filename,
      'b.jpg',
    );
  });

  test(
    'logout waits for a running job without resurrecting its journal',
    () async {
      final entered = Completer<void>(), release = Completer<void>();
      respond = (options, handler) async {
        entered.complete();
        await release.future;
        success(options, handler, assets(options.data as FormData));
      };
      final operation = service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [await picked('a.jpg')],
      );
      await entered.future;
      final logout = service.dropScope(_actor);
      await Future<void>.delayed(Duration.zero);
      release.complete();
      await operation;
      await logout;
      expect(await jobs(), isEmpty);
    },
  );

  test(
    'cancelling a pending cover drops its intent without clearing the server',
    () async {
      mode = PhotoApiMode.unknown;
      final source = await picked('a.jpg');
      final first = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [source],
        setCover: true,
        coverPhoto: source,
      );
      final cancelled = await service.apply(
        target: _target,
        actorId: _actor,
        removedPendingPaths: first.pendingPaths,
      );
      expect(cancelled.pending, isFalse);
      expect(cancelled.coverPhoto, isNull);
      mode = PhotoApiMode.assets;
      await service.drain(_actor);
      expect(requests, isEmpty);
    },
  );

  test(
    'a rejected pending cover never clears an existing server cover',
    () async {
      final source = await picked(
        'large.jpg',
        size: AssetPhotoClient.maxFileBytes + 1,
      );
      final rejected = await service.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [source],
        setCover: true,
        coverPhoto: source,
      );
      expect(rejected.pending, isFalse);
      expect(rejected.rejectedCount, 1);
      expect(requests, isEmpty);
    },
  );
}
