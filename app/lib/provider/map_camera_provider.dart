import 'package:latlong2/latlong.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'map_camera_provider.g.dart';

class MapCameraState {
  final LatLng center;
  final double zoom;
  const MapCameraState({required this.center, required this.zoom});
}

@Riverpod(keepAlive: true)
class MapCamera extends _$MapCamera {
  @override
  MapCameraState? build() => null;

  void save(LatLng center, double zoom) {
    state = MapCameraState(center: center, zoom: zoom);
  }
}
