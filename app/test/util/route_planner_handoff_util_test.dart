import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/gpx_conversion_util.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/route_planner_handoff_util.dart';

// Tests for the pure handoff helpers (no network/navigation), plus
// buildDraftTrail, which since 34-05 builds its draft trail entirely
// on-device via the ported `trailFromGpx` (see buildLocalTrail in
// trail_import_util.dart) and so needs a WidgetRef only for its optional,
// online-only reverse-geocode fill (D-07). finishPlanning's own
// orchestration is still not unit-tested here — it has no seam beyond
// buildDraftTrail worth re-testing.

/// Fakes any Dio traffic `buildDraftTrail`'s optional reverse-geocode step
/// might issue while online. Mirrors the `_FakeApi` pattern in
/// `test/provider/route_anchor_provider_test.dart`.
class _FakeApi extends Api {
  _FakeApi({this.response, this.shouldFail = false});

  final Map<String, dynamic>? response;
  final bool shouldFail;

  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (shouldFail) {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.connectionError,
              ),
            );
            return;
          }
          handler.resolve(
            Response(
              requestOptions: options,
              statusCode: 200,
              data: response,
            ),
          );
        },
      ),
    );
    return dio;
  }
}

class _FakeOnlineStatus extends OnlineStatus {
  _FakeOnlineStatus(this._initial);

  final bool _initial;

  @override
  bool build() => _initial;
}

/// Pumps a bare `Consumer` inside a `ProviderScope` overriding [apiProvider]
/// with [_FakeApi] and [onlineStatusProvider] with [_FakeOnlineStatus]
/// (defaulting to offline, so the default test path performs no
/// reverse-geocode request at all), and hands back the captured [WidgetRef]
/// for the test to call ref-requiring functions with.
Future<WidgetRef> _pumpRef(
  WidgetTester tester, {
  Map<String, dynamic>? response,
  bool shouldFail = false,
  bool online = false,
}) async {
  late WidgetRef capturedRef;
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        apiProvider.overrideWith(
          () => _FakeApi(response: response, shouldFail: shouldFail),
        ),
        onlineStatusProvider.overrideWith(() => _FakeOnlineStatus(online)),
      ],
      child: Consumer(
        builder: (context, ref, _) {
          capturedRef = ref;
          return const SizedBox();
        },
      ),
    ),
  );
  return capturedRef;
}

/// Fakes `/valhalla/height`'s per-chunk response so [fetchHeightsForShape]'s
/// batching can be tested without a real server. [respond] receives the
/// request's `shape` list and returns the `height` list to answer with — a
/// length mismatch or a thrown error lets tests exercise the silent-fallback
/// paths per-chunk.
class _FakeHeightApi extends Api {
  _FakeHeightApi(this.respond);

  final List<num> Function(List<dynamic> shape) respond;
  int requestCount = 0;

  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requestCount++;
          try {
            final shape = (options.data as Map)['shape'] as List;
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {'height': respond(shape)},
              ),
            );
          } catch (e) {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.connectionError,
                error: e,
              ),
            );
          }
        },
      ),
    );
    return dio;
  }
}

Future<WidgetRef> _pumpHeightRef(
  WidgetTester tester,
  List<num> Function(List<dynamic> shape) respond,
) async {
  late WidgetRef capturedRef;
  await tester.pumpWidget(
    ProviderScope(
      overrides: [apiProvider.overrideWith(() => _FakeHeightApi(respond))],
      child: Consumer(
        builder: (context, ref, _) {
          capturedRef = ref;
          return const SizedBox();
        },
      ),
    ),
  );
  return capturedRef;
}

/// Fakes both `/valhalla/trace-route` and `/valhalla/height` so
/// [buildFinalPlannedGpx]'s snap-then-heights pipeline (Task 2, D-14/T-34-28)
/// can be exercised without a real server. [snapShape] answers any
/// trace-route request unconditionally — one fixed shape is enough to prove
/// the boundary re-pin, since it is deliberately built to differ from every
/// real anchor coordinate. [heights] answers any height request; when
/// [shouldFailAll] is true, every request (either endpoint) is rejected as a
/// connection error, doubling as a "no request was attempted" guard.
class _FakeSnapApi extends Api {
  _FakeSnapApi({required this.snapShape, this.heights, this.shouldFailAll = false});

  final List<Map<String, double>> snapShape;
  final List<num> Function(List<dynamic> shape)? heights;
  final bool shouldFailAll;

  int traceRouteCalls = 0;
  int heightCalls = 0;

  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          if (shouldFailAll) {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.connectionError,
              ),
            );
            return;
          }
          if (options.path.contains('trace-route')) {
            traceRouteCalls++;
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {'shape': snapShape},
              ),
            );
            return;
          }
          if (options.path.contains('height')) {
            heightCalls++;
            final shape = (options.data as Map)['shape'] as List;
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {
                  'height': heights?.call(shape) ?? List<num>.filled(shape.length, 0),
                },
              ),
            );
            return;
          }
          handler.reject(
            DioException(
              requestOptions: options,
              type: DioExceptionType.connectionError,
            ),
          );
        },
      ),
    );
    return dio;
  }
}

Future<WidgetRef> _pumpSnapRef(WidgetTester tester, _FakeSnapApi api) async {
  late WidgetRef capturedRef;
  await tester.pumpWidget(
    ProviderScope(
      overrides: [apiProvider.overrideWith(() => api)],
      child: Consumer(
        builder: (context, ref, _) {
          capturedRef = ref;
          return const SizedBox();
        },
      ),
    ),
  );
  return capturedRef;
}

/// A 3-anchor / 2-leg route, as `segmentPolylinesFromTrack` would hand it to
/// `seedFromTrack`: one polyline per leg, each sharing its boundary anchor
/// with its neighbour, and each with an interior point so a straight-line
/// fallback is distinguishable from the real geometry.
final _twoLegRoute = <List<ml.Geographic>>[
  [
    ml.Geographic(lat: 47.000, lon: 9.000),
    ml.Geographic(lat: 47.0005, lon: 9.0015), // off the direct line
    ml.Geographic(lat: 47.001, lon: 9.001),
  ],
  [
    ml.Geographic(lat: 47.001, lon: 9.001),
    ml.Geographic(lat: 47.0015, lon: 9.0025),
    ml.Geographic(lat: 47.002, lon: 9.002),
  ],
];

