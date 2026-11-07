import 'package:maplibre/maplibre.dart';
import 'package:wanderer/util/route/map_matcher.dart';

/// Resolves live GPS fixes to a distance along a GPX track in that track's
/// OWN cumulative-metre coordinate — the elevation chart's x-axis (see
/// `buildElevationTrackPoints`'s raw-axis comment).
///
/// Deliberately reuses [RouteMapMatcher] (an HMM with hairpin/out-and-back
/// robustness) rather than a nearest-point projection, which flips legs on a
/// self-overlapping trail. This is NOT the `Navigation` provider's own
/// matcher: that one runs on the Valhalla map-snapped `NavigateResponse.shape`
/// — a different, map-matched, ≤500-point axis — while this one runs on the
/// GPX's own raw polyline so its result lands directly on the chart's axis.
class TrackPositionMatcher {
  TrackPositionMatcher({
    required List<Geographic> shape,
    required List<double> cumulativeMeters,
    this.onTrackThresholdMeters = 50.0,
  }) : assert(
         shape.length == cumulativeMeters.length,
         'shape and cumulativeMeters must be the same length',
       ),
       _shape = shape,
       _cumulative = cumulativeMeters,
       _matcher = RouteMapMatcher(
         shape: shape,
         shapeCumulativeMeters: cumulativeMeters,
       );

  final List<Geographic> _shape;
  final List<double> _cumulative;
  final RouteMapMatcher _matcher;

  /// Distance (metres) from a fix to the point the matcher committed to,
  /// beyond which [update] returns null (hides the marker) rather than a
  /// possibly-misleading position. Generous — 15–30 m GPS accuracy is
  /// routine outdoors, more under canopy or in a gorge — while still hiding
  /// the marker once the user has clearly left the track. Aligned with
  /// [RouteMapMatcher.offRouteCrossTrackThresholdMeters] (40 m) plus slack;
  /// a discretionary choice, not a derived constant.
  final double onTrackThresholdMeters;

  /// Feeds one GPS fix into the underlying matcher and returns the along-
  /// track distance (chart x-axis units) when [pos] is within
  /// [onTrackThresholdMeters] of the point the matcher committed to, or null
  /// when off-track (or uncertain — see below).
  ///
  /// The gate measures distance from [pos] to the matcher's COMMITTED point
  /// (not a perpendicular cross-track distance) — so it also reads null
  /// while [RouteMapMatcher] is coasting on an unconfirmed leg jump (see its
  /// `commitConfirmFixes` docs), which is the desired "uncertain → hide"
  /// behaviour rather than reporting a possibly-wrong leg.
  double? update(
    Geographic pos, {
    double? heading,
    double? headingAccuracy,
    double? speed,
    double? accuracy,
  }) {
    if (_shape.length < 2) return null;

    final result = _matcher.update(
      pos: pos,
      heading: heading,
      headingAccuracy: headingAccuracy,
      speed: speed,
      accuracy: accuracy,
    );
    final matched = pointAtAlongTrack(
      result.alongTrackMeters,
      result.shapeIndex,
    );
    final distance = SphericalGreatCircle(matched).distanceTo(pos);
    return distance <= onTrackThresholdMeters ? result.alongTrackMeters : null;
  }

  /// The point on the shape's polyline at [alongTrackMeters], interpolated
  /// within the segment starting at [shapeIndex] (clamped into range).
  Geographic pointAtAlongTrack(double alongTrackMeters, int shapeIndex) {
    final i = shapeIndex.clamp(0, _shape.length - 2);
    final segLen = _cumulative[i + 1] - _cumulative[i];
    if (segLen <= 0) return _shape[i];
    final t = ((alongTrackMeters - _cumulative[i]) / segLen).clamp(0.0, 1.0);
    return SphericalGreatCircle(
      _shape[i],
    ).intermediatePointTo(_shape[i + 1], fraction: t);
  }
}
