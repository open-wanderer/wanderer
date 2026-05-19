import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

class WandererMap extends ConsumerStatefulWidget {
  final Trail trail;
  const WandererMap({super.key, required this.trail});

  @override
  ConsumerState<WandererMap> createState() => _WandererMapState();
}

class _WandererMapState extends ConsumerState<WandererMap> {
  Style? style;

  @override
  void initState() {
    super.initState();
    _initializeStyle();
  }

  Future<void> _initializeStyle() async {
    final auth = ref.read(authProvider).requireValue!;
    final serverUrl = auth.serverUrl;

    final originalStyle = await StyleReader.asset(
      'assets/styles/ofm.json',
    ).read();

    if (mounted) {
      setState(() {
        style = originalStyle;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (style == null) {
      return const Center(child: CircularProgressIndicator());
    }

    final bounds = widget.trail.expand?.gpx?.getBounds();

    return FlutterMap(
      options: MapOptions(
        initialCameraFit: bounds != null
            ? CameraFit.bounds(
                bounds: bounds,
                padding: const EdgeInsets.all(40),
              )
            : null,
        initialCenter: LatLng(widget.trail.lat ?? 0, widget.trail.lon ?? 0),
        initialZoom: 18,
      ),
      children: [
        VectorTileLayer(
          tileProviders: style!.providers,
          theme: style!.theme,
          tileOffset: TileOffset.DEFAULT,
        ),
        if (widget.trail.expand?.gpx != null) TrailLayer(trail: widget.trail),
      ],
    );
  }
}
