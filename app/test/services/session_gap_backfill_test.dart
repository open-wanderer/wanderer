import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:tracelet/tracelet.dart' as tl;
import 'package:wanderer/entities/active_navigation_entity.dart';
import 'package:wanderer/services/session_gap_backfill.dart';
import 'package:wanderer/util/geo/polyline.dart';

// The recording that died at 10:38:43 in issue #1321, reduced to two points.
final _lastFixUtc = DateTime.utc(2026, 10, 11, 10, 38, 43);

ActiveNavigationEntity _row({bool isPaused = false}) {
  final points = [
    ml.Geographic(lat: 53.11561, lon: 9.4443),
    ml.Geographic(lat: 53.11571, lon: 9.44441),
  ];
  return ActiveNavigationEntity(
    sessionType: ActiveSessionType.rec,
    breadcrumbPolyline: PolylineUtil.encode(points),
    elevations: [70.0, 73.63],
    timestampsUtc: [
      _lastFixUtc.subtract(const Duration(seconds: 5)).millisecondsSinceEpoch,
      _lastFixUtc.millisecondsSinceEpoch,
    ],
    distanceMeters: 1322.0,
    currentElapsedSeconds: 100,
    isPaused: isPaused,
    updatedAtUtc: DateTime.utc(2026, 10, 11, 10, 38, 50),
  );
}

tl.Location _fix(
  DateTime at, {
  required double lat,
  required double altitude,
  bool isMoving = true,
}) => tl.Location.fromMap({
  'uuid': at.toIso8601String(),
  'timestamp': at.toIso8601String(),
  'is_moving': isMoving,
  'coords': {
    'latitude': lat,
    'longitude': 9.44441,
    'altitude': altitude,
    'altitudeAccuracy': 3.0,
    'accuracy': 5.0,
    'speed': 2.8,
    'heading': 0.0,
  },
});

/// What tracelet's store holds for the gap: one fix already in the breadcrumb,
/// three moving fixes ~11 m apart, a stationary one, and a late moving fix
/// 75 s after it.
List<tl.Location> _store() => [
  _fix(
    _lastFixUtc.subtract(const Duration(seconds: 13)),
    lat: 53.11565,
    altitude: 73.0,
  ),
  _fix(
    _lastFixUtc.add(const Duration(seconds: 17)),
    lat: 53.11581,
    altitude: 73.7,
  ),
  _fix(
    _lastFixUtc.add(const Duration(seconds: 22)),
    lat: 53.11591,
    altitude: 76.0,
  ),
  _fix(
    _lastFixUtc.add(const Duration(seconds: 27)),
    lat: 53.11601,
    altitude: 76.5,
  ),
  _fix(
    _lastFixUtc.add(const Duration(seconds: 32)),
    lat: 53.11601,
    altitude: 76.5,
    isMoving: false,
  ),
  _fix(
    _lastFixUtc.add(const Duration(seconds: 107)),
    lat: 53.11701,
    altitude: 72.0,
  ),
];

void main() {
  group('backfillSessionGap', () {
    test('readies tracelet before it reads the store', () async {
      // Mirrors the native SDK: `getLocations()` answers an empty list as a
      // plain success until `ready()` has run in the process. A relaunch after
      // the process was killed is such a process, and it used to read the store
      // before anything readied it — splicing nothing, silently, every time.
      final calls = <String>[];
      var ready = false;
      final row = _row();

      final added = await backfillSessionGap(
        row,
        ensureTraceletReady: () async {
          calls.add('ready');
          ready = true;
        },
        readStoredLocations: () async {
          calls.add('read');
          return ready ? _store() : <tl.Location>[];
        },
      );

      expect(calls, ['ready', 'read']);
      expect(added, 4);
    });

    test('splices only moving fixes newer than the breadcrumb', () async {
      final row = _row();

      final added = await backfillSessionGap(
        row,
        ensureTraceletReady: () async {},
        readStoredLocations: () async => _store(),
      );

      // The fix older than the last breadcrumb point and the stationary one
      // are left out; the four moving ones are appended in order.
      expect(added, 4);
      final points = PolylineUtil.decode(row.breadcrumbPolyline!);
      expect(points, hasLength(6));
      expect(points.last.lat, closeTo(53.11701, 1e-9));
      expect(row.elevations, hasLength(6));
      expect(row.timestampsUtc, hasLength(6));
      expect(
        row.timestampsUtc!.last,
        _lastFixUtc.add(const Duration(seconds: 107)).millisecondsSinceEpoch,
      );
    });

    test('accumulates distance, elevation and time by the live rules', () async {
      final row = _row();

      await backfillSessionGap(
        row,
        ensureTraceletReady: () async {},
        readStoredLocations: () async => _store(),
      );

      // Three ~11.1 m hops plus one ~111 m hop from the last breadcrumb point.
      expect(row.distanceMeters, closeTo(1322.0 + 144.5, 2.0));
      // 73.63 -> 76.0 clears the 2 m noise floor (the 73.7 reading does not),
      // and 76.0 -> 72.0 is the loss.
      expect(row.elevationGainMeters, closeTo(2.37, 1e-6));
      expect(row.elevationLossMeters, closeTo(4.0, 1e-6));
      // 17 + 5 + 5 s, then the 75 s stretch after the stationary fix is capped
      // at 60 s so a stall is not credited as walking.
      expect(row.currentElapsedSeconds, 100 + 17 + 5 + 5 + 60);
    });

    test('leaves the row untouched when readying tracelet fails', () async {
      final row = _row();
      final polylineBefore = row.breadcrumbPolyline;

      final added = await backfillSessionGap(
        row,
        ensureTraceletReady: () async => throw StateError('no native SDK'),
        readStoredLocations: () async => _store(),
      );

      expect(added, 0);
      expect(row.breadcrumbPolyline, polylineBefore);
      expect(row.timestampsUtc, hasLength(2));
      expect(row.distanceMeters, 1322.0);
    });

    test('does not ready tracelet for a paused session', () async {
      // Paused sessions accumulate nothing live, so they backfill nothing —
      // and have no business restarting the native pipeline to find that out.
      var readied = false;

      final added = await backfillSessionGap(
        _row(isPaused: true),
        ensureTraceletReady: () async => readied = true,
        readStoredLocations: () async => _store(),
      );

      expect(added, 0);
      expect(readied, isFalse);
    });
  });
}
