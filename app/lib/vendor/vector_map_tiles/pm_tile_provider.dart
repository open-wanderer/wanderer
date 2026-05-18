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
