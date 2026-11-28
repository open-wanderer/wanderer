import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

const target = PhotoTarget.trail('trail0000000001');
const actor = 'actor0000000001';
const photoUrl = '/api/v1/files/assets/asset0000000001/picked.jpg';

void main() {
  late Directory documents;
  late File source;
  late Dio api;
  late PhotoApiMode mode;
  late int uploads;
  late List<RequestOptions> requests;

  PhotoMutationService build({
    void Function(PhotoTarget, String, String, List<String>)? cancelled,
  }) => PhotoMutationService(
    api: api,
    resolveMode: () async => mode,
    documentsDirectory: () async => documents,
    onCancelledSources: cancelled,
  );
  Future<List<File>> jobs() async => [
    await for (final file in documents.list(recursive: true))
      if (file is File && file.path.endsWith('.json')) file,
  ];

  setUp(() {
    documents = Directory.systemTemp.createTempSync('queued-photo-receipts-');
    source = File('${documents.path}/unsynced/account/local-1-1/picked.jpg');
    source.parent.createSync(recursive: true);
    source.writeAsBytesSync([1]);
    mode = PhotoApiMode.assets;
    uploads = 0;
    requests = [];
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) {
          requests.add(request);
          if (request.method == 'PUT') uploads++;
          handler.resolve(
            Response(
              requestOptions: request,
              statusCode: 200,
              data: [
                {
                  'id': 'asset0000000001',
                  'collectionId': 'assets',
                  'file': 'picked.jpg',
                },
              ],
            ),
          );
        },
      ),
    );
  });
  tearDown(() => documents.deleteSync(recursive: true));

  test(
    'queued receipts survive an editor acknowledgement and confirmed sign-out',
    () async {
      final service = build();
      final completed = await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
      );
      expect(completed.completedPaths, [source.path]);
      await service.acknowledge(target, actor);
      expect(await jobs(), hasLength(1));
      final renamed = await service.apply(
        target: target,
        actorId: actor,
        existingPhotos: const [],
        newPhotos: [source.path],
      );
      expect(renamed.photos, [photoUrl]);
      await service.dropScope(actor);
      expect(await jobs(), hasLength(1));
      final signedIn = await build().apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
        existingPhotos: const [],
      );
      expect(signedIn.photos, [photoUrl]);
      expect(signedIn.completedPaths, [source.path]);
      expect(uploads, 1);
      await service.pruneCompleted(target, actor);
      expect(await jobs(), isEmpty);
    },
  );

  test(
    'scope close rejects a queued cancellation without replacing a receipt',
    () async {
      final entered = Completer<void>(), release = Completer<void>();
      api.interceptors.insert(
        0,
        InterceptorsWrapper(
          onRequest: (request, handler) async {
            if (!entered.isCompleted) {
              entered.complete();
              await release.future;
            }
            handler.next(request);
          },
        ),
      );
      final service = build();
      final running = service.apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
      );
      await entered.future;
      final stale = build().apply(
        target: target,
        actorId: actor,
        removedPendingPaths: [source.path],
      );
      final staleCheck = expectLater(stale, throwsStateError);
      await Future<void>.delayed(Duration.zero);
      final cleanup = build().dropScope(actor);
      await Future<void>.delayed(Duration.zero);
      release.complete();
      await running;
      await staleCheck;
      await cleanup;
      final retained =
          jsonDecode(await (await jobs()).single.readAsString()) as Map;
      expect(retained['photos'], [photoUrl]);
      expect((retained['files'] as List).single['photo'], photoUrl);
      final resumed = await build().apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
      );
      expect(resumed.photos, [photoUrl]);
      expect(uploads, 1);
    },
  );

  test(
    'an orphan queue job drains and prunes after its capture row disappears',
    () async {
      mode = PhotoApiMode.unknown;
      await build().apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
      );
      mode = PhotoApiMode.assets;
      final service = PhotoMutationService(
        api: api,
        resolveMode: () async => mode,
        documentsDirectory: () async => documents,
        hasQueuedTarget: (checkedTarget, checkedActor, base) {
          expect(checkedTarget.id, target.id);
          expect(checkedActor, actor);
          expect(base, api.options.baseUrl);
          return false;
        },
      );
      final resumed = await service.drain(actor);
      expect(resumed.pending, isFalse);
      expect(uploads, 1);
      expect(await jobs(), isEmpty);
    },
  );

  test('an older unmarked queue receipt survives scope cleanup', () async {
    final service = build();
    await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [source.path],
    );
    final file = (await jobs()).single;
    final job = jsonDecode(await file.readAsString()) as Map<String, dynamic>;
    job.remove('queueOwned');
    await file.writeAsString(jsonEncode(job));
    await service.dropScope(actor);
    expect(await jobs(), hasLength(1));
    final signedIn = await build().apply(
      target: target,
      actorId: actor,
      newPhotos: [source.path],
    );
    expect(signedIn.completedPaths, [source.path]);
    expect(uploads, 1);
  });

  test('journal drain leaves queued sources for the row-owned drain', () async {
    final service = build();
    mode = PhotoApiMode.unknown;
    await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [source.path],
    );
    mode = PhotoApiMode.assets;
    await service.drain(actor);
    expect(uploads, 0);
    expect(await service.pendingPaths(target, actor), hasLength(1));
    final completed = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [source.path],
      retryImmediately: false,
    );
    expect(completed.pending, isFalse);
    expect(uploads, 1);
  });

  test(
    'a cancelled row source remains a receipt if row persistence fails',
    () async {
      mode = PhotoApiMode.unknown;
      final service = build(
        cancelled: (cancelledTarget, cancelledActor, base, paths) {
          expect(cancelledTarget.id, target.id);
          expect(cancelledActor, actor);
          expect(base, api.options.baseUrl);
          expect(paths, [source.path]);
          throw StateError('Database temporarily unavailable');
        },
      );
      await service.apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
      );
      expect(
        await service.pendingPaths(
          target,
          actor,
          excludeSources: [source.path],
        ),
        isEmpty,
      );
      await expectLater(
        service.apply(
          target: target,
          actorId: actor,
          removedPendingPaths: [source.path],
        ),
        throwsStateError,
      );
      mode = PhotoApiMode.assets;
      final recovered = await build().apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
      );
      expect(recovered.pending, isFalse);
      expect(recovered.completedPaths, [source.path]);
      expect(recovered.rejectedCount, 0);
      expect(uploads, 0);
      await build().pruneCompleted(target, actor);
      expect(await jobs(), isEmpty);
    },
  );

  test(
    'a removed row source is durable before its first journal entry exists',
    () async {
      expect(await build().hasJob(target, actor), isFalse);
      final service = build(
        cancelled: (cancelledTarget, cancelledActor, base, paths) {
          expect(paths, [source.path]);
          throw StateError('Database temporarily unavailable');
        },
      );
      await expectLater(
        service.apply(
          target: target,
          actorId: actor,
          removedPendingPaths: [source.path],
        ),
        throwsStateError,
      );
      expect(await build().hasJob(target, actor), isTrue);
      expect(await jobs(), hasLength(1));
      final recovered = await build().apply(
        target: target,
        actorId: actor,
        newPhotos: [source.path],
        setCover: true,
        coverPhoto: source.path,
      );
      expect(recovered.pending, isFalse);
      expect(recovered.completedPaths, [source.path]);
      expect(recovered.rejectedCount, 0);
      expect(uploads, 0);
      await build().pruneCompleted(target, actor);
      expect(await jobs(), isEmpty);
      expect(await build().hasJob(target, actor), isFalse);
    },
  );

  test('removing a completed row source deletes its upload without sending it again', () async {
    final service = build();
    final attached = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [source.path],
    );
    expect(attached.photos, [photoUrl]);
    final removed = await service.apply(
      target: target,
      actorId: actor,
      removedPendingPaths: [source.path],
      setCover: true,
      coverPhoto: source.path,
    );
    expect(removed.pending, isFalse);
    expect(removed.photos, isEmpty);
    expect(removed.completedPaths, [source.path]);
    expect(removed.rejectedCount, 0);
    final resumed = await service.apply(
      target: target,
      actorId: actor,
      newPhotos: [source.path],
      existingPhotos: [photoUrl],
    );
    expect(resumed.photos, isEmpty);
    expect(requests.map((request) => request.method), ['PUT', 'DELETE']);
    expect(uploads, 1);
    await service.pruneCompleted(target, actor);
    expect(await jobs(), isEmpty);
  });
}