/// Seeds [routeAnchorsProvider] with an edit session over [legs], exactly as
/// RoutePlannerScreen does on open, and lets each segment's fire-and-forget
/// height fetch settle. Anchors are the legs' shared boundary points.
///
/// Runs under [WidgetTester.runAsync]: Dio schedules its interceptor pipeline
/// via `Timer.run`, a real timer that `testWidgets`' fake clock never fires.
Future<void> _seedRoute(
  WidgetTester tester,
  WidgetRef ref,
  List<List<ml.Geographic>> legs,
) async {
  final anchors = [
    if (legs.isNotEmpty) legs.first.first,
    for (final leg in legs) leg.last,
  ];
  await tester.runAsync(() async {
    ref
        .read(routeAnchorsProvider.notifier)
        .seedFromTrack(anchors, 'pedestrian', null, segmentPolylines: legs);
    await Future<void>.delayed(const Duration(milliseconds: 50));
  });
}

/// Calls [buildDraftTrail] inside [WidgetTester.runAsync] — `testWidgets`
/// runs under a fake clock that never fires real `Timer`s on its own, and
/// Dio schedules its interceptor pipeline via `Timer.run` (a real timer), so
/// awaiting the call directly hangs forever (relevant whenever the online
/// reverse-geocode branch is exercised). `runAsync` steps outside the fake
/// zone into a real one so that timer actually fires.
Future<Trail> _buildDraftTrail(
  WidgetTester tester,
  WidgetRef ref,
  Gpx gpx, {
  String? category,
  String? subcategory,
  double? durationSeconds,
  Duration? movingDuration,
}) async {
  return (await tester.runAsync(
    () => buildDraftTrail(
      ref,
      gpx,
      category: category,
      subcategory: subcategory,
      durationSeconds: durationSeconds,
      movingDuration: movingDuration,
    ),
  ))!;
}

