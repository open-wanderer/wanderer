import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter/foundation.dart'
    show debugPrint, visibleForTesting;
import 'package:maplibre/maplibre.dart' show LngLatBounds;
import 'package:pmtiles/pmtiles.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/services/map_source_persistence.dart';
import 'package:wanderer/services/tile_proxy_identity.dart';
import 'package:wanderer/services/tile_repository_manager.dart';
import 'package:wanderer/util/geo/xyz_tile_bounds.dart';
import 'package:wanderer/util/region/map_cache_path.dart';

/// TileJSON document for the operator's hillshade DEM source. Must stay
/// byte-identical to `hillshadeSource.url` in `assets/map/wanderer_*.json`.
///
/// It is a TileJSON URL, not an XYZ template, so the DEM redirect target must
/// be resolved from its `tiles[0]` — unlike the vector source, whose
/// `/map/style-sources` `tileUrl` is already a template.
const String kDemTileJsonUrl = 'https://tiles.mapterhorn.com/tilejson.json';

/// Loopback-only `HttpServer` serving four route families: vector/DEM tiles
/// from a downloaded region's `.pmtiles` archive, and glyphs/sprites from the
/// shared `map_cache`.
///
/// Tiles are never reverse-proxied — their per-pan volume must stay off the
/// root isolate, so a coverage miss answers 302/503 and lets MapLibre fetch
/// from the CDN itself. Glyphs and sprites are, since a style load issues only
/// a handful and write-through caching is what lets a first-run-offline map
/// render labels at all.
///
/// Every composed style bakes in one static XYZ source per kind rather than
/// adding and removing per-region `pmtiles://` sources as the camera moves.
/// That incremental reconcile used to desync its tracking sets under
/// overlapping camera-idle events; here there is nothing left to race, because
/// each request resolves its own region via [resolveRegionForTile].
///
/// Binds `InternetAddress.loopbackIPv4` on a port drawn once at install from
/// the IANA dynamic range and persisted (`tile_proxy_identity.dart`) —
/// MapLibre's ambient cache keys on the loopback URL, so a port that changes
/// every launch orphans the tile cache. Unguessability comes from that port
/// being random and from every route requiring a per-install 128-bit secret
/// path segment.
class TileProxyServer {
  final HttpServer _server;
  final Store _store;
  final String _secret;

  /// Shared glyph/sprite cache root, `<app-docs>/map_cache` — resolved once in
  /// [start] so the proxy and the render path agree on the layout.
  final String _cacheRoot;

  /// In-flight upstream glyph/sprite fetches keyed by local cache path, so a
  /// style load requesting the same range from several layers fetches once.
  final Map<String, Future<List<int>?>> _inFlightAssetFetches = {};
  final _ArchiveCache _archiveCache = _ArchiveCache();

  /// Short-TTL memo of the region table — a full `getAll()` per tile is a
  /// table scan per tile during a pan. Regions change on user actions, so a
  /// few seconds of staleness is imperceptible.
  List<RegionEntity>? _regionCache;
  DateTime? _regionCacheAt;
  static const _regionCacheTtl = Duration(seconds: 5);

  /// Short-TTL memo of the operator's upstream templates. Read per request
  /// rather than injected at construction: the proxy starts before the first
  /// `/map/style-sources` fetch completes, so a first-run-online user's
  /// templates land after the server is already serving.
  String? _vectorTemplate;
  String? _demTemplate;
  DateTime? _templatesAt;
  static const _templateCacheTtl = Duration(seconds: 30);

  /// True while a DEM TileJSON resolve is in flight, so at most one is ever
  /// outstanding. Cleared on every exit path.
  bool _demResolveInFlight = false;

  void _refreshTemplatesIfStale() {
    final now = DateTime.now();
    final at = _templatesAt;
    if (at != null && now.difference(at) < _templateCacheTtl) return;
    _vectorTemplate = readPersistedMapStyleSources(_store)?.tileUrl;
    _demTemplate = readPersistedDemTileTemplate(_store);
    _templatesAt = now;
  }

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

  TileProxyServer._(
    this._server,
    this._store,
    this._secret,
    this._cacheRoot,
  );

