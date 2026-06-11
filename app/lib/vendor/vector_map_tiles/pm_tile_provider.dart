import 'dart:typed_data';

import 'package:pmtiles/pmtiles.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';

/// Use this [VectorTileProvider] to provide vector or raster tiles from a
/// PMTiles archive.
///
class PmTilesVectorTileProvider extends VectorTileProvider {
  /// Create a tile provider directly with a [PmTilesArchive] from the
  /// `pmtiles` package.
  PmTilesVectorTileProvider.fromArchive(
    this.archive, {
    this.type = TileProviderType.vector,
    @Deprecated(
      'This option is no longer used and will get removed in a future update.',
    )
    this.silenceTileNotFound = false,
  });

  /// Tile [PmTilesArchive] instance used to request tiles from.
  final PmTilesArchive archive;

  /// The type of the tile provider.
  ///
  /// Either [TileProviderType.vector] or [TileProviderType.raster].
  @override
  final TileProviderType type;

  @Deprecated(
    'This option is no longer used and will get removed in a future update.',
  )
  /// Flag no longer in use.
  final bool silenceTileNotFound;

  /// Create a tile provider by specifying the source of the PMTiles file.
  ///
  /// [source] can either be a URL or path on your file system.
  static Future<PmTilesVectorTileProvider> fromSource(
    String source, {
    TileProviderType type = TileProviderType.vector,
    @Deprecated(
      'This option is no longer used and will get removed in a future update.',
    )
    bool silenceTileNotFound = false,
  }) async {
    final archive = await PmTilesArchive.from(source);
    return PmTilesVectorTileProvider.fromArchive(archive, type: type);
  }

  /// The maximum zoom level that the tile provider supports.
  @override
  int get maximumZoom => archive.maxZoom;

  /// The minimum zoom level that the tile provider supports.
  @override
  int get minimumZoom => archive.minZoom;

  /// Used by the vector_map_tiles package to request a specific tile
  @override
  Future<Uint8List> provide(TileIdentity tile) async {
    final tileId = ZXY(tile.z, tile.x, tile.y).toTileId();
    try {
      final data = await archive.tile(tileId);
      return Uint8List.fromList(data.bytes());
    } on TileNotFoundException {
      throw ProviderException(
        message: 'Tile not found: $tile',
        retryable: Retryable.none,
        statusCode: 404,
      );
    }
  }

  @override
  TileOffset get tileOffset => TileOffset.DEFAULT;
}

/// A [VectorTileProvider] that fans tile requests across multiple
/// [PmTilesArchive] instances and returns the first successful result.
///
/// Use this when a trail's offline tiles are split across several `.pmtiles`
/// files (e.g. one per bounding-box cell produced by the download service).
class MultiPmTilesVectorTileProvider extends VectorTileProvider {
  /// Creates a provider from an already-opened list of [PmTilesArchive]s.
  MultiPmTilesVectorTileProvider.fromArchives(
    this.archives, {
    this.type = TileProviderType.vector,
  }) {
    if (archives.isEmpty) {
      throw ArgumentError('archives must not be empty');
    }
  }

  /// The list of archives to fan requests across.
  final List<PmTilesArchive> archives;

  /// The type of the tile provider.
  @override
  final TileProviderType type;

  /// Opens all [sources] in parallel and returns a [MultiPmTilesVectorTileProvider].
  ///
  /// [sources] can be file-system paths or URLs, matching the convention of
  /// [PmTilesVectorTileProvider.fromSource].
  /// Throws [ArgumentError] if [sources] is empty.
  static Future<MultiPmTilesVectorTileProvider> fromSources(
    List<String> sources, {
    TileProviderType type = TileProviderType.vector,
  }) async {
    if (sources.isEmpty) {
      throw ArgumentError('sources must not be empty');
    }
    final archives = await Future.wait(
      sources.map((s) => PmTilesArchive.from(s)),
    );
    return MultiPmTilesVectorTileProvider.fromArchives(
      archives,
      type: type,
    );
  }

  /// Union maximum zoom: the highest `maxZoom` across all archives.
  @override
  int get maximumZoom =>
      archives.map((a) => a.maxZoom).reduce((a, b) => a > b ? a : b);

  /// Union minimum zoom: the lowest `minZoom` across all archives.
  @override
  int get minimumZoom =>
      archives.map((a) => a.minZoom).reduce((a, b) => a < b ? a : b);

  /// Requests a tile by trying each archive in order.
  ///
  /// Returns the first successful result. Throws a [ProviderException] with
  /// status 404 only when no archive contains the requested tile.
  @override
  Future<Uint8List> provide(TileIdentity tile) async {
    final tileId = ZXY(tile.z, tile.x, tile.y).toTileId();
    for (final archive in archives) {
      try {
        final data = await archive.tile(tileId);
        return Uint8List.fromList(data.bytes());
      } on TileNotFoundException {
        continue;
      }
    }
    throw ProviderException(
      message: 'Tile not found: $tile',
      retryable: Retryable.none,
      statusCode: 404,
    );
  }

  @override
  TileOffset get tileOffset => TileOffset.DEFAULT;
}
