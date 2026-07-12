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
              // Wedge base sits just outside the dot's painted edge (radius
              // + border) so it stays visible regardless of how dotSize/size
              // are tuned relative to each other.
              painter: _HeadingWedgePainter(dotSize / 2 + 3),
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
  const _HeadingWedgePainter(this.innerRadius);

  /// Radius of the dot (including its border) that the wedge must clear —
  /// the wedge is drawn behind the dot, so any part inside this radius would
  /// be fully occluded regardless of canvas size.
  final double innerRadius;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()..color = Colors.blue.withValues(alpha: .5);
    final center = Offset(size.width / 2, size.height / 2);
    final outerRadius = size.width / 2;
    // Guard against a canvas no larger than the dot — nothing to draw.
    if (outerRadius <= innerRadius) return;
    final backRadius = innerRadius;
    final wingRadius = backRadius + (outerRadius - backRadius) * 0.3;
    const wingAngle = 20 * math.pi / 180;

    Offset polar(double radius, double angleFromUp) =>
        center +
        Offset(radius * math.sin(angleFromUp), -radius * math.cos(angleFromUp));

    final path = Path()
      ..moveTo(polar(outerRadius, 0).dx, polar(outerRadius, 0).dy)
      ..lineTo(polar(wingRadius, wingAngle).dx, polar(wingRadius, wingAngle).dy)
      ..lineTo(polar(backRadius, 0).dx, polar(backRadius, 0).dy)
      ..lineTo(
        polar(wingRadius, -wingAngle).dx,
        polar(wingRadius, -wingAngle).dy,
      )
      ..close();
    canvas.drawPath(path, paint);
  }

  @override
  bool shouldRepaint(covariant _HeadingWedgePainter oldDelegate) =>
      oldDelegate.innerRadius != innerRadius;
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
