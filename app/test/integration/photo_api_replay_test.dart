import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/server_capabilities_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/store/app_photo_paths.dart';

const _actor = 'ccccccccccccccc';
const _account = 'ddddddddddddddd';
const _stamp = '2026-01-01T00:00:00Z';
const _legacyKeys = {'photos', 'photos+', '+photos', 'photos-', '-photos'};

class _Request {
  _Request(RequestOptions options)
    : method = options.method,
      uri = options.uri,
      fileCount = options.data is FormData
          ? (options.data as FormData).files.length
          : 0,
      keys = options.data is FormData
          ? {
              for (final entry in (options.data as FormData).fields) entry.key,
              for (final entry in (options.data as FormData).files) entry.key,
            }
          : options.data is Map
          ? (options.data as Map).keys.cast<String>().toSet()
          : <String>{};
  final String method;
  final Uri uri;
  final Set<String> keys;
  final int fileCount;
}

/// Answers the two server contracts and rejects unrecognised routes and fields.
/// No production network or native database is involved in the replay.
class _PhotoServer implements HttpClientAdapter {
  _PhotoServer({required this.assets});
  bool assets;
  int? capabilityStatus;
  final requests = <_Request>[];
  final trails = <String, Map<String, dynamic>>{};
  final waypoints = <String, Map<String, dynamic>>{};
  final targetPhotos = <String, List<String>>{};
  final originals = <String, List<int>>{};
  var trailCreates = 0;
  var waypointCreates = 0;
  var fileUploads = 0;
  final uploadedFilenames = <String>[];
  var failAfterFirstFile = false;
  var failedWaypointWrites = 0;
  int? uploadStatus;
  var tagCreates = 0;
  var rejectLegacyPhotos = false;
  var proxyMissing = false;
  int? maxFilesPerRequest;
  var emptyUploadErrors = 0;
  String? invalidLegacyName;

  List<String> photos(String kind, String id) =>
      targetPhotos.putIfAbsent('$kind:$id', () => []);

  Map<String, dynamic> trail(String id) => {
    ...trails[id]!,
    'photos': List<String>.of(photos('trail', id)),
  };

  Map<String, dynamic> waypoint(String id) => {
    ...waypoints[id]!,
    'photos': List<String>.of(photos('waypoint', id)),
  };

  Map<String, dynamic> _asset() {
    fileUploads++;
    final id = 'asset${fileUploads.toString().padLeft(10, '0')}';
    final file = 'photo$fileUploads.jpg';
    final url = '/api/v1/files/assets/$id/$file';
    originals[url] = [fileUploads, 2, 3];
    return {'id': id, 'collectionId': 'assets', 'file': file};
  }

