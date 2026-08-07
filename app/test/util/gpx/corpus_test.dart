import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/theme/icons.dart';

/// The corpus is a single, on-disk, language-neutral fixture set at
/// the repo root, sibling to app/, web/, db/ — not duplicated per language.
/// See fixtures/gpx-corpus/README.md for the full contract this file
/// implements, and web/src/lib/models/gpx/gpx-corpus.test.ts for the TS
/// counterpart this suite must stay field-for-field comparable with.
///
/// `flutter test` runs with CWD == the package root
/// (`app/`), so this relative path needs no pubspec `assets:` entry.
final Directory _corpusRoot = Directory('../fixtures/gpx-corpus');

/// Absolute tolerances, declared once here and never inline in a test
/// body. Empirically justified in fixtures/gpx-corpus/README.md: a
/// 5000-point haversine accumulation was bit-identical between Node/V8 and
/// the Dart VM to 17 significant digits, so any failure at these
/// tolerances is an algorithmic divergence, not floating-point noise.
const double kMetreTolerance = 1e-6; // metres — distance/elevation fields
const double kDegreeTolerance = 1e-9; // degrees — centroid fields only

class _CorpusFixture {
  final String dir;
  final String inputGpx;
  final Map<String, dynamic> expected;

  const _CorpusFixture({
    required this.dir,
    required this.inputGpx,
    required this.expected,
  });

  Map<String, dynamic> get metrics =>
      expected['metrics'] as Map<String, dynamic>;
  Map<String, dynamic> get trail => expected['trail'] as Map<String, dynamic>;
  Map<String, dynamic> get boundingBox =>
      metrics['boundingBox'] as Map<String, dynamic>;
  Map<String, dynamic> get centroid =>
      metrics['centroid'] as Map<String, dynamic>;
}

/// Reads every fixture directory under fixtures/gpx-corpus/ from disk.
/// Asserts at least 12 fixtures are discovered so a corpus that
/// silently shrinks (a fixture accidentally deleted or excluded) fails this
/// suite rather than passing vacuously.
List<_CorpusFixture> _loadFixtures() {
  final dirs =
      _corpusRoot
          .listSync()
          .whereType<Directory>()
          .map((d) => d.path.split(Platform.pathSeparator).last)
          .toList()
        ..sort();

  final fixtures = dirs.map((dir) {
    final base = '${_corpusRoot.path}/$dir';
    final inputGpx = File('$base/input.gpx').readAsStringSync();
    final expected =
        jsonDecode(File('$base/expected.json').readAsStringSync())
            as Map<String, dynamic>;
    return _CorpusFixture(dir: dir, inputGpx: inputGpx, expected: expected);
  }).toList();

  // `expect()` cannot run outside a `test()` body (main() runs at load
  // time), so a shrunk corpus is surfaced as a thrown StateError here,
  // matching the existsSync guard above's loud-failure style.
  if (fixtures.length < 12) {
    throw StateError(
      'GPX corpus at ${_corpusRoot.path} shrank below 12 fixtures '
      '(found ${fixtures.length})',
    );
  }
  return fixtures;
}

/// Asserts `(actual - expected).abs() <= tol`, naming the failing field via
/// [label] so a failure identifies which value diverged.
void _expectClose(double actual, num expected, double tol, String label) {
  expect(
    (actual - expected.toDouble()).abs(),
    lessThanOrEqualTo(tol),
    reason:
        '$label: $actual not within $tol of $expected (diff ${(actual - expected.toDouble()).abs()})',
  );
}

