import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/provider/photo_api_provider.dart';
import 'package:wanderer/provider/server_capabilities_provider.dart';
import 'package:wanderer/provider/trail/trail_save_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/provider/waypoint/waypoint_provider.dart';
import 'package:wanderer/services/asset_photo_client.dart';
import 'package:wanderer/services/photo_mutation_service.dart';
import 'package:wanderer/services/trail_download_service.dart';
import 'package:wanderer/store/local_photo_store.dart';
import 'package:wanderer/store/photo_queue_upgrade.dart';
import 'package:wanderer/store/app_photo_paths.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/provider/toast_provider.dart';

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
  int? deleteStatus;
  var waypointDeletes = 0;
  final deletedWaypoints = <String>{};
  var tagCreates = 0;
  var rejectLegacyPhotos = false;
  var proxyMissing = false;
  var guest = false;
  var guestLegacyStatus = 401;
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
    if (path == '/user/$_account') {
      return guest
          ? _json({'message': 'The requested resource was not found.'}, 404)
          : _json({'id': _account});
    }
    if (guest &&
        ((options.method == 'GET' &&
                (path.startsWith('/trail/') ||
                    path.startsWith('/waypoint/'))) ||
            path == '/assets' ||
            path.startsWith('/assets/') ||
            path.startsWith('/trail/form/') ||
            path.endsWith('/file'))) {
      final status = options.method == 'GET'
          ? 404
          : path.startsWith('/trail/form/') || path.endsWith('/file')
          ? guestLegacyStatus
          : 401;
      return _json({
        'message': 'The requested resource was not found.',
      }, status);
    }

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
    if (point != null && options.method == 'DELETE') {
      waypointDeletes++;
      deletedWaypoints.add(point[1]!);
      return _json({});
    }
    if (point != null && {'POST', 'GET'}.contains(options.method)) {
      if (deletedWaypoints.contains(point[1]))
        return _json({'message': 'The requested resource was not found.'}, 404);
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
      if (kind == 'waypoint' && deletedWaypoints.contains(fields[kind]))
        return _json({'message': 'The requested resource was not found.'}, 404);
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
      if (deleteStatus != null)
        return _json({'message': 'Try again'}, deleteStatus!);
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

class _ReplayAuth extends Auth {
  @override
  UserEntity? build() => null;
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

  String queuePhoto(String filename) {
    final root = unsyncedTrailPhotoDir(directory.path, _account, 'local-1-1');
    final file = File('$root/$filename');
    file.parent.createSync(recursive: true);
    if (!file.existsSync()) file.writeAsBytesSync([1, 2, filename.length]);
    return file.path;
  }

  TrailEntity queuedTrail(
    Store store, {
    String id = 'local-1-1',
    List<String> photos = const [],
    List<String> picks = const [],
    List<WaypointEntity> points = const [],
  }) {
    final trail = TrailEntity(
      id: id,
      localId: 'local-1-1',
      owner: _account,
      syncState: TrailSyncState.pending,
      name: 'Queued hike',
      created: DateTime(2026),
      updated: DateTime(2026),
    );
    trail.photos = photos;
    trail.localPhotos = picks;
    trail.waypoints.addAll(points);
    store.box<TrailEntity>().put(trail);
    return trail;
  }

  WaypointEntity pendingPoint(
    String id, {
    List<String> photos = const [],
    List<String> picks = const [],
  }) => WaypointEntity(
    id: id,
    localKey: id.startsWith('local-') ? id : null,
    lat: 46,
    lon: 7,
    author: _actor,
    photos: photos,
    localPhotos: picks,
    created: DateTime(2026),
    updated: DateTime(2026),
  );

  Future<
    ({ProviderContainer container, Dio api, _PhotoServer server, Store store})
  >
  harness(
    bool assets, {
    int? capabilityStatus,
    PhotoApiModeResolver? resolveMode,
    bool useProductionJournal = false,
    bool stubAuth = false,
    Future<void> Function(Object)? onSessionInvalid,
    bool Function(PhotoTarget, String, String)? hasQueuedTarget,
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
        if (stubAuth) authProvider.overrideWith(() => _ReplayAuth()),
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
              currentAccountId: () => _account,
              sessionEpoch: () => ref.read(apiProvider.notifier).sessionEpoch,
              onSessionInvalid: onSessionInvalid,
              onCancelledSources: cancellationHook,
              hasQueuedTarget: hasQueuedTarget,
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

  for (final assets in [true, false]) {
    test(
      '${assets ? 'asset' : 'legacy'} offline drain sends trail and waypoint photos',
      () async {
        final h = await harness(assets);
        final point = WaypointEntity(
          id: 'local-2-1',
          localKey: 'local-2-1',
          lat: 46,
          lon: 7,
          author: _actor,
          created: DateTime(2026),
          updated: DateTime(2026),
          localPhotos: [queuePhoto('second.jpg')],
        );
        h.store.box<WaypointEntity>().put(point);
        final queued = TrailEntity(
          id: 'local-1-1',
          localId: 'local-1-1',
          owner: _account,
          syncState: TrailSyncState.pending,
          name: 'Offline hike',
          created: DateTime(2026),
          updated: DateTime(2026),
        );
        queued.localPhotos = [queuePhoto('first.jpg')];
        queued.waypoints.add(point);
        h.store.box<TrailEntity>().put(queued);
        await h.container.read(trailSyncProvider.notifier).drainIfOnline();
        expect(h.server.trailCreates, 1);
        expect(h.server.waypointCreates, 1);
        expect(h.server.fileUploads, 2);
        expect(
          h.server.photos('trail', h.server.trails.keys.single),
          hasLength(1),
        );
        expect(
          h.server.photos('waypoint', h.server.waypoints.keys.single),
          hasLength(1),
        );
        expect(h.store.box<TrailEntity>().getAll(), isEmpty);
        final uploads = h.server.requests
            .where((request) => request.fileCount > 0)
            .toList();
        if (assets) {
          expect(uploads.map((request) => request.uri.path), [
            '/api/v1/assets',
            '/api/v1/assets',
          ]);
          expect(uploads[0].keys, contains('trail'));
          expect(uploads[1].keys, contains('waypoint'));
        } else {
          expect(uploads[0].uri.path, '/api/v1/trail/form');
          expect(
            uploads[1].uri.path,
            '/api/v1/waypoint/${h.server.waypoints.keys.single}/file',
          );
          expect(
            uploads.every((request) => request.keys.contains('photos')),
            isTrue,
          );
        }
      },
    );
  }

  test(
    'a trail queued before a server upgrade drains only through assets',
    () async {
      final h = await harness(false);
      final capabilities = h.container.read(serverCapabilitiesProvider);
      expect(await capabilities.discover(), isFalse);
      final point = WaypointEntity(
        id: 'local-2-1',
        localKey: 'local-2-1',
        lat: 46,
        lon: 7,
        author: _actor,
        created: DateTime(2026),
        updated: DateTime(2026),
        localPhotos: [queuePhoto('second.jpg')],
      );
      h.store.box<WaypointEntity>().put(point);
      final queued = TrailEntity(
        id: 'local-1-1',
        localId: 'local-1-1',
        owner: _account,
        syncState: TrailSyncState.pending,
        name: 'Upgrade hike',
        created: DateTime(2026),
        updated: DateTime(2026),
      );
      queued.localPhotos = [queuePhoto('first.jpg')];
      queued.waypoints.add(point);
      h.store.box<TrailEntity>().put(queued);
      expect(h.server.trailCreates, 0);
      expect(h.server.waypointCreates, 0);
      h.server.assets = true;
      capabilities.newSession();
      expect(await capabilities.discover(), isTrue);
      final sync = h.container.read(trailSyncProvider.notifier);
      await sync.drainIfOnline();
      await sync.drainIfOnline();
      expect(h.server.trailCreates, 1);
      expect(h.server.waypointCreates, 1);
      expect(h.server.fileUploads, 2);
      expect(
        h.server.photos('trail', h.server.trails.keys.single),
        hasLength(1),
      );
      expect(
        h.server.photos('waypoint', h.server.waypoints.keys.single),
        hasLength(1),
      );
      final transfers = h.server.requests
          .where((request) => request.fileCount > 0)
          .toList();
      expect(transfers.map((request) => request.uri.path), [
        '/api/v1/assets',
        '/api/v1/assets',
      ]);
      expect(transfers[0].keys, {'trail', 'files'});
      expect(transfers[1].keys, {'waypoint', 'files'});
      expect(
        h.server.requests
            .expand((request) => request.keys)
            .toSet()
            .intersection(_legacyKeys),
        isEmpty,
      );
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
    },
  );
  for (final assets in [true, false]) {
    test(
      '${assets ? 'asset' : 'legacy'} resumed drain never uploads server references',
      () async {
        final h = await harness(assets);
        h.server.failedWaypointWrites = 1;
        queuedTrail(
          h.store,
          picks: [queuePhoto('trail.jpg')],
          points: [pendingPoint('local-2-1'), pendingPoint('local-2-2')],
        );
        final sync = h.container.read(trailSyncProvider.notifier);
        await sync.drainIfOnline();
        final remaining = h.store.box<TrailEntity>().getAll().single;
        expect(remaining.localPhotos, isEmpty);
        expect(remaining.photos, hasLength(1));
        expect(h.server.fileUploads, 1);
        await sync.retry('local-1-1');
        expect(h.server.trailCreates, 1);
        expect(h.server.waypointCreates, 2);
        expect(h.server.fileUploads, 1);
        expect(h.store.box<TrailEntity>().getAll(), isEmpty);
      },
    );

    test(
      '${assets ? 'asset' : 'legacy'} older acknowledged rows do not upload their copies again',
      () async {
        final h = await harness(assets);
        const id = 'trail0000000010';
        h.server.trails[id] = {
          'id': id,
          'name': 'Older hike',
          'author': _actor,
          'created': _stamp,
          'updated': _stamp,
        };
        queuedTrail(
          h.store,
          id: id,
          photos: ['sent.jpg'],
          picks: [queuePhoto('sent.jpg')],
          points: [
            pendingPoint(
              'point0000000010',
              photos: ['sent-point.jpg'],
              picks: [queuePhoto('sent-point.jpg')],
            ),
            pendingPoint('local-2-1'),
          ],
        );
        await h.container.read(trailSyncProvider.notifier).drainIfOnline();
        expect(h.server.fileUploads, 0);
        expect(h.server.trailCreates, 0);
        expect(h.server.waypointCreates, 1);
        expect(h.store.box<TrailEntity>().getAll(), isEmpty);
        expect(await upgradePhotoQueue(h.store), isFalse);
      },
    );
  }

  test(
    'queue receipt upgrade runs once and keeps an unconfirmed waypoint pick',
    () async {
      final h = await harness(true);
      const id = 'trail0000000010';
      final pending = pendingPoint(
        'point0000000010',
        picks: [queuePhoto('pending.jpg')],
      );
      final cached = File('${directory.path}/library/$id/point.jpg');
      cached.parent.createSync(recursive: true);
      cached.writeAsBytesSync([4, 5, 6]);
      final confirmed = pendingPoint(
        'point0000000020',
        photos: ['sent.jpg'],
        picks: [cached.path],
      );
      queuedTrail(
        h.store,
        id: id,
        picks: [queuePhoto('sent.jpg')],
        points: [pending, confirmed],
      );
      expect(await upgradePhotoQueue(h.store), isTrue);
      final point = h.store.box<WaypointEntity>().get(pending.obxId)!;
      expect(point.localPhotos, [queuePhoto('pending.jpg')]);
      final old = h.store.box<WaypointEntity>().get(confirmed.obxId)!;
      expect(old.localPhotos, isEmpty);
      expect(old.downloadedPhotos.map(resolveStoredPhotoPath), [cached.path]);
      final trail = h.store.box<TrailEntity>().getAll().single;
      trail.localPhotos = [queuePhoto('new.jpg')];
      h.store.box<TrailEntity>().put(trail);
      expect(await upgradePhotoQueue(h.store), isFalse);
      expect(
        h.store.box<TrailEntity>().getAll().single.localPhotos,
        isNotEmpty,
      );
    },
  );

  test('pending photo passes create each tag once and retain backoff without progress', () async {
    final h = await harness(true);
    h.server.uploadStatus = 503;
    final trail = queuedTrail(h.store, picks: [queuePhoto('pending.jpg')]);
    trail.tagsJson = encodeTrailTags([const Tag(name: 'Nature')]);
    h.store.box<TrailEntity>().put(trail);
    final sync = h.container.read(trailSyncProvider.notifier);
    await sync.drainIfOnline();
    var pending = h.store.box<TrailEntity>().getAll().single;
    expect(jsonDecode(pending.tagsJson!)[0]['id'], isNotEmpty);
    await sync.drainIfOnline();
    pending = h.store.box<TrailEntity>().getAll().single;
    expect(pending.syncAttempts, 1);
    expect(pending.syncNextAttemptAt, isNotNull);
    await sync.retry('local-1-1');
    expect(h.server.tagCreates, 1);
    expect(h.server.trailCreates, 1);
    expect(
      h.store.box<TrailEntity>().getAll().single.syncState,
      TrailSyncState.pending,
    );
  });

  test('one drain resolves photo mode once for every queued target', () async {
    var discoveries = 0;
    final h = await harness(
      true,
      resolveMode: () async {
        discoveries++;
        return PhotoApiMode.assets;
      },
    );
    queuedTrail(
      h.store,
      picks: [queuePhoto('trail.jpg')],
      points: [
        pendingPoint('local-2-1', picks: [queuePhoto('point.jpg')]),
      ],
    );
    await h.container.read(trailSyncProvider.notifier).drainIfOnline();
    expect(discoveries, 1);
    expect(h.server.fileUploads, 2);
  });

  test(
    'legacy photo rejection keeps the queue pending and switches to assets',
    () async {
      final h = await harness(false);
      expect(
        await h.container.read(serverCapabilitiesProvider).discover(),
        false,
      );
      h.server.assets = true;
      h.server.rejectLegacyPhotos = true;
      queuedTrail(h.store, picks: [queuePhoto('trail.jpg')]);
      final sync = h.container.read(trailSyncProvider.notifier);
      await sync.drainIfOnline();
      final row = h.store.box<TrailEntity>().getAll().single;
      expect(row.syncState, TrailSyncState.pending);
      expect(row.syncAttempts, 0);
      expect(h.container.read(serverCapabilitiesProvider).assetApi, true);
      await sync.drainIfOnline();
      expect(h.server.trailCreates, 1);
      expect(h.server.fileUploads, 1);
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
    },
  );

  test(
    'download preserves pending captures and only unsynced picks are uploaded',
    () async {
      final h = await harness(true);
      await upgradePhotoQueue(h.store);
      const id = 'trail0000000010';
      const pendingId = 'point0000000010';
      const completedId = 'point0000000020';
      const firstPhoto = '/api/v1/files/assets/asset0000000030/one.jpg';
      const secondPhoto = '/api/v1/files/assets/asset0000000040/two.jpg';
      h.server.trails[id] = {
        'id': id,
        'name': 'Downloaded capture',
        'author': _actor,
        'created': _stamp,
        'updated': _stamp,
      };
      for (final entry in {
        pendingId: firstPhoto,
        completedId: secondPhoto,
      }.entries) {
        h.server.waypoints[entry.key] = {
          'id': entry.key,
          'lat': 46.0,
          'lon': 7.0,
          'author': _actor,
          'trail': id,
          'created': _stamp,
          'updated': _stamp,
        };
        h.server.photos('waypoint', entry.key).add(entry.value);
        h.server.originals[entry.value] = [1, 2, 3];
      }
      final pending = pendingPoint(
        pendingId,
        photos: [firstPhoto],
        picks: [queuePhoto('new-point.jpg')],
      );
      queuedTrail(
        h.store,
        id: id,
        points: [
          pending,
          pendingPoint(completedId, photos: [secondPhoto]),
          pendingPoint('local-2-1'),
        ],
      );
      final model = Trail.fromJson(h.server.trail(id)).copyWith(
        expand: TrailExpand(
          waypointsViaTrail: [
            Waypoint.fromJson(h.server.waypoint(pendingId)),
            Waypoint.fromJson(h.server.waypoint(completedId)),
          ],
        ),
      );
      await TrailDownloadService(
        h.store,
        h.api,
        photoApi: h.api,
      ).downloadTrail(model, savedByUserId: _account);
      final downloaded = h.store.box<TrailEntity>().getAll().single;
      expect(downloaded.waypoints, hasLength(3));
      final held = downloaded.waypoints.singleWhere(
        (point) => point.id == pendingId,
      );
      expect(held.localPhotos, [queuePhoto('new-point.jpg')]);
      expect(held.downloadedPhotos, hasLength(1));
      final completed = downloaded.waypoints.singleWhere(
        (point) => point.id == completedId,
      );
      expect(completed.localPhotos, isEmpty);
      // A persisted downloaded copy is never a permissible upload input.
      completed.localPhotos = completed.downloadedPhotos;
      h.store.box<WaypointEntity>().put(completed);
      await h.container.read(trailSyncProvider.notifier).drainIfOnline();
      expect(h.server.waypointCreates, 1);
      expect(h.server.fileUploads, 1);
      expect(
        h.store.box<TrailEntity>().getAll().single.syncState,
        TrailSyncState.synced,
      );
      expect(
        h.server.requests
            .where((request) => request.fileCount > 0)
            .single
            .uri
            .path,
        '/api/v1/assets',
      );
    },
  );

  test('queued relative photo paths survive a mobile container relocation', () async {
    final h = await harness(true);
    final oldDocs = Directory('${directory.path}/Documents')..createSync();
    final picked = File(
      '${unsyncedTrailPhotoDir(oldDocs.path, _account, 'local-1-1')}/picked.jpg',
    );
    picked.parent.createSync(recursive: true);
    picked.writeAsBytesSync([1, 2, 3]);
    queuedTrail(h.store, picks: [storePhotoPath(oldDocs.path, picked.path)]);
    final relocated = Directory('${directory.path}/relocated')..createSync();
    final newDocs = oldDocs.renameSync('${relocated.path}/Documents');
    PathProviderPlatform.instance = _Documents(newDocs.path);
    configurePhotoDocumentsDirectory(newDocs.path);
    await h.container.read(trailSyncProvider.notifier).drainIfOnline();
    expect(h.server.fileUploads, 1);
    expect(h.server.trailCreates, 1);
    expect(h.store.box<TrailEntity>().getAll(), isEmpty);
  });
  test('a journal upgrade stops the remaining legacy queue pass', () async {
    final h = await harness(false);
    final capabilities = h.container.read(serverCapabilitiesProvider);
    expect(await capabilities.discover(), false);
    const id = 'trail0000000010';
    h.server.trails[id] = {
      'id': id,
      'name': 'Earlier edit',
      'author': _actor,
      'created': _stamp,
      'updated': _stamp,
    };
    await h.container
        .read(photoMutationServiceProvider)
        .apply(
          target: const PhotoTarget.trail(id),
          actorId: _actor,
          newPhotos: [first.path],
          mode: PhotoApiMode.unknown,
        );
    queuedTrail(h.store, picks: [queuePhoto('next.jpg')]);
    h.server.assets = true;
    h.server.rejectLegacyPhotos = true;
    final sync = h.container.read(trailSyncProvider.notifier);
    await sync.drainIfOnline();
    final rejected = h.server.requests
        .where((request) => request.keys.intersection(_legacyKeys).isNotEmpty)
        .toList();
    expect(rejected, hasLength(1));
    expect(rejected.single.uri.path, '/api/v1/trail/form/$id');
    expect(h.server.trailCreates, 0);
    await sync.drainIfOnline();
    expect(h.server.trailCreates, 1);
    expect(h.server.fileUploads, 2);
    expect(h.store.box<TrailEntity>().getAll(), isEmpty);
  });

  test('partial queue receipts survive journal cleanup on sign out', () async {
    final h = await harness(true);
    final firstPath = queuePhoto('one.jpg');
    final secondPath = queuePhoto('two.jpg');
    queuedTrail(h.store, picks: [firstPath, secondPath]);
    h.server.failAfterFirstFile = true;
    final sync = h.container.read(trailSyncProvider.notifier);
    await sync.drainIfOnline();
    final remaining = h.store.box<TrailEntity>().getAll().single;
    expect(remaining.localPhotos.map(resolveStoredPhotoPath), [secondPath]);
    expect(remaining.photos, hasLength(1));
    await h.container.read(photoMutationServiceProvider).dropScope(_actor);
    h.container.invalidate(photoMutationServiceProvider);
    h.container.read(apiProvider.notifier).renewSession();
    h.container.read(serverCapabilitiesProvider).newSession();
    now = now.add(const Duration(hours: 1));
    await sync.retry('local-1-1');
    expect(h.server.fileUploads, 2);
    expect(h.server.trailCreates, 1);
    expect(h.server.photos('trail', h.server.trails.keys.single), hasLength(2));
    expect(h.store.box<TrailEntity>().getAll(), isEmpty);
  });

  for (final assets in [true, false]) {
    test(
      '${assets ? 'asset' : 'legacy'} proxy text 404 retains queued photos until recovery',
      () async {
        final h = await harness(assets);
        final picks = [
          queuePhoto('proxy-one.jpg'),
          queuePhoto('proxy-two.jpg'),
        ];
        queuedTrail(h.store, picks: picks);
        final sync = h.container.read(trailSyncProvider.notifier);
        if (assets) {
          h.server.uploadStatus = 503;
          await sync.drainIfOnline();
          h.server.uploadStatus = null;
        }
        h.server.proxyMissing = true;
        now = now.add(const Duration(hours: 1));
        await sync.retry('local-1-1');
        final retained = h.store.box<TrailEntity>().getAll().single;
        expect(retained.localPhotos.map(resolveStoredPhotoPath), picks);
        expect(picks.every((path) => File(path).existsSync()), isTrue);
        expect(h.server.fileUploads, 0);
        h.server.proxyMissing = false;
        now = now.add(const Duration(hours: 1));
        await sync.retry('local-1-1');
        expect(h.server.fileUploads, 2);
        expect(h.server.trailCreates, 1);
        expect(h.store.box<TrailEntity>().getAll(), isEmpty);
      },
    );
  }

  test(
    'a proxy batch limit splits queued requests without rejecting good photos',
    () async {
      final h = await harness(true);
      queuedTrail(
        h.store,
        picks: [for (var i = 0; i < 5; i++) queuePhoto('batch-$i.jpg')],
      );
      h.server.maxFilesPerRequest = 2;
      await h.container.read(trailSyncProvider.notifier).drainIfOnline();
      expect(h.server.fileUploads, 5);
      expect(
        h.server.photos('trail', h.server.trails.keys.single),
        hasLength(5),
      );
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
      expect(h.container.read(toastProvider), isEmpty);
    },
  );

  test(
    'a transient empty-data 400 keeps every queued photo for the next pass',
    () async {
      final h = await harness(true);
      final picks = [
        queuePhoto('transient-one.jpg'),
        queuePhoto('transient-two.jpg'),
      ];
      queuedTrail(h.store, picks: picks);
      h.server.emptyUploadErrors = 10;
      final sync = h.container.read(trailSyncProvider.notifier);
      await sync.drainIfOnline();
      expect(
        h.store.box<TrailEntity>().getAll().single.localPhotos.map(
          resolveStoredPhotoPath,
        ),
        picks,
      );
      expect(h.server.fileUploads, 0);
      expect(h.container.read(toastProvider), isEmpty);
      h.server.emptyUploadErrors = 0;
      now = now.add(const Duration(hours: 1));
      await sync.retry('local-1-1');
      expect(h.server.fileUploads, 2);
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
    },
  );

  test(
    'a legacy validation failure rejects only its oversized queued file',
    () async {
      final h = await harness(false);
      await upgradePhotoQueue(h.store);
      const id = 'trail0000000010';
      h.server.trails[id] = {
        'id': id,
        'name': 'Older server',
        'author': _actor,
        'created': _stamp,
        'updated': _stamp,
      };
      final oversized = File(queuePhoto('too-large.jpg'))
          .openSync(mode: FileMode.write);
      oversized.truncateSync(21 * 1024 * 1024);
      oversized.closeSync();
      queuedTrail(
        h.store,
        id: id,
        picks: [
          queuePhoto('good-first.jpg'),
          queuePhoto('too-large.jpg'),
          queuePhoto('good-last.jpg'),
        ],
      );
      h.server.invalidLegacyName = 'too-large.jpg';
      await h.container.read(trailSyncProvider.notifier).drainIfOnline();
      expect(h.server.fileUploads, 2);
      expect(h.server.photos('trail', id), hasLength(2));
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
      expect(
        h.container.read(toastProvider).single.text,
        contains('One photo'),
      );
      expect(
        h.container.read(toastProvider).single.text,
        isNot(contains('50 MiB')),
      );
    },
  );

  Future<void> seedResumedCapture(
    ({ProviderContainer container, Dio api, _PhotoServer server, Store store})
    h,
  ) async {
    await upgradePhotoQueue(h.store);
    const trailId = 'trail0000000010';
    const pointId = 'point0000000010';
    h.server.trails[trailId] = {
      'id': trailId,
      'name': 'Pending capture',
      'author': _actor,
      'created': _stamp,
      'updated': _stamp,
    };
    h.server.waypoints[pointId] = {
      'id': pointId,
      'lat': 46.0,
      'lon': 7.0,
      'author': _actor,
      'trail': trailId,
      'created': _stamp,
      'updated': _stamp,
    };
    queuedTrail(
      h.store,
      id: trailId,
      picks: [queuePhoto('trail-pick.jpg')],
      points: [
        pendingPoint(pointId, picks: [queuePhoto('point-pick.jpg')]),
      ],
    );
  }

  test('an editor rename keeps queued trail and waypoint upload receipts until the next drain', () async {
    final h = await harness(true);
    await seedResumedCapture(h);
    h.server.uploadStatus = 503;
    final sync = h.container.read(trailSyncProvider.notifier);
    await sync.drainIfOnline();
    h.server.uploadStatus = null;
    now = now.add(const Duration(hours: 1));
    final before = h.store.box<TrailEntity>().getAll().single.toModel();
    final edited = before.copyWith(
      name: 'Renamed',
      expand: before.expand!.copyWith(
        waypointsViaTrail: [
          before.expand!.waypointsViaTrail!.single.copyWith(
            name: 'Renamed point',
          ),
        ],
      ),
    );
    await h.container
        .read(trailSaveProvider.notifier)
        .updateTrail(
          before,
          edited,
          authorId: _actor,
          newPhotos: const [],
          removedPhotoFilenames: const [],
        );
    expect(h.server.fileUploads, 2);
    expect(h.store.box<TrailEntity>().getAll().single.localPhotos, isNotEmpty);
    await sync.retry('local-1-1');
    expect(h.server.fileUploads, 2);
    expect(h.server.photos('trail', 'trail0000000010'), hasLength(1));
    expect(h.server.photos('waypoint', 'point0000000010'), hasLength(1));
    expect(h.store.box<TrailEntity>().getAll(), isEmpty);
  });

  for (final removalDue in [false, true]) {
    test(
      'a waypoint removal keeps its capture pending ${removalDue ? 'when due' : 'during backoff'} and completes after recovery',
      () async {
        late ({
          ProviderContainer container,
          Dio api,
          _PhotoServer server,
          Store store,
        })
        h;
        h = await harness(
          true,
          hasQueuedTarget: (target, actor, base) => hasQueuedPhotoTarget(
            h.store,
            accountId: _account,
            kind: target.kind,
            targetId: target.id,
          ),
          cancellationHook: (target, actor, base, paths) =>
              removeQueuedPhotoSources(
                h.store,
                accountId: _account,
                kind: target.kind,
                targetId: target.id,
                paths: paths,
              ),
        );
        await seedResumedCapture(h);
        h.server.uploadStatus = 503;
        final sync = h.container.read(trailSyncProvider.notifier);
        await sync.drainIfOnline();
        h.server.uploadStatus = null;
        now = now.add(const Duration(hours: 1));
        final before = h.store.box<TrailEntity>().getAll().single.toModel();
        final uploaded =
            (await h.container
                    .read(trailSaveProvider.notifier)
                    .updateTrail(
                      before,
                      before.copyWith(
                        name: 'Renamed',
                        expand: before.expand!.copyWith(
                          waypointsViaTrail: [
                            before.expand!.waypointsViaTrail!.single.copyWith(
                              name: 'Renamed point',
                            ),
                          ],
                        ),
                      ),
                      authorId: _actor,
                      newPhotos: const [],
                      removedPhotoFilenames: const [],
                    ))
                .trail;
        expect(h.server.fileUploads, 2);
        final point = uploaded.expand!.waypointsViaTrail!.single;
        final sources = h.store
            .box<TrailEntity>()
            .getAll()
            .single
            .waypoints
            .single
            .localPhotos;
        h.server.deleteStatus = 503;
        await h.container
            .read(waypointSaveProvider.notifier)
            .updateWaypoint(
              point,
              point.copyWith(
                photos: const [],
                localPhotos: const [],
                removedPendingPhotos: sources,
              ),
              authorId: _actor,
              mode: PhotoApiMode.assets,
            );
        final journal = h.container.read(photoMutationServiceProvider);
        expect(await journal.pendingCount(_actor), 1);
        expect(
          h.store
              .box<TrailEntity>()
              .getAll()
              .single
              .waypoints
              .single
              .localPhotos,
          isEmpty,
        );
        if (removalDue) now = now.add(const Duration(hours: 1));
        await sync.retry('local-1-1');
        expect(h.store.box<TrailEntity>().getAll(), hasLength(1));
        final deletes = h.server.requests
            .where(
              (request) =>
                  request.method == 'DELETE' &&
                  request.uri.path.contains('/assets/'),
            )
            .length;
        expect(deletes, removalDue ? 2 : 1);
        await journal.drain(_actor, mode: PhotoApiMode.assets);
        expect(
          h.server.requests.where(
            (request) =>
                request.method == 'DELETE' &&
                request.uri.path.contains('/assets/'),
          ),
          hasLength(deletes),
        );
        h.server.deleteStatus = null;
        now = now.add(const Duration(hours: 6));
        await sync.retry('local-1-1');
        expect(h.server.photos('waypoint', point.id), isEmpty);
        expect(
          h.server.requests.where(
            (request) =>
                request.method == 'DELETE' &&
                request.uri.path.contains('/assets/'),
          ),
          hasLength(deletes + 1),
        );
        expect(h.store.box<TrailEntity>().getAll(), isEmpty);
        expect(await journal.pendingCount(_actor), 0);
        expect(
          await journal.hasJob(PhotoTarget.waypoint(point.id), _actor),
          isFalse,
        );
      },
    );
  }

  test(
    'a queue-owned removal without a capture resumes in the standalone drain',
    () async {
      late ({
        ProviderContainer container,
        Dio api,
        _PhotoServer server,
        Store store,
      })
      h;
      h = await harness(
        true,
        hasQueuedTarget: (target, actor, base) => hasQueuedPhotoTarget(
          h.store,
          accountId: _account,
          kind: target.kind,
          targetId: target.id,
        ),
        cancellationHook: (target, actor, base, paths) =>
            removeQueuedPhotoSources(
              h.store,
              accountId: _account,
              kind: target.kind,
              targetId: target.id,
              paths: paths,
            ),
      );
      await seedResumedCapture(h);
      final point = h.store.box<TrailEntity>().getAll().single.waypoints.single;
      final source = point.localPhotos.single;
      final journal = h.container.read(photoMutationServiceProvider);
      final uploaded = await journal.apply(
        target: PhotoTarget.waypoint(point.id),
        actorId: _actor,
        newPhotos: [source],
        mode: PhotoApiMode.assets,
      );
      expect(uploaded.pending, isFalse);
      h.server.deleteStatus = 503;
      final removed = await journal.apply(
        target: PhotoTarget.waypoint(point.id),
        actorId: _actor,
        removedPendingPaths: [source],
        existingPhotos: uploaded.photos,
        mode: PhotoApiMode.assets,
      );
      expect(removed.pending, isTrue);
      deleteLocalTrailRow(h.store, 'local-1-1', accountId: _account);
      h.server.deleteStatus = null;
      now = now.add(const Duration(hours: 1));
      await journal.drain(_actor, mode: PhotoApiMode.assets);
      expect(h.server.photos('waypoint', point.id), isEmpty);
      expect(await journal.pendingCount(_actor), 0);
      expect(
        await journal.hasJob(PhotoTarget.waypoint(point.id), _actor),
        isFalse,
      );
      expect(
        h.server.requests.where(
          (request) =>
              request.method == 'DELETE' &&
              request.uri.path.contains('/assets/'),
        ),
        hasLength(2),
      );
    },
  );

  for (final scenario in [
    (assets: true, receipt: false, legacyStatus: 401, logout: false),
    (assets: true, receipt: false, legacyStatus: 401, logout: true),
    (assets: true, receipt: true, legacyStatus: 401, logout: false),
    (assets: false, receipt: false, legacyStatus: 401, logout: false),
    (assets: false, receipt: false, legacyStatus: 404, logout: false),
  ]) {
    test(
      'a guest ${scenario.assets ? 'asset' : 'legacy'} capture keeps its waypoint and copies (${scenario.receipt ? 'removal receipt' : scenario.legacyStatus}${scenario.logout ? ', account invalidation' : ''}) and finishes after the account returns',
      () async {
        var signOuts = 0;
        late ({
          ProviderContainer container,
          Dio api,
          _PhotoServer server,
          Store store,
        })
        h;
        h = await harness(
          scenario.assets,
          onSessionInvalid: (_) async {
            signOuts++;
            if (scenario.logout) {
              h.container.read(apiProvider.notifier).renewSession();
              h.container.invalidate(waypointSaveProvider);
              h.store.box<UserEntity>().removeAll();
            }
          },
        );
        await seedResumedCapture(h);
        final row = h.store.box<TrailEntity>().getAll().single;
        row.localPhotos = [];
        h.store.box<TrailEntity>().put(row);
        final point = row.waypoints.single;
        final source = resolveStoredPhotoPath(point.localPhotos.single);
        final journal = h.container.read(photoMutationServiceProvider);
        await journal.apply(
          target: PhotoTarget.waypoint(point.id),
          actorId: _actor,
          newPhotos: [source],
          mode: scenario.receipt ? PhotoApiMode.assets : PhotoApiMode.unknown,
        );
        if (scenario.receipt) {
          await journal.apply(
            target: PhotoTarget.waypoint(point.id),
            actorId: _actor,
            removedPhotos: ['legacy-old.jpg'],
            mode: PhotoApiMode.unknown,
          );
        }
        h.server.guest = true;
        h.server.guestLegacyStatus = scenario.legacyStatus;
        final sync = h.container.read(trailSyncProvider.notifier);
        await sync.drainIfOnline();
        final retained = h.store.box<TrailEntity>().getAll().single;
        expect(retained.waypoints.map((point) => point.id), [point.id]);
        if (!scenario.receipt) {
          expect(retained.waypoints.single.localPhotos, isNotEmpty);
          expect(File(source).existsSync(), isTrue);
        } else {
          expect(retained.waypoints.single.photos, hasLength(1));
        }
        expect(
          await journal.hasJob(PhotoTarget.waypoint(point.id), _actor),
          isTrue,
        );
        expect(await journal.pendingCount(_actor), 1);
        expect(h.container.read(toastProvider), isEmpty);
        expect(signOuts, greaterThan(0));
        h.server.guest = false;
        if (scenario.logout) {
          expect(h.store.box<UserEntity>().getAll(), isEmpty);
          _seedAccount(h.store, 'https://a.example');
        }
        now = now.add(const Duration(hours: 1));
        await sync.retry('local-1-1');
        await sync.drainIfOnline();
        expect(h.store.box<TrailEntity>().getAll(), isEmpty);
        expect(h.server.fileUploads, 1);
        expect(h.server.photos('waypoint', point.id), hasLength(1));
        expect(
          await journal.hasJob(PhotoTarget.waypoint(point.id), _actor),
          isFalse,
        );
        expect(File(source).existsSync(), isFalse);
        expect(h.container.read(toastProvider), isEmpty);
      },
    );
  }

  test(
    'a confirmed deleted waypoint does not block the next queued waypoint',
    () async {
      final h = await harness(true);
      await seedResumedCapture(h);
      final row = h.store.box<TrailEntity>().getAll().single;
      final removedSource = resolveStoredPhotoPath(
        row.waypoints.single.localPhotos.single,
      );
      row.waypoints.add(
        pendingPoint('local-2-1', picks: [queuePhoto('next-point.jpg')]),
      );
      h.store.box<TrailEntity>().put(row);
      await h.api.delete('/waypoint/point0000000010');
      await h.container.read(trailSyncProvider.notifier).drainIfOnline();
      expect(h.server.waypointCreates, 1);
      expect(h.server.photos('waypoint', 'point0000000001'), hasLength(1));
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
      expect(File(removedSource).existsSync(), isFalse);
      expect(h.container.read(toastProvider), isEmpty);
      expect(h.server.waypointDeletes, 1);
      expect(
        h.server.requests.where(
          (request) =>
              request.method == 'GET' &&
              request.uri.path.endsWith('/waypoint/point0000000010'),
        ),
        hasLength(1),
      );
    },
  );

  test(
    'an editor waypoint deletion reconciles the capture before its next drain',
    () async {
      final h = await harness(true, stubAuth: true);
      await seedResumedCapture(h);
      final row = h.store.box<TrailEntity>().getAll().single;
      final removedSource = resolveStoredPhotoPath(
        row.waypoints.single.localPhotos.single,
      );
      row.waypoints.add(
        pendingPoint('local-2-1', picks: [queuePhoto('next-point.jpg')]),
      );
      h.store.box<TrailEntity>().put(row);
      h.server.uploadStatus = 503;
      h.server.failedWaypointWrites = 1;
      final sync = h.container.read(trailSyncProvider.notifier);
      await sync.drainIfOnline();
      h.server.uploadStatus = null;
      now = now.add(const Duration(hours: 1));
      final before = h.store.box<TrailEntity>().getAll().single.toModel();
      final edited = before.copyWith(
        expand: before.expand!.copyWith(
          waypointsViaTrail: before.expand!.waypointsViaTrail!
              .where((point) => point.id != 'point0000000010')
              .toList(),
        ),
      );
      final saved = await h.container
          .read(trailSaveProvider.notifier)
          .updateTrail(
            before,
            edited,
            authorId: _actor,
            newPhotos: const [],
            removedPhotoFilenames: const [],
          );
      expect(saved.hadWaypointFailures, isFalse);
      expect(
        h.store.box<TrailEntity>().getAll().single.waypoints.map(
          (point) => point.id,
        ),
        ['local-2-1'],
      );
      expect(File(removedSource).existsSync(), isFalse);
      await sync.retry('local-1-1');
      expect(h.server.waypointCreates, 1);
      expect(h.server.photos('waypoint', 'point0000000001'), hasLength(1));
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
      expect(h.container.read(toastProvider), isEmpty);
      expect(h.server.waypointDeletes, 1);
      expect(
        await h.container
            .read(photoMutationServiceProvider)
            .pendingCount(_actor),
        0,
      );
    },
  );

  test('a backed-off queued job stays with its row through journal drain and confirmed sign-out', () async {
    final h = await harness(true);
    await seedResumedCapture(h);
    h.server.uploadStatus = 503;
    final sync = h.container.read(trailSyncProvider.notifier);
    await sync.drainIfOnline();
    h.server.uploadStatus = null;
    now = now.add(const Duration(hours: 1));
    final journal = h.container.read(photoMutationServiceProvider);
    await journal.drain(_actor, mode: PhotoApiMode.assets);
    expect(
      h.server.fileUploads,
      0,
      reason: 'The queue must persist its own receipts.',
    );
    await journal.dropScope(_actor);
    h.container.invalidate(photoMutationServiceProvider);
    _seedAccount(h.store, 'https://a.example');
    h.container.read(apiProvider.notifier).renewSession();
    await sync.retry('local-1-1');
    expect(h.server.fileUploads, 2);
    expect(h.store.box<TrailEntity>().getAll(), isEmpty);
  });

  for (final seedJournal in [true, false]) {
    test(
      seedJournal
          ? 'removing queued trail and waypoint photos cancels their sources before the next drain'
          : 'removing queued photos before their first journal job prevents a later upload',
      () async {
        final h = await harness(true, useProductionJournal: true);
        await seedResumedCapture(h);
        final row = h.store.box<TrailEntity>().getAll().single;
        final journal = h.container.read(photoMutationServiceProvider);
        final point = row.waypoints.single;
        if (seedJournal) {
          await journal.apply(
            target: PhotoTarget.trail(row.id),
            actorId: _actor,
            newPhotos: row.localPhotos,
            mode: PhotoApiMode.unknown,
          );
          await journal.apply(
            target: PhotoTarget.waypoint(point.id),
            actorId: _actor,
            newPhotos: point.localPhotos,
            mode: PhotoApiMode.unknown,
          );
        }
        final before = row.toModel();
        final edited = before.copyWith(
          localPhotos: const [],
          expand: before.expand!.copyWith(
            waypointsViaTrail: [
              before.expand!.waypointsViaTrail!.single.copyWith(
                localPhotos: const [],
                removedPendingPhotos: point.localPhotos,
              ),
            ],
          ),
        );
        await h.container
            .read(trailSaveProvider.notifier)
            .updateTrail(
              before,
              edited,
              authorId: _actor,
              newPhotos: const [],
              removedPhotoFilenames: const [],
              removedPendingPaths: row.localPhotos,
            );
        final updated = h.store.box<TrailEntity>().getAll().single;
        expect(updated.localPhotos, isEmpty);
        expect(updated.waypoints.single.localPhotos, isEmpty);
        await h.container.read(trailSyncProvider.notifier).retry('local-1-1');
        expect(h.server.fileUploads, 0);
        expect(
          h.server.requests.where(
            (request) => request.uri.path.endsWith('/assets'),
          ),
          isEmpty,
        );
        final journalRoot = Directory('${directory.path}/photo-mutations');
        final remainingJobs = await journalRoot.exists()
            ? await journalRoot
                  .list(recursive: true)
                  .where((file) => file is File && file.path.endsWith('.json'))
                  .toList()
            : const <FileSystemEntity>[];
        expect(remainingJobs, isEmpty);
      },
    );
  }

  test('a failed waypoint edit keeps its queued source until cancellation can be saved', () async {
    final h = await harness(true, useProductionJournal: true);
    await seedResumedCapture(h);
    final row = h.store.box<TrailEntity>().getAll().single;
    final point = row.waypoints.single;
    final journal = h.container.read(photoMutationServiceProvider);
    await journal.apply(
      target: PhotoTarget.waypoint(point.id),
      actorId: _actor,
      newPhotos: point.localPhotos,
      mode: PhotoApiMode.unknown,
    );
    h.api.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (options.method == 'POST' &&
              options.uri.path.endsWith('/waypoint/${point.id}')) {
            handler.reject(
              DioException(
                requestOptions: options,
                response: Response(requestOptions: options, statusCode: 503),
              ),
            );
          } else {
            handler.next(options);
          }
        },
      ),
    );
    final before = row.toModel();
    final edited = before.copyWith(
      expand: before.expand!.copyWith(
        waypointsViaTrail: [
          before.expand!.waypointsViaTrail!.single.copyWith(
            localPhotos: const [],
            removedPendingPhotos: point.localPhotos,
          ),
        ],
      ),
    );
    final result = await h.container
        .read(trailSaveProvider.notifier)
        .updateTrail(
          before,
          edited,
          authorId: _actor,
          newPhotos: const [],
          removedPhotoFilenames: const [],
        );
    expect(result.hadWaypointFailures, isTrue);
    expect(
      h.store.box<TrailEntity>().getAll().single.waypoints.single.localPhotos,
      point.localPhotos,
    );
    expect(
      await journal.pendingPaths(PhotoTarget.waypoint(point.id), _actor),
      hasLength(1),
    );
  });

  test(
    'a cancelled queued source stays cancelled when its database write fails',
    () async {
      final h = await harness(
        true,
        cancellationHook: (target, actor, base, paths) {
          throw StateError('Database is temporarily unavailable');
        },
      );
      await seedResumedCapture(h);
      final row = h.store.box<TrailEntity>().getAll().single;
      final journal = h.container.read(photoMutationServiceProvider);
      await journal.apply(
        target: PhotoTarget.trail(row.id),
        actorId: _actor,
        newPhotos: row.localPhotos,
        mode: PhotoApiMode.unknown,
      );
      await expectLater(
        journal.apply(
          target: PhotoTarget.trail(row.id),
          actorId: _actor,
          removedPendingPaths: row.localPhotos,
          mode: PhotoApiMode.unknown,
        ),
        throwsStateError,
      );
      expect(
        h.store.box<TrailEntity>().getAll().single.localPhotos,
        isNotEmpty,
      );
      final waypointId = row.waypoints.single.id;
      await h.container.read(trailSyncProvider.notifier).retry('local-1-1');
      expect(h.server.photos('trail', row.id), isEmpty);
      expect(h.server.photos('waypoint', waypointId), hasLength(1));
      expect(h.server.fileUploads, 1);
      expect(h.store.box<TrailEntity>().getAll(), isEmpty);
    },
  );

  for (final waypointTarget in [false, true]) {
    test(
      '${waypointTarget ? 'waypoint' : 'trail'} replacement photo remains queued after its old source is cancelled',
      () async {
        late Store callbackStore;
        final h = await harness(
          true,
          cancellationHook: (target, actor, base, paths) {
            removeQueuedPhotoSources(
              callbackStore,
              accountId: _account,
              kind: target.kind,
              targetId: target.id,
              paths: paths,
            );
          },
        );
        callbackStore = h.store;
        await seedResumedCapture(h);
        final row = h.store.box<TrailEntity>().getAll().single;
        final point = row.waypoints.single;
        if (waypointTarget) {
          row.localPhotos = [];
          h.store.box<TrailEntity>().put(row);
        } else {
          point.localPhotos = [];
          h.store.box<WaypointEntity>().put(point);
        }
        final target = waypointTarget
            ? PhotoTarget.waypoint(point.id)
            : PhotoTarget.trail(row.id);
        final oldPaths = List<String>.of(
          waypointTarget ? point.localPhotos : row.localPhotos,
        );
        final journal = h.container.read(photoMutationServiceProvider);
        await journal.apply(
          target: target,
          actorId: _actor,
          newPhotos: oldPaths,
          mode: PhotoApiMode.unknown,
        );
        final replacement = File('${directory.path}/replacement.jpg')
          ..writeAsBytesSync([7, 8, 9]);
        h.server.uploadStatus = 503;
        if (waypointTarget) {
          final before = point.toModel();
          await h.container
              .read(waypointSaveProvider.notifier)
              .updateWaypoint(
                before,
                before.copyWith(
                  localPhotos: [replacement.path],
                  removedPendingPhotos: oldPaths,
                ),
                authorId: _actor,
                mode: PhotoApiMode.assets,
              );
        } else {
          final before = row.toModel();
          await h.container
              .read(trailSaveProvider.notifier)
              .updateTrail(
                before,
                before.copyWith(localPhotos: [replacement.path]),
                authorId: _actor,
                newPhotos: [replacement],
                removedPhotoFilenames: const [],
                removedPendingPaths: oldPaths,
              );
        }
        final persisted = h.store.box<TrailEntity>().getAll().single;
        expect(
          waypointTarget
              ? persisted.waypoints.single.localPhotos
              : persisted.localPhotos,
          isEmpty,
        );
        final sync = h.container.read(trailSyncProvider.notifier);
        now = now.add(const Duration(hours: 1));
        await sync.retry('local-1-1');
        expect(
          h.store.box<TrailEntity>().getAll(),
          hasLength(1),
          reason: 'A pending journal replacement keeps its capture queued even without row source paths.',
        );
        expect(h.server.fileUploads, 0);
        h.server.uploadStatus = null;
        now = now.add(const Duration(hours: 1));
        await sync.retry('local-1-1');
        expect(h.server.fileUploads, 1);
        expect(h.server.uploadedFilenames, ['replacement.jpg']);
        expect(h.server.photos(target.kind, target.id), hasLength(1));
        expect(h.store.box<TrailEntity>().getAll(), isEmpty);
      },
    );
  }
}
