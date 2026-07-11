import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/provider/foreground_position_stream_provider.dart';

/// The on-map "you are here" indicator: a blue dot with a white border,
/// optionally with a rotating heading wedge behind it.
class LocationPuck extends StatelessWidget {
  const LocationPuck({
    super.key,
    this.size = 18,
    this.dotSize = 18,
    this.heading,
    this.headingAccuracy,
    this.showHeading = false,
  });

  /// Overall bounding box (matches the enclosing [ml.Marker]'s size) — also
  /// the wedge's canvas when [showHeading] is on.
  final double size;

  /// Diameter of the static dot.
  final double dotSize;

  final double? heading;
  final double? headingAccuracy;

  /// Whether to draw the heading wedge at all. Off by default since most
  /// callers (trail/search maps) have no heading data.
  final bool showHeading;

  @override
  Widget build(BuildContext context) {
    // geolocator reports negative heading/accuracy when stationary or
    // low-confidence — treat those as "no heading" rather than jittering.
    final hasValidHeading =
        showHeading &&
        heading != null &&
        heading! >= 0 &&
        headingAccuracy != null &&
        headingAccuracy! >= 0;
    return Stack(
      alignment: Alignment.center,
      children: [
        if (hasValidHeading)
          Transform.rotate(
            angle: heading! * math.pi / 180,
            child: CustomPaint(
              size: Size(size, size),
              painter: const _HeadingWedgePainter(),
            ),
          ),
        Container(
          width: dotSize,
          height: dotSize,
          decoration: BoxDecoration(
            color: Colors.blue,
            shape: BoxShape.circle,
            border: Border.all(color: Colors.white, width: 3),
            boxShadow: [
              BoxShadow(
                color: Colors.black.withValues(alpha: .3),
                blurRadius: 4,
                offset: const Offset(0, 1),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _HeadingWedgePainter extends CustomPainter {
  const _HeadingWedgePainter();

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()..color = Colors.blue.withValues(alpha: .35);
    final path = Path()
      ..moveTo(size.width / 2, 0)
      ..lineTo(size.width * 0.78, size.height * 0.42)
      ..lineTo(size.width / 2, size.height * 0.28)
      ..lineTo(size.width * 0.22, size.height * 0.42)
      ..close();
    canvas.drawPath(path, paint);
  }

  @override
  bool shouldRepaint(covariant _HeadingWedgePainter oldDelegate) => false;
}

/// Location layer for screens driven by [foregroundPositionStreamProvider]
/// (no heading data available — [LocationPuck.showHeading] stays false).
class LocationMarkerLayer extends ConsumerWidget {
  const LocationMarkerLayer({super.key, this.dotSize = 18});

  final double dotSize;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final positionStream = ref.watch(foregroundPositionStreamProvider);
    return StreamBuilder<LocationMarkerPosition?>(
      stream: positionStream,
      builder: (context, snapshot) {
        final position = snapshot.data;
        if (position == null) return const SizedBox.shrink();
        return ml.WidgetLayer(
          markers: [
            ml.Marker(
              point: ml.Geographic(
                lat: position.latitude,
                lon: position.longitude,
              ),
              size: Size(dotSize, dotSize),
              child: LocationPuck(size: dotSize, dotSize: dotSize),
            ),
          ],
        );
      },
    );
  }
}