  ResponseBody _json(Object value, [int status = 200]) =>
      ResponseBody.fromString(
        jsonEncode(value),
        status,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final request = _Request(options);
    requests.add(request);
    final path = options.uri.path.replaceFirst('/api/v1', '');
    if (proxyMissing && (path == '/assets' || path.startsWith('/trail/form'))) {
      return ResponseBody.fromString(
        '404 page not found',
        404,
        headers: {
          Headers.contentTypeHeader: ['text/plain'],
        },
      );
    }
    if (options.data is FormData &&
        (options.data as FormData).files.isNotEmpty) {
      final files = (options.data as FormData).files;
      if (maxFilesPerRequest != null && files.length > maxFilesPerRequest!) {
        return ResponseBody.fromString(
          'Request too large',
          413,
          headers: {
            Headers.contentTypeHeader: ['text/plain'],
          },
        );
      }
      if (!assets &&
          invalidLegacyName != null &&
          files.any((file) => file.value.filename == invalidLegacyName)) {
        return _json({
          'message': 'Failed to update record',
          'data': {
            'photos': {
              'code': 'validation_file_size_limit',
              'message': 'File is too large',
            },
          },
        }, 400);
      }
    }
    if (rejectLegacyPhotos &&
        request.keys.intersection(_legacyKeys).isNotEmpty) {
      return _json({'message': 'trail_photos_form_field_removed'}, 400);
    }
    if (assets) {
      expect(
        request.keys.intersection(_legacyKeys),
        isEmpty,
        reason: '${request.method} ${request.uri} sent a legacy photo key',
      );
    }
    if (path == '/capabilities') {
      expect(options.method, 'GET');
      final status = capabilityStatus ?? (assets ? 200 : 404);
      return _json(assets ? {'assetApi': true} : {}, status);
    }
    if (path == '/health') return _json({'status': 'ok'});
    if (options.method == 'GET' && originals.containsKey(options.uri.path)) {
      final bytes = originals[options.uri.path]!;
      return ResponseBody.fromBytes(
        bytes,
        200,
        headers: {
          Headers.contentLengthHeader: ['${bytes.length}'],
          Headers.contentTypeHeader: [
            options.uri.path.endsWith('.svg') ? 'image/svg+xml' : 'image/jpeg',
          ],
        },
      );
    }
    if (path == '/tag' && options.method == 'PUT') {
      return _json({
        'id': 'tag${(++tagCreates).toString().padLeft(12, '0')}',
        'name': (options.data as Map)['name'],
        'created': _stamp,
        'updated': _stamp,
      });
    }
    if (path == '/trail/form' && options.method == 'PUT') {
      final form = options.data as FormData;
      final fields = Map.fromEntries(form.fields);
      expect(fields['author'], _actor);
      expect(fields['name'], isNotEmpty);
      final id = 'trail${(++trailCreates).toString().padLeft(10, '0')}';
      trails[id] = {
        'id': id,
        'name': fields['name'],
        'author': _actor,
        'created': _stamp,
        'updated': _stamp,
      };
      _legacyForm(form, 'trail', id);
      return _json(trail(id));
    }
    if (path.startsWith('/trail/form/') && options.method == 'POST') {
      final id = path.split('/').last;
      final form = options.data as FormData;
      expect(Map.fromEntries(form.fields)['id'], id);
      _legacyForm(form, 'trail', id);
      return _json(trail(id));
    }
    if (path == '/waypoint' && options.method == 'PUT') {
      if (failedWaypointWrites > 0) {
        failedWaypointWrites--;
        return _json({'message': 'Try again'}, 503);
      }
      final body = options.data as Map;
      expect(body['trail'], isNotEmpty);
      final id = 'point${(++waypointCreates).toString().padLeft(10, '0')}';
      waypoints[id] = {
        'id': id,
        'lat': body['lat'],
        'lon': body['lon'],
        'author': _actor,
        'trail': body['trail'],
        'created': _stamp,
        'updated': _stamp,
      };
      return _json(waypoint(id));
    }
    final pointFile = RegExp(r'^/waypoint/([^/]+)/file$').firstMatch(path);
    if (pointFile != null && options.method == 'POST') {
      expect(assets, isFalse);
      _legacyForm(options.data as FormData, 'waypoint', pointFile[1]!);
      return _json(waypoint(pointFile[1]!));
    }
    final point = RegExp(r'^/waypoint/([^/]+)$').firstMatch(path);
    if (point != null && {'POST', 'GET'}.contains(options.method)) {
      return _json(waypoint(point[1]!));
    }
    if (path == '/assets' && options.method == 'PUT') {
      expect(assets, isTrue);
      if (emptyUploadErrors > 0) {
        emptyUploadErrors--;
        return _json({'message': 'Failed to create record', 'data': {}}, 400);
      }
      if (uploadStatus != null)
        return _json({'message': 'Try again'}, uploadStatus!);
      final form = options.data as FormData;
      final fields = Map.fromEntries(form.fields);
      final kind = [
        'trail',
        'waypoint',
        'summit_log',
      ].singleWhere(fields.containsKey);
      expect(form.files.map((entry) => entry.key).toSet(), {'files'});
      final added = <Map<String, dynamic>>[];
      for (final file in form.files) {
        uploadedFilenames.add(file.value.filename!);
        final asset = _asset();
        added.add(asset);
        photos(kind, fields[kind]!).add(assetPhotoUrl(asset));
        if (failAfterFirstFile) {
          failAfterFirstFile = false;
          return _json({
            'message': 'Upload interrupted',
            'detail': {
              'uploadedAssets': added,
              'attachedAssets': added,
              'completedFileCount': 1,
            },
          }, 503);
        }
      }
      return _json(added);
    }
    final asset = RegExp(r'^/assets/([^/]+)$').firstMatch(path);
    if (asset != null && options.method == 'DELETE') {
      expect(assets, isTrue);
      final query = options.uri.queryParameters;
      final kind = [
        'trail',
        'waypoint',
        'summit_log',
      ].singleWhere(query.containsKey);
      photos(
        kind,
        query[kind]!,
      ).removeWhere((photo) => photoAssetId(photo) == asset[1]);
      return _json({});
    }
    final cover = RegExp(r'^/trail/([^/]+)/thumbnail$').firstMatch(path);
    if (cover != null && options.method == 'PATCH') {
      expect(assets, isTrue);
      expect((options.data as Map).keys, ['asset']);
      expect((options.data as Map)['asset'], isNotNull);
      return _json(trail(cover[1]!));
    }
    final trailPath = RegExp(r'^/trail/([^/]+)$').firstMatch(path);
    if (trailPath != null && {'GET', 'POST'}.contains(options.method)) {
      if (options.method == 'POST') {
        expect((options.data as Map).keys, ['thumbnail']);
        expect(assets, isFalse);
      }
      return _json(trail(trailPath[1]!));
    }
    fail('Unexpected request: ${options.method} ${options.uri}');
  }

