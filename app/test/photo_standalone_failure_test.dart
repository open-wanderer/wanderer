import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/store/app_photo_paths.dart';

const _actor = 'actor0000000001';
const _account = 'account00000001';
const _target = PhotoTarget.trail('trail0000000099');

class _Documents extends PathProviderPlatform {
  _Documents(this.path);
  final String path;
  @override
  Future<String?> getApplicationDocumentsPath() async => path;
}

class _Api extends Api {
  _Api(this.client);
  final Dio client;
  @override
  Dio build() => client;
}

class _ThrowingDrain extends PhotoMutationService {
  _ThrowingDrain(Dio client, Directory directory)
    : super(
        api: client,
        resolveMode: () async => PhotoApiMode.unknown,
        documentsDirectory: () async => directory,
      );

  int calls = 0;
  @override
  Future<PhotoMutationResult> drain(
    String actorId, {
    PhotoApiMode? mode,
  }) async {
    calls++;
    throw StateError('Photo journal unavailable');
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('a standalone failure keeps its job and allows capture uploads on every pass', () async {
    final directory = Directory.systemTemp.createTempSync(
      'photo-drain-failure-',
    );
    final previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Documents(directory.path);
    configurePhotoDocumentsDirectory(directory.path);
    final store = await openStore(directory: '${directory.path}/objectbox');
    final logs = <String>[];
    final previousDebugPrint = debugPrint;
    debugPrint = (message, {wrapWidth}) {
      if (message != null) logs.add(message);
    };
    addTearDown(() async {
      debugPrint = previousDebugPrint;
      store.close();
      PathProviderPlatform.instance = previousPaths;
      await directory.delete(recursive: true);
    });
    final now = DateTime.utc(2026);
    store.box<UserEntity>().put(
      UserEntity(
        id: _account,
        collectionId: 'users',
        collectionName: 'users',
        actorId: _actor,
        username: 'hiker',
        preferredUsername: 'hiker',
        email: 'hiker@a.example',
        iri: 'https://a.example/hiker',
        serverUrl: 'https://a.example',
        created: now,
        updated: now,
      ),
    );
    var creates = 0;
    final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) {
          Object data;
          if (request.uri.path.endsWith('/health')) {
            data = {'status': 'ok'};
          } else if (request.uri.path.endsWith('/trail/form') &&
              request.method == 'PUT') {
            creates++;
            data = {
              'id': 'trail${creates.toString().padLeft(10, '0')}',
              'name': 'Queued hike',
              'author': _actor,
              'photos': <String>[],
              'created': now.toIso8601String(),
              'updated': now.toIso8601String(),
            };
          } else {
            throw StateError('Unexpected ${request.method} ${request.uri}');
          }
          handler.resolve(
            Response<Object>(
              requestOptions: request,
              statusCode: 200,
              data: data,
            ),
          );
        },
      ),
    );
    final service = _ThrowingDrain(api, directory);
    final container = ProviderContainer(
      overrides: [
        apiProvider.overrideWith(() => _Api(api)),
        objectBoxProvider.overrideWithValue(store),
        photoApiModeProvider.overrideWithValue(() async => PhotoApiMode.legacy),
        photoMutationServiceProvider.overrideWithValue(service),
      ],
    );
    addTearDown(container.dispose);
    final pick = File('${directory.path}/pending.jpg')
      ..writeAsBytesSync([1, 2, 3]);
    final pending = await service.apply(
      target: _target,
      actorId: _actor,
      newPhotos: [pick.path],
    );
    expect(pending.pendingPaths, hasLength(1));
    final copy = File(pending.pendingPaths.single);

    for (var pass = 1; pass <= 2; pass++) {
      final localId = 'local-$pass-1';
      store.box<TrailEntity>().put(
        TrailEntity(
          id: localId,
          localId: localId,
          owner: _account,
          syncState: TrailSyncState.pending,
          name: 'Queued hike',
          created: now,
          updated: now,
        ),
      );
      await container.read(trailSyncProvider.notifier).drainIfOnline();
      expect(creates, pass);
      expect(store.box<TrailEntity>().getAll(), isEmpty);
      expect(service.calls, pass);
      expect(await service.hasJob(_target, _actor), isTrue);
      expect(await copy.exists(), isTrue);
    }
    expect(
      logs.where(
        (message) => message.contains('standalone photo drain failed'),
      ),
      hasLength(2),
    );
    expect(
      logs.where((message) => message.contains('Photo journal unavailable')),
      hasLength(2),
    );
  });
}
