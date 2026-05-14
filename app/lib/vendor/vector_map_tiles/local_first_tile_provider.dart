import 'dart:io';
import 'dart:typed_data';

import 'package:vector_map_tiles/vector_map_tiles.dart';

class LocalFirstTileProvider extends NetworkVectorTileProvider {
  final String trailId;
  final String baseAppPath;

  LocalFirstTileProvider({
    required super.urlTemplate,
    required this.trailId,
    required this.baseAppPath,
    super.maximumZoom = 14,
    super.minimumZoom = 0,
  });

  @override
  Future<Uint8List> provide(TileIdentity tile) async {
    final localPath =
        '$baseAppPath/library/$trailId/map/${tile.z}/${tile.x}/${tile.y}.pbf';
    final localFile = File(localPath);

    if (await localFile.exists()) {
      try {
        return await localFile.readAsBytes();
      } catch (e) {
        print("Local tile read error: $e");
      }
    }
    return super.provide(tile);
  }
}
