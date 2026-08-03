import 'dart:io';

import 'package:dio/dio.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/glyph_sprite_cache_paths.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/map_style_sources_provider.dart';
import 'package:wanderer/util/region/map_cache_path.dart';

part 'glyph_sprite_cache_provider.g.dart';

/// Number of 256-codepoint glyph ranges spanning the full 0-65535 codepoint
/// space, so all 4 fontstacks are cached across the complete range set.
/// Ranges the font does not cover simply 404 and are skipped.
const int _rangeCount = 256;

/// Max concurrent downloads. Caps socket usage so warming the full
/// 4-fontstack × 256-range set does not open hundreds of connections at once.
const int _maxConcurrentDownloads = 8;

/// The files that make up one sprite variant, keyed by the suffix the MapLibre
/// sprite loader appends to the sprite base. Both the 1x (`.json`/`.png`) and
/// 2x (`@2x.json`/`@2x.png`) pairs are cached: MapLibre Native requests the
/// pixel-ratio-appropriate pair, so a hi-dpi device fetches `@2x.json` +
/// `@2x.png`. Omitting `@2x.json` makes the offline sprite load fail on hi-dpi
/// devices ("Failed to load sprite"), dropping every map icon.
const List<String> _spriteSuffixes = <String>[
  '.json',
  '.png',
  '@2x.json',
  '@2x.png',
];

/// Resolves the on-disk layout of the shared glyph/sprite cache under
/// `<app-docs>/map_cache` — pure path construction, no network and no
/// downloads. Shared by [GlyphSpriteCache] (which then populates these paths
/// online) and [offlineGlyphSpritePaths] (which returns them as-is for the
/// network-free offline render path).
Future<GlyphSpriteCachePaths> resolveGlyphSpriteCachePaths() async {
  final docs = await getApplicationDocumentsDirectory();
  final root = p.join(docs.path, 'map_cache');
  return GlyphSpriteCachePaths(
    root: root,
    glyphBase: p.join(root, 'glyphs'),
    spriteLightBase: spriteCacheBasePath(root, dark: false),
    spriteDarkBase: spriteCacheBasePath(root, dark: true),
  );
}

/// The offline counterpart to [GlyphSpriteCache]: returns the local
/// glyph/sprite cache paths **without any network call or download**. Used by
/// the offline map render path, where the cache was already populated online at
/// download time and the style rewriter only needs the `file://` bases. Kept
/// separate from [GlyphSpriteCache] so an offline map open never awaits (or
/// hangs on) `mapStyleSourcesProvider`.
@Riverpod(keepAlive: true)
Future<GlyphSpriteCachePaths> offlineGlyphSpritePaths(Ref ref) {
  return resolveGlyphSpriteCachePaths();
}

/// The one shared app-wide glyph/sprite cache.
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op). Every local path is built via
/// the path-safety helpers in `map_cache_path.dart`, so no operator-controlled
/// token is ever concatenated into a path.
@Riverpod(keepAlive: true)
class GlyphSpriteCache extends _$GlyphSpriteCache {
  @override
  Future<GlyphSpriteCachePaths> build() async {
    final sources = await ref.watch(mapStyleSourcesProvider.future);
    final api = ref.read(apiProvider);

    final paths = await resolveGlyphSpriteCachePaths();
    final root = paths.root;

    // Build the full download job list (glyphs + both sprite variants). Every
    // local path comes from map_cache_path.dart — never string-concatenated.
    final jobs = <_DownloadJob>[];

    for (final fontstack in allowedFontstacks) {
      for (var i = 0; i < _rangeCount; i++) {
        final start = i * 256;
        final range = '$start-${start + 255}';
        final localPath = glyphCacheFilePath(root, fontstack, range);
        // {fontstack} carries spaces — encode only for the REMOTE fetch URL; the
        // on-disk directory keeps the literal name so file:// substitution
        // matches at render time.
        final url = sources.glyphUrl
            .replaceAll('{fontstack}', Uri.encodeComponent(fontstack))
            .replaceAll('{range}', range);
        jobs.add(_DownloadJob(url: url, localPath: localPath));
      }
    }

    for (final dark in <bool>[false, true]) {
      final localBase = spriteCacheBasePath(root, dark: dark);
      final remoteBase = '${sources.spriteUrl}/${dark ? 'dark' : 'light'}';
      for (final suffix in _spriteSuffixes) {
        jobs.add(
          _DownloadJob(
            url: '$remoteBase$suffix',
            localPath: '$localBase$suffix',
          ),
        );
      }
    }

    await _runPooled(api, jobs, _maxConcurrentDownloads);
    return paths;
  }

  /// Download [job] unless its file already exists on disk (idempotent skip).
  /// A missing range (404) or transient failure is swallowed best-effort —
  /// not every fontstack covers every Unicode range, and a glyph miss must
  /// not fail the whole warm.
  Future<void> _download(Dio api, _DownloadJob job) async {
    if (await File(job.localPath).exists()) return;
    final dir = Directory(p.dirname(job.localPath));
    if (!await dir.exists()) {
      await dir.create(recursive: true);
    }
    try {
      await api.download(job.url, job.localPath);
    } on DioException {
      // Best-effort: clean up any partial file so a later run retries cleanly.
      final partial = File(job.localPath);
      if (await partial.exists()) {
        await partial.delete();
      }
    }
  }

  /// Run [jobs] with at most [concurrency] downloads in flight at once. The
  /// shared iterator is only advanced synchronously (between awaits), so no two
  /// workers ever claim the same job.
  Future<void> _runPooled(
    Dio api,
    List<_DownloadJob> jobs,
    int concurrency,
  ) async {
    final iterator = jobs.iterator;
    Future<void> worker() async {
      while (iterator.moveNext()) {
        await _download(api, iterator.current);
      }
    }

    await Future.wait(<Future<void>>[
      for (var i = 0; i < concurrency; i++) worker(),
    ]);
  }
}

/// A single glyph/sprite file to fetch: a remote URL and its on-disk target.
class _DownloadJob {
  const _DownloadJob({required this.url, required this.localPath});

  final String url;
  final String localPath;
}
