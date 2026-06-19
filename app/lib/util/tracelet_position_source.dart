import 'dart:async';

import 'package:geolocator/geolocator.dart' as geo;
import 'package:tracelet/tracelet.dart' as tl;

/// Bridges tracelet's background-location engine into a [geo.Position] stream
/// so the navigation screen's existing consumers (maneuver provider, stats
/// provider, CurrentLocationLayer) remain type-compatible.
///
/// Mirrors the prior Geolocator.getPositionStream contract:
/// - High accuracy, 5 m distance filter
/// - Continues delivering fixes while the screen is locked (background)
///
/// Lifecycle: call [start] once in initState, [dispose] in dispose().
class TraceletPositionSource {
  final _controller = StreamController<geo.Position>.broadcast();
  StreamSubscription<tl.Location>? _locationSub;

  Stream<geo.Position> get stream => _controller.stream;

  Future<void> start() async {
    _locationSub = tl.Tracelet.onLocation(_onLocation);

    await tl.Tracelet.ready(
      tl.Config.balanced().copyWith(
        geo: const tl.GeoConfig(
          desiredAccuracy: tl.DesiredAccuracy.high,
          distanceFilter: 5.0,
        ),
        app: const tl.AppConfig(stopOnTerminate: false),
      ),
    );

    await tl.Tracelet.start();
  }

  void _onLocation(tl.Location location) {
    if (_controller.isClosed) return;
    final c = location.coords;
    _controller.add(
      geo.Position(
        latitude: c.latitude,
        longitude: c.longitude,
        altitude: c.altitude,
        altitudeAccuracy: c.altitudeAccuracy,
        speed: c.speed,
        speedAccuracy: c.speedAccuracy,
        heading: c.heading,
        headingAccuracy: c.headingAccuracy,
        accuracy: c.accuracy,
        timestamp: DateTime.now(),
      ),
    );
  }

  Future<void> dispose() async {
    await _locationSub?.cancel();
    _locationSub = null;
    await tl.Tracelet.stop();
    await _controller.close();
  }
}
