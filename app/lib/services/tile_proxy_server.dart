import 'dart:async';
import 'dart:io';

import 'package:maplibre/maplibre.dart' show LngLatBounds;
import 'package:pmtiles/pmtiles.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/services/tile_repository_manager.dart';
import 'package:wanderer/util/geo/xyz_tile_bounds.dart';

/// Loopback-only `HttpServer` that serves vector/DEM map tiles from a
/// downloaded region's `.pmtiles` archive.
///
/// Both `TrailMap` and `navigation_screen` bake a single STATIC
/// `tiles: ['<baseUrl>/vector/{z}/{x}/{y}.pbf']` /
/// `['<baseUrl>/dem/{z}/{x}/{y}.png']` XYZ source into every composed offline
/// style (see `offline_style_rewriter.dart`'s `rewriteStyleForProxy`) instead
/// of incrementally `addSource`/`removeSource`-ing per-region `pmtiles://`
/// archives. That incremental reconcile (an earlier
/// `_reconcileRegionComposition` in `navigation_screen.dart`) had no
/// reentrancy guard and desynced its tracking sets from the real native style
/// under overlapping camera-idle events. Routing every tile request
/// through this single static-source server structurally eliminates that bug
/// class: there is no reconcile call left to race, because MapLibre Native's
/// own viewport tracking decides which tiles to request, and every request
/// resolves its winning region fresh, per-request, via [resolveRegionForTile].
///
/// Binds `InternetAddress.loopbackIPv4` on an OS-assigned ephemeral port
/// (never the wildcard-bind address, never a fixed port):
/// the server must never be reachable from the LAN/hotspot, and a co-resident
/// app must not be
/// able to pre-target a guessable port.
class TileProxyServer {
  final HttpServer _server;
  final Store _store;
  final _ArchiveCache _archiveCache = _ArchiveCache();

  /// Short-TTL memo of the region table. Every tile request used to run a
  /// full `RegionEntity` `getAll()` — during a pan that's an ObjectBox table
  /// scan per tile. Regions change on the order of user actions (download /
  /// delete), so a few seconds of staleness is imperceptible: a deleted
  /// region's file-vanished path already 404s via [_ArchiveCache.forPath],
  /// and a fresh download's tiles appear within the TTL.
  List<RegionEntity>? _regionCache;
  DateTime? _regionCacheAt;
  static const _regionCacheTtl = Duration(seconds: 5);

  List<RegionEntity> _regions() {
    final now = DateTime.now();
    final cachedAt = _regionCacheAt;
    final cached = _regionCache;
    if (cached != null &&
        cachedAt != null &&
        now.difference(cachedAt) < _regionCacheTtl) {
      return cached;
    }
    final fresh = _store.box<RegionEntity>().getAll();
    _regionCache = fresh;
    _regionCacheAt = now;
    return fresh;
  }

  TileProxyServer._(this._server, this._store);

  /// The resolved loopback base URL, e.g. `http://127.0.0.1:54321` — exposed
  /// to the widget tree via the [tile_proxy_provider.dart] `keepAlive`
  /// provider, overridden in `main.dart` immediately after this starts.
  String get baseUrl => 'http://127.0.0.1:${_server.port}';

