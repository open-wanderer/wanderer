import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/gpx_util.dart';

class WandererMap extends ConsumerStatefulWidget {
  final Trail trail;
  final MapController? mapController;
  final bool disabled;
  final List<Widget>? controls;

  final bool showTrail;

  final TapCallback? onTap;
  final Function(MapEvent)? onMapEvent;
  final Function(Waypoint wp)? onWaypointTap;
  const WandererMap({
    super.key,
    required this.trail,
    this.mapController,
    this.onTap,
    this.onWaypointTap,
    this.onMapEvent,
    this.disabled = false,
    this.controls = const [],
    this.showTrail = true,
  });

  @override
  ConsumerState<WandererMap> createState() => _WandererMapState();
}

class _WandererMapState extends ConsumerState<WandererMap> {
  Style? style;
  LatLngBounds? bounds;

  @override
  void initState() {
    super.initState();
    _initializeStyle();

    bounds = widget.trail.expand?.gpx?.getBounds();
  }

  Future<void> _initializeStyle() async {
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

    return FlutterMap(
      mapController: widget.mapController,
      options: MapOptions(
        onTap: widget.onTap,
        onMapEvent: (e) => widget.onMapEvent,
        interactionOptions: widget.disabled
            ? const InteractionOptions(flags: InteractiveFlag.none)
            : const InteractionOptions(),
        initialCameraFit: bounds != null
            ? CameraFit.bounds(
                bounds: bounds!,
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
        if (widget.trail.expand?.gpx != null && widget.showTrail)
          TrailLayer(trail: widget.trail, onWaypointTap: widget.onWaypointTap),

        Align(
          alignment: Alignment.topRight,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: widget.controls!,
          ),
        ),
      ],
    );
  }
}