void main() {
  group('buildDraftTrail', () {
    // Two track points, lat 47.000/lon 9.000/ele 400 and lat 47.001/lon
    // 9.001/ele 410 — matches the CONTEXT.md-mandated fixture shape for this
    // plan's re-pointed assertions.
    Gpx buildSampleGpx() {
      final gpx = Gpx();
      gpx.trks = [
        Trk(
          trksegs: [
            Trkseg(
              trkpts: [
                Wpt(lat: 47.000, lon: 9.000, ele: 400),
                Wpt(lat: 47.001, lon: 9.001, ele: 410),
              ],
            ),
          ],
        ),
      ];
      return gpx;
    }

    /// The same two points, but with `time` stamped [span] apart, so
    /// duration comes from the GPX rather than a timeless-fallback.
    Gpx buildTimestampedGpx(Duration span) {
      final start = DateTime.utc(2026, 1, 1, 8, 0, 0);
      final gpx = Gpx();
      gpx.trks = [
        Trk(
          trksegs: [
            Trkseg(
              trkpts: [
                Wpt(lat: 47.000, lon: 9.000, ele: 400, time: start),
                Wpt(
                  lat: 47.001,
                  lon: 9.001,
                  ele: 410,
                  time: start.add(span),
                ),
              ],
            ),
          ],
        ),
      ];
      return gpx;
    }

    testWidgets(
      'sets a non-empty expand.gpxData containing "<gpx" (Pitfall 1)',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.expand?.gpxData, isNotNull);
        expect(result.expand!.gpxData, isNotEmpty);
        expect(result.expand!.gpxData, contains('<gpx'));
      },
    );

    testWidgets(
      'leaves expand.waypointsViaTrail empty for a track with no <wpt>s',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.expand?.waypointsViaTrail, isEmpty);
      },
    );

    testWidgets('keeps expand.gpx identical to the finalGpx passed in', (
      tester,
    ) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester);

      final result = await _buildDraftTrail(tester, ref, gpx);

      expect(identical(result.expand!.gpx, gpx), isTrue);
    });

    testWidgets('passes a supplied category id through', (tester) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester);

      final result = await _buildDraftTrail(
        tester,
        ref,
        gpx,
        category: 'bike-id',
      );

      expect(result.category, 'bike-id');
    });

    testWidgets('leaves category null when none is supplied', (
      tester,
    ) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester);

      final result = await _buildDraftTrail(tester, ref, gpx);

      expect(result.category, isNull);
    });

    testWidgets(
      "clears subcategory to '' when none is supplied "
      '(trail_create_screen clear convention)',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.subcategory, '');
      },
    );

    testWidgets('a supplied subcategory id round-trips through', (
      tester,
    ) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester);

      final result = await _buildDraftTrail(
        tester,
        ref,
        gpx,
        subcategory: 'hiking-sub-id',
      );

      expect(result.subcategory, 'hiking-sub-id');
    });

    testWidgets(
      'sets bounds from the track and lat/lon from its first point '
      '(local computation reproduces gpx_util.ts:78-87)',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.maxLat, 47.001);
        expect(result.minLat, 47.000);
        expect(result.maxLon, 9.001);
        expect(result.minLon, 9.000);
        expect(result.lat, 47.000);
        expect(result.lon, 9.000);
      },
    );

    testWidgets(
      'distance and elevationGain are computed locally over the track',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.distance, computeTrailMetrics(gpx).distance);
        // The 400-to-410 climb clears the 5 m noise threshold.
        expect(result.elevationGain, 10);
      },
    );

    testWidgets(
      'applies the durationSeconds fallback for a timeless GPX',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(
          tester,
          ref,
          gpx,
          durationSeconds: 1200,
        );

        expect(result.duration, 1200);
      },
    );

    testWidgets(
      'does not apply the durationSeconds fallback when the GPX carries '
      'real timestamps',
      (tester) async {
        final gpx = buildTimestampedGpx(const Duration(minutes: 30));
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(
          tester,
          ref,
          gpx,
          durationSeconds: 999999,
        );

        expect(result.duration, const Duration(minutes: 30).inSeconds.toDouble());
      },
    );

    // D-12: the session-to-trail moving-time hand-off, pinned by its own
    // test rather than the shared corpus (moving time is not a function of
    // a GPX — the same GPX legitimately yields different values by
    // provenance).
    testWidgets(
      'movingDuration flows to trail.movingDuration while duration stays '
      'GPX-derived (D-11)',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(
          tester,
          ref,
          gpx,
          movingDuration: const Duration(hours: 1, minutes: 5),
        );

        expect(result.movingDuration, 3900.0);
        expect(result.duration, 0);
      },
    );

    testWidgets(
      'movingDuration is null when the parameter is omitted (import/planner '
      'paths keep reporting elapsed time, CONV-06)',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.movingDuration, isNull);
      },
    );

    testWidgets(
      'a recorded-shape GPX keeps duration (GPX elapsed) and movingDuration '
      "(the session's own) independent",
      (tester) async {
        final gpx = buildTimestampedGpx(const Duration(minutes: 90));
        final ref = await _pumpRef(tester);

        final result = await _buildDraftTrail(
          tester,
          ref,
          gpx,
          movingDuration: const Duration(minutes: 70),
        );

        expect(result.duration, 5400.0);
        expect(result.movingDuration, 4200.0);
      },
    );

    testWidgets(
      'performs no reverse-geocode request and leaves location null while '
      'offline (default test path)',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester, shouldFail: true);

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.location, isNull);
      },
    );

    testWidgets(
      'fills location via reverse geocode when online',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(
          tester,
          online: true,
          response: {
            'features': [
              {
                'properties': {
                  'address': {
                    'city': 'Testtown',
                    'country': 'Testland',
                  },
                },
              },
            ],
          },
        );

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.location, 'Testtown, Testland');
      },
    );
  });

  group('buildFinalPlannedGpx', () {
    // The regression these guard: the export used to flatten every leg into
    // a single trkseg (buildNavShape + mergeHeightsIntoGpx), so re-opening a
    // finished multi-leg route in the planner collapsed it to a bare
    // start/finish pair. The round-trip assertions below — export, then read
    // back through anchorsFromTrack/segmentPolylinesFromTrack, exactly as
    // trail_create_screen does — are the real guard; trkseg counts alone
    // would not catch a boundary landing one point off an anchor.

    testWidgets('a route with no legs exports a bare Gpx', (tester) async {
      final ref = await _pumpHeightRef(tester, (shape) => const []);

      final gpx = (await tester.runAsync(() => buildFinalPlannedGpx(ref)))!;

      expect(gpx.trks, isEmpty);
    });

    testWidgets('a 3-anchor / 2-leg route round-trips back to 3 anchors and '
        '2 full-resolution legs', (tester) async {
      final ref = await _pumpHeightRef(
        tester,
        (shape) => List<num>.filled(shape.length, 500),
      );
      await _seedRoute(tester, ref, _twoLegRoute);

      final gpx = (await tester.runAsync(() => buildFinalPlannedGpx(ref)))!;

      expect(gpx.trks.single.trksegs, hasLength(2));

      final anchors = anchorsFromTrack(gpx);
      expect(anchors, hasLength(3));
      expect(anchors[0].lat, _twoLegRoute[0].first.lat);
      expect(anchors[1].lat, _twoLegRoute[0].last.lat);
      expect(anchors[2].lat, _twoLegRoute[1].last.lat);

      final polylines = segmentPolylinesFromTrack(gpx, anchors);
      expect(polylines, hasLength(2));
      // Not the 2-point straight-line fallback: each leg's interior points
      // survived, so the user can still see (and edit) the real geometry.
      expect(polylines[0], hasLength(_twoLegRoute[0].length));
      expect(polylines[1], hasLength(_twoLegRoute[1].length));
      expect(polylines[0][1].lon, _twoLegRoute[0][1].lon);

      // Heights from each segment's own fetch are carried onto the export.
      expect(gpx.trks.single.trksegs.first.trkpts.first.ele, 500);
    });

    testWidgets('the flattened point stream carries no duplicated boundary '
        'point, so a re-export is byte-stable', (tester) async {
      final ref = await _pumpHeightRef(
        tester,
        (shape) => List<num>.filled(shape.length, 500),
      );
      await _seedRoute(tester, ref, _twoLegRoute);

      final first = (await tester.runAsync(() => buildFinalPlannedGpx(ref)))!;
      final total = _twoLegRoute[0].length + _twoLegRoute[1].length - 1;
      expect(first.allPoints, hasLength(total));

      // Feed the export back in as a fresh edit session and re-export: point
      // count must not grow. A shared-endpoint layout would gain one point
      // per boundary on every save/re-edit cycle.
      final anchors = anchorsFromTrack(first);
      await _seedRoute(tester, ref, segmentPolylinesFromTrack(first, anchors));
      final second = (await tester.runAsync(() => buildFinalPlannedGpx(ref)))!;

      expect(second.allPoints, hasLength(total));
      expect(anchorsFromTrack(second), hasLength(3));
    });

    testWidgets('a leg whose height fetch never resolved is backfilled at '
        'export time', (tester) async {
      var failUntilExport = true;
      final ref = await _pumpHeightRef(tester, (shape) {
        if (failUntilExport) throw StateError('height unavailable');
        return List<num>.filled(shape.length, 750);
      });
      await _seedRoute(tester, ref, _twoLegRoute);

      // Seeding left both segments without elevations.
      expect(
        ref.read(routeAnchorsProvider).segments.every((s) => s.elevations == null),
        isTrue,
      );

      failUntilExport = false;
      final gpx = (await tester.runAsync(() => buildFinalPlannedGpx(ref)))!;

      expect(gpx.trks.single.trksegs, hasLength(2));
      expect(gpx.trks.single.trksegs.first.trkpts.first.ele, 750);
    });

    testWidgets('a leg whose height fetch fails at export time still exports '
        'its geometry, with ele left null', (tester) async {
      final ref = await _pumpHeightRef(
        tester,
        (shape) => throw StateError('height unavailable'),
      );
      await _seedRoute(tester, ref, _twoLegRoute);

      final gpx = (await tester.runAsync(() => buildFinalPlannedGpx(ref)))!;

      expect(gpx.trks.single.trksegs, hasLength(2));
      expect(anchorsFromTrack(gpx), hasLength(3));
      expect(gpx.trks.single.trksegs.first.trkpts.first.ele, isNull);
    });

    // T-34-28 / Task 2: the leg-boundary re-pin. snapShapeToRoads's fake
    // response is deliberately built to differ from every real anchor
    // coordinate but keep a comparable bounding-box diagonal (so
    // snapResultAcceptable doesn't reject it as a truncation) — proving the
    // re-pin, not merely that the pre-snap shape happened to survive.
    testWidgets(
      'snapCosting supplied: a snapped leg whose endpoints differ from the '
      "anchors still round-trips through anchorsFromTrack to the route's "
      'original anchors, and trkseg count still equals the leg count',
      (tester) async {
        const driftedSnapShape = [
          {'lat': 47.0001, 'lon': 9.0002},
          {'lat': 47.0006, 'lon': 9.0017},
          {'lat': 47.0011, 'lon': 9.0012},
        ];
        final api = _FakeSnapApi(
          snapShape: driftedSnapShape,
          heights: (shape) => List<num>.filled(shape.length, 500),
        );
        final ref = await _pumpSnapRef(tester, api);
        await _seedRoute(tester, ref, _twoLegRoute);

        final gpx = (await tester.runAsync(
          () => buildFinalPlannedGpx(ref, snapCosting: 'pedestrian'),
        ))!;

        expect(gpx.trks.single.trksegs, hasLength(_twoLegRoute.length));

        final anchors = anchorsFromTrack(gpx);
        expect(anchors, hasLength(3));
        expect(anchors[0].lat, _twoLegRoute[0].first.lat);
        expect(anchors[0].lon, _twoLegRoute[0].first.lon);
        expect(anchors[1].lat, _twoLegRoute[0].last.lat);
        expect(anchors[1].lon, _twoLegRoute[0].last.lon);
        expect(anchors[2].lat, _twoLegRoute[1].last.lat);
        expect(anchors[2].lon, _twoLegRoute[1].last.lon);

        // The drifted midpoint survives (proving a real snap happened, not
        // a silent fallback to the pre-snap shape).
        final midEle = gpx.trks.single.trksegs.first.trkpts[1];
        expect(midEle.lat, isNot(_twoLegRoute[0][1].lat));
        expect(api.traceRouteCalls, greaterThan(0));
      },
    );

    testWidgets(
      'both flags false, api throws on any request: completes without '
      'throwing for a route whose legs already have elevations — proves '
      'the offline path issues no network call',
      (tester) async {
        // Seed once against a working api so every leg resolves real
        // elevations, exactly as a genuinely-online planning session would.
        final workingRef = await _pumpHeightRef(
          tester,
          (shape) => List<num>.filled(shape.length, 500),
        );
        await _seedRoute(tester, workingRef, _twoLegRoute);
        final seededState = workingRef.read(routeAnchorsProvider);
        expect(
          seededState.segments.every((s) => s.elevations != null),
          isTrue,
        );

        // A fresh session whose api rejects every request — shouldFailAll
        // doubles as a "no request was attempted" guard: if
        // buildFinalPlannedGpx regressed to calling the api while both
        // flags are off, this fake would reject and the test would fail
        // with a DioException instead of completing normally.
        final failingApi = _FakeSnapApi(snapShape: const [], shouldFailAll: true);
        final ref = await _pumpSnapRef(tester, failingApi);
        // Transplant the already-elevated state directly (bypassing the
        // network) so this session starts with real elevations already
        // resolved, same as the working session above.
        ref.read(routeAnchorsProvider.notifier).state = seededState;

        final gpx = await tester.runAsync(() => buildFinalPlannedGpx(ref));

        expect(gpx, isNotNull);
        expect(gpx!.trks.single.trksegs, hasLength(_twoLegRoute.length));
        expect(failingApi.traceRouteCalls, 0);
        expect(failingApi.heightCalls, 0);
      },
    );
  });

  group('anchorsFromTrack', () {
    test('single-trkseg track yields exactly [first, last] (2 anchors)', () {
      final gpx = Gpx();
      gpx.trks = [
        Trk(
          trksegs: [
            Trkseg(
              trkpts: [
                Wpt(lat: 47.000, lon: 9.000),
                Wpt(lat: 47.001, lon: 9.001),
                Wpt(lat: 47.002, lon: 9.002),
              ],
            ),
          ],
        ),
      ];

      final anchors = anchorsFromTrack(gpx);

      expect(anchors, hasLength(2));
      expect(anchors[0].lat, 47.000);
      expect(anchors[0].lon, 9.000);
      expect(anchors[1].lat, 47.002);
      expect(anchors[1].lon, 9.002);
    });

    test(
      'two-trkseg track yields [seg0.first, seg1.first, seg1.last] '
      '(3 anchors)',
      () {
        final gpx = Gpx();
        gpx.trks = [
          Trk(
            trksegs: [
              Trkseg(
                trkpts: [
                  Wpt(lat: 47.000, lon: 9.000),
                  Wpt(lat: 47.001, lon: 9.001),
                ],
              ),
              Trkseg(
                trkpts: [
                  Wpt(lat: 47.010, lon: 9.010),
                  Wpt(lat: 47.011, lon: 9.011),
                ],
              ),
            ],
          ),
        ];

        final anchors = anchorsFromTrack(gpx);

        expect(anchors, hasLength(3));
        expect(anchors[0].lat, 47.000);
        expect(anchors[0].lon, 9.000);
        expect(anchors[1].lat, 47.010);
        expect(anchors[1].lon, 9.010);
        expect(anchors[2].lat, 47.011);
        expect(anchors[2].lon, 9.011);
      },
    );

    test('an empty/trackless Gpx yields an empty list', () {
      expect(anchorsFromTrack(Gpx()), isEmpty);
    });

    test('a trk with an empty trksegs list yields an empty list (CR-01/WR-01 regression)', () {
      final gpx = Gpx();
      gpx.trks = [Trk(trksegs: const [])];

      expect(anchorsFromTrack(gpx), isEmpty);
    });

    test(
      'a trailing empty trkseg does not drop the true final point '
      '(WR-02 regression)',
      () {
        final gpx = Gpx();
        gpx.trks = [
          Trk(
            trksegs: [
              Trkseg(
                trkpts: [
                  Wpt(lat: 47.000, lon: 9.000),
                  Wpt(lat: 47.001, lon: 9.001),
                ],
              ),
              Trkseg(trkpts: const []),
            ],
          ),
        ];

        final anchors = anchorsFromTrack(gpx);

        expect(anchors, hasLength(2));
        expect(anchors[0].lat, 47.000);
        expect(anchors[0].lon, 9.000);
        expect(anchors[1].lat, 47.001);
        expect(anchors[1].lon, 9.001);
      },
    );

    test(
      'a trkpt with a null lat/lon is dropped rather than force-unwrapped '
      '(CR-01 regression)',
      () {
        final gpx = Gpx();
        gpx.trks = [
          Trk(
            trksegs: [
              Trkseg(
                trkpts: [
                  Wpt(lat: null, lon: null),
                  Wpt(lat: 47.001, lon: 9.001),
                  Wpt(lat: 47.002, lon: 9.002),
                ],
              ),
            ],
          ),
        ];

        expect(() => anchorsFromTrack(gpx), returnsNormally);
        final anchors = anchorsFromTrack(gpx);

        expect(anchors, hasLength(2));
        expect(anchors[0].lat, 47.001);
        expect(anchors[0].lon, 9.001);
        expect(anchors[1].lat, 47.002);
        expect(anchors[1].lon, 9.002);
      },
    );
  });

  group('segmentPolylinesFromTrack', () {
    test(
      'a single-trkseg track yields one segment whose polyline is every '
      'recorded point (preserves an off-road recording, not a straight '
      'line between the endpoints)',
      () {
        final gpx = Gpx();
        gpx.trks = [
          Trk(
            trksegs: [
              Trkseg(
                trkpts: [
                  Wpt(lat: 47.000, lon: 9.000),
                  Wpt(lat: 47.0005, lon: 9.0015), // off the direct line
                  Wpt(lat: 47.001, lon: 9.001),
                  Wpt(lat: 47.002, lon: 9.002),
                ],
              ),
            ],
          ),
        ];
        final anchors = anchorsFromTrack(gpx);

        final polylines = segmentPolylinesFromTrack(gpx, anchors);

        expect(polylines, hasLength(1));
        expect(polylines[0], hasLength(4));
        expect(polylines[0][1].lat, 47.0005);
        expect(polylines[0][1].lon, 9.0015);
      },
    );

    test(
      'a two-trkseg track yields one polyline per consecutive anchor pair, '
      'each a contiguous slice of the flattened recorded points',
      () {
        final gpx = Gpx();
        gpx.trks = [
          Trk(
            trksegs: [
              Trkseg(
                trkpts: [
                  Wpt(lat: 47.000, lon: 9.000),
                  Wpt(lat: 47.001, lon: 9.001),
                ],
              ),
              Trkseg(
                trkpts: [
                  Wpt(lat: 47.010, lon: 9.010),
                  Wpt(lat: 47.0105, lon: 9.0105),
                  Wpt(lat: 47.011, lon: 9.011),
                ],
              ),
            ],
          ),
        ];
        final anchors = anchorsFromTrack(gpx);

        final polylines = segmentPolylinesFromTrack(gpx, anchors);

        expect(anchors, hasLength(3)); // seg0.first, seg1.first, seg1.last
        expect(polylines, hasLength(2));
        // seg0.first -> seg1.first: seg0's own points then the jump into seg1
        expect(polylines[0], hasLength(3));
        expect(polylines[0][0].lat, 47.000);
        expect(polylines[0][2].lat, 47.010);
        // seg1.first -> seg1.last: all of seg1's own points
        expect(polylines[1], hasLength(3));
        expect(polylines[1][1].lat, 47.0105);
      },
    );

    test('fewer than 2 anchors yields an empty list', () {
      expect(segmentPolylinesFromTrack(Gpx(), const []), isEmpty);
    });

    // WR-01 regression. The not-found fallback used to resume at `k = i`,
    // dropping the still-outstanding (i-1, i) pair — so the list came back
    // one short AND mis-ordered relative to the anchor pairs. Its consumer
    // (`RouteAnchorsNotifier.seedFromTrack`) indexes positionally.
    group('not-found fallback (defensive path)', () {
      Gpx trackOf(List<(double, double)> pts) {
        final gpx = Gpx();
        gpx.trks = [
          Trk(
            trksegs: [
              Trkseg(
                trkpts: [for (final p in pts) Wpt(lat: p.$1, lon: p.$2)],
              ),
            ],
          ),
        ];
        return gpx;
      }

      test('still returns anchors.length - 1 polylines when a middle anchor '
          'is absent from the track', () {
        final gpx = trackOf(const [
          (47.000, 9.000),
          (47.001, 9.001),
          (47.002, 9.002),
        ]);
        final anchors = [
          ml.Geographic(lat: 47.000, lon: 9.000),
          ml.Geographic(lat: 47.001, lon: 9.001),
          // Not on the track at all.
          ml.Geographic(lat: 48.500, lon: 10.500),
          ml.Geographic(lat: 47.002, lon: 9.002),
        ];

        final polylines = segmentPolylinesFromTrack(gpx, anchors);

        expect(polylines, hasLength(anchors.length - 1));
        // Pair 0 was resolved against the real track...
        expect(polylines[0].first.lat, 47.000);
        expect(polylines[0].last.lat, 47.001);
        // ...and every remaining pair is the straight line for ITS OWN pair.
        expect(polylines[1].first.lat, 47.001);
        expect(polylines[1].last.lat, 48.500);
        expect(polylines[2].first.lat, 48.500);
        expect(polylines[2].last.lat, 47.002);
      });

      test('still returns anchors.length - 1 polylines when the FIRST anchor '
          'is absent from the track', () {
        final gpx = trackOf(const [(47.000, 9.000), (47.001, 9.001)]);
        final anchors = [
          ml.Geographic(lat: 48.500, lon: 10.500),
          ml.Geographic(lat: 47.000, lon: 9.000),
          ml.Geographic(lat: 47.001, lon: 9.001),
        ];

        final polylines = segmentPolylinesFromTrack(gpx, anchors);

        expect(polylines, hasLength(anchors.length - 1));
        expect(polylines[0].first.lat, 48.500);
        expect(polylines[0].last.lat, 47.000);
        expect(polylines[1].first.lat, 47.000);
        expect(polylines[1].last.lat, 47.001);
      });
    });
  });

  group('mergeRouteIntoTrail', () {
    Trail buildSampleTrail() {
      return Trail(
        id: 'trail-1',
        name: 'Sample Trail',
        description: 'A description',
        public: true,
        created: DateTime(2026),
        updated: DateTime(2026),
        expand: TrailExpand(
          waypointsViaTrail: [
            Waypoint(
              id: 'wp-1',
              lat: 47.5,
              lon: 9.5,
              created: DateTime(2026),
              updated: DateTime(2026),
            ),
          ],
        ),
      );
    }

    Gpx buildFinalGpx() {
      final gpx = Gpx();
      gpx.trks = [
        Trk(
          trksegs: [
            Trkseg(
              trkpts: [
                Wpt(lat: 48.000, lon: 10.000),
                Wpt(lat: 48.001, lon: 10.001),
              ],
            ),
          ],
        ),
      ];
      return gpx;
    }

    test('sets both expand.gpx and expand.gpxData (Pitfall 1)', () {
      final existing = buildSampleTrail();
      final finalGpx = buildFinalGpx();

      final result = mergeRouteIntoTrail(existing, finalGpx);

      expect(identical(result.expand!.gpx, finalGpx), isTrue);
      expect(result.expand!.gpxData, isNotEmpty);
      expect(result.expand!.gpxData, contains('<gpx'));
    });

    test(
      'preserves title/description/id and existing waypoints unchanged',
      () {
        final existing = buildSampleTrail();
        final finalGpx = buildFinalGpx();

        final result = mergeRouteIntoTrail(existing, finalGpx);

        expect(result.id, existing.id);
        expect(result.name, existing.name);
        expect(result.description, existing.description);
        expect(result.public, existing.public);
        expect(
          result.expand!.waypointsViaTrail,
          existing.expand!.waypointsViaTrail,
        );
      },
    );

    test('recomputes lat/lon/bounds from the finalGpx bounds', () {
      final existing = buildSampleTrail();
      final finalGpx = buildFinalGpx();

      final result = mergeRouteIntoTrail(existing, finalGpx);

      expect(result.maxLat, 48.001);
      expect(result.minLat, 48.000);
      expect(result.maxLon, 10.001);
      expect(result.minLon, 10.000);
      expect(result.lat, closeTo(48.0005, 1e-9));
      expect(result.lon, closeTo(10.0005, 1e-9));
    });

    test('falls back to existing bounds/lat/lon for a trackless finalGpx', () {
      final existing = buildSampleTrail().copyWith(
        lat: 1,
        lon: 2,
        maxLat: 3,
        minLat: 4,
        maxLon: 5,
        minLon: 6,
      );

      final result = mergeRouteIntoTrail(existing, Gpx());

      expect(result.lat, 1);
      expect(result.lon, 2);
      expect(result.maxLat, 3);
      expect(result.minLat, 4);
      expect(result.maxLon, 5);
      expect(result.minLon, 6);
    });
  });

  group('mergeHeightsIntoGpx', () {
    test('aligns ele to shape indices', () {
      final shape = [
        {'lat': 47.000, 'lon': 9.000},
        {'lat': 47.001, 'lon': 9.001},
        {'lat': 47.002, 'lon': 9.002},
      ];
      final heights = [400, 410, 420];

      final gpx = mergeHeightsIntoGpx(shape, heights);
      final points = gpx.trks.single.trksegs.single.trkpts;

      expect(points, hasLength(3));
      expect(points[0].lat, 47.000);
      expect(points[0].lon, 9.000);
      expect(points[0].ele, 400.0);
      expect(points[1].ele, 410.0);
      expect(points[2].ele, 420.0);
    });

    test('assigns null ele when heights is shorter than shape', () {
      final shape = [
        {'lat': 47.000, 'lon': 9.000},
        {'lat': 47.001, 'lon': 9.001},
        {'lat': 47.002, 'lon': 9.002},
      ];
      final heights = [400, 410];

      final gpx = mergeHeightsIntoGpx(shape, heights);
      final points = gpx.trks.single.trksegs.single.trkpts;

      expect(points[0].ele, 400.0);
      expect(points[1].ele, 410.0);
      expect(points[2].ele, isNull);
    });

    test('returns an empty Gpx (no tracks) for an empty shape', () {
      final gpx = mergeHeightsIntoGpx(const [], const []);

      expect(gpx.trks, isEmpty);
    });

    // CR-01 regression. Before the `source:` parameter existed this helper
    // returned a bare `Gpx()` carrying only `trks`, so a file import with
    // either post-capture toggle enabled permanently discarded the imported
    // document's waypoints, name and description — the stripped document was
    // both what the draft trail was built from AND what got uploaded.
    group('source: preserves the original document\'s non-track content', () {
      const sourceXml = '''
<?xml version="1.0"?>
<gpx version="1.1" creator="unit-test">
  <metadata>
    <name>Alpine Traverse</name>
    <desc>A long day out.</desc>
  </metadata>
  <wpt lat="47.0005" lon="9.0005">
    <name>Hut</name>
    <desc>Overnight stop</desc>
    <sym>campground</sym>
  </wpt>
  <rte><name>Planned variant</name>
    <rtept lat="47.0" lon="9.0"/>
  </rte>
  <trk>
    <name>Day 1</name>
    <desc>Track description</desc>
    <trkseg>
      <trkpt lat="47.000" lon="9.000"><ele>400</ele></trkpt>
      <trkpt lat="47.001" lon="9.001"><ele>410</ele></trkpt>
    </trkseg>
  </trk>
</gpx>
''';

      final shape = [
        {'lat': 47.000, 'lon': 9.000},
        {'lat': 47.001, 'lon': 9.001},
      ];

      test('carries metadata, wpts, rtes and the trk name onto the '
          'rebuilt document', () {
        final source = parseGpxSafely(sourceXml);

        final merged = mergeHeightsIntoGpx(
          shape,
          const [500, 510],
          source: source,
        );

        expect(merged.metadata?.name, 'Alpine Traverse');
        expect(merged.metadata?.desc, 'A long day out.');
        expect(merged.wpts, hasLength(1));
        expect(merged.wpts.single.name, 'Hut');
        expect(merged.rtes, hasLength(1));
        expect(merged.creator, 'unit-test');
        expect(merged.trks.single.name, 'Day 1');
        expect(merged.trks.single.desc, 'Track description');
        // The geometry IS replaced — that is the whole point of the merge.
        expect(merged.trks.single.trksegs.single.trkpts, hasLength(2));
        expect(merged.trks.single.trksegs.single.trkpts.first.ele, 500.0);
      });

      test('the draft trail built from the merged document keeps the '
          'GPX\'s own name, description and waypoints', () {
        final source = parseGpxSafely(sourceXml);

        final merged = mergeHeightsIntoGpx(
          shape,
          const [500, 510],
          source: source,
        );
        final trail = trailFromGpx(merged, fallbackName: 'track.gpx');

        expect(trail.name, 'Alpine Traverse');
        expect(trail.description, 'A long day out.');
        expect(trail.expand?.waypointsViaTrail, hasLength(1));
        expect(trail.expand?.waypointsViaTrail?.single.name, 'Hut');
      });

      test('the re-serialised document still contains the waypoint and '
          'metadata name (this is what gets uploaded)', () {
        final source = parseGpxSafely(sourceXml);

        final merged = mergeHeightsIntoGpx(
          shape,
          const [500, 510],
          source: source,
        );
        final xml = GpxWriter().asString(merged);

        expect(xml, contains('Alpine Traverse'));
        expect(xml, contains('<wpt'));
        expect(xml, contains('Hut'));
      });

      test('preserves non-track content even when the shape is empty', () {
        final source = parseGpxSafely(sourceXml);

        final merged = mergeHeightsIntoGpx(
          const [],
          const [],
          source: source,
        );

        expect(merged.trks, isEmpty);
        expect(merged.metadata?.name, 'Alpine Traverse');
        expect(merged.wpts, hasLength(1));
        expect(merged.rtes, hasLength(1));
      });

      test('without source: still returns a bare document (recording and '
          'planner paths are unchanged)', () {
        final merged = mergeHeightsIntoGpx(shape, const [500, 510]);

        expect(merged.metadata, isNull);
        expect(merged.wpts, isEmpty);
        expect(merged.rtes, isEmpty);
        expect(merged.trks.single.name, isNull);
      });
    });
  });

  // WR-02/WR-03 regression. Every caller passes buildNavShape's ≤500-point
  // request hint, so returning `shape` on the fallback path silently
  // persisted a decimation of a full-resolution track whenever the network
  // hiccuped — and gave callers no way to tell a real snap from a fallback.
  group('snapShapeToRoads fallback', () {
    // 600 points: past buildNavShape's 500 cap, so the request hint is a
    // strictly lossy decimation of this list.
    final fullResolution = [
      for (var i = 0; i < 600; i++)
        {'lat': 47.0 + i * 0.0001, 'lon': 9.0 + i * 0.0001},
    ];
    final fullResolutionPoints = [
      for (final p in fullResolution)
        ml.Geographic(lat: p['lat']!, lon: p['lon']!),
    ];

    testWidgets('a failed request returns the caller\'s full-resolution '
        'fallbackShape, not the downsampled request hint', (tester) async {
      final api = _FakeSnapApi(snapShape: const [], shouldFailAll: true);
      final ref = await _pumpSnapRef(tester, api);

      final result = await tester.runAsync(
        () => snapShapeToRoadsResult(
          ref,
          buildNavShape(fullResolutionPoints),
          'pedestrian',
          fallbackShape: fullResolution,
        ),
      );

      expect(result!.snapped, isFalse);
      expect(result.shape, hasLength(600));
      expect(result.shape, same(fullResolution));
    });

    testWidgets('a snap rejected by snapResultAcceptable also returns the '
        'full-resolution fallbackShape', (tester) async {
      // A 2-point degenerate reply: bbox diagonal far below 0.6x the
      // original's, so snapResultAcceptable rejects it as a truncation.
      final api = _FakeSnapApi(
        snapShape: [
          {'lat': 47.0, 'lon': 9.0},
          {'lat': 47.0001, 'lon': 9.0001},
        ],
      );
      final ref = await _pumpSnapRef(tester, api);

      final result = await tester.runAsync(
        () => snapShapeToRoadsResult(
          ref,
          buildNavShape(fullResolutionPoints),
          'pedestrian',
          fallbackShape: fullResolution,
        ),
      );

      expect(api.traceRouteCalls, 1);
      expect(result!.snapped, isFalse);
      expect(result.shape, hasLength(600));
    });

    testWidgets('an accepted snap reports snapped: true and returns the '
        'matched path', (tester) async {
      final matched = [
        for (var i = 0; i < 40; i++)
          {'lat': 47.0 + i * 0.0015, 'lon': 9.0 + i * 0.0015},
      ];
      final api = _FakeSnapApi(snapShape: matched);
      final ref = await _pumpSnapRef(tester, api);

      final result = await tester.runAsync(
        () => snapShapeToRoadsResult(
          ref,
          buildNavShape(fullResolutionPoints),
          'pedestrian',
          fallbackShape: fullResolution,
        ),
      );

      expect(result!.snapped, isTrue);
      expect(result.shape, hasLength(40));
    });

    testWidgets('without fallbackShape the request hint is still the '
        'fallback (unchanged for callers whose hint IS their geometry)',
        (tester) async {
      final api = _FakeSnapApi(snapShape: const [], shouldFailAll: true);
      final ref = await _pumpSnapRef(tester, api);
      final hint = [
        {'lat': 47.0, 'lon': 9.0},
        {'lat': 47.001, 'lon': 9.001},
      ];

      final result = await tester.runAsync(
        () => snapShapeToRoads(ref, hint, 'pedestrian'),
      );

      expect(result, same(hint));
    });
  });

  group('snapResultAcceptable', () {
    // A 5-point original shape spanning a 0.010° x 0.010° bbox
    // (diagonal ~= 0.014142).
    final original = [
      {'lat': 47.000, 'lon': 9.000},
      {'lat': 47.0025, 'lon': 9.0025},
      {'lat': 47.005, 'lon': 9.005},
      {'lat': 47.0075, 'lon': 9.0075},
      {'lat': 47.010, 'lon': 9.010},
    ];

    List<Map<String, double>> scaledBboxShape(double scale, {int points = 5}) {
      return [
        for (var i = 0; i < points; i++)
          {
            'lat': 47.000 + (0.010 * scale) * (i / (points - 1)),
            'lon': 9.000 + (0.010 * scale) * (i / (points - 1)),
          },
      ];
    }

    test('returns false for an empty snapped shape', () {
      expect(snapResultAcceptable(original, const []), isFalse);
    });

    test(
      'returns true when the snapped bbox diagonal is ~0.9x the original '
      '(comparable, not truncated)',
      () {
        final snapped = scaledBboxShape(0.9);

        expect(snapResultAcceptable(original, snapped), isTrue);
      },
    );

    test(
      'returns false when the snapped bbox diagonal is ~0.3x the original '
      '(partial map-match truncation, valhalla#4802)',
      () {
        final snapped = scaledBboxShape(0.3);

        expect(snapResultAcceptable(original, snapped), isFalse);
      },
    );

    test(
      'returns true for a snapped shape with far fewer points but a '
      'comparable bbox (Valhalla re-vertexes; point count is not the '
      'rejection signal)',
      () {
        final snapped = [
          {'lat': 47.000, 'lon': 9.000},
          {'lat': 47.010, 'lon': 9.010},
        ];

        expect(snapResultAcceptable(original, snapped), isTrue);
      },
    );
  });

  group('fetchHeightsForShape', () {
    List<Map<String, double>> buildShape(int length) => [
      for (var i = 0; i < length; i++) {'lat': 47.0 + i * 0.0001, 'lon': 9.0},
    ];

    testWidgets('returns an empty list for an empty shape with no request', (
      tester,
    ) async {
      var calls = 0;
      final ref = await _pumpHeightRef(tester, (shape) {
        calls++;
        return List<num>.filled(shape.length, 0);
      });

      final result = await tester.runAsync(
        () => fetchHeightsForShape(ref, const []),
      );

      expect(result, isEmpty);
      expect(calls, 0);
    });

    testWidgets(
      'a shape of exactly 500 points is sent as a single chunk',
      (tester) async {
        final calls = <int>[];
        final ref = await _pumpHeightRef(tester, (shape) {
          calls.add(shape.length);
          return List<num>.generate(shape.length, (i) => i.toDouble());
        });
        final shape = buildShape(500);

        final result = await tester.runAsync(
          () => fetchHeightsForShape(ref, shape),
        );

        expect(calls, [500]);
        expect(result, hasLength(500));
      },
    );

    testWidgets(
      'a shape over 500 points is batched into multiple chunks and '
      'concatenated 1:1 — the CR-01 fix (no longer downsampled/truncated)',
      (tester) async {
        final calls = <int>[];
        final ref = await _pumpHeightRef(tester, (shape) {
          calls.add(shape.length);
          return List<num>.generate(shape.length, (i) => calls.length * 1000.0 + i);
        });
        final shape = buildShape(650);

        final result = await tester.runAsync(
          () => fetchHeightsForShape(ref, shape),
        );

        expect(calls, [500, 150]);
        expect(result, hasLength(650));
        // First chunk's heights come first, second chunk's follow — no
        // reordering/dropping across the batch boundary.
        expect(result![0], 1000.0);
        expect(result[499], 1499.0);
        expect(result[500], 2000.0);
        expect(result[649], 2149.0);
      },
    );

    testWidgets(
      'falls back to an empty list when any chunk request fails '
      '(silent fallback, no partially-heighted track)',
      (tester) async {
        final ref = await _pumpHeightRef(tester, (shape) {
          throw StateError('simulated network failure');
        });
        final shape = buildShape(10);

        final result = await tester.runAsync(
          () => fetchHeightsForShape(ref, shape),
        );

        expect(result, isEmpty);
      },
    );

    testWidgets(
      'falls back to an empty list when a chunk response length does not '
      'match the chunk it answered (malformed upstream body)',
      (tester) async {
        final ref = await _pumpHeightRef(tester, (shape) {
          return List<num>.filled(shape.length - 1, 0);
        });
        final shape = buildShape(10);

        final result = await tester.runAsync(
          () => fetchHeightsForShape(ref, shape),
        );

        expect(result, isEmpty);
      },
    );
  });
}
