// SPIKE 15-01 — THROWAWAY, delete after the file:// glyph gate resolves.
//
// One-off pre-seed helper for the Phase 15 risk-gate spike (D-01/D-02/D-03,
// RESEARCH.md "Spike Design"). Fetches a SMALL set of glyph ranges + the sprite
// files into the app documents directory WHILE ONLINE, so the spike screen can
// then resolve them via `file://` URLs on a physical device in airplane mode.
//
// Intentionally NOT generalized — no full 4-fontstack / 256-range warm, no
// sanitize/optimize. That is later plan work (GLYPH-04, gated on this PASS).
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/map_style_sources_provider.dart';

/// The single hard-coded fontstack the spike proves labels for (T-15-01-01:
/// a literal, no external input reaches the constructed path).
const spikeFontstack = 'Noto Sans Regular';

/// The minimal glyph ranges to pre-seed — enough to cover the ASCII label
/// codepoints. Hard-coded numeric ranges (no external input).
const spikeGlyphRanges = <String>['0-255', '256-511'];

/// Result of a pre-seed pass: the absolute app-docs cache dir and a human log
/// of what was fetched / skipped / failed (surfaced on the spike screen so
/// Christian can confirm the files landed before enabling airplane mode).
class SpikeGlyphSeedResult {
  const SpikeGlyphSeedResult({required this.cacheDir, required this.log});

  /// Absolute path to `<app-docs>/spike_glyphs` (the file:// root).
  final String cacheDir;

  /// Ordered, human-readable fetch/skip/fail lines.
  final List<String> log;
}

/// Pre-seed the spike glyph + sprite cache into `<app-docs>/spike_glyphs`.
///
/// Mirrors `trail_download_service.dart`'s convention: resolve the app documents
/// directory, create it `recursive: true`, skip any file already on disk
/// (idempotent), and fetch via the existing dio api client (`_api.download`).
///
/// Reads the operator-controlled glyph/sprite URLs from `mapStyleSourcesProvider`
/// and substitutes the `{fontstack}` / `{range}` tokens to build each fetch URL.
Future<SpikeGlyphSeedResult> seedSpikeGlyphCache(WidgetRef ref) async {
  final sources = await ref.read(mapStyleSourcesProvider.future);
  final api = ref.read(apiProvider);

  final docs = await getApplicationDocumentsDirectory();
  final cacheDir = '${docs.path}/spike_glyphs';
  final fontDir = Directory('$cacheDir/$spikeFontstack');
  if (!await fontDir.exists()) {
    await fontDir.create(recursive: true);
  }

  final log = <String>['cache: $cacheDir'];

  // --- Glyphs (A1: the risk gate) --------------------------------------------
  for (final range in spikeGlyphRanges) {
    final localPath = '${fontDir.path}/$range.pbf';
    if (await File(localPath).exists()) {
      log.add('skip glyph $spikeFontstack/$range (already cached)');
      continue;
    }
    // {fontstack} carries a space — encode only for the REMOTE fetch URL; the
    // local file:// path keeps the literal directory name so MapLibre's runtime
    // token substitution matches on-disk.
    final url = sources.glyphUrl
        .replaceAll('{fontstack}', Uri.encodeComponent(spikeFontstack))
        .replaceAll('{range}', range);
    try {
      await api.download(url, localPath);
      log.add('fetched glyph $spikeFontstack/$range');
    } catch (e) {
      log.add('FAILED glyph $spikeFontstack/$range: $e');
    }
  }

  // --- Sprite (A2: same file:// scheme, secondary signal) --------------------
  // spriteUrl is a theme-agnostic base (e.g. `.../sprites/v4`); the light
  // variant's files live under `<base>/light{.json,.png,@2x.png}`. MapLibre
  // resolves a `sprite` key by appending those suffixes to the base, so we cache
  // them locally as `sprite{.json,.png,@2x.png}` and point the style at
  // `file://<cache>/sprite`. Best-effort — a sprite miss must not abort the
  // primary glyph gate.
  final spriteBase = '${sources.spriteUrl}/light';
  const spriteFiles = <String, String>{
    'sprite.json': '.json',
    'sprite.png': '.png',
    'sprite@2x.png': '@2x.png',
  };
  for (final entry in spriteFiles.entries) {
    final localPath = '$cacheDir/${entry.key}';
    if (await File(localPath).exists()) {
      log.add('skip ${entry.key} (already cached)');
      continue;
    }
    try {
      await api.download('$spriteBase${entry.value}', localPath);
      log.add('fetched ${entry.key}');
    } catch (e) {
      log.add('FAILED ${entry.key}: $e');
    }
  }

  return SpikeGlyphSeedResult(cacheDir: cacheDir, log: log);
}