  /// Resolved loopback base URL, e.g.
  /// `http://127.0.0.1:54321/<32-hex-secret>`. Carries the per-install secret
  /// — never log it or surface it in UI.
  String get baseUrl => 'http://127.0.0.1:${_server.port}/$_secret';

  /// Binds and starts serving. Must run after `openStore()` and before the
  /// first composed style is built, since [baseUrl] is baked into that style.
  ///
  /// Binds the persisted port so the ambient cache survives a cold start. If
  /// it is occupied, mints and persists a fresh port (never re-minting the
  /// secret) and retries, up to 4 attempts, then falls back to an OS-assigned
  /// port — which orphans the cache for one launch only.
  static Future<TileProxyServer> start(Store store) async {
    final identity = resolveTileProxyIdentity(store);
    final secret = identity.secret;
    var port = identity.port;
    final cachePaths = await resolveGlyphSpriteCachePaths();

    HttpServer? server;
    const maxBindAttempts = 4;
    for (var attempt = 1; attempt <= maxBindAttempts; attempt++) {
      try {
        server = await HttpServer.bind(InternetAddress.loopbackIPv4, port);
        break;
      } catch (e) {
        // Catch every bind failure, not just SocketException: anything else
        // would escape `start()` and crash before `runApp`, skipping the
        // fallback above.
        debugPrint(
          'TileProxyServer: bind attempt $attempt/$maxBindAttempts on port '
          '$port failed — $e',
        );
        if (attempt == maxBindAttempts) break;
        port = mintTileProxyIdentity().port;
        persistTileProxyPort(store, port);
      }
    }

    if (server == null) {
      debugPrint(
        'TileProxyServer: exhausted $maxBindAttempts bind attempts on the '
        'persisted port; falling back to an OS-assigned port. The ambient '
        'tile cache will be orphaned this launch only.',
      );
      server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    }

    final proxy = TileProxyServer._(server, store, secret, cachePaths.root);
    proxy._serve();
    return proxy;
  }

  /// Subscribes to the request stream.
  ///
  /// `Stream.listen` with `cancelOnError: false` rather than `await for`,
  /// which terminates on the first stream error and would silently kill the
  /// proxy for the rest of the process lifetime.
  void _serve() {
    _server.listen(
      (request) {
        unawaited(
          _handle(request).catchError((Object _) {
            request.response.statusCode = HttpStatus.internalServerError;
            return request.response.close();
          }),
        );
      },
      onError: (Object e) {
        debugPrint('TileProxyServer: request stream error (continuing) — $e');
      },
      cancelOnError: false,
    );
  }

  /// Closes the server and every cached archive handle. Tests and defensive
  /// shutdown only — production keeps it alive for the process lifetime, since
  /// a loopback bind costs nothing while idle.
  Future<void> stop() async {
    await _archiveCache.closeAll();
    await _server.close(force: true);
  }

