import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'map_camera_provider.g.dart';

class MapCameraState {
  final Geographic center;
  final double zoom;
  const MapCameraState({required this.center, required this.zoom});
}

@Riverpod(keepAlive: true)
class MapCamera extends _$MapCamera {
  @override
  MapCameraState? build() => null;

  void save(Geographic center, double zoom) {
    state = MapCameraState(center: center, zoom: zoom);
  }
}
