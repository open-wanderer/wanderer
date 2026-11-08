import 'dart:math';

import 'package:collection/collection.dart';
import 'package:duration/duration.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/util/format.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/gpx/gpx.dart';

class ElevationProfile extends ConsumerStatefulWidget {
  final Trail? trail;
  final Gpx gpx;

  final double chartHeight;
  final int smoothingWindowSize;
  final Function(TrackPoint? point)? onLineTouch;
  final bool enableLineTouch;

  /// Overrides the header's total-duration stat when [gpx] has no `time`
  /// data (e.g. the route planner's in-progress `Gpx`, which has no
  /// timestamps). Ignored when `null`.
  final Duration? durationOverride;

  /// The user's live along-track distance in the chart's own x units (raw
  /// cumulative metres — see [buildElevationTrackPoints]). Non-null draws a
  /// "you are here" vertical guide + dot on the curve; null hides it. A
  /// [ValueListenable] so a per-fix update rebuilds only the chart, never
  /// the parent widget. Producers should compute this with
  /// `TrackPositionMatcher` over [buildRawTrackPoints] of the SAME [gpx], so
  /// the marker lands on the identical axis the chart plots on.
  final ValueListenable<double?>? livePositionMeters;

  const ElevationProfile({
    super.key,
    this.trail,
    required this.gpx,
    this.chartHeight = 150,
    this.smoothingWindowSize = 30,
    this.onLineTouch,
    this.enableLineTouch = true,
    this.durationOverride,
    this.livePositionMeters,
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
    // identical() first: gpx 2.3.0's == has no identity short-circuit — it
    // deep-walks every trackpoint even when comparing an object to itself,
    // which made this check O(track) on every parent rebuild.
    if (!identical(oldWidget.gpx, widget.gpx) && oldWidget.gpx != widget.gpx ||
        oldWidget.smoothingWindowSize != widget.smoothingWindowSize) {
      _points = _parseGpx(widget.gpx, widget.smoothingWindowSize);
      _selectedIndex = null;
    }
  }

  List<TrackPoint> _parseGpx(Gpx gpx, int windowSize) =>
      buildElevationTrackPoints(gpx, windowSize);

  @override
  Widget build(BuildContext context) {
    // Watched here so the widget rebuilds on unit change; helper methods
    // read it via ref.read within the same frame.
    ref.watch(unitProvider);

    // A single point is not a profile: every axis range collapses to zero and
    // the chart degenerates to one dot with no line. Reachable now that points
    // lacking elevation are skipped rather than plotted at 0 — a recording
    // whose first breadcrumb point came from the altitude-less seed fix has
    // exactly one plottable point until its second real fix lands.
    if (_points.length < 2) {
      return _EmptyState(height: widget.chartHeight);
    }

    final minElev = _points.map((p) => p.elevationM).reduce(min);
    final maxElev = _points.map((p) => p.elevationM).reduce(max);
    final maxDist = _points.last.distanceM;
    final maxDur = widget.durationOverride ?? _points.last.duration;

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
      final trail = widget.trail;
      final double elevationGain;
      final double elevationLoss;
      if (trail != null) {
        elevationGain = trail.elevationGain;
        elevationLoss = trail.elevationLoss;
      } else {
        final metrics = computeTrailMetrics(widget.gpx);
        elevationGain = metrics.elevationGain;
        elevationLoss = metrics.elevationLoss;
      }
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
            formatElevation(elevationGain, unit: ref.read(unitProvider)),
            FontAwesomeIcons.arrowTrendUp,
          ),
          _buildStatText(
            formatElevation(elevationLoss, unit: ref.read(unitProvider)),
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

  // Must match leftTitles reservedSize so icon X positions align with the
  // plot area — a private State field (not local) since both _buildChart
  // and _lineChart need it.
  static const _leftAxisWidth = 36.0;
  static const _iconSize = 14.0;

  Widget _buildChart(
    double minElev,
    double yMin,
    double yMax,
    double maxDist,
    double xInterval,
    double yInterval,
  ) {
    final waypoints = (widget.trail?.expand?.waypointsViaTrail ?? [])
        .where(
          (w) => w.distanceFromStart != null && w.distanceFromStart! <= maxDist,
        )
        .toList();

    return LayoutBuilder(
      builder: (context, constraints) {
        final plotWidth = constraints.maxWidth - _leftAxisWidth;

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
            widget.livePositionMeters == null
                ? _lineChart(
                    context,
                    null,
                    barData: barData,
                    spots: spots,
                    waypoints: waypoints,
                    yMin: yMin,
                    yMax: yMax,
                    maxDist: maxDist,
                    xInterval: xInterval,
                    yInterval: yInterval,
                  )
                : ValueListenableBuilder<double?>(
                    valueListenable: widget.livePositionMeters!,
                    builder: (context, liveMeters, _) => _lineChart(
                      context,
                      liveMeters,
                      barData: barData,
                      spots: spots,
                      waypoints: waypoints,
                      yMin: yMin,
                      yMax: yMax,
                      maxDist: maxDist,
                      xInterval: xInterval,
                      yInterval: yInterval,
                    ),
                  ),

            // ── Waypoint icon markers ─────────────────────────────────────────
            ...waypoints.map((w) {
              final fraction = (w.distanceFromStart! / maxDist).clamp(0.0, 1.0);
              final left =
                  (_leftAxisWidth + fraction * plotWidth - (_iconSize + 8) / 2)
                      .clamp(
                        _leftAxisWidth,
                        constraints.maxWidth - (_iconSize + 8),
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
                    size: _iconSize,
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

  /// Builds the actual `LineChart` — factored out of [_buildChart] so the
  /// live-marker branch there can rebuild only this widget (via
  /// [ValueListenableBuilder]) on every GPS fix, never the surrounding
  /// [LayoutBuilder]/waypoint-icon tree. [liveMeters] is the along-track
  /// distance of the live marker (chart x units), or null to hide it.
  Widget _lineChart(
    BuildContext context,
    double? liveMeters, {
    required LineChartBarData barData,
    required List<FlSpot> spots,
    required List<Waypoint> waypoints,
    required double yMin,
    required double yMax,
    required double maxDist,
    required double xInterval,
    required double yInterval,
  }) {
    final lineBarsData = <LineChartBarData>[barData];

    final verticalLines = waypoints.map((w) {
      return VerticalLine(
        x: w.distanceFromStart!,
        color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.2),
        strokeWidth: 1,
        dashArray: [4, 4],
      );
    }).toList();

    if (liveMeters != null && maxDist > 0) {
      final x = liveMeters.clamp(0.0, maxDist);
      final y = elevationAtDistance(_points, x);
      final primary = Theme.of(context).colorScheme.primary;
      // Solid — the waypoint lines above are dashed [4, 4] and the grid
      // [6, 4], so solid reads as "live" rather than a static reference.
      verticalLines.add(
        VerticalLine(
          x: x,
          color: primary.withValues(alpha: 0.6),
          strokeWidth: 1.5,
        ),
      );
      // A single-spot bar: fl_chart's generateNormalBarPath explicitly
      // supports `size == 1`, and its dot is drawn by drawDots when
      // dotData.show is true. The dot mirrors the scrub indicator's
      // radius-5/stroked look so the two read as the same family.
      lineBarsData.add(
        LineChartBarData(
          spots: [FlSpot(x, y)],
          isCurved: false,
          color: Colors.transparent,
          barWidth: 0,
          dotData: FlDotData(
            show: true,
            getDotPainter: (spot, percent, bar, index) => FlDotCirclePainter(
              radius: 5,
              color: primary,
              strokeColor: Theme.of(context).colorScheme.surface,
              strokeWidth: 2,
            ),
          ),
        ),
      );
    }

    return LineChart(
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
              reservedSize: _leftAxisWidth,
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
            } else {
              // Touch collects the nearest spot from EVERY bar and sorts by
              // pixel distance — with the live-marker bar present, `.first`
              // could be its single spot instead of the profile line's.
              // Only barIndex 0 (the profile line) should ever drive
              // scrubbing.
              final spot = response?.lineBarSpots?.firstWhereOrNull(
                (s) => s.barIndex == 0,
              );
              if (spot != null) {
                final index = spot.spotIndex;
                setState(() {
                  _selectedIndex = index;
                });
                final point = _points[index];
                widget.onLineTouch?.call(point);
              }
            }
          },
          getTouchedSpotIndicator: (barData, spotIndexes) {
            // fl_chart passes no bar index here. The profile line's own
            // dotData is `show: false` while the live-marker bar's is
            // `show: true`, so that flag is what discriminates the two —
            // the marker bar must never render the scrub indicator.
            if (barData.dotData.show) {
              return List<TouchedSpotIndicatorData?>.filled(
                spotIndexes.length,
                null,
              );
            }
            return spotIndexes.map((i) {
              return TouchedSpotIndicatorData(
                FlLine(
                  color: Theme.of(
                    context,
                  ).colorScheme.onSurface.withValues(alpha: 0.4),
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
            tooltipBorderRadius: const BorderRadius.all(Radius.circular(8)),
            getTooltipItems: (spots) {
              return spots.map((spot) => null).toList();
            },
          ),
        ),
        lineBarsData: lineBarsData,

        // ── Waypoint (+ live marker) vertical lines ────────────────────────
        extraLinesData: ExtraLinesData(verticalLines: verticalLines),
      ),
      duration: const Duration(milliseconds: 0),
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

/// Shown when fewer than two points carry elevation. The track itself may be
/// complete, so the copy must not claim that it is missing. Unfilled because
/// the profile sits on several different surfaces.
class _EmptyState extends StatelessWidget {
  final double height;

  const _EmptyState({required this.height});

  @override
  Widget build(BuildContext context) {
    final color = Theme.of(context).colorScheme.onSurfaceVariant;
    return SizedBox(
      height: height,
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.terrain, size: 32, color: color),
            const SizedBox(height: 8),
            Text(
              AppLocalizations.of(context)!.no_elevation_data,
              textAlign: TextAlign.center,
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(color: color),
            ),
          ],
        ),
      ),
    );
  }
}

/// Douglas-Peucker-style thinning for the chart. Top-level (was a State
/// member) so buildElevationTrackPoints below can be tested outside a
/// widget. Pure — it never touched instance state.
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

/// Every plottable sample of [gpx], unsmoothed and unsimplified.
///
/// `distanceM` is the elevation chart's x-axis coordinate; `lonlat` the
/// sample position — together they are the polyline `TrackPositionMatcher`
/// matches live fixes against, so the marker lands on the chart's own axis.
/// Keep this the single source of that axis: the chart's own points (see
/// [buildElevationTrackPoints]) are a subset of these with identical
/// `distanceM`.
///
/// Top-level (was a private State method, which put the chart's distance
/// accumulation out of reach of any unit test — and that is precisely where
/// the raw-vs-smoothed mismatch below hid) — also called directly by
/// `navigation_screen.dart` to build the polyline `TrackPositionMatcher`
/// matches against.
List<TrackPoint> buildRawTrackPoints(Gpx gpx) {
  if (gpx.allWaypoints.isEmpty) return [];

  final result = <TrackPoint>[];
  Duration cumDuration = Duration.zero;

  // RAW per-sample cumulative distance — deliberately NOT the smoothed total.
  //
  // These two are different things and must not be conflated: the smoothed
  // accumulator (computeTrailMetrics / trail.distance) is a DENOISED ESTIMATE
  // OF TRAIL LENGTH, while this is a PLOTTING COORDINATE saying where each
  // sample actually sits along the track. They legitimately differ, and the
  // gap grows with GPS jitter.
  //
  // Driving the axis from the smoothed accumulator was tried and reverted: it
  // only advances once per ~5 m, so consecutive samples share an x, and the
  // gradient below (dElev / dDist) then divides a one-sample numerator by a
  // zero or multi-sample denominator. Measured on a true constant 10% grade at
  // 1.5 m sampling: 46 of 60 points read 0.0% and the rest 2.5%, all of which
  // fall in _gradientColor's "flat" bucket — the chart's gradient colouring is
  // destroyed for any 1 Hz recording. Waypoint markers broke too, since
  // Waypoint.distanceFromStart is raw from both producers.
  //
  // If the axis maximum must equal the trail's reported distance, scale this
  // axis by (smoothed / raw) and keep computing the gradient from raw deltas —
  // do not swap the accumulator.
  double cumDist = 0;
  Wpt? prevPoint;

  for (final track in gpx.trks) {
    for (final segment in track.trksegs) {
      for (final wpt in segment.trkpts) {
        if (wpt.lat == null || wpt.lon == null) continue;
        // A point with no usable elevation is skipped rather than plotted at
        // 0. `wpt.ele ?? 0` was the third of the three disagreeing answers to
        // "is this elevation usable?" that conversion.dart's hasUsablePosition
        // doc comment warns about, and it is visible: a live recording's first
        // breadcrumb point carries no altitude when the session seeded from an
        // already-resolved map marker, so plotting it at 0 drew a cliff from
        // sea level to the device's real altitude at the start of the chart.
        // computeTrailMetrics already skips these; the chart now agrees.
        final elevation = parseGpxElevation(wpt.ele);
        if (elevation == null) continue;

        if (prevPoint != null) {
          final hop = haversineMeters(prevPoint, wpt);
          if (hop.isFinite) cumDist += hop;

          final prevTime = prevPoint.time;
          final currTime = wpt.time;
          if (prevTime != null && currTime != null) {
            final delta = currTime.difference(prevTime);
            if (delta > Duration.zero) cumDuration += delta;
          }
        }
        prevPoint = wpt;

        result.add(
          TrackPoint(
            distanceM: cumDist,
            elevationM: elevation,
            lonlat: Geographic(lat: wpt.lat!, lon: wpt.lon!),
            duration: cumDuration,
            gradient: 0,
            color: Colors.white,
          ),
        );
      }
    }
  }

  return result;
}

/// Builds the elevation chart's track points from [gpx]: the same raw walk
/// as [buildRawTrackPoints], smoothed, coloured by gradient, and thinned to
/// 250 points for the chart.
///
/// Top-level and `@visibleForTesting` purely so it can be exercised without a
/// widget: this was a private State method, which put the chart's distance
/// accumulation out of reach of any unit test — and that is precisely where
/// the raw-vs-smoothed mismatch below hid.
@visibleForTesting
List<TrackPoint> buildElevationTrackPoints(Gpx gpx, int windowSize) {
  final result = buildRawTrackPoints(gpx);
  if (result.isEmpty) return [];

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

/// Linear-interpolated elevation at [distanceM] along [points] (chart x-axis
/// units — see [buildElevationTrackPoints]). Clamps to the first/last
/// point's elevation outside the track's range. Requires `points.length >=
/// 2` and strictly ascending `distanceM` (guaranteed by the builders above).
@visibleForTesting
double elevationAtDistance(List<TrackPoint> points, double distanceM) {
  assert(points.length >= 2, 'elevationAtDistance needs at least 2 points');

  if (distanceM <= points.first.distanceM) return points.first.elevationM;
  if (distanceM >= points.last.distanceM) return points.last.elevationM;

  var lo = 0;
  var hi = points.length - 1;
  while (hi - lo > 1) {
    final mid = (lo + hi) >> 1;
    if (points[mid].distanceM <= distanceM) {
      lo = mid;
    } else {
      hi = mid;
    }
  }

  final a = points[lo];
  final b = points[hi];
  final span = b.distanceM - a.distanceM;
  if (span <= 0) return a.elevationM;
  final t = (distanceM - a.distanceM) / span;
  return a.elevationM + (b.elevationM - a.elevationM) * t;
}
