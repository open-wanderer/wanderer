import 'dart:math';

import 'package:duration/duration.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:gpx/gpx.dart';
import 'package:latlong2/latlong.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/gpx_util.dart';

class ElevationProfile extends ConsumerStatefulWidget {
  final Trail trail;
  final Gpx gpx;

  final double chartHeight;
  final int smoothingWindowSize;
  final Function(TrackPoint? point)? onLineTouch;
  final bool enableLineTouch;
  const ElevationProfile({
    super.key,
    required this.trail,
    required this.gpx,
    this.chartHeight = 150,
    this.smoothingWindowSize = 30,
    this.onLineTouch,
    this.enableLineTouch = true,
  });

  @override
  ConsumerState<ElevationProfile> createState() => _ElevationProfileState();
}

class _ElevationProfileState extends ConsumerState<ElevationProfile> {
  late List<TrackPoint> _points;
  int? _selectedIndex;

  @override
  void initState() {
    super.initState();
    _points = _parseGpx(widget.gpx, widget.smoothingWindowSize);
  }

  @override
  void didUpdateWidget(ElevationProfile oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.gpx != widget.gpx ||
        oldWidget.smoothingWindowSize != widget.smoothingWindowSize) {
      _points = _parseGpx(widget.gpx, widget.smoothingWindowSize);
      _selectedIndex = null;
    }
  }

  List<TrackPoint> _parseGpx(Gpx gpx, int windowSize) {
    final rawPoints = gpx.allWaypoints;

    if (rawPoints.isEmpty) return [];

    final result = <TrackPoint>[];
    double cumDist = 0;
    Duration cumDuration = Duration.zero;

    for (int i = 0; i < rawPoints.length; i++) {
      final wpt = rawPoints[i];
      if (i > 0) {
        cumDist += _haversine(rawPoints[i - 1], wpt);

        final prevTime = rawPoints[i - 1].time;
        final currTime = wpt.time;
        if (prevTime != null && currTime != null) {
          final delta = currTime.difference(prevTime);
          if (delta > Duration.zero) cumDuration += delta;
        }
      }
      result.add(
        TrackPoint(
          distanceM: cumDist,
          elevationM: wpt.ele ?? 0,
          lonlat: Geographic(lat: wpt.lat ?? 0, lon: wpt.lon ?? 0),
          duration: cumDuration,
          gradient: 0,
          color: Colors.white,
        ),
      );
    }

    _smoothElevations(result, windowSize: windowSize);

    for (int i = 0; i < result.length; i++) {
      if (i == 0) {
        result[i].gradient = 0;
      } else {
        final dElev = result[i].elevationM - result[i - 1].elevationM;
        final dDist = result[i].distanceM - result[i - 1].distanceM;
        result[i].gradient = dDist > 0 ? (dElev / dDist) * 100 : 0;
      }
      result[i].color = _gradientColor(result[i].gradient);
    }
    result[0].color = result.length > 1 ? result[1].color : _gradientColor(0);

    return _simplifyTrackPoints(result, 250);
  }

  @override
  Widget build(BuildContext context) {
    // Establish the rebuild dependency on the unit provider; helper methods
    // read the same value synchronously via ref.read during this frame.
    ref.watch(unitProvider);

    if (_points.isEmpty) {
      return const _EmptyState();
    }

    final minElev = _points.map((p) => p.elevationM).reduce(min);
    final maxElev = _points.map((p) => p.elevationM).reduce(max);
    final maxDist = _points.last.distanceM;
    final maxDur = _points.last.duration;

    final yMin = (minElev / 100).floor() * 100.0;
    final yMax = ((maxElev + 100) / 50).ceil() * 50.0;

    final xInterval = _niceInterval(maxDist, 5);
    final yInterval = _niceInterval(yMax - yMin, 4);

    final showScrubStats =
        _selectedIndex != null &&
        _selectedIndex! > 0 &&
        _selectedIndex! < _points.length;

    final Widget statsHeader;
    if (showScrubStats) {
      final pt = _points[_selectedIndex!];
      statsHeader = Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          _buildStatText(
            pt.duration.pretty(
              abbreviated: true,
              tersity: DurationTersity.minute,
            ),
            FontAwesomeIcons.clock,
          ),
          _buildStatText(
            formatDistance(pt.distanceM, unit: ref.read(unitProvider)),
            FontAwesomeIcons.ruler,
          ),
          _buildStatText(
            formatElevation(pt.elevationM, unit: ref.read(unitProvider)),
            FontAwesomeIcons.mountain,
          ),
          _buildStatText(
            '${pt.gradient >= 0 ? '+' : ''}${pt.gradient.toStringAsFixed(1)}%',
            pt.gradient >= 0
                ? FontAwesomeIcons.arrowTrendUp
                : FontAwesomeIcons.arrowTrendDown,
          ),
        ],
      );
    } else {
      statsHeader = Row(
        mainAxisAlignment: MainAxisAlignment.spaceAround,
        children: [
          _buildStatText(
            maxDur.pretty(abbreviated: true, tersity: DurationTersity.minute),
            FontAwesomeIcons.clock,
          ),
          _buildStatText(
            formatDistance(maxDist, unit: ref.read(unitProvider)),
            FontAwesomeIcons.ruler,
          ),
          _buildStatText(
            formatElevation(
              widget.trail.elevationGain,
              unit: ref.read(unitProvider),
            ),
            FontAwesomeIcons.arrowTrendUp,
          ),
          _buildStatText(
            formatElevation(
              widget.trail.elevationLoss,
              unit: ref.read(unitProvider),
            ),
            FontAwesomeIcons.arrowTrendDown,
          ),
        ],
      );
    }

    return Column(
      children: [
        statsHeader,
        SizedBox(
          height: widget.chartHeight,
          child: _buildChart(
            minElev,
            yMin,
            yMax,
            maxDist,
            xInterval,
            yInterval,
          ),
        ),
      ],
    );
  }

  Widget _buildStatText(String text, FaIconData icon) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        FaIcon(icon, size: 16),
        SizedBox(width: 4),
        Text(text, style: Theme.of(context).textTheme.labelLarge),
      ],
    );
  }

  Widget _buildChart(
    double minElev,
    double yMin,
    double yMax,
    double maxDist,
    double xInterval,
    double yInterval,
  ) {
    final waypoints = (widget.trail.expand?.waypointsViaTrail ?? [])
        .where(
          (w) => w.distanceFromStart != null && w.distanceFromStart! <= maxDist,
        )
        .toList();

    // Must match leftTitles reservedSize so icon X positions align with the plot area
    const leftAxisWidth = 36.0;
    const iconSize = 14.0;

    return LayoutBuilder(
      builder: (context, constraints) {
        final plotWidth = constraints.maxWidth - leftAxisWidth;

        final spots = _points
            .map((p) => FlSpot(p.distanceM, p.elevationM))
            .toList();

        final barData = LineChartBarData(
          spots: spots,
          isCurved: true,
          curveSmoothness: 0.35,
          barWidth: 2.5,
          isStrokeCapRound: true,
          dotData: const FlDotData(show: false),
          gradient: _buildLineGradient(),
          belowBarData: BarAreaData(show: true, gradient: _buildFillGradient()),
        );

        return Stack(
          clipBehavior: Clip.none,
          children: [
            // ── Chart ──────────────────────────────────────────────────────────
            LineChart(
              LineChartData(
                minX: 0,
                maxX: _points.last.distanceM,
                minY: yMin,
                maxY: yMax,
                showingTooltipIndicators:
                    _selectedIndex != null &&
                        _selectedIndex! > 0 &&
                        _selectedIndex! < spots.length
                    ? [
                        ShowingTooltipIndicators([
                          LineBarSpot(barData, 0, spots[_selectedIndex!]),
                        ]),
                      ]
                    : [],
                clipData: const FlClipData.all(),
                backgroundColor: Colors.transparent,
                gridData: FlGridData(
                  show: true,
                  drawVerticalLine: false,
                  horizontalInterval: yInterval,
                  getDrawingHorizontalLine: (value) => FlLine(
                    color: Colors.black.withValues(alpha: 0.1),
                    strokeWidth: 1,
                    dashArray: [6, 4],
                  ),
                ),
                borderData: FlBorderData(show: false),
                titlesData: FlTitlesData(
                  leftTitles: AxisTitles(
                    sideTitles: SideTitles(
                      showTitles: true,
                      reservedSize: leftAxisWidth,
                      interval: yInterval,
                      getTitlesWidget: (value, meta) {
                        if (value == meta.min || value == meta.max) {
                          return const SizedBox.shrink();
                        }
                        return Text(
                          formatElevation(value, unit: ref.read(unitProvider)),
                          style: const TextStyle(
                            color: Color(0xFF888899),
                            fontSize: 10,
                            fontFamily: 'monospace',
                          ),
                        );
                      },
                    ),
                  ),
                  bottomTitles: AxisTitles(
                    sideTitles: SideTitles(
                      showTitles: true,
                      reservedSize: 22,
                      interval: xInterval,
                      getTitlesWidget: (value, meta) {
                        if (value == meta.max) return const SizedBox.shrink();
                        return Padding(
                          padding: const EdgeInsets.only(top: 4),
                          child: Text(
                            formatDistance(value, unit: ref.read(unitProvider)),
                            style: const TextStyle(
                              color: Color(0xFF888899),
                              fontSize: 10,
                              fontFamily: 'monospace',
                            ),
                          ),
                        );
                      },
                    ),
                  ),
                  topTitles: const AxisTitles(
                    sideTitles: SideTitles(showTitles: false),
                  ),
                  rightTitles: const AxisTitles(
                    sideTitles: SideTitles(showTitles: false),
                  ),
                ),
                lineTouchData: LineTouchData(
                  enabled: widget.enableLineTouch,
                  handleBuiltInTouches: true,
                  getTouchLineEnd: (_, _) {
                    return yMax;
                  },
                  touchCallback: (event, response) {
                    if (event is FlLongPressEnd ||
                        event is FlPanEndEvent ||
                        event is FlTapUpEvent) {
                      setState(() {
                        _selectedIndex = null;
                      });
                      widget.onLineTouch?.call(null);
                    } else if (response?.lineBarSpots?.isNotEmpty == true) {
                      final index = response!.lineBarSpots!.first.spotIndex;
                      setState(() {
                        _selectedIndex = index;
                      });
                      final point = _points[index];
                      widget.onLineTouch?.call(point);
                    }
                  },
                  getTouchedSpotIndicator: (barData, spotIndexes) {
                    return spotIndexes.map((i) {
                      return TouchedSpotIndicatorData(
                        FlLine(
                          color: Theme.of(
                            context,
                          ).primaryColor.withValues(alpha: 0.4),
                          strokeWidth: 1,
                        ),
                        FlDotData(
                          getDotPainter: (spot, percent, bar, index) =>
                              FlDotCirclePainter(
                                radius: 5,
                                color: _gradientColor(_points[index].gradient),
                                strokeColor: Colors.white,
                                strokeWidth: 1.5,
                              ),
                        ),
                      );
                    }).toList();
                  },
                  touchTooltipData: LineTouchTooltipData(
                    getTooltipColor: (_) => Theme.of(context).primaryColor,
                    tooltipBorderRadius: const BorderRadius.all(
                      Radius.circular(8),
                    ),
                    getTooltipItems: (spots) {
                      return spots.map((spot) => null).toList();
                    },
                  ),
                ),
                lineBarsData: [barData],

                // ── Waypoint vertical lines ───────────────────────────────────
                extraLinesData: ExtraLinesData(
                  verticalLines: waypoints.map((w) {
                    return VerticalLine(
                      x: w.distanceFromStart!,
                      color: Theme.of(
                        context,
                      ).colorScheme.onSurface.withValues(alpha: 0.2),
                      strokeWidth: 1,
                      dashArray: [4, 4],
                    );
                  }).toList(),
                ),
              ),
              duration: const Duration(milliseconds: 0),
            ),

            // ── Waypoint icon markers ─────────────────────────────────────────
            ...waypoints.map((w) {
              final fraction = (w.distanceFromStart! / maxDist).clamp(0.0, 1.0);
              final left =
                  (leftAxisWidth + fraction * plotWidth - (iconSize + 8) / 2)
                      .clamp(
                        leftAxisWidth,
                        constraints.maxWidth - (iconSize + 8),
                      );
              return Positioned(
                left: left,
                top: 4,
                child: Container(
                  padding: EdgeInsets.all(4),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.primary,
                    borderRadius: BorderRadius.all(Radius.circular(16)),
                  ),
                  child: FaIcon(
                    w.icon,
                    size: iconSize,
                    color: Theme.of(context).colorScheme.onPrimaryContainer,
                  ),
                ),
              );
            }),
          ],
        );
      },
    );
  }

  LinearGradient _buildLineGradient() {
    final totalDist = _points.last.distanceM;
    if (totalDist == 0) {
      return const LinearGradient(colors: [Colors.purple, Colors.purple]);
    }

    final colors = <Color>[];
    final stops = <double>[];

    for (final pt in _points) {
      final stop = (pt.distanceM / totalDist).clamp(0.0, 1.0);
      if (stops.isNotEmpty && stop <= stops.last) continue;
      stops.add(stop);
      colors.add(_gradientColor(pt.gradient));
    }

    if (stops.first > 0.0) {
      stops.insert(0, 0.0);
      colors.insert(0, colors.first);
    }
    if (stops.last < 1.0) {
      stops.add(1.0);
      colors.add(colors.last);
    }

    return LinearGradient(
      begin: Alignment.centerLeft,
      end: Alignment.centerRight,
      colors: colors,
      stops: stops,
    );
  }

  LinearGradient _buildFillGradient() {
    final totalDist = _points.last.distanceM;
    if (totalDist == 0) {
      return LinearGradient(
        colors: [
          const Color(0xFF6D28D9).withValues(alpha: 0.3),
          Colors.transparent,
        ],
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
      );
    }

    final colors = <Color>[];
    final stops = <double>[];

    for (final pt in _points) {
      final stop = (pt.distanceM / totalDist).clamp(0.0, 1.0);
      if (stops.isNotEmpty && stop <= stops.last) continue;
      stops.add(stop);
      colors.add(_gradientColor(pt.gradient).withValues(alpha: 0.18));
    }

    if (stops.first > 0.0) {
      stops.insert(0, 0.0);
      colors.insert(0, colors.first);
    }
    if (stops.last < 1.0) {
      stops.add(1.0);
      colors.add(colors.last);
    }

    return LinearGradient(
      begin: Alignment.centerLeft,
      end: Alignment.centerRight,
      colors: colors,
      stops: stops,
    );
  }

  List<TrackPoint> _simplifyTrackPoints(
    List<TrackPoint> points,
    int targetCount,
  ) {
    if (points.length <= targetCount) return points;

    final numBuckets = targetCount ~/ 2;
    final bucketSize = points.length / numBuckets;
    final result = <TrackPoint>[];

    result.add(points.first);

    for (int i = 0; i < numBuckets; i++) {
      final start = max(1, (i * bucketSize).floor());
      final end = min(points.length - 1, ((i + 1) * bucketSize).floor());
      if (start >= end) continue;

      int minIdx = start;
      int maxIdx = start;
      double minEle = points[start].elevationM;
      double maxEle = points[start].elevationM;

      for (int j = start + 1; j < end; j++) {
        final ele = points[j].elevationM;
        if (ele < minEle) {
          minEle = ele;
          minIdx = j;
        }
        if (ele > maxEle) {
          maxEle = ele;
          maxIdx = j;
        }
      }

      if (minIdx == maxIdx) {
        result.add(points[minIdx]);
      } else if (minIdx < maxIdx) {
        result.add(points[minIdx]);
        result.add(points[maxIdx]);
      } else {
        result.add(points[maxIdx]);
        result.add(points[minIdx]);
      }
    }

    if (result.last != points.last) {
      result.add(points.last);
    }

    final uniqueResult = <TrackPoint>[];
    for (final pt in result) {
      if (uniqueResult.isEmpty || uniqueResult.last != pt) {
        uniqueResult.add(pt);
      }
    }

    return uniqueResult;
  }
}