  /// Routes four families under `/<secret>/…`: `vector/{z}/{x}/{y}.pbf` and
  /// `dem/{z}/{x}/{y}.png` (redirect-or-503 only), plus
  /// `glyphs/{fontstack}/{range}.pbf` and `sprite/<fileName>` (local-first,
  /// reverse-proxied with write-through on a miss).
  ///
  /// 404/400 is reserved for requests that can never succeed: empty path,
  /// wrong secret, unknown route shape, out-of-range z/x/y, or a
  /// non-whitelisted fontstack or filename. Every retryable miss — uncovered
  /// tile, region with no package path, vanished archive, tile absent from a
  /// covering archive, uncached asset with no known upstream — goes to
  /// [_redirectOrUnavailable] instead.
  Future<void> _handle(HttpRequest request) async {
    final segments = request.uri.pathSegments;
    if (segments.isEmpty) {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }
    if (!_constantTimeEquals(segments[0], _secret)) {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    final kind = segments.length >= 2 ? segments[1] : '';

    if (kind == 'vector' || kind == 'dem') {
      if (segments.length != 5) {
        request.response.statusCode = HttpStatus.notFound;
        return request.response.close();
      }

      final z = int.tryParse(segments[2]);
      final x = int.tryParse(segments[3]);
      final y = int.tryParse(segments[4].split('.').first);

      // Bounds-check before constructing a ZXY — its constructor `assert` is
      // stripped in release builds.
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

      // Computed once and reused by every retryable-miss branch below.
      final target = _upstreamRedirectTargetFor(kind, z: z, x: x, y: y);

      final tileBounds = tileToBounds(z, x, y);
      // resolveRegionForTile is @visibleForTesting so its pure shape stays
      // testable without a live Store; this handler is its one production
      // caller.
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
        // No downloaded region covers this tile — redirect upstream rather
        // than 404: it could still succeed via the CDN.
        return _redirectOrUnavailable(request, target);
      }

      // Archive path comes ONLY from the winning region's own localFilePath,
      // validated at write time — never assembled from the request path. That
      // is what eliminates traversal structurally.
      final localFilePath = kind == 'dem'
          ? region.demPackage.target?.localFilePath
          : region.vectorPackage.target?.localFilePath;
      if (localFilePath == null) {
        return _redirectOrUnavailable(request, target);
      }

      final archive = await _archiveCache.forPath(localFilePath);
      if (archive == null) {
        // File gone mid-session (region deleted) — same retryable outcome as
        // no coverage.
        return _redirectOrUnavailable(request, target);
      }

      final tile = await archive.tile(ZXY(z, x, y).toTileId());
      List<int> bytes;
      try {
        bytes = tile.bytes();
      } on TileNotFoundException {
        // In the archive's index but absent from its data — still retryable
        // via the operator's upstream copy.
        return _redirectOrUnavailable(request, target);
      }

      // Decompressed bytes with no Content-Encoding header — safer than
      // serving compressed bytes plus the header.
      request.response.headers.contentType = ContentType.parse(
        tile.type.mimeType(),
      );
      // Without Cache-Control MapLibre treats every tile as immediately
      // expired and re-runs the whole resolve + pmtiles read per pan revisit.
      // One day balances that against a re-downloaded region's updated tiles
      // (same URLs) becoming visible.
      request.response.headers.set(
        HttpHeaders.cacheControlHeader,
        'public, max-age=86400',
      );
      request.response.add(bytes);
      return request.response.close();
    }

    if (kind == 'glyphs') {
      return _handleGlyphRequest(request, segments);
    }
    if (kind == 'sprite') {
      return _handleSpriteRequest(request, segments);
    }

    request.response.statusCode = HttpStatus.notFound;
    return request.response.close();
  }

