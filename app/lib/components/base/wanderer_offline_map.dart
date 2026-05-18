import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:vector_tile_renderer/vector_tile_renderer.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/vendor/vector_map_tiles/pm_tile_provider.dart';

class WandererOfflineMap extends ConsumerStatefulWidget {
  final Trail trail;

  const WandererOfflineMap({super.key, required this.trail});

  @override
  ConsumerState<WandererOfflineMap> createState() => _WandererOfflineMapState();
}

class _WandererOfflineMapState extends ConsumerState<WandererOfflineMap> {
  Future<PmTilesVectorTileProvider>? _futureTileProvider;

  final mapTheme = ProvidedThemes.protomapsLight(logger: Logger.console());

  @override
  void initState() {
    super.initState();
    _futureTileProvider = PmTilesVectorTileProvider.fromSource(
      widget.trail.pmTiles[0],
    );
  }

  @override
  Widget build(BuildContext context) {
    final bounds = widget.trail.expand?.gpx?.getBounds();

    return FutureBuilder<PmTilesVectorTileProvider>(
      future: _futureTileProvider,
      builder: (context, snapshot) {
        if (snapshot.hasData) {
          final tileProvider = snapshot.data!;
          return FlutterMap(
            options: MapOptions(
              initialCameraFit: bounds != null
                  ? CameraFit.bounds(
                      bounds: bounds,
                      padding: const EdgeInsets.all(40),
                    )
                  : null, // firenze
              maxZoom: 18,
              minZoom: 0,
            ),
            children: [
              VectorTileLayer(
                fileCacheTtl: Duration.zero,
                theme: mapTheme,
                tileProviders: TileProviders({'protomaps': tileProvider}),
                tileOffset: TileOffset.DEFAULT,
              ),
              if (widget.trail.expand?.gpx != null)
                TrailLayer(trail: widget.trail),
            ],
          );
        }
        if (snapshot.hasError) {
          debugPrint(snapshot.error.toString());
          debugPrintStack(stackTrace: snapshot.stackTrace);
          return Center(child: Text(snapshot.error.toString()));
        }
        return const Center(child: CircularProgressIndicator());
      },
    );
  }
}
