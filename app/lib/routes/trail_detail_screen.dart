import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';
import 'package:wanderer/components/map/trail_layer.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/gpx_util.dart';

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
    final serverUrl = ref.read(authProvider).requireValue!.serverUrl;
    Future<Style> readStyle() =>
        StyleReader(uri: '$serverUrl/styles/ofm.json').read();

    readStyle().then((style) {
      this.style = style;
      setState(() {});
    });
  }

  @override
  Widget build(BuildContext context) {
    final trailAsync = ref.watch(trailProvider(widget.id));

    return Scaffold(
      body: SafeArea(
        child: trailAsync.when(
          data: (trail) => buildMap(trailAsync.requireValue),
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, stack) => Center(child: Text('Error: $err')),
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
              child: _buildPanelContent(context, trail, scrollController),
            );
          },
        ),
      ],
    );
  }

  Widget _buildPanelContent(
    BuildContext context,
    Trail trail,
    ScrollController scrollController,
  ) {
    return SingleChildScrollView(
      controller: scrollController,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Center(
            child: Container(
              margin: const EdgeInsets.symmetric(vertical: 12),
              width: 40,
              height: 5,
              decoration: BoxDecoration(
                color: Colors.grey.withValues(alpha: 0.3),
                borderRadius: BorderRadius.circular(10),
              ),
            ),
          ),

          Padding(
            padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  trail.name,
                  style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 12),

                Wrap(
                  spacing: 10,
                  runSpacing: 10,
                  children: [
                    _buildStatChip(
                      context,
                      FontAwesomeIcons.ruler,
                      formatDistance(trail.distance),
                    ),
                    _buildStatChip(
                      context,
                      FontAwesomeIcons.clock,
                      Duration(seconds: trail.duration.toInt()).pretty(
                        abbreviated: true,
                        tersity: DurationTersity.minute,
                      ),
                    ),
                    _buildStatChip(
                      context,
                      FontAwesomeIcons.arrowTrendUp,
                      formatElevation(trail.elevationGain),
                    ),
                    _buildStatChip(
                      context,
                      FontAwesomeIcons.arrowTrendDown,
                      formatElevation(trail.elevationLoss),
                    ),
                    if (trail.expand?.category != null)
                      _buildStatChip(
                        context,
                        FontAwesomeIcons.route,
                        trail.expand!.category!.name,
                      ),
                  ],
                ),

                const Divider(height: 32),

                Text(
                  "Description",
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  trail.description.isNotEmpty
                      ? trail.description
                      : "No description provided for this trail.",
                  style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                    color: Colors.black87,
                    height: 1.5,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStatChip(BuildContext context, FaIconData icon, String label) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      decoration: BoxDecoration(
        color: Colors.grey.withValues(alpha: 0.1),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          FaIcon(icon, size: 16, color: Colors.blueGrey),
          const SizedBox(width: 8),
          Text(label, style: Theme.of(context).textTheme.bodySmall),
        ],
      ),
    );
  }
}
