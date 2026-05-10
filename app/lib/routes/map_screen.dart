import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:latlong2/latlong.dart';
import 'package:vector_map_tiles/vector_map_tiles.dart';

class MapScreen extends StatefulWidget {
  const MapScreen({super.key});

  @override
  State<MapScreen> createState() => _MapScreenState();
}

class _MapScreenState extends State<MapScreen> {
  Style? style;

  @override
  void initState() {
    super.initState();
    Future<Style> readStyle() =>
        StyleReader(uri: 'https://demo.wanderer.to/styles/ofm.json').read();

    readStyle().then((style) {
      this.style = style;
      setState(() {});
    });
  }

  @override
  Widget build(BuildContext context) {
    if (style != null) {
      return FlutterMap(
        options: MapOptions(
          initialCenter: LatLng(49.246292, -123.116226),
          initialZoom: 10,
          maxZoom: 22,
        ),
        children: [
          SizedBox.expand(
            child: VectorTileLayer(
              tileProviders: style!.providers,
              theme: style!.theme,
              tileOffset: TileOffset.DEFAULT,
            ),
          ),
        ],
      );
    } else {
      return const SizedBox.expand(
        child: Center(child: CircularProgressIndicator()),
      );
    }
  }
}