  /// Binds the loopback server and starts serving. Must be called after
  /// `openStore()` (the request handler queries [RegionEntity] rows) and
  /// before the first composed style is built, since the resolved [baseUrl]
  /// is baked into that style's tile source templates.
  static Future<TileProxyServer> start(Store store) async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    final proxy = TileProxyServer._(server, store);
    unawaited(proxy._serve());
    return proxy;
  }

  Future<void> _serve() async {
    await for (final request in _server) {
      unawaited(
        _handle(request).catchError((Object _) {
          request.response.statusCode = HttpStatus.internalServerError;
          return request.response.close();
        }),
      );
    }
  }

  /// Closes the server and every cached archive handle. Used by tests /
  /// defensive shutdown — production keeps the server alive for the process
  /// lifetime (matches the `keepAlive` ObjectBox [Store] precedent; no
  /// app-lifecycle-driven start/stop, since loopback binding has no
  /// battery/network cost while idle).
  Future<void> stop() async {
    await _archiveCache.closeAll();
    await _server.close(force: true);
  }

  /// Routes `/vector/{z}/{x}/{y}.pbf` and `/dem/{z}/{x}/{y}.png`. Any other
  /// shape, an out-of-range z/x/y, an unresolvable region, or a missing tile
  /// all resolve to the same user-visible outcome as today's blank-basemap
  /// behavior: HTTP 404, empty body.
  Future<void> _handle(HttpRequest request) async {
    final segments = request.uri.pathSegments;
    if (segments.length != 4) {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    final kind = segments[0];
    if (kind != 'vector' && kind != 'dem') {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    final z = int.tryParse(segments[1]);
    final x = int.tryParse(segments[2]);
    final y = int.tryParse(segments[3].split('.').first);

    // Explicit bounds check BEFORE constructing a ZXY — never rely on ZXY's
    // constructor `assert`, which is stripped in release builds
    // — trusting that assert is an anti-pattern.
    if (z == null ||
        x == null ||
        y == null ||
        z < 0 ||
        z > 26 ||
        x < 0 ||
        x >= (1 << z) ||
        y < 0 ||
        y >= (1 << z)) {
      request.response.statusCode = HttpStatus.badRequest;
      return request.response.close();
    }

    final tileBounds = tileToBounds(z, x, y);
    // resolveRegionForTile is @visibleForTesting so its pure-function shape
    // stays unit-testable without a live Store (matches bboxOverlaps'/
    // splitRegionTilePaths' precedent) — this proxy handler is its one
    // sanctioned production caller, mirroring the pmtiles package's own
    // `fromReadAt` cross-file @visibleForTesting usage
    // (pmtiles-1.2.0/lib/src/archive.dart).
    // ignore: invalid_use_of_visible_for_testing_member
    final region = resolveRegionForTile(
      _regions(),
      LngLatBounds(
        longitudeWest: tileBounds.west,
        longitudeEast: tileBounds.east,
        latitudeSouth: tileBounds.south,
        latitudeNorth: tileBounds.north,
      ),
      dem: kind == 'dem',
    );

    if (region == null) {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    // The archive path is read ONLY from the winning region's own
    // DownloadedTilePackageEntity.localFilePath — a DB-derived value already
    // validated at write time via util/region/file_path.dart's
    // assertValidRegionPath, NEVER assembled from the request path. This
    // structurally eliminates path traversal.
    final localFilePath = kind == 'dem'
        ? region.demPackage.target?.localFilePath
        : region.vectorPackage.target?.localFilePath;
    if (localFilePath == null) {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    final archive = await _archiveCache.forPath(localFilePath);
    if (archive == null) {
      // The winning region's file no longer exists on disk (mid-session
      // delete) — same 404 outcome as no coverage.
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    final tile = await archive.tile(ZXY(z, x, y).toTileId());
    List<int> bytes;
    try {
      bytes = tile.bytes();
    } on TileNotFoundException {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    // Serve decompressed bytes with NO Content-Encoding header (safer
    // default than serving compressed bytes + Content-Encoding: gzip).
    request.response.headers.contentType = ContentType.parse(
      tile.type.mimeType(),
    );
    // Without a Cache-Control header MapLibre treats every offline tile as
    // immediately expired and re-runs the whole HTTP → region-resolve →
    // pmtiles read per pan revisit; with one, its ambient cache serves
    // revisits directly. One day balances that against a re-downloaded
    // region's updated tiles (same URLs) becoming visible.
    request.response.headers.set(
      HttpHeaders.cacheControlHeader,
      'public, max-age=86400',
    );
    request.response.add(bytes);
    return request.response.close();
  }
}

/// Bounded LRU-style cache of open [PmTilesArchive] handles, keyed by
/// absolute file path — never reopen `PmTilesArchive.fromFile` per request
/// (each open re-reads/parses the archive header and root directory).
/// Capped at a small fixed size so an unbounded number of distinct regions
/// visited in one session can't leave an ever-growing set of open file
/// handles. Evicts (and treats as a cache miss) any path whose
/// backing file no longer exists on disk, covering a mid-session region
/// delete.
class _ArchiveCache {
  static const int _capacity = 8;

  final Map<String, PmTilesArchive> _open = {};

  /// Returns the archive at [path], opening (and caching) it if not already
  /// cached. Returns `null` when [path] no longer exists on disk.
  Future<PmTilesArchive?> forPath(String path) async {
    if (!File(path).existsSync()) {
      await _evict(path);
      return null;
    }

    final cached = _open[path];
    if (cached != null) return cached;

    if (_open.length >= _capacity) {
      final oldestPath = _open.keys.first;
      await _evict(oldestPath);
    }

    final archive = await PmTilesArchive.fromFile(File(path));
    _open[path] = archive;
    return archive;
  }

  Future<void> _evict(String path) async {
    final archive = _open.remove(path);
    await archive?.close();
  }

  Future<void> closeAll() async {
    for (final archive in _open.values) {
      await archive.close();
    }
    _open.clear();
  }
}