  /// Handles `/<secret>/glyphs/{fontstack}/{range}.pbf`.
  ///
  /// A non-whitelisted fontstack or malformed range 404s without touching the
  /// filesystem. A whitelisted request is served local-first, falling back to
  /// a write-through-cached fetch of the operator's `glyphUrl`.
  Future<void> _handleGlyphRequest(
    HttpRequest request,
    List<String> segments,
  ) async {
    if (segments.length != 4) {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }
    final fontstack = segments[2];
    final range = segments[3].split('.').first;

    String localPath;
    try {
      localPath = glyphCacheFilePath(_cacheRoot, fontstack, range);
    } on ArgumentError {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    // {fontstack} carries spaces — encode only for the remote URL; the
    // on-disk directory keeps the literal name.
    final sources = readPersistedMapStyleSources(_store);
    final upstreamUrl = sources?.glyphUrl
        .replaceAll('{fontstack}', Uri.encodeComponent(fontstack))
        .replaceAll('{range}', range);

    return _serveCachedAsset(
      request,
      localPath: localPath,
      upstreamUrl: upstreamUrl,
      contentType: ContentType('application', 'x-protobuf'),
    );
  }

  /// Handles `/<secret>/sprite/<fileName>`.
  ///
  /// A non-whitelisted filename 404s without touching the filesystem.
  /// `spriteUrl` is a theme-agnostic base and the whitelisted filename already
  /// carries its `light`/`dark` variant, so no variant is appended.
  Future<void> _handleSpriteRequest(
    HttpRequest request,
    List<String> segments,
  ) async {
    if (segments.length != 3) {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }
    final fileName = segments[2];

    String localPath;
    try {
      localPath = spriteCacheFilePath(_cacheRoot, fileName);
    } on ArgumentError {
      request.response.statusCode = HttpStatus.notFound;
      return request.response.close();
    }

    final sources = readPersistedMapStyleSources(_store);
    final upstreamUrl = sources == null
        ? null
        : '${sources.spriteUrl}/$fileName';

    final contentType = fileName.endsWith('.png')
        ? ContentType('image', 'png')
        : ContentType('application', 'json');

    return _serveCachedAsset(
      request,
      localPath: localPath,
      upstreamUrl: upstreamUrl,
      contentType: contentType,
    );
  }

  /// Serves a glyph/sprite local-first, falling back to a write-through-cached
  /// upstream fetch — the one place this proxy fetches bytes itself.
  ///
  /// 1. Serve [localPath] if it exists.
  /// 2. If [upstreamUrl] is null or unsafe, answer 503, never 404: the
  ///    template may simply not have been fetched yet.
  /// 3. Otherwise fetch it (deduplicated per [localPath]), write through via
  ///    `<localPath>.part` + [File.rename], and serve. Any failure deletes the
  ///    partial file and answers 503.
  Future<void> _serveCachedAsset(
    HttpRequest request, {
    required String localPath,
    required String? upstreamUrl,
    required ContentType contentType,
  }) async {
    final cached = File(localPath);
    if (await cached.exists()) {
      final bytes = await cached.readAsBytes();
      request.response.headers.contentType = contentType;
      request.response.headers.set(
        HttpHeaders.cacheControlHeader,
        'public, max-age=86400',
      );
      request.response.add(bytes);
      return request.response.close();
    }

    final upstreamUri = upstreamUrl == null
        ? null
        : Uri.tryParse(upstreamUrl);
    if (upstreamUri == null || !isSafeRedirectTarget(upstreamUri)) {
      return _redirectOrUnavailable(request, null);
    }

    final bytes = await _fetchAndCacheAsset(localPath, upstreamUrl!);
    if (bytes == null) {
      return _redirectOrUnavailable(request, null);
    }

    request.response.headers.contentType = contentType;
    request.response.headers.set(
      HttpHeaders.cacheControlHeader,
      'public, max-age=86400',
    );
    request.response.add(bytes);
    return request.response.close();
  }

  /// Fetches [upstreamUrl] into [localPath], deduplicated so concurrent
  /// requests for the same asset produce one fetch. `null` on failure.
  Future<List<int>?> _fetchAndCacheAsset(String localPath, String upstreamUrl) {
    final inFlight = _inFlightAssetFetches[localPath];
    if (inFlight != null) return inFlight;
    final future = _downloadAndWriteThrough(localPath, upstreamUrl);
    _inFlightAssetFetches[localPath] = future;
    unawaited(
      future.whenComplete(() => _inFlightAssetFetches.remove(localPath)),
    );
    return future;
  }

  /// Downloads [upstreamUrl] and writes it through atomically — to
  /// `<localPath>.part`, then [File.rename]d — so a truncated file is never
  /// left where a later local-first read would serve it as complete.
  Future<List<int>?> _downloadAndWriteThrough(
    String localPath,
    String upstreamUrl,
  ) async {
    final partialPath = '$localPath.part';
    final client = HttpClient();
    try {
      final httpRequest = await client.getUrl(Uri.parse(upstreamUrl));
      final response = await httpRequest.close();
      if (response.statusCode != HttpStatus.ok) {
        return null;
      }
      final builder = BytesBuilder();
      await for (final chunk in response) {
        builder.add(chunk);
      }
      final bytes = builder.takeBytes();
      final file = File(localPath);
      await file.parent.create(recursive: true);
      final partialFile = File(partialPath);
      await partialFile.writeAsBytes(bytes);
      await partialFile.rename(localPath);
      return bytes;
    } catch (e) {
      debugPrint('TileProxyServer: asset fetch failed for $upstreamUrl: $e');
      final partialFile = File(partialPath);
      if (await partialFile.exists()) {
        await partialFile.delete();
      }
      return null;
    } finally {
      client.close(force: true);
    }
  }

  /// Resolves this request's upstream redirect target from the persisted
  /// templates, refreshing the TTL memo first.
  ///
  /// A DEM request with no known template kicks off a one-shot background
  /// resolve and answers `null` (503) immediately; the resolve usually
  /// completes within MapLibre's ~1s retry window. The vector template needs
  /// no such step — `tileUrl` is already an XYZ template.
  String? _upstreamRedirectTargetFor(
    String kind, {
    required int z,
    required int x,
    required int y,
  }) {
    _refreshTemplatesIfStale();
    if (kind == 'dem') {
      if (_demTemplate == null && !_demResolveInFlight) {
        _demResolveInFlight = true;
        unawaited(_resolveDemTemplate());
      }
      return buildUpstreamRedirect(_demTemplate, z: z, x: x, y: y);
    }
    return buildUpstreamRedirect(_vectorTemplate, z: z, x: x, y: y);
  }

  /// One-shot resolution of the DEM XYZ template from [kDemTileJsonUrl].
  ///
  /// Uses a plain `dart:io` client, not Dio: the proxy runs before
  /// `ProviderScope` and must not depend on the authenticated API client.
  ///
  /// Every failure — non-2xx, malformed JSON, missing `tiles`, or a candidate
  /// [buildUpstreamRedirect] rejects — is swallowed with a `debugPrint`. DEM
  /// requests keep answering 503 until a later attempt succeeds. Only a
  /// validated candidate is persisted.
  Future<void> _resolveDemTemplate() async {
    final client = HttpClient();
    try {
      final request = await client.getUrl(Uri.parse(kDemTileJsonUrl));
      final response = await request.close();
      if (response.statusCode != HttpStatus.ok) {
        debugPrint(
          'TileProxyServer: DEM TileJSON fetch returned '
          '${response.statusCode}',
        );
        return;
      }
      final body = await response.transform(utf8.decoder).join();
      final decoded = jsonDecode(body);
      if (decoded is! Map<String, dynamic>) return;
      final tiles = decoded['tiles'];
      if (tiles is! List || tiles.isEmpty) return;
      final candidate = tiles[0];
      if (candidate is! String || candidate.isEmpty) return;
      if (buildUpstreamRedirect(candidate, z: 0, x: 0, y: 0) == null) return;
      _demTemplate = candidate;
      writePersistedDemTileTemplate(_store, candidate);
    } catch (e) {
      debugPrint('TileProxyServer: DEM TileJSON resolve failed: $e');
    } finally {
      client.close(force: true);
      _demResolveInFlight = false;
    }
  }

  /// Answers a retryable miss: 302 to [target] when one is known, 503 when not.
  ///
  /// The choice is load-bearing. 503 (`Reason::Server`) backs off 1s ×3 then
  /// exponentially and IS retried; 404 (`Reason::NotFound`) backs off to
  /// `Duration::max()` and 204 is persisted as an empty tile — both terminal.
  /// Never answer a tile that might succeed later with either.
  Future<void> _redirectOrUnavailable(HttpRequest request, String? target) {
    if (target != null) {
      request.response.statusCode = HttpStatus.found;
      request.response.headers.set(HttpHeaders.locationHeader, target);
      return request.response.close();
    }
    request.response.statusCode = HttpStatus.serviceUnavailable;
    return request.response.close();
  }
}

/// Whether [uri] is safe as a redirect `Location`: absolute `http`/`https`,
/// non-empty host, not `localhost` or a loopback/link-local address.
///
/// Rejecting local targets prevents the proxy redirecting into itself — an
/// infinite loop that would exhaust OkHttp's 20-hop limit — and stops a
/// tampered or stale template turning it into a relay to another local
/// listener.
@visibleForTesting
bool isSafeRedirectTarget(Uri uri) {
  if (!uri.hasScheme || (uri.scheme != 'http' && uri.scheme != 'https')) {
    return false;
  }
  if (uri.host.isEmpty || uri.host == 'localhost') return false;
  final address = InternetAddress.tryParse(uri.host);
  if (address != null) {
    // `0.0.0.0` / `::` are the unspecified addresses: neither loopback nor
    // link-local, so the checks below miss them, yet they resolve to
    // localhost — the same local relay this guard exists to prevent.
    if (address == InternetAddress.anyIPv4 ||
        address == InternetAddress.anyIPv6) {
      return false;
    }
    if (address.isLoopback || address.isLinkLocal) return false;
  }
  return true;
}

/// Substitutes `{z}`/`{x}`/`{y}` into [template] and validates the result,
/// returning `null` if the template is empty, missing a token, unparseable
/// once substituted, or rejected by [isSafeRedirectTarget].
///
/// Validation runs against the substituted URI, so a host that is only
/// malformed once tokens are filled is still caught. The vector template
/// legitimately carries a `?key=` query string; substitution leaves it alone.
@visibleForTesting
String? buildUpstreamRedirect(
  String? template, {
  required int z,
  required int x,
  required int y,
}) {
  if (template == null || template.isEmpty) return null;
  if (!template.contains('{z}') ||
      !template.contains('{x}') ||
      !template.contains('{y}')) {
    return null;
  }
  final substituted = template
      .replaceAll('{z}', '$z')
      .replaceAll('{x}', '$x')
      .replaceAll('{y}', '$y');
  final uri = Uri.tryParse(substituted);
  if (uri == null) return null;
  if (!isSafeRedirectTarget(uri)) return null;
  return substituted;
}

/// Constant-time equality for the request's secret path segment. Compares
/// lengths once, then XOR-accumulates every code unit with no early exit — an
/// early return would let a co-resident process time-oracle the secret one
/// character at a time.
bool _constantTimeEquals(String a, String b) {
  if (a.length != b.length) return false;
  var accumulator = 0;
  for (var i = 0; i < a.length; i++) {
    accumulator |= a.codeUnitAt(i) ^ b.codeUnitAt(i);
  }
  return accumulator == 0;
}

/// Bounded LRU cache of open [PmTilesArchive] handles keyed by absolute path —
/// each open re-reads the archive header and root directory, so never reopen
/// per request. Capped so many distinct regions in one session cannot leave an
/// ever-growing set of open handles. Treats a vanished file as a miss.
class _ArchiveCache {
  static const int _capacity = 8;

