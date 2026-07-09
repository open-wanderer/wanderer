import 'dart:io';

import 'package:dio/dio.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/glyph_sprite_cache_paths.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/map_style_sources_provider.dart';
import 'package:wanderer/util/map_cache_path.dart';

part 'glyph_sprite_cache_provider.g.dart';

/// Number of 256-codepoint glyph ranges spanning the full 0-65535 codepoint
/// space (D-08 "complete range set" — all 4 fontstacks are cached across this
/// full set). Ranges the font does not cover simply 404 and are skipped.
const int _rangeCount = 256;

/// Max concurrent downloads. Caps socket usage so warming the full
/// 4-fontstack × 256-range set does not open hundreds of connections at once
/// (DoS mitigation T-15-03-03).
const int _maxConcurrentDownloads = 8;

/// The three files that make up one sprite variant, keyed by the suffix the
/// MapLibre sprite loader appends to the sprite base.
const List<String> _spriteSuffixes = <String>['.json', '.png', '@2x.png'];

/// The one shared app-wide glyph/sprite cache (GLYPH-04, D-08).
///
/// On first read this `keepAlive` provider downloads every glyph range for the
/// 4 whitelisted fontstacks plus both light and dark sprite sheets into
/// `<app-docs>/map_cache`, idempotently (files already on disk are skipped, so a
/// second trail download / map open is a no-op — OFFL-01, D-10). Every local
/// path is built via the Task-1 path-safety helpers, so no operator-controlled
/// token is ever concatenated into a path (T-15-03-01).
@Riverpod(keepAlive: true)
class GlyphSpriteCache extends _$GlyphSpriteCache {
  @override
  Future<GlyphSpriteCachePaths> build() async {
    final sources = await ref.watch(mapStyleSourcesProvider.future);
    final api = ref.read(apiProvider);

    final docs = await getApplicationDocumentsDirectory();
    final root = p.join(docs.path, 'map_cache');

    final paths = GlyphSpriteCachePaths(
      root: root,
      glyphBase: p.join(root, 'glyphs'),
      spriteLightBase: spriteCacheBasePath(root, dark: false),
      spriteDarkBase: spriteCacheBasePath(root, dark: true),
    );

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
        // matches at render time (15-06).
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

  /// Download [job] unless its file already exists on disk (idempotent skip →
  /// OFFL-01 / D-10 no-op). A missing range (404) or transient failure is
  /// swallowed best-effort — not every fontstack covers every Unicode range,
  /// and a glyph miss must not fail the whole warm.
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
