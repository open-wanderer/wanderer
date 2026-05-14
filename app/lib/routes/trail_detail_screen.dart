import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';
import 'package:path_provider/path_provider.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/vendor/vector_map_tiles/local_first_tile_provider.dart';

class TrailDetailScreen extends ConsumerStatefulWidget {
  final String id;
  const TrailDetailScreen({super.key, required this.id});

  @override
  ConsumerState<TrailDetailScreen> createState() => _TrailDetailScreenState();
}

class _TrailDetailScreenState extends ConsumerState<TrailDetailScreen> {
  Style? style;

  @override
  void initState() {
    super.initState();
    _initializeStyle();
  }

  Future<void> _initializeStyle() async {
    final auth = ref.read(authProvider).requireValue!;
    final serverUrl = auth.serverUrl;

    final appDir = await getApplicationDocumentsDirectory();

    final originalStyle = await StyleReader(
      uri: '$serverUrl/styles/ofm.json',
    ).read();

    final Map<String, VectorTileProvider> patchedProviders = {};

    for (var entry in originalStyle.providers.tileProviderBySource.entries) {
      final sourceId = entry.key;
      final originalProvider = entry.value;

      if (originalProvider is NetworkVectorTileProvider) {
        patchedProviders[sourceId] = LocalFirstTileProvider(
          urlTemplate:
              "https://tiles.openfreemap.org/planet/latest/{z}/{x}/{y}.pbf",
          trailId: widget.id,
          baseAppPath: appDir.path,
        );
      } else {
        patchedProviders[sourceId] = originalProvider;
      }
    }

    if (mounted) {
      setState(() {
        style = Style(
          theme: originalStyle.theme,
          providers: TileProviders(patchedProviders),
          sprites: originalStyle.sprites,
          center: originalStyle.center,
          name: originalStyle.name,
          zoom: originalStyle.zoom,
        );
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final trailAsync = ref.watch(trailProvider(widget.id));

    return Scaffold(
      body: SafeArea(
        child: trailAsync.when(
          data: (trail) => buildMap(trailAsync.requireValue),
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, stack) => WandererError(err: err, stack: stack),
        ),
      ),
    );
  }

  Widget buildMap(Trail trail) {
    if (style == null) {
      return const Center(child: CircularProgressIndicator());
    }

    final bounds = trail.expand?.gpx?.getBounds();

    return Stack(
      children: [
        FlutterMap(
          options: MapOptions(
            initialCameraFit: bounds != null
                ? CameraFit.bounds(
                    bounds: bounds,
                    padding: const EdgeInsets.all(40),
                  )
                : null,
            initialCenter: LatLng(trail.lat ?? 0, trail.lon ?? 0),
            initialZoom: 18,
          ),
          children: [
            VectorTileLayer(
              tileProviders: style!.providers,
              theme: style!.theme,
              tileOffset: TileOffset.DEFAULT,
            ),
            if (trail.expand?.gpx != null) TrailLayer(trail: trail),
          ],
        ),
        DraggableScrollableSheet(
          initialChildSize: 0.3,
          minChildSize: 0.15,
          maxChildSize: 0.9,
          builder: (context, scrollController) {
            return Container(
              decoration: BoxDecoration(
                color: Theme.of(context).canvasColor,
                borderRadius: const BorderRadius.vertical(
                  top: Radius.circular(20),
                ),
                boxShadow: [
                  BoxShadow(
                    blurRadius: 10,
                    color: Colors.black.withValues(alpha: 0.1),
                  ),
                ],
              ),
              // The extracted content function
              child: TrailPanel(
                trail: trail,
                scrollController: scrollController,
              ),
            );
          },
        ),
      ],
    );
  }
}
