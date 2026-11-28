import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:dio/dio.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:objectbox/objectbox.dart';
import 'package:path_provider_platform_interface/path_provider_platform_interface.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/services/trail_download_service.dart';
import 'package:wanderer/objectbox.g.dart' as database;
import 'package:wanderer/store/app_photo_paths.dart';
import 'package:wanderer/util/image_cache.dart';

const _account = 'ddddddddddddddd';

/// Returns real response bytes and records every URL the download service uses.
class _PhotoDownloadServer implements HttpClientAdapter {
  final originals = <String, List<int>>{};
  final statuses = <String, int>{};
  final requests = <RequestOptions>[];
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    expect(options.method, 'GET');
    final bytes = originals[options.uri.path];
    expect(bytes, isNotNull, reason: 'Unexpected photo URL ${options.uri}');
    return ResponseBody.fromBytes(
      bytes!,
      statuses[options.uri.path] ?? 200,
      headers: {
        Headers.contentLengthHeader: ['${bytes.length}'],
        Headers.contentTypeHeader: [
          options.uri.path.endsWith('.svg') ? 'image/svg+xml' : 'image/jpeg',
        ],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

class _DownloadServer implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    if (options.uri.host == 'a.example' &&
        options.uri.path.endsWith('redirect.jpg')) {
      return ResponseBody.fromBytes(
        [],
        302,
        headers: {
          'location': ['https://sub.a.example/photo.jpg'],
        },
      );
    }
    return ResponseBody.fromBytes(
      [7, 8, 9],
      200,
      headers: {
        Headers.contentLengthHeader: ['3'],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

class _Documents extends PathProviderPlatform {
  _Documents(this.path);
  final String path;
  @override
  Future<String?> getApplicationDocumentsPath() async => path;
}

/// Queries in this fixture address one row per entity type. The production
/// query and bookkeeping code run normally against these in-memory boxes.
class _MemoryStore extends Fake implements Store {
  final _boxes = <Type, Object>{};
  @override
  Box<T> box<T>() => _boxes.putIfAbsent(T, _MemoryBox<T>.new) as Box<T>;
  @override
  R runInTransaction<R>(TxMode mode, R Function() work) => work();
}

class _MemoryBox<T> extends Fake implements Box<T> {
  final rows = <T>[];
  @override
  int put(T object, {PutMode mode = PutMode.put}) {
    final row = object as dynamic;
    if (row.obxId == 0) row.obxId = rows.length + 1;
    final previous = rows.indexWhere(
      (entry) => (entry as dynamic).id == row.id,
    );
    if (previous >= 0) {
      row.obxId = (rows[previous] as dynamic).obxId;
      rows[previous] = object;
    } else {
      rows.add(object);
    }
    return row.obxId as int;
  }

  @override
  List<T> getAll() => List.of(rows);
  @override
  bool remove(int id) {
    final count = rows.length;
    rows.removeWhere((row) => (row as dynamic).obxId == id);
    return rows.length != count;
  }

  @override
  QueryBuilder<T> query([Condition<T>? condition]) => _MemoryBuilder<T>(rows);
}

class _MemoryBuilder<T> extends Fake implements QueryBuilder<T> {
  _MemoryBuilder(this.rows);
  final List<T> rows;
  @override
  Query<T> build() => _MemoryQuery<T>(rows);
}

class _MemoryQuery<T> extends Fake implements Query<T> {
  _MemoryQuery(this.rows);
  final List<T> rows;
  @override
  List<T> find() => List.of(rows);
  @override
  T? findFirst() => rows.firstOrNull;
  @override
  void close() {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  late Directory directory;
  late PathProviderPlatform previousPaths;
  setUp(() {
    directory = Directory.systemTemp.createTempSync('trail-photo-download-');
    previousPaths = PathProviderPlatform.instance;
    PathProviderPlatform.instance = _Documents(directory.path);
  });
  tearDown(() async {
    PathProviderPlatform.instance = previousPaths;
    await directory.delete(recursive: true);
  });

  ({Dio api, _PhotoDownloadServer server, _MemoryStore store}) harness(
    bool assets,
  ) {
    final server = _PhotoDownloadServer();
    final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'))
      ..httpClientAdapter = server;
    addTearDown(api.close);
    return (api: api, server: server, store: _MemoryStore());
  }

  for (final assets in [true, false]) {
    test(
      '${assets ? 'asset' : 'legacy'} download replay persists resolved photo bytes',
      () async {
        final h = harness(assets);
        const id = 'trail0000000001';
        const firstAsset = 'asset0000000001';
        const secondAsset = 'asset0000000002';
        final photos = assets
            ? [
                '/api/v1/assets/$firstAsset/file',
                '/api/v1/assets/$secondAsset/file',
                '/api/v1/files/assets/asset0000000003/preview.svg',
              ]
            : ['one.jpg', 'two.jpg', 'preview.svg'];
        final expected = <List<int>>[
          [1, 2, 3],
          [4, 5, 6],
          utf8.encode('<svg xmlns="http://www.w3.org/2000/svg"/>'),
        ];
        for (var index = 0; index < photos.length; index++) {
          final url = assets
              ? photos[index]
              : '/api/v1/files/trails/$id/${photos[index]}';
          h.server.originals[url] = expected[index];
        }
        final model = Trail.empty().copyWith(
          id: id,
          name: 'Downloaded hike',
          photos: photos,
        );
        await TrailDownloadService(
          h.store,
          h.api,
          photoApi: h.api,
        ).downloadTrail(model, savedByUserId: _account);
        final saved = h.store.box<TrailEntity>().getAll().single;
        expect(saved.photos, hasLength(3));
        expect(saved.photos.toSet(), hasLength(3));
        expect(
          saved.photos.where((path) => path.endsWith('.svg')),
          hasLength(1),
        );
        for (var index = 0; index < expected.length; index++) {
          expect(
            await File(resolveStoredPhotoPath(saved.photos[index]))
                .readAsBytes(),
            expected[index],
          );
        }
        final downloads = h.server.requests
            .where((request) => request.method == 'GET')
            .toList();
        expect(downloads, hasLength(3));
        expect(
          downloads.every((request) => request.uri.host == 'a.example'),
          isTrue,
        );
        expect(
          downloads.every((request) => request.uri.path.startsWith('/api/v1/')),
          isTrue,
        );
      },
    );
  }

  test('updating downloads gives different assets distinct paths and removes old files', () async {
    final h = harness(true);
    final service = TrailDownloadService(h.store, h.api, photoApi: h.api);
    const first = '/api/v1/assets/asset0000000001/file';
    const second = '/api/v1/assets/asset0000000002/file';
    const replacement = '/api/v1/assets/asset0000000003/file';
    h.server.originals.addAll({
      first: [1],
      second: [2],
      replacement: [3],
    });
    final model = Trail.empty().copyWith(
      id: 'trail0000000001',
      photos: [first, second],
    );
    await service.downloadTrail(model);
    final previous = List<String>.of(
      h.store.box<TrailEntity>().getAll().single.photos,
    );
    await service.downloadTrail(model.copyWith(photos: [second, first]));
    final reordered = h.store.box<TrailEntity>().getAll().single.photos;
    expect(reordered, [previous[1], previous[0]]);
    await service.downloadTrail(model.copyWith(photos: [replacement]));
    final current = h.store.box<TrailEntity>().getAll().single.photos.single;
    expect(previous, isNot(contains(current)));
    expect(await File(resolveStoredPhotoPath(current)).readAsBytes(), [3]);
    expect(
      previous.every(
        (path) => !File(resolveStoredPhotoPath(path)).existsSync(),
      ),
      isTrue,
    );
  });

  test('replacing a cached file evicts its decoded image', () async {
    final h = harness(true);
    const url = '/api/v1/assets/asset0000000001/file';
    h.server.originals[url] = [1];
    final service = TrailDownloadService(h.store, h.api, photoApi: h.api);
    final model = Trail.empty().copyWith(id: 'trail0000000001', photos: [url]);
    await service.downloadTrail(model);
    final path = h.store.box<TrailEntity>().getAll().single.photos.single;
    final key = FileImage(File(resolveStoredPhotoPath(path)));
    final cache = PaintingBinding.instance.imageCache;
    cache.putIfAbsent(
      key,
      () => OneFrameImageStreamCompleter(Completer<ImageInfo>().future),
    );
    expect(cache.containsKey(key), isTrue);
    h.server.originals[url] = [2];
    await service.downloadTrail(model);
    expect(cache.containsKey(key), isFalse);
    expect(await File(resolveStoredPhotoPath(path)).readAsBytes(), [2]);
  });

  for (final status in [300, 302, 304, 307]) {
    test('a final $status response is not saved as a photo', () async {
      final h = harness(true);
      const url = '/api/v1/bad.jpg';
      h.server.originals[url] = utf8.encode('<html>moved</html>');
      h.server.statuses[url] = status;
      await TrailDownloadService(h.store, h.api, photoApi: h.api).downloadTrail(
        Trail.empty().copyWith(id: 'trail0000000001', photos: [url]),
      );
      expect(h.store.box<TrailEntity>().getAll().single.photos, isEmpty);
      final photoFiles = Directory(directory.path)
          .listSync(recursive: true)
          .whereType<File>();
      expect(photoFiles, isEmpty);
    });
  }

  test('downloaded waypoint copies preserve pending photos and uncreated children', () async {
    final h = harness(true);
    const trailId = 'trail0000000001';
    const pointId = 'point0000000001';
    const url = '/api/v1/assets/asset0000000001/file';
    h.server.originals[url] = [1, 2, 3];
    final pending = File(
      '${directory.path}/unsynced/$_account/local-1-1/waypoints/local-2-1/pending.jpg',
    );
    await pending.parent.create(recursive: true);
    await pending.writeAsBytes([8]);
    final old = TrailEntity.fromModel(Trail.empty().copyWith(id: trailId));
    old.localId = 'local-1-1';
    old.owner = _account;
    old.syncState = TrailSyncState.pending;
    final created = WaypointEntity(
      id: pointId,
      lat: 1,
      lon: 2,
      author: 'actor',
      created: DateTime(2026),
      updated: DateTime(2026),
      localKey: 'local-2-1',
      localPhotos: [pending.path],
      photos: [url],
    )..obxId = 17;
    final uncreated = WaypointEntity(
      id: 'local-3-1',
      lat: 3,
      lon: 4,
      author: 'actor',
      created: DateTime(2026),
      updated: DateTime(2026),
      localKey: 'local-3-1',
    );
    old.waypoints.addAll([created, uncreated]);
    h.store.box<TrailEntity>().put(old);
    final remote = Trail.empty().copyWith(
      id: trailId,
      expand: TrailExpand(
        waypointsViaTrail: [
          Waypoint(
            id: pointId,
            lat: 1,
            lon: 2,
            created: DateTime(2026),
            updated: DateTime(2026),
            photos: [url],
          ),
        ],
      ),
    );
    await TrailDownloadService(
      h.store,
      h.api,
      photoApi: h.api,
    ).downloadTrail(remote, savedByUserId: _account);
    final saved = h.store.box<TrailEntity>().getAll().single;
    expect(saved.obxId, old.obxId);
    expect(saved.waypoints, hasLength(2));
    final point = saved.waypoints.singleWhere(
      (waypoint) => waypoint.id == pointId,
    );
    expect(point.obxId, 17);
    expect(point.localKey, 'local-2-1');
    expect(point.localPhotos, [pending.path]);
    expect(point.photos, [url]);
    expect(point.downloadedPhotos, hasLength(1));
    expect(point.downloadedPhotos, isNot(contains(pending.path)));
    expect(
      saved.waypoints.any((waypoint) => waypoint.id == 'local-3-1'),
      isTrue,
    );
    expect(await pending.readAsBytes(), [8]);
  });

  test('download refresh prunes missing waypoint rows but preserves omitted expands', () async {
    final store = await database.openStore(
      directory: '${directory.path}/database',
    );
    addTearDown(store.close);
    final server = _PhotoDownloadServer();
    final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'))
      ..httpClientAdapter = server;
    addTearDown(api.close);
    const firstPhoto = '/api/v1/assets/asset0000000001/file';
    const secondPhoto = '/api/v1/assets/asset0000000002/file';
    server.originals.addAll({
      firstPhoto: [1],
      secondPhoto: [2],
    });
    Waypoint point(String id, String photo) => Waypoint(
      id: id,
      lat: 1,
      lon: 2,
      created: DateTime(2026),
      updated: DateTime(2026),
      photos: [photo],
    );
    final first = point('point0000000001', firstPhoto);
    final second = point('point0000000002', secondPhoto);
    final model = Trail.empty().copyWith(
      id: 'trail0000000001',
      expand: TrailExpand(waypointsViaTrail: [first, second]),
    );
    final service = TrailDownloadService(store, api, photoApi: api);
    await service.downloadTrail(model);
    final original = store.box<TrailEntity>().getAll().single;
    final removedPath = original.waypoints
        .singleWhere((point) => point.id == second.id)
        .downloadedPhotos
        .single;
    await service.downloadTrail(
      model.copyWith(expand: TrailExpand(waypointsViaTrail: [first])),
    );
    final refreshed = store.box<TrailEntity>().get(original.obxId)!;
    expect(refreshed.waypoints.map((point) => point.id), [first.id]);
    expect(store.box<WaypointEntity>().getAll(), hasLength(1));
    expect(File(resolveStoredPhotoPath(removedPath)).existsSync(), isFalse);
    final keptPath = refreshed.waypoints.single.downloadedPhotos.single;
    await service.downloadTrail(model.copyWith(expand: null));
    final withoutExpand = store.box<TrailEntity>().get(original.obxId)!;
    expect(withoutExpand.waypoints.single.id, first.id);
    expect(File(resolveStoredPhotoPath(keptPath)).existsSync(), isTrue);
  });

  test('offline download keeps the selected cover first', () async {
    final h = harness(true);
    const firstUrl = '/api/v1/assets/asset0000000001/file';
    const selectedUrl = '/api/v1/assets/asset0000000002/file';
    h.server.originals[firstUrl] = [1, 2, 3];
    h.server.originals[selectedUrl] = [4, 5, 6];
    final model = Trail.empty().copyWith(
      id: 'trail0000000001',
      name: 'Selected cover',
      photos: [firstUrl, selectedUrl],
      thumbnail: 1,
    );
    await TrailDownloadService(
      h.store,
      h.api,
      photoApi: h.api,
    ).downloadTrail(model);
    final saved = h.store.box<TrailEntity>().getAll().single;
    expect(saved.photos, hasLength(2));
    expect(
      await File(resolveStoredPhotoPath(saved.photos.first)).readAsBytes(),
      [4, 5, 6],
    );
    expect(
      await File(resolveStoredPhotoPath(saved.photos.last)).readAsBytes(),
      [1, 2, 3],
    );
    expect(
      saved.toModel().localPhotos.first,
      resolveStoredPhotoPath(saved.photos.first),
    );
  });

  test(
    'offline photo cookies stay on the configured host across redirects',
    () async {
      final server = _DownloadServer();
      final api = Dio(BaseOptions(baseUrl: 'https://a.example/api/v1'))
        ..httpClientAdapter = server;
      final store = _MemoryStore();
      final jar = CookieJar();
      await jar.saveFromResponse(Uri.parse('https://a.example'), [
        Cookie('pb_auth', 'session')
          ..domain = 'a.example'
          ..path = '/'
          ..secure = true,
      ]);
      expect(
        await jar.loadForRequest(Uri.parse('https://sub.a.example/photo.jpg')),
        isNotEmpty,
      );
      final previousJar = WandererImageCacheManager.cookieJar;
      final previousHost = WandererImageCacheManager.serverHost;
      WandererImageCacheManager.cookieJar = jar;
      WandererImageCacheManager.serverHost = 'a.example';
      try {
        final model = Trail.empty().copyWith(
          id: 'trail0000000001',
          name: 'Private hike',
          photos: [
            '/api/v1/direct.jpg',
            '/api/v1/redirect.jpg',
            'https://sub.a.example/foreign.jpg',
          ],
        );
        await TrailDownloadService(
          store,
          api,
          photoApi: api,
        ).downloadTrail(model);
        final local = server.requests.where(
          (request) => request.uri.host == 'a.example',
        );
        expect(local, hasLength(2));
        expect(
          local.every(
            (request) => request.headers['cookie'] == 'pb_auth=session',
          ),
          isTrue,
        );
        final other = server.requests.where(
          (request) => request.uri.host == 'sub.a.example',
        );
        expect(other, hasLength(2));
        expect(
          other.every((request) => !request.headers.containsKey('cookie')),
          isTrue,
        );
        expect(store.box<TrailEntity>().getAll().single.photos, hasLength(3));
      } finally {
        WandererImageCacheManager.cookieJar = previousJar;
        WandererImageCacheManager.serverHost = previousHost;
        api.close();
      }
    },
  );
}
