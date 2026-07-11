import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

/// Plain data class carrying a foreground GPS fix for map location markers.
class LocationMarkerPosition {
  final double latitude;
  final double longitude;
  final double accuracy;
  final double? heading;
  final double? headingAccuracy;

  const LocationMarkerPosition({
    required this.latitude,
    required this.longitude,
    required this.accuracy,
    this.heading,
    this.headingAccuracy,
  });
}

/// Signals that the device's location service is disabled.
class ServiceDisabledException implements Exception {
  const ServiceDisabledException();
}

/// A singleton foreground position stream shared across all map screens.
///
/// The ghost subscription keeps the broadcast controller alive between
/// navigations so [onListen] fires only once per app session — the GPS
/// activation dialog appears at most once. When the user declines and
/// later enables GPS manually, [getServiceStatusStream] fires [enabled]
/// and we restart [getPositionStream] so the location marker appears
/// without requiring another navigation or app restart.
final foregroundPositionStreamProvider =
    NotifierProvider<ForegroundPositionStream, Stream<LocationMarkerPosition?>>(
  ForegroundPositionStream.new,
);

class ForegroundPositionStream
    extends Notifier<Stream<LocationMarkerPosition?>> {
  StreamSubscription<Position>? _positionSub;
  StreamSubscription<ServiceStatus>? _statusSub;
  late StreamController<LocationMarkerPosition?> _controller;

  void _cancelPosition() {
    _positionSub?.cancel();
    _positionSub = null;
  }

  void _startPositionStream() {
    _cancelPosition();
    _positionSub = Geolocator.getPositionStream().listen(
      (pos) {
        if (!_controller.isClosed) {
          _controller.add(LocationMarkerPosition(
            latitude: pos.latitude,
            longitude: pos.longitude,
            accuracy: pos.accuracy,
            heading: pos.heading,
            headingAccuracy: pos.headingAccuracy,
          ));
        }
      },
      onError: (_) {
        // Swallow position stream errors (GPS declined or toggled off);
        // service status listener handles the disabled → enabled transition.
      },
    );
  }

  @override
  Stream<LocationMarkerPosition?> build() {
    _controller = StreamController<LocationMarkerPosition?>.broadcast(
      onListen: () async {
        // Mirror GPS on/off transitions to the layer.
        _statusSub = Geolocator.getServiceStatusStream().listen((status) {
          if (_controller.isClosed) return;
          if (status == ServiceStatus.enabled) {
            _controller.add(null); // reset layer to "locating"
            _startPositionStream(); // restart after GPS was re-enabled
          } else {
            _cancelPosition();
            _controller.addError(const ServiceDisabledException());
          }
        });

        // Signal immediately if GPS is already off so the layer shows
        // "service disabled" while the activation dialog is pending.
        final serviceEnabled = await Geolocator.isLocationServiceEnabled();
        if (_controller.isClosed) return;
        if (!serviceEnabled) {
          _controller.addError(const ServiceDisabledException());
        }

        // Subscribe to the live position stream.
        // On Android + Google Play Services this triggers the "Turn on GPS?"
        // system dialog when location services are disabled.
        // Because the ghost subscription below keeps this controller alive,
        // onListen fires only once per session — the dialog appears at most once.
        _startPositionStream();
      },
    );

    ref.onDispose(() {
      _statusSub?.cancel();
      _cancelPosition();
      _controller.close();
    });

    // Ghost subscription: keeps the broadcast stream alive between navigations.
    final ghost = _controller.stream.listen(null, onError: (_) {});
    ref.onDispose(ghost.cancel);

    return _controller.stream;
  }
}