class TrackPoint {
  final double distanceM;
  double elevationM;
  Duration duration;
  double gradient;
  Color color;
  Geographic lonlat;

  TrackPoint({
    required this.distanceM,
    required this.elevationM,
    required this.lonlat,

    this.duration = Duration.zero,
    this.gradient = 0,
    this.color = Colors.white,
  });
}

void _smoothElevations(List<TrackPoint> points, {required int windowSize}) {
  if (windowSize < 1) windowSize = 1;
  if (windowSize == 1 || points.length < 2) return;

  final half = windowSize ~/ 2;
  final smoothed = List<double>.filled(points.length, 0);

  for (int i = 0; i < points.length; i++) {
    final start = max(0, i - half);
    final end = min(points.length, i + half + 1); // exclusive

    double weightedSum = 0;
    double weightTotal = 0;

    for (int j = start; j < end; j++) {
      final weight = (j - start + 1).toDouble();
      weightedSum += points[j].elevationM * weight;
      weightTotal += weight;
    }

    smoothed[i] = weightedSum / weightTotal;
  }

  for (int i = 0; i < points.length; i++) {
    points[i].elevationM = smoothed[i];
  }
}

Color _gradientColor(double gradientPct) {
  final g = gradientPct.clamp(-20.0, 20.0);

  // ── Descents (negative) ───────────────────────────────────────────────────
  if (g <= -10) return const Color(0xFF1565C0); // deep blue
  if (g <= -6) return const Color(0xFF2196F3); // blue
  if (g <= -3) return const Color(0xFF4DD0E1); // cyan
  if (g < 0) return const Color(0xFF90CAF9); // light blue

  // ── Flat ──────────────────────────────────────────────────────────────────
  if (g < 3) return const Color(0xFF9575CD); // soft purple

  // ── Ascents (positive) ────────────────────────────────────────────────────
  if (g < 6) return const Color(0xFFCDDC39); // yellow-green
  if (g < 9) return const Color(0xFFFFA726); // amber
  if (g < 13) return const Color(0xFFEF6C00); // deep orange
  return const Color(0xFFD32F2F); // crimson red
}

