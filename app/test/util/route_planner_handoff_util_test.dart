import 'package:dio/dio.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/route_planner_handoff_util.dart';

// Tests for the pure handoff helpers (no network/navigation), plus
// buildDraftTrail, which now round-trips through `/trail/convert` (see
// convertGpxToTrail in trail_import_util.dart) and so needs a WidgetRef and a
// faked Api. finishPlanning's own orchestration is still not unit-tested here
// — it has no seam beyond buildDraftTrail worth re-testing.

/// Fakes the `/trail/convert` response so `buildDraftTrail` can be tested
/// without a real server. Mirrors the `_FakeApi` pattern in
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

/// Pumps a bare `Consumer` inside a `ProviderScope` overriding [apiProvider]
/// with [_FakeApi], and hands back the captured [WidgetRef] for the test to
/// call ref-requiring functions with.
Future<WidgetRef> _pumpRef(
  WidgetTester tester, {
  Map<String, dynamic>? response,
  bool shouldFail = false,
}) async {
  late WidgetRef capturedRef;
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        apiProvider.overrideWith(
          () => _FakeApi(response: response, shouldFail: shouldFail),
        ),
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

/// Calls [buildDraftTrail] inside [WidgetTester.runAsync] — `testWidgets`
/// runs under a fake clock that never fires real `Timer`s on its own, and
/// Dio schedules its interceptor pipeline via `Timer.run` (a real timer), so
/// awaiting the call directly hangs forever. `runAsync` steps outside the
/// fake zone into a real one so that timer actually fires.
Future<Trail> _buildDraftTrail(
  WidgetTester tester,
  WidgetRef ref,
  Gpx gpx, {
  String? category,
}) async {
  return (await tester.runAsync(
    () => buildDraftTrail(ref, gpx, category: category),
  ))!;
}

void main() {
  group('buildDraftTrail', () {
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

    /// A minimal, valid `/trail/convert` response — enough for
    /// `Trail.fromJson` (plus `convertGpxToTrail`'s injected id/created/
    /// updated placeholders) to parse successfully.
    Map<String, dynamic> buildServerResponse({
      Map<String, dynamic>? overrides,
    }) {
      return {
        'name': 'Sample Trail',
        'lat': 47.000,
        'lon': 9.000,
        'max_lat': 47.001,
        'min_lat': 47.000,
        'max_lon': 9.001,
        'min_lon': 9.000,
        'expand': {
          'gpx_data': '<gpx><trk><trkseg></trkseg></trk></gpx>',
          'waypoints_via_trail': <dynamic>[],
        },
        ...?overrides,
      };
    }

    testWidgets(
      'sets a non-empty expand.gpxData containing "<gpx" (Pitfall 1)',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester, response: buildServerResponse());

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.expand?.gpxData, isNotNull);
        expect(result.expand!.gpxData, isNotEmpty);
        expect(result.expand!.gpxData, contains('<gpx'));
      },
    );

    testWidgets('leaves expand.waypointsViaTrail empty (D-07)', (
      tester,
    ) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester, response: buildServerResponse());

      final result = await _buildDraftTrail(tester, ref, gpx);

      expect(result.expand?.waypointsViaTrail, isEmpty);
    });

    testWidgets('keeps expand.gpx identical to the finalGpx passed in', (
      tester,
    ) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester, response: buildServerResponse());

      final result = await _buildDraftTrail(tester, ref, gpx);

      expect(identical(result.expand!.gpx, gpx), isTrue);
    });

    testWidgets('passes a supplied category id through', (tester) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester, response: buildServerResponse());

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
      final ref = await _pumpRef(tester, response: buildServerResponse());

      final result = await _buildDraftTrail(tester, ref, gpx);

      expect(result.category, isNull);
    });

    testWidgets(
      'passes through the bounds/lat/lon the server computed from the track',
      (tester) async {
        final gpx = buildSampleGpx();
        final ref = await _pumpRef(tester, response: buildServerResponse());

        final result = await _buildDraftTrail(tester, ref, gpx);

        expect(result.maxLat, 47.001);
        expect(result.minLat, 47.000);
        expect(result.maxLon, 9.001);
        expect(result.minLon, 9.000);
        expect(result.lat, 47.000);
        expect(result.lon, 9.000);
      },
    );

    testWidgets('propagates a /trail/convert failure to the caller', (
      tester,
    ) async {
      final gpx = buildSampleGpx();
      final ref = await _pumpRef(tester, shouldFail: true);

      // runAsync's own returned Future doesn't reliably propagate an error
      // thrown inside the callback back through expectLater/throwsA — catch
      // it inside the real-zone callback instead and assert on that.
      Object? caught;
      await tester.runAsync(() async {
        try {
          await buildDraftTrail(ref, gpx);
        } catch (e) {
          caught = e;
        }
      });

      expect(caught, isA<DioException>());
    });
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