void main() {
  // A wrong CWD must fail loudly (zero fixtures discovered), not
  // silently pass a vacuous suite. `flutter test` must be run from `app/`.
  if (!_corpusRoot.existsSync()) {
    throw StateError(
      'GPX corpus not found at resolved path "${_corpusRoot.path}" '
      '(cwd: ${Directory.current.path}). This test must be run with '
      '`flutter test`\'s working directory set to "app/" '
      '(e.g. "cd app && flutter test"), not the repo root.',
    );
  }

  final fixtures = _loadFixtures();

  group('gpx corpus - metrics', () {
    for (final fixture in fixtures) {
      test('${fixture.dir}: computeTrailMetrics reproduces the corpus\'s metrics', () {
        final gpx = parseGpxSafely(fixture.inputGpx);
        final metrics = computeTrailMetrics(gpx);

        _expectClose(
          metrics.distance,
          fixture.metrics['distance'] as num,
          kMetreTolerance,
          '${fixture.dir}: metrics.distance',
        );
        _expectClose(
          metrics.elevationGain,
          fixture.metrics['elevationGain'] as num,
          kMetreTolerance,
          '${fixture.dir}: metrics.elevationGain',
        );
        _expectClose(
          metrics.elevationLoss,
          fixture.metrics['elevationLoss'] as num,
          kMetreTolerance,
          '${fixture.dir}: metrics.elevationLoss',
        );

        _expectClose(
          metrics.centroidLat,
          fixture.centroid['lat'] as num,
          kDegreeTolerance,
          '${fixture.dir}: metrics.centroid.lat',
        );
        _expectClose(
          metrics.centroidLon,
          fixture.centroid['lon'] as num,
          kDegreeTolerance,
          '${fixture.dir}: metrics.centroid.lon',
        );

        expect(
          metrics.minLat,
          fixture.boundingBox['minLat'],
          reason: '${fixture.dir}: metrics.minLat',
        );
        expect(
          metrics.maxLat,
          fixture.boundingBox['maxLat'],
          reason: '${fixture.dir}: metrics.maxLat',
        );
        expect(
          metrics.minLon,
          fixture.boundingBox['minLon'],
          reason: '${fixture.dir}: metrics.minLon',
        );
        expect(
          metrics.maxLon,
          fixture.boundingBox['maxLon'],
          reason: '${fixture.dir}: metrics.maxLon',
        );
        expect(
          metrics.durationMs,
          fixture.metrics['durationMs'],
          reason: '${fixture.dir}: metrics.durationMs',
        );
        expect(
          metrics.pointCount,
          fixture.metrics['pointCount'],
          reason: '${fixture.dir}: metrics.pointCount',
        );

        // Deliberately not asserted anywhere in this file: the raw
        // index-aligned per-point distance array, or the track-shape hash.
        // Both exist on the TS class but are outside the corpus's
        // public-metrics-only contract. Fixture 03 (the empty-ele canary)
        // reaching this assertion without GpxReader throwing is the proof
        // that the sanitize pass (parseGpxSafely) is wired into
        // the read path.
      });
    }
  });

  group('gpx corpus - trail assembly', () {
    for (final fixture in fixtures) {
      test('${fixture.dir}: trailFromGpx reproduces the corpus\'s trail block', () {
        final gpx = parseGpxSafely(fixture.inputGpx);
        final trail = trailFromGpx(gpx, gpxData: fixture.inputGpx);

        expect(
          trail.name,
          fixture.trail['name'],
          reason: '${fixture.dir}: trail.name',
        );
        // The TS model leaves `description` `undefined`/`null`; the Dart
        // model's field is non-nullable with a `''` default — null and
        // empty string are treated as equivalent for this field only.
        expect(
          trail.description,
          fixture.trail['description'] ?? '',
          reason: '${fixture.dir}: trail.description',
        );
        expect(
          trail.lat,
          fixture.trail['lat'],
          reason: '${fixture.dir}: trail.lat',
        );
        expect(
          trail.lon,
          fixture.trail['lon'],
          reason: '${fixture.dir}: trail.lon',
        );

        // trail.date is only asserted when the fixture supplies a
        // non-null expected date. Trail's constructor otherwise defaults
        // date to a non-deterministic "now" value when trailFromGpx finds
        // no start/end trkpt time to derive it from — that default is a
        // Trail model implementation detail, not part of the GPX-derived
        // contract this corpus pins.
        final expectedDate = fixture.trail['date'] as String?;
        if (expectedDate != null) {
          final actualDate = trail.date;
          expect(
            actualDate,
            isNotNull,
            reason: '${fixture.dir}: trail.date expected non-null',
          );
          final formatted =
              '${actualDate!.year.toString().padLeft(4, '0')}-'
              '${actualDate.month.toString().padLeft(2, '0')}-'
              '${actualDate.day.toString().padLeft(2, '0')}';
          expect(
            formatted,
            expectedDate,
            reason: '${fixture.dir}: trail.date',
          );
        }

        _expectClose(
          trail.distance,
          fixture.trail['distance'] as num,
          kMetreTolerance,
          '${fixture.dir}: trail.distance',
        );
        _expectClose(
          trail.elevationGain,
          fixture.trail['elevationGain'] as num,
          kMetreTolerance,
          '${fixture.dir}: trail.elevationGain',
        );
        _expectClose(
          trail.elevationLoss,
          fixture.trail['elevationLoss'] as num,
          kMetreTolerance,
          '${fixture.dir}: trail.elevationLoss',
        );

        expect(
          trail.duration,
          (fixture.trail['duration'] as num).toDouble(),
          reason: '${fixture.dir}: trail.duration',
        );

        expect(
          trail.minLat,
          fixture.trail['minLat'],
          reason: '${fixture.dir}: trail.minLat',
        );
        expect(
          trail.maxLat,
          fixture.trail['maxLat'],
          reason: '${fixture.dir}: trail.maxLat',
        );
        expect(
          trail.minLon,
          fixture.trail['minLon'],
          reason: '${fixture.dir}: trail.minLon',
        );
        expect(
          trail.maxLon,
          fixture.trail['maxLon'],
          reason: '${fixture.dir}: trail.maxLon',
        );

        // The corpus is a pure function of a GPX and can never pin
        // moving time — trailFromGpx is called with no movingDuration
        // override, so this must always be null.
        expect(
          trail.movingDuration,
          isNull,
          reason: '${fixture.dir}: trail.movingDuration must be null',
        );

        final actualWaypoints = trail.expand?.waypointsViaTrail ?? [];
        final expectedWaypoints = fixture.trail['waypoints'] as List<dynamic>;
        expect(
          actualWaypoints.length,
          expectedWaypoints.length,
          reason: '${fixture.dir}: waypoint count',
        );

        for (var i = 0; i < expectedWaypoints.length; i++) {
          final actualWaypoint = actualWaypoints[i];
          final expectedWaypoint =
              expectedWaypoints[i] as Map<String, dynamic>;

          expect(
            actualWaypoint.lat,
            expectedWaypoint['lat'],
            reason: '${fixture.dir}: waypoint[$i].lat',
          );
          expect(
            actualWaypoint.lon,
            expectedWaypoint['lon'],
            reason: '${fixture.dir}: waypoint[$i].lon',
          );
          expect(
            actualWaypoint.name,
            expectedWaypoint['name'] ?? '',
            reason: '${fixture.dir}: waypoint[$i].name',
          );
          expect(
            actualWaypoint.description,
            expectedWaypoint['description'] ?? '',
            reason: '${fixture.dir}: waypoint[$i].description',
          );

          // Assert the map lookup SUCCEEDED before comparing.
          // Computing the expected value with the same
          // `fontAwesomeIconsMap[...] ?? circle` expression the production
          // code uses made this assertion tautological — if the map lost the
          // `campground` or `mountain` key both sides would collapse to
          // `circle` and it would still pass, while the TS counterpart
          // (which compares against the literal sym string) would fail. The
          // two suites are only field-for-field comparable here if this side
          // pins the key's presence independently.
          final expectedIcon = expectedWaypoint['icon'] as String?;
          if (expectedIcon != null) {
            expect(
              fontAwesomeIconsMap[expectedIcon],
              isNotNull,
              reason:
                  '${fixture.dir}: waypoint[$i] expects sym "$expectedIcon", '
                  'which is missing from theme/icons.dart\'s '
                  'fontAwesomeIconsMap',
            );
          }
          expect(
            actualWaypoint.icon,
            expectedIcon != null
                ? fontAwesomeIconsMap[expectedIcon]
                : FontAwesomeIcons.circle,
            reason: '${fixture.dir}: waypoint[$i].icon',
          );
        }
      });
    }
  });

  group('gpx corpus - integrity', () {
    // Iterated per fixture (rather than one aggregate assertion loop) so a
    // failure identifies the offending fixture by name in the test
    // description, mirroring the TS suite's integrity describe.
    for (final fixture in fixtures) {
      test(
        '${fixture.dir}: expected.json id matches its directory name and, if hand-derived, has a DERIVATION.md sibling',
        () {
          expect(
            fixture.expected['id'],
            fixture.dir,
            reason: 'fixture directory "${fixture.dir}"',
          );

          if (fixture.expected['derivation'] == 'hand') {
            final derivationFile = File(
              '${_corpusRoot.path}/${fixture.dir}/DERIVATION.md',
            );
            expect(
              derivationFile.existsSync(),
              isTrue,
              reason: '${fixture.dir}/DERIVATION.md',
            );
          }
        },
      );
    }
  });
}