  /// Insertion-ordered, doubling as the LRU recency list: [forPath] re-inserts
  /// on every hit, so `_open.keys.first` is the least-recently-used entry.
  final Map<String, PmTilesArchive> _open = {};

  /// Opens currently in flight, keyed by path.
  ///
  /// Without this, two concurrent requests for the same uncached path both
  /// miss, both open a handle, and the second assignment orphans the first
  /// without closing it. MapLibre bursts tile requests when the camera enters
  /// a region, so that race is the normal case.
  final Map<String, Future<PmTilesArchive?>> _opening = {};

  /// Returns the archive at [path], opening (and caching) it if not already
  /// cached. Returns `null` when [path] no longer exists on disk.
  Future<PmTilesArchive?> forPath(String path) async {
    if (!File(path).existsSync()) {
      await _evict(path);
      return null;
    }

    // Hit: remove + re-insert to move this entry to most-recently-used.
    final cached = _open.remove(path);
    if (cached != null) {
      _open[path] = cached;
      return cached;
    }

    final inFlight = _opening[path];
    if (inFlight != null) return inFlight;

    final future = _openAndCache(path);
    _opening[path] = future;
    return future;
  }

  /// Opens [path] and installs it in [_open], evicting down to [_capacity]
  /// first. Always clears its own [_opening] entry.
  Future<PmTilesArchive?> _openAndCache(String path) async {
    try {
      final archive = await PmTilesArchive.fromFile(File(path));

      // Never overwrite a live handle: if another caller installed one while
      // this open was in flight, close the loser rather than orphaning it.
      final existing = _open[path];
      if (existing != null) {
        await archive.close();
        return existing;
      }

      // Evict AFTER a successful open, so a failed open never costs a live
      // handle. Briefly exceeding [_capacity] beats closing a handle a
      // concurrent request is about to read.
      while (_open.length >= _capacity) {
        await _evict(_open.keys.first);
      }

      _open[path] = archive;
      return archive;
    } finally {
      _opening.remove(path);
    }
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
