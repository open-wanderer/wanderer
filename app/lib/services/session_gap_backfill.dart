import 'package:geolocator/geolocator.dart' as geo;
import 'package:maplibre/maplibre.dart' as ml;
import 'package:tracelet/tracelet.dart' as tl;
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart'
    show NavigationStatsNotifier;
import 'package:wanderer/services/tracelet_position_source.dart'
    show hasUsableAltitude;
import 'package:wanderer/util/geo/polyline.dart';

/// Upper bound on the moving time credited between two consecutive fixes.
///
/// Moving time is normally accumulated by a 1s ticker in
/// `NavigationStatsNotifier`, which cannot run while the app is dead. Summing
/// the intervals between recorded fixes reconstructs it, but a single long
/// gap — GPS lost in a valley, the OS throttling the service — would otherwise
/// be credited in full as walking. Capping each step keeps a stall from
/// inflating the total.
const Duration _kMaxStepElapsed = Duration(seconds: 60);

/// Splices the locations tracelet recorded while the app was dead into [row].
///
/// `stopOnTerminate: false` keeps tracelet recording after the process is
/// killed, and `PersistMode.all` writes every fix to its own SQLite store —
/// but nothing in the app was reading that back. The breadcrumb is rehydrated
/// from this row, whose snapshot is written by `NavigationScreen._persistNow`
/// (dead along with the app), and `Tracelet.onLocation` only delivers fixes
/// from subscription onward. So the walked-while-closed segment was invisible
/// from both directions even though tracelet had every point.
///
/// Accumulates by the same rules as the live path — raw Haversine distance,
/// [NavigationStatsNotifier.kAltitudeNoiseFloorMeters] elevation floor,
/// [hasUsableAltitude] gating — so a resumed session's totals do not depend on
/// where the app happened to be killed.
///
/// Best-effort: any failure leaves [row] untouched and the session resumes
/// with what it already had. Returns the number of points spliced in.
Future<int> backfillSessionGap(ActiveNavigationEntity row) async {
  // Paused sessions accumulate nothing live, so they must not backfill either.
  if (row.isPaused) return 0;

  final timestamps = row.timestampsUtc;
  final polyline = row.breadcrumbPolyline;
  // Without a last timestamp there is no gap boundary to splice from, and
  // without a breadcrumb there is nothing to splice onto.
  if (timestamps == null || timestamps.isEmpty) return 0;
  if (polyline == null || polyline.isEmpty) return 0;

  final lastMs = timestamps.last;
  if (lastMs <= 0) return 0;

  final List<tl.Location> stored;
  try {
    // Fetched unfiltered and narrowed here: tracelet does not export SQLQuery.
    // Retention is bounded by PersistenceConfig.maxDaysToPersist (1 by
    // default), so the set stays small.
    stored = await tl.Tracelet.getLocations();
  } catch (_) {
    return 0;
  }

  final existing = PolylineUtil.decode(polyline);
  if (existing.isEmpty) return 0;

  final elevations = List<double>.from(row.elevations ?? const <double>[]);
  final points = List<ml.Geographic>.from(existing);
  final times = List<int>.from(timestamps);

  var lastPoint = existing.last;
  // Anchor elevation on the last point that carried a usable reading, not
  // simply the last point: the persistence path writes 0.0 for a fix with no
  // altitude, and diffing against that phantom would register the next real
  // reading as a climb of the device's full absolute altitude.
  double? lastAltitude;
  for (var i = elevations.length - 1; i >= 0; i--) {
    if (elevations[i] != 0.0) {
      lastAltitude = elevations[i];
      break;
    }
  }

  var distance = 0.0;
  var gain = 0.0;
  var loss = 0.0;
  var elapsed = Duration.zero;
  var lastTimeMs = lastMs;
  var added = 0;

  for (final location in stored) {
    final DateTime time;
    try {
      time = DateTime.parse(location.timestamp);
    } catch (_) {
      continue;
    }
    final ms = time.toUtc().millisecondsSinceEpoch;
    if (ms <= lastMs) continue; // Already in the breadcrumb.

    // Mirrors the live breadcrumb gate: a stationary fix is not walked
    // distance, and appending it would plant a cluster at every rest stop.
    if (!location.isMoving) {
      lastTimeMs = ms;
      continue;
    }

    final c = location.coords;
    final here = ml.Geographic(lat: c.latitude, lon: c.longitude);
    distance += ml.SphericalGreatCircle(lastPoint).distanceTo(here);
    lastPoint = here;

    // hasUsableAltitude reads a geo.Position; build the minimum it inspects.
    final usable = hasUsableAltitude(
      geo.Position(
        latitude: c.latitude,
        longitude: c.longitude,
        altitude: c.altitude,
        altitudeAccuracy: c.altitudeAccuracy,
        accuracy: c.accuracy,
        heading: c.heading,
        headingAccuracy: c.headingAccuracy,
        speed: c.speed,
        speedAccuracy: c.speedAccuracy,
        timestamp: time,
      ),
    );
    if (usable) {
      if (lastAltitude != null) {
        final delta = c.altitude - lastAltitude;
        if (delta.abs() >= NavigationStatsNotifier.kAltitudeNoiseFloorMeters) {
          if (delta > 0) {
            gain += delta;
          } else {
            loss += -delta;
          }
          lastAltitude = c.altitude;
        }
      } else {
        lastAltitude = c.altitude;
      }
    }

    final step = Duration(milliseconds: ms - lastTimeMs);
    elapsed += step > _kMaxStepElapsed ? _kMaxStepElapsed : step;
    lastTimeMs = ms;

    points.add(here);
    // 0.0 for an unusable reading, matching what _persistNow writes for a
    // breadcrumb point whose Wpt carries no `ele`.
    elevations.add(usable ? c.altitude : 0.0);
    times.add(ms);
    added++;
  }

  if (added == 0) return 0;

  row.breadcrumbPolyline = PolylineUtil.encode(points);
  row.elevations = elevations;
  row.timestampsUtc = times;
  row.distanceMeters += distance;
  row.elevationGainMeters += gain;
  row.elevationLossMeters += loss;
  row.currentElapsedSeconds += elapsed.inSeconds;
  row.updatedAtUtc = DateTime.now().toUtc();

  return added;
}
