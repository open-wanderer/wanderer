import 'package:flutter_test/flutter_test.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/util/route_planner_handoff_util.dart';

// ---------------------------------------------------------------------------
// Tests for the pure handoff helpers (no network/navigation): buildDraftTrail
// and mergeHeightsIntoGpx. finishPlanning's orchestration (network fetch +
// pendingImportedTrail + navigation) is intentionally not unit-tested here —
// it has no pure/synchronous seam without a WidgetRef/BuildContext harness.
// ---------------------------------------------------------------------------

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

    test('sets a non-empty expand.gpxData containing "<gpx" (Pitfall 1)', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.expand?.gpxData, isNotNull);
      expect(result.expand!.gpxData, isNotEmpty);
      expect(result.expand!.gpxData, contains('<gpx'));
    });

    test('leaves expand.waypointsViaTrail empty (D-07)', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.expand?.waypointsViaTrail, isEmpty);
    });

    test('keeps expand.gpx identical to the finalGpx passed in', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(identical(result.expand!.gpx, gpx), isTrue);
    });

    test('passes a supplied category id through', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx, category: 'bike-id');

      expect(result.category, 'bike-id');
    });

    test('leaves category null when none is supplied', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.category, isNull);
    });

    test('sets lat/lon/bounds from the track so the map centers on the '
        'route instead of null island', () {
      final gpx = buildSampleGpx();
      final result = buildDraftTrail(gpx);

      expect(result.maxLat, 47.001);
      expect(result.minLat, 47.000);
      expect(result.maxLon, 9.001);
      expect(result.minLon, 9.000);
      expect(result.lat, closeTo(47.0005, 1e-9));
      expect(result.lon, closeTo(9.0005, 1e-9));
    });

    test('leaves lat/lon null and bounds zeroed for an empty track', () {
      final result = buildDraftTrail(Gpx());

      expect(result.lat, isNull);
      expect(result.lon, isNull);
      expect(result.maxLat, 0);
      expect(result.minLat, 0);
      expect(result.maxLon, 0);
      expect(result.minLon, 0);
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
}