double _haversine(Wpt a, Wpt b) {
  const r = 6371.0;
  final lat1 = _deg2rad(a.lat ?? 0);
  final lat2 = _deg2rad(b.lat ?? 0);
  final dLat = _deg2rad((b.lat ?? 0) - (a.lat ?? 0));
  final dLon = _deg2rad((b.lon ?? 0) - (a.lon ?? 0));
  final h =
      sin(dLat / 2) * sin(dLat / 2) +
      cos(lat1) * cos(lat2) * sin(dLon / 2) * sin(dLon / 2);
  return r * 1000 * 2 * atan2(sqrt(h), sqrt(1 - h));
}

double _deg2rad(double deg) => deg * pi / 180;

double _niceInterval(double range, int targetCount) {
  if (range <= 0) return 1.0; // guard against flat/zero-range data
  final raw = range / targetCount;
  final magnitude = pow(10, (log(raw) / ln10).floor()).toDouble();
  final residual = raw / magnitude;
  if (residual <= 1) return magnitude;
  if (residual <= 2) return 2 * magnitude;
  if (residual <= 5) return 5 * magnitude;
  return 10 * magnitude;
}

class _EmptyState extends StatelessWidget {
  const _EmptyState();

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 200,
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.secondaryContainer,
        borderRadius: BorderRadius.circular(16),
      ),
      alignment: Alignment.center,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.terrain, size: 48),
          SizedBox(height: 8),
          Text(
            'No track data',
            style: TextStyle(
              color: Theme.of(context).colorScheme.onSurface,
              fontSize: 14,
            ),
          ),
        ],
      ),
    );
  }
}
