import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';

const _target = PhotoTarget.trail('trail0000000001');
const _actor = 'actor0000000001';

void main() {
  late Directory docs;
  late Dio api;
  late Completer<void> entered, release;
  late List<String> uploads;
  late File source;
  PhotoMutationService build() => PhotoMutationService(
    api: api,
    resolveMode: () async => PhotoApiMode.assets,
    documentsDirectory: () async => docs,
  );
  Future<List<File>> jobs() async => [
    await for (final file in docs.list(recursive: true))
      if (file is File && file.path.endsWith('.json')) file,
  ];
  setUp(() {
    docs = Directory.systemTemp.createTempSync('photo-journal-concurrency-');
    source = File('${docs.path}/picked.jpg')..writeAsBytesSync([1]);
    entered = Completer<void>();
    release = Completer<void>();
    uploads = [];
    api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) async {
          final files = (request.data as FormData).files;
          uploads.addAll(files.map((entry) => entry.value.filename!));
          if (!entered.isCompleted) {
            entered.complete();
            await release.future;
          }
          handler.resolve(
            Response(
              requestOptions: request,
              statusCode: 200,
              data: [
                for (final entry in files)
                  {
                    'id': 'asset0000000001',
                    'collectionId': 'assets',
                    'file': entry.value.filename,
                  },
              ],
            ),
          );
        },
      ),
    );
  });
  tearDown(() async {
    if (!release.isCompleted) release.complete();
    await docs.delete(recursive: true);
  });

  test('replacement instances serialize the same account journal', () async {
    final first = build().apply(
      target: _target,
      actorId: _actor,
      newPhotos: [source.path],
      existingPhotosAreAuthoritative: false,
    );
    await entered.future;
    // A provider rebuilt during sign-out/sign-in has a distinct service instance.
    final replacement = build().apply(
      target: _target,
      actorId: _actor,
      newPhotos: [source.path],
      existingPhotosAreAuthoritative: false,
    );
    await Future<void>.delayed(Duration.zero);
    expect(uploads, ['picked.jpg']);
    release.complete();
    final results = await Future.wait([first, replacement]);
    expect(uploads, ['picked.jpg']);
    expect(results.last.photos, results.first.photos);
    final job = jsonDecode(await (await jobs()).single.readAsString()) as Map;
    expect((job['files'] as List).single['photo'], results.first.photos.single);
  });

  test(
    'an old session cannot create a job after its document lookup resumes',
    () async {
      final locating = Completer<void>(), located = Completer<void>();
      var epoch = 1;
      final oldSession = PhotoMutationService(
        api: api,
        resolveMode: () async => PhotoApiMode.assets,
        sessionEpoch: () => epoch,
        documentsDirectory: () async {
          locating.complete();
          await located.future;
          return docs;
        },
      );
      final stale = oldSession.apply(
        target: _target,
        actorId: _actor,
        newPhotos: [source.path],
      );
      final staleCheck = expectLater(stale, throwsStateError);
      await locating.future;
      epoch++; // Logout, followed by the same account signing in again.
      await build().dropScope(_actor);
      located.complete();
      await staleCheck;
      expect(uploads, isEmpty);
      expect(await jobs(), isEmpty);
    },
  );

  test('scope cleanup fences queued work across replacement instances', () async {
    final firstService = build();
    final first = firstService.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [source.path],
    );
    await entered.future;
    final stale = build().apply(
      target: _target,
      actorId: _actor,
      newPhotos: [source.path],
    );
    final staleCheck = expectLater(stale, throwsStateError);
    await Future<void>.delayed(Duration.zero);
    final purge = build().dropScope(_actor);
    await Future<void>.delayed(Duration.zero);
    release.complete();
    await first;
    await staleCheck;
    await purge;
    expect(uploads, ['picked.jpg']);
    expect(await jobs(), isEmpty);
    // A later legitimate session can use the same scope after cleanup finishes.
    await build().apply(
      target: _target,
      actorId: _actor,
      newPhotos: [source.path],
    );
    expect(uploads, ['picked.jpg', 'picked.jpg']);
  });
}
