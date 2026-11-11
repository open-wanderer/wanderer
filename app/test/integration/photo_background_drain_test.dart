import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/server_capabilities_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/services/server_capabilities.dart';
import 'package:wanderer/store/app_photo_paths.dart';

const _actor = 'actor0000000001';
const _target = PhotoTarget.trail('trail0000000001');
const _stamp = '2026-01-01T00:00:00Z';

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

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform previousPaths;
  late Store store;
  late ProviderContainer container;
  late DateTime now;
  late bool assets;
  late int capabilityStatus;
  late int uploadStatus;
  late int uploadedFiles;
  late int metadataCreates;
  late int capabilityRequests;
  late List<String> serverPhotos;
  late List<String> uploadedFilenames;

  Future<void> open({
    required bool assetServer,
    bool unknownAtSave = false,
  }) async {
    assets = assetServer;
    capabilityStatus = unknownAtSave ? 503 : (assets ? 200 : 404);
    uploadStatus = assets ? 503 : 200;
    uploadedFiles = 0;
    metadataCreates = 0;
    capabilityRequests = 0;
    serverPhotos = [];
    uploadedFilenames = [];
    final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'));
    Map<String, dynamic> trail() => {
      'id': _target.id,
      'name': 'Interactive hike',
      'author': _actor,
      'photos': List<String>.of(serverPhotos),
      'created': _stamp,
      'updated': _stamp,
    };
    api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (request, handler) {
          final path = request.uri.path;
          var status = 200;
          Object data;
          if (path.endsWith('/capabilities')) {
            capabilityRequests++;
            status = capabilityStatus;
            data = assets ? {'assetApi': true} : <String, dynamic>{};
          } else if (path.endsWith('/trail/form') && request.method == 'PUT') {
            metadataCreates++;
            expect((request.data as FormData).files, isEmpty);
            data = trail();
          } else if (path.endsWith('/assets') && request.method == 'PUT') {
            expect(assets, isTrue);
            status = uploadStatus;
            if (status == 200) {
              final files = (request.data as FormData).files;
              uploadedFiles += files.length;
              uploadedFilenames.addAll(
                files.map((file) => file.value.filename!),
              );
              serverPhotos.add(
                '/api/v1/files/assets/asset0000000001/photo.jpg',
              );
              data = [
                {
                  'id': 'asset0000000001',
                  'collectionId': 'assets',
                  'file': 'photo.jpg',
                },
              ];
            } else {
              data = {'message': 'Try again'};
            }
          } else if (path.endsWith('/trail/form/${_target.id}') &&
              request.method == 'POST') {
            expect(assets, isFalse);
            final form = request.data as FormData;
            expect(form.files.map((file) => file.key).toSet(), {'photos+'});
            uploadedFiles += form.files.length;
            uploadedFilenames.addAll(
              form.files.map((file) => file.value.filename!),
            );
            serverPhotos.add('legacy.jpg');
            data = trail();
          } else {
            throw StateError('Unexpected ${request.method} ${request.uri}');
          }
          final response = Response<Object>(
            requestOptions: request,
            statusCode: status,
            data: data,
            headers: Headers.fromMap({
              Headers.contentTypeHeader: [Headers.jsonContentType],
            }),
          );
          if (status >= 400 && request.validateStatus(status) != true) {
            handler.reject(
              DioException(requestOptions: request, response: response),
            );
          } else {
            handler.resolve(response);
          }
        },
      ),
    );
    container = ProviderContainer(
      overrides: [
        apiProvider.overrideWith(() => _Api(api)),
        objectBoxProvider.overrideWithValue(store),
        serverCapabilitiesProvider.overrideWith((ref) {
          final capabilities = ServerCapabilities(api, now: () => now);
          ref.onDispose(capabilities.dispose);
          return capabilities;
        }),
        photoMutationServiceProvider.overrideWith(
          (ref) => PhotoMutationService(
            api: api,
            resolveMode: () => ref.read(photoApiModeProvider)(),
            documentsDirectory: () async => directory,
            now: () => now,
          ),
        ),
      ],
    );
    addTearDown(container.dispose);
  }

  setUp(() async {
    now = DateTime.utc(2026);
    directory = Directory.systemTemp.createTempSync('photo-background-drain-');
    previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Documents(directory.path);
    configurePhotoDocumentsDirectory(directory.path);
    store = await openStore(directory: '${directory.path}/objectbox');
    store.box<UserEntity>().put(
      UserEntity(
        id: 'account00000001',
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
  });
  tearDown(() async {
    store.close();
    PathProviderPlatform.instance = previousPaths;
    await directory.delete(recursive: true);
  });

  Future<void> savePending() async {
    final picked = File('${directory.path}/picked.jpg')
      ..writeAsBytesSync([1, 2, 3]);
    final result = await container
        .read(trailSaveProvider.notifier)
        .createTrail(
          Trail.empty().copyWith(name: 'Interactive hike'),
          authorId: _actor,
          newPhotos: [picked],
        );
    expect(result.hasPendingPhotos, isTrue);
    expect(uploadedFiles, 0);
    expect(store.box<TrailEntity>().getAll(), isEmpty);
    expect(
      await container
          .read(photoMutationServiceProvider)
          .pendingPaths(_target, _actor),
      hasLength(1),
    );
  }

  Future<void> resumeOnce() async {
    now = now.add(const Duration(days: 1));
    await container.read(trailSyncProvider.notifier).drainIfOnline();
    expect(uploadedFiles, 1);
    expect(uploadedFilenames, ['picked.jpg']);
    expect(metadataCreates, 1);
    expect(
      await container
          .read(photoMutationServiceProvider)
          .pendingPaths(_target, _actor),
      isEmpty,
    );
    await container.read(trailSyncProvider.notifier).drainIfOnline();
    expect(
      uploadedFiles,
      1,
      reason: 'A later app trigger must reuse the receipt.',
    );
  }

  test(
    'app trigger resumes an interactive asset upload after a transient 503',
    () async {
      await open(assetServer: true);
      await savePending();
      uploadStatus = 200;
      await resumeOnce();
      expect(serverPhotos, ['/api/v1/files/assets/asset0000000001/photo.jpg']);
    },
  );

  test('app trigger discovers an older server after an unknown save and uploads once', () async {
    await open(assetServer: false, unknownAtSave: true);
    await savePending();
    capabilityStatus = 404;
    await resumeOnce();
    expect(capabilityRequests, 2);
    expect(serverPhotos, ['legacy.jpg']);
  });
}
