import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/waypoint_card.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/format_util.dart';

const double _kLineX = 20;
const double _kLeftColWidth = 40.0;
const double _kRowGap = 16.0;

class TrailTimeline extends StatelessWidget {
  final List<Waypoint> waypoints;
  final double? totalDistance;

  const TrailTimeline({super.key, required this.waypoints, this.totalDistance});

  @override
  Widget build(BuildContext context) {
    if (waypoints.isEmpty) return const SizedBox.shrink();

    final rows = <_RowData>[
      _RowData.cap(icon: FontAwesomeIcons.bullseye, label: 'Start'),
      for (final wp in waypoints) _RowData.waypoint(waypoint: wp),
      _RowData.cap(
        icon: FontAwesomeIcons.flagCheckered,
        label: 'Finish',
        distanceLabel: totalDistance != null
            ? formatDistance(totalDistance)
            : null,
      ),
    ];

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Stack(
        children: [
          Positioned.fill(
            child: CustomPaint(painter: _ContinuousLinePainter()),
          ),

          Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              for (int i = 0; i < rows.length; i++) ...[
                _TimelineRow(data: rows[i]),
                if (i < rows.length - 1) const SizedBox(height: _kRowGap),
              ],
            ],
          ),
        ],
      ),
    );
  }
}

class _RowData {
  final FaIconData icon;
  final String? label;
  final String? distanceLabel;
  final Waypoint? waypoint;

  const _RowData.cap({
    required this.icon,
    required this.label,
    this.distanceLabel,
  }) : waypoint = null;

  const _RowData.waypoint({required Waypoint this.waypoint})
    : icon = FontAwesomeIcons.circle,
      label = null,
      distanceLabel = null;

  bool get isCap => waypoint == null;
}

class _TimelineRow extends StatelessWidget {
  final _RowData data;

  const _TimelineRow({super.key, required this.data});

  @override
  Widget build(BuildContext context) {
    final wp = data.waypoint;
    final icon = wp?.icon ?? data.icon;
    final distLabel = wp?.distanceFromStart != null
        ? formatDistance(wp!.distanceFromStart!)
        : data.distanceLabel;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Container(
          width: _kLeftColWidth,
          color: Theme.of(context).canvasColor,
          padding: EdgeInsets.symmetric(vertical: 4),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              FaIcon(
                icon,
                size: 14,
                color: Theme.of(context).colorScheme.onSurface,
              ),
              if (distLabel != null) ...[
                const SizedBox(height: 4),
                Text(
                  distLabel,
                  style: TextStyle(
                    fontSize: 10,
                    color: Colors.grey[500],
                    fontWeight: FontWeight.w500,
                    height: 1.2,
                  ),
                  textAlign: TextAlign.center,
                ),
              ],
            ],
          ),
        ),

        const SizedBox(width: 12),
        Expanded(
          child: data.isCap
              ? Text(
                  data.label ?? '',
                  style: const TextStyle(
                    fontWeight: FontWeight.w600,
                    fontSize: 15,
                  ),
                )
              : Padding(
                  padding: const EdgeInsets.only(right: 16, bottom: 0),
                  child: WaypointCard(waypoint: wp!),
                ),
        ),
      ],
    );
  }
}

class _ContinuousLinePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = Colors.grey[300]!
      ..strokeWidth = 1.5
      ..style = PaintingStyle.stroke;

    const dashHeight = 5.0;
    const dashSpace = 4.0;
    double y = 0;

    while (y < size.height) {
      final end = (y + dashHeight).clamp(0.0, size.height);
      canvas.drawLine(Offset(_kLineX, y), Offset(_kLineX, end), paint);
      y += dashHeight + dashSpace;
    }
  }

  @override
  bool shouldRepaint(_ContinuousLinePainter old) => false;
}