  void _legacyForm(FormData form, String kind, String id) {
    if (assets) return;
    for (final field in form.fields.where((entry) => entry.key == 'photos-')) {
      photos(kind, id).removeWhere(
        (photo) => Uri.parse(photo).pathSegments.last == field.value,
      );
    }
    for (final file in form.files.where(
      (entry) => _legacyKeys.contains(entry.key),
    )) {
      expect(
        file.key,
        kind == 'trail' && form.fields.any((field) => field.key == 'id')
            ? 'photos+'
            : 'photos',
      );
      uploadedFilenames.add(file.value.filename!);
      fileUploads++;
      final name = 'photo$fileUploads.jpg';
      photos(kind, id).add(name);
      originals['/api/v1/files/${kind == 'trail' ? 'trails' : 'waypoints'}/$id/$name'] =
          [fileUploads, 2, 3];
    }
  }

  @override
  void close({bool force = false}) {}
}

class _ReplayApi extends Api {
  _ReplayApi(this.client);
  final Dio client;
  @override
  Dio build() => client;
}

class _Documents extends PathProviderPlatform {
  _Documents(this.path);
  final String path;
  @override
  Future<String?> getApplicationDocumentsPath() async => path;
}

void _seedAccount(Store store, String serverUrl) {
  store.box<UserEntity>().removeAll();
  store.box<UserEntity>().put(
    UserEntity(
      id: _account,
      collectionId: 'users',
      collectionName: 'users',
      actorId: _actor,
      username: 'hiker',
      preferredUsername: 'hiker',
      email: 'hiker@a.example',
      iri: '$serverUrl/hiker',
      serverUrl: serverUrl,
      created: DateTime(2026),
      updated: DateTime(2026),
    ),
  );
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform previousPaths;
  late File first;
  late File second;
  late DateTime now;

  setUp(() {
    now = DateTime.utc(2026);
    directory = Directory.systemTemp.createTempSync('photo-api-replay-');
    previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Documents(directory.path);
    configurePhotoDocumentsDirectory(directory.path);
    first = File('${directory.path}/first.jpg')..writeAsBytesSync([1, 2, 3]);
    second = File('${directory.path}/second.jpg')..writeAsBytesSync([4, 5, 6]);
  });
  tearDown(() async {
    PathProviderPlatform.instance = previousPaths;
    await directory.delete(recursive: true);
  });

  Future<
    ({ProviderContainer container, Dio api, _PhotoServer server, Store store})
  >
  harness(
    bool assets, {
    int? capabilityStatus,
    PhotoApiModeResolver? resolveMode,
    bool useProductionJournal = false,
    void Function(PhotoTarget, String, String, List<String>)? cancellationHook,
  }) async {
    final server = _PhotoServer(assets: assets)
      ..capabilityStatus = capabilityStatus;
    final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'))
      ..httpClientAdapter = server;
    final store = await openStore(directory: '${directory.path}/objectbox');
    addTearDown(store.close);
    _seedAccount(store, 'https://a.example');
    final container = ProviderContainer(
      overrides: [
        apiProvider.overrideWith(() => _ReplayApi(api)),
        objectBoxProvider.overrideWithValue(store),
        if (resolveMode != null)
          photoApiModeProvider.overrideWithValue(resolveMode),
        if (!useProductionJournal)
          photoMutationServiceProvider.overrideWith(
            (ref) => PhotoMutationService(
              api: api,
              resolveMode: () => ref.read(photoApiModeProvider)(),
              markAssetApi: () =>
                  ref.read(serverCapabilitiesProvider).markAssetApi(),
              now: () => now,
              onCancelledSources: cancellationHook,
            ),
          ),
      ],
    );
    addTearDown(container.dispose);
    container.read(serverCapabilitiesProvider).newSession();
    return (container: container, api: api, server: server, store: store);
  }

  for (final assets in [true, false]) {
    test(
      '${assets ? 'asset' : 'legacy'} contract replay: create, edit, remove, cover and waypoint',
      () async {
        final h = await harness(assets);
        final save = h.container.read(trailSaveProvider.notifier);
        final created = (await save.createTrail(
          Trail.empty().copyWith(name: 'Recorded hike'),
          authorId: _actor,
          newPhotos: [first],
        )).trail;
        expect(created.photos, hasLength(1));
        final original = created.photos.single;
        final updated = (await save.updateTrail(
          created,
          created.copyWith(name: 'Edited hike'),
          authorId: _actor,
          newPhotos: [second],
          removedPhotoFilenames: [original],
        )).trail;
        expect(updated.photos, hasLength(1));
        expect(updated.photos, isNot(contains(original)));
        final service = h.container.read(photoMutationServiceProvider);
        final covered = await service.apply(
          target: PhotoTarget.trail(created.id),
          actorId: _actor,
          existingPhotos: updated.photos,
          coverPhoto: updated.photos.single,
          setCover: true,
        );
        expect(covered.pending, isFalse);
        final pointSave = h.container.read(waypointSaveProvider.notifier);
        final point = await pointSave.create(
          Waypoint(
            id: '',
            lat: 46,
            lon: 7,
            created: DateTime(2026),
            updated: DateTime(2026),
            localPhotos: [first.path],
          ),
          authorId: _actor,
          trailId: created.id,
        );
        expect(point.photos, hasLength(1));
        final editedPoint = await pointSave.updateWaypoint(
          point,
          point.copyWith(photos: [], localPhotos: [second.path]),
          authorId: _actor,
        );
        expect(editedPoint.photos, hasLength(1));
        expect(editedPoint.photos, isNot(contains(point.photos.single)));
        expect(h.server.trailCreates, 1);
        expect(h.server.waypointCreates, 1);
        final paths = h.server.requests.map(
          (request) => '${request.method} ${request.uri.path}',
        );
        expect(paths, contains('PUT /api/v1/trail/form'));
        expect(paths, contains('POST /api/v1/trail/form/${created.id}'));
        expect(paths, contains('PUT /api/v1/waypoint'));
        expect(paths, contains('POST /api/v1/waypoint/${point.id}'));
        if (assets) {
          expect(
            paths.where((path) => path == 'PUT /api/v1/assets'),
            hasLength(4),
          );
          expect(
            paths.where((path) => path.startsWith('DELETE /api/v1/assets/')),
            hasLength(2),
          );
          expect(
            paths,
            contains('PATCH /api/v1/trail/${created.id}/thumbnail'),
          );
          expect(
            h.server.requests
                .expand((request) => request.keys)
                .toSet()
                .intersection(_legacyKeys),
            isEmpty,
          );
        } else {
          expect(paths.any((path) => path.contains('/assets')), isFalse);
          expect(paths, contains('POST /api/v1/waypoint/${point.id}/file'));
          expect(
            h.server.requests.any((request) => request.keys.contains('photos')),
            isTrue,
          );
          expect(
            h.server.requests.any(
              (request) => request.keys.contains('photos+'),
            ),
            isTrue,
          );
          expect(
            h.server.requests.any(
              (request) => request.keys.contains('photos-'),
            ),
            isTrue,
          );
        }
      },
    );
  }

  test(
    'partial upload replay retries only the uncompleted file prefix',
    () async {
      final h = await harness(true);
      const id = 'trail0000000001';
      h.server.trails[id] = {
        'id': id,
        'name': 'Hike',
        'created': _stamp,
        'updated': _stamp,
      };
      h.server.failAfterFirstFile = true;
      final service = h.container.read(photoMutationServiceProvider);
      final result = await service.apply(
        target: const PhotoTarget.trail(id),
        actorId: _actor,
        newPhotos: [first.path, second.path],
      );
      expect(result.pending, isTrue);
      expect(result.photos, hasLength(1));
      now = now.add(const Duration(minutes: 1));
      final resumed = PhotoMutationService(
        api: h.api,
        now: () => now,
        resolveMode: () => h.container.read(photoApiModeProvider)(),
        documentsDirectory: () async => directory,
      );
      await resumed.drain(_actor);
      expect(h.server.photos('trail', id), hasLength(2));
      expect(h.server.photos('trail', id).toSet(), hasLength(2));
      expect(h.server.fileUploads, 2);
      expect(
        h.server.requests
            .where((request) => request.uri.path == '/api/v1/assets')
            .map((request) => request.fileCount),
        [2, 1],
      );
    },
  );

  test(
    'unknown support keeps photos while metadata saves; an upgrade drains them',
    () async {
      final h = await harness(false, capabilityStatus: 503);
      final saved =
          (await h.container
                  .read(trailSaveProvider.notifier)
                  .createTrail(
                    Trail.empty().copyWith(name: 'Queued hike'),
                    authorId: _actor,
                    newPhotos: [first],
                  ))
              .trail;
      expect(h.server.trailCreates, 1);
      expect(h.server.fileUploads, 0);
      expect(saved.photos, isEmpty);
      final jobFiles = directory
          .listSync(recursive: true)
          .whereType<File>()
          .where((file) => file.path.endsWith('.json'))
          .toList();
      expect(jobFiles, isNotEmpty);
      expect(
        directory
            .listSync(recursive: true)
            .whereType<File>()
            .where(
              (file) =>
                  file.path.contains('photo-mutations') &&
                  file.path.endsWith('.jpg'),
            ),
        isNotEmpty,
      );
      h.server.assets = true;
      h.server.capabilityStatus = 200;
      h.container.read(serverCapabilitiesProvider).reset();
      await h.container.read(photoMutationServiceProvider).drain(_actor);
      expect(h.server.trailCreates, 1);
      expect(h.server.fileUploads, 1);
      expect(h.server.photos('trail', saved.id), hasLength(1));
      expect(
        h.server.requests
            .expand((request) => request.keys)
            .toSet()
            .intersection(_legacyKeys),
        isEmpty,
      );
    },
  );

  test(
    'switching servers chooses each contract and keeps pending photos scoped',
    () async {
      final h = await harness(true, capabilityStatus: 503);
      const id = 'trail0000000001';
      h.server.trails[id] = {
        'id': id,
        'name': 'Hike',
        'created': _stamp,
        'updated': _stamp,
      };
      h.server.capabilityStatus = 503;
      final service = h.container.read(photoMutationServiceProvider);
      expect(
        (await service.apply(
          target: const PhotoTarget.trail(id),
          actorId: _actor,
          newPhotos: [first.path],
        )).pending,
        isTrue,
      );
      h.api.options.baseUrl = 'https://b.example/api/v1';
      _seedAccount(h.store, 'https://b.example');
      h.server.assets = false;
      h.server.capabilityStatus = 404;
      await service.drain(_actor);
      expect(h.server.fileUploads, 0);
      final legacy = await service.apply(
        target: const PhotoTarget.trail(id),
        actorId: _actor,
        newPhotos: [second.path],
      );
      expect(legacy.pending, isFalse);
      expect(
        h.server.requests
            .where((request) => request.fileCount > 0)
            .single
            .uri
            .host,
        'b.example',
      );
      h.api.options.baseUrl = 'https://a.example/api/v1';
      _seedAccount(h.store, 'https://a.example');
      h.server.assets = true;
      h.server.capabilityStatus = 200;
      await service.drain(_actor);
      final transfers = h.server.requests
          .where((request) => request.fileCount > 0)
          .toList();
      expect(transfers, hasLength(2));
      expect(transfers[0].uri.path, '/api/v1/trail/form/$id');
      expect(transfers[0].keys, contains('photos+'));
      expect(transfers[1].uri.host, 'a.example');
      expect(transfers[1].uri.path, '/api/v1/assets');
      expect(transfers[1].keys.intersection(_legacyKeys), isEmpty);
    },
  );
}
