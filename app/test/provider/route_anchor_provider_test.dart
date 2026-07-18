import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/polyline_util.dart';

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

const _anchorA = Geographic(lat: 47.000, lon: 9.000);
const _anchorB = Geographic(lat: 47.001, lon: 9.000);
const _anchorC = Geographic(lat: 47.002, lon: 9.000);

const _profile = 'pedestrian';

/// Waits long enough for a fire-and-forget resolve dispatched by a `void`
/// mutation method (e.g. `appendAnchor`) to fully settle.
///
/// Dio's own interceptor-wrapping pipeline (`dio_mixin.dart`'s
/// `requestInterceptorWrapper`) schedules each step via the `Future(...)`
/// constructor, which Dart implements with `Timer.run` — a macrotask, not a
/// microtask — so a pure microtask flush (`await Future.value()` in a loop)
/// is NOT sufficient here; a real, if brief, elapsed-time wait is required.
Future<void> _flushAsyncWork() async {
  await Future<void>.delayed(const Duration(milliseconds: 50));
}

Map<String, dynamic> _tripData(String shape) {
  return {
    'trip': {
      'legs': [
        {'shape': shape},
      ],
      'summary': {'time': 90},
    },
  };
}

// ---------------------------------------------------------------------------
// Fake API harness
//
// Empirically verified against dio 5.9.2 (see 19-02-SUMMARY.md "Deviations"
// for the full trace): every interceptor step Dio's `fetch()` builds is
// wrapped via `listenCancelForAsyncTask`, which races the step against
// `cancelToken.whenCancel` using `Future.any`. Because our fake resolves
// entirely inside that same interceptor step (no real socket), a request
// that gets cancelled *before* its own response has already been handed to
// `handler.resolve()`/`reject()` ALWAYS loses that race and settles with
// `DioException(type: cancel)` — never with a late, "successful" response.
// `_resolveSegment` always cancels the previous in-flight token for a given
// segment key before dispatching a new one, so for two `retrySegment` calls
// issued back-to-back for the same key, the FIRST one is reliably caught by
// this cancellation path (the `if (e.type == DioExceptionType.cancel)
// return;` branch), not by the generation-counter check in the success
// path — that check remains valuable defense-in-depth for a narrower race
// (an already-resolved-but-not-yet-applied response racing a fresh
// dispatch) that a real network layer can produce but this fast, in-process
// fake cannot reliably force. The test below verifies the OBSERVABLE
// contract instead: a stale, superseded dispatch never corrupts state and
// never throws uncaught, regardless of which of the two guards catches it.
// ---------------------------------------------------------------------------

class _CannedResponse {
  const _CannedResponse.success(this.data) : isSuccess = true;
  const _CannedResponse.failure() : isSuccess = false, data = null;

  final bool isSuccess;
  final Map<String, dynamic>? data;
}

typedef _Responder =
    FutureOr<_CannedResponse> Function(RequestOptions options, int callIndex);

class _FakeApi extends Api {
  _FakeApi(this.responder);

  final _Responder responder;

  @override
  Dio build() {
    var callIndex = 0;
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) async {
          // appendAnchor/dragAnchor/insertAnchorOnSegment now also fire a
          // reverse-geocode search (route_anchor_provider.dart's
          // _resolveAnchorLocation, quick-260717-t7q follow-up) through the
          // same apiProvider. Intercept it here with a fixed "no address
          // found" response so it never consumes a Valhalla-call-index slot
          // or corrupts the call-count assertions the tests below make
          // against `/valhalla/route` specifically — the location search is
          // a separate concern from every test in this file.
          if (options.path.contains('/geocoding/reverse')) {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {'features': <dynamic>[]},
              ),
            );
            return;
          }

          // Every segment creation/update now also fires a fire-and-forget
          // `/valhalla/height` fetch (route_anchor_provider.dart's
          // `_resolveElevation`) through this same apiProvider. Carved out
          // exactly like `/geocoding/reverse` above so it never consumes a
          // `/valhalla/route`-call-index slot or corrupts the call-count
          // assertions the tests below make against that endpoint
          // specifically — elevation is a separate concern from every test
          // in this file.
          if (options.path.contains('/valhalla/height')) {
            final shape = options.data is Map
                ? (options.data as Map)['shape'] as List?
                : null;
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: {'height': List.filled(shape?.length ?? 0, 0)},
              ),
            );
            return;
          }

          final index = callIndex++;
          final canned = await responder(options, index);
          if (canned.isSuccess) {
            handler.resolve(
              Response(
                requestOptions: options,
                statusCode: 200,
                data: canned.data,
              ),
            );
          } else {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.badResponse,
              ),
            );
          }
        },
      ),
    );
    return dio;
  }
}

ProviderContainer _buildContainer(_Responder responder) {
  final container = ProviderContainer(
    overrides: [apiProvider.overrideWith(() => _FakeApi(responder))],
  );
  addTearDown(container.dispose);
  // routeAnchorsProvider is autoDispose; keep it alive for the test's
  // duration via a persistent listener (mirrors how a real screen would
  // `ref.watch` it continuously), so in-flight fire-and-forget resolution
  // work isn't torn down across an `await` gap.
  container.listen(routeAnchorsProvider, (_, _) {});
  return container;
}

/// A `RouteAnchors` subclass whose `build()` returns a pre-populated state
/// instead of the empty default — lets the segment-resolution engine's own
/// tests (`_resolveSegment`/`retrySegment`/`toggleAutoRouting`) seed a
/// 2-anchor/1-segment fixture directly, without depending on the anchor
/// mutation methods (`appendAnchor` et al.), which are a separate concern.
class _SeededRouteAnchors extends RouteAnchors {
  _SeededRouteAnchors(
    this._anchors,
    this._segments,
    this._autoRoutingEnabled, {
    String travelProfile = _profile,
    Map<String, dynamic>? costingOptions,
  }) : _travelProfile = travelProfile,
       _costingOptions = costingOptions;

  final List<RouteAnchor> _anchors;
  final List<RouteSegment> _segments;
  final bool _autoRoutingEnabled;
  final String _travelProfile;
  final Map<String, dynamic>? _costingOptions;

  @override
  RouteAnchorsState build() {
    return RouteAnchorsState(
      anchors: _anchors,
      segments: _segments,
      autoRoutingEnabled: _autoRoutingEnabled,
      travelProfile: _travelProfile,
      costingOptions: _costingOptions,
      undoStack: const [],
      redoStack: const [],
    );
  }
}

/// Builds a container seeded with a single straight segment between two
/// fixed-id anchors ([_anchorIdA] before, [_anchorIdB] after).
ProviderContainer _buildSeededContainer(
  _Responder responder, {
  bool autoRoutingEnabled = true,
}) {
  final anchors = [
    RouteAnchor(id: _anchorIdA, lat: _anchorA.lat, lon: _anchorA.lon),
    RouteAnchor(id: _anchorIdB, lat: _anchorB.lat, lon: _anchorB.lon),
  ];
  final segments = [
    const RouteSegment(
      beforeAnchorId: _anchorIdA,
      afterAnchorId: _anchorIdB,
      polyline: [_anchorA, _anchorB],
      state: SegmentState.straight,
    ),
  ];
  final container = ProviderContainer(
    overrides: [
      apiProvider.overrideWith(() => _FakeApi(responder)),
      routeAnchorsProvider.overrideWith(
        () => _SeededRouteAnchors(anchors, segments, autoRoutingEnabled),
      ),
    ],
  );
  addTearDown(container.dispose);
  container.listen(routeAnchorsProvider, (_, _) {});
  return container;
}

const _anchorIdA = 'anchor-a';
const _anchorIdB = 'anchor-b';
const _anchorIdC = 'anchor-c';

/// Builds a container seeded with 3 fixed-id anchors (A, B, C) where the
/// A-B segment is straight and the B-C segment is already
/// [SegmentState.routed] with a distinctive multi-point polyline —
/// simulating a previously-resolved Valhalla segment for the reorder
/// reuse-unchanged-segment tests.
ProviderContainer _buildThreeAnchorSeededContainer(
  _Responder responder, {
  bool autoRoutingEnabled = true,
}) {
  final anchors = [
    RouteAnchor(id: _anchorIdA, lat: _anchorA.lat, lon: _anchorA.lon),
    RouteAnchor(id: _anchorIdB, lat: _anchorB.lat, lon: _anchorB.lon),
    RouteAnchor(id: _anchorIdC, lat: _anchorC.lat, lon: _anchorC.lon),
  ];
  final segments = [
    const RouteSegment(
      beforeAnchorId: _anchorIdA,
      afterAnchorId: _anchorIdB,
      polyline: [_anchorA, _anchorB],
      state: SegmentState.straight,
    ),
    const RouteSegment(
      beforeAnchorId: _anchorIdB,
      afterAnchorId: _anchorIdC,
      polyline: [_anchorB, _anchorC, _anchorC],
      state: SegmentState.routed,
    ),
  ];
  final container = ProviderContainer(
    overrides: [
      apiProvider.overrideWith(() => _FakeApi(responder)),
      routeAnchorsProvider.overrideWith(
        () => _SeededRouteAnchors(anchors, segments, autoRoutingEnabled),
      ),
    ],
  );
  addTearDown(container.dispose);
  container.listen(routeAnchorsProvider, (_, _) {});
  return container;
}

void main() {
  group('RouteAnchors - segment resolution engine', () {
    test(
      '_resolveSegment success path decodes the shape at precision 6 and '
      'marks the segment routed',
      () async {
        final shape = PolylineUtil.encode([_anchorA, _anchorB], precision: 6);
        final container = _buildSeededContainer(
          (options, index) async => _CannedResponse.success(_tripData(shape)),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);

        await notifier.retrySegment(_anchorIdA, _anchorIdB);

        final segment = container.read(routeAnchorsProvider).segments.single;
        expect(segment.state, SegmentState.routed);
        expect(segment.polyline, PolylineUtil.decode(shape, precision: 6));
      },
    );

    test(
      '_resolveSegment failure path marks the segment blocked and leaves '
      'its prior polyline unchanged (ROUTE-05)',
      () async {
        final container = _buildSeededContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        final priorPolyline = container
            .read(routeAnchorsProvider)
            .segments
            .single
            .polyline;

        await notifier.retrySegment(_anchorIdA, _anchorIdB);

        final segment = container.read(routeAnchorsProvider).segments.single;
        expect(segment.state, SegmentState.blocked);
        expect(segment.polyline, priorPolyline);
      },
    );

    test(
      '_resolveSegment includes costing_options in the POST body keyed by '
      'travelProfile when state.costingOptions is non-null',
      () async {
        Map<String, dynamic>? capturedBody;
        final shape = PolylineUtil.encode([_anchorA, _anchorB], precision: 6);
        final container = ProviderContainer(
          overrides: [
            apiProvider.overrideWith(
              () => _FakeApi((options, index) async {
                capturedBody = options.data as Map<String, dynamic>;
                return _CannedResponse.success(_tripData(shape));
              }),
            ),
            routeAnchorsProvider.overrideWith(
              () => _SeededRouteAnchors(
                [
                  RouteAnchor(id: _anchorIdA, lat: _anchorA.lat, lon: _anchorA.lon),
                  RouteAnchor(id: _anchorIdB, lat: _anchorB.lat, lon: _anchorB.lon),
                ],
                [
                  const RouteSegment(
                    beforeAnchorId: _anchorIdA,
                    afterAnchorId: _anchorIdB,
                    polyline: [_anchorA, _anchorB],
                    state: SegmentState.straight,
                  ),
                ],
                true,
                travelProfile: 'bicycle',
                costingOptions: const {'bicycle_type': 'Road'},
              ),
            ),
          ],
        );
        addTearDown(container.dispose);
        container.listen(routeAnchorsProvider, (_, _) {});
        final notifier = container.read(routeAnchorsProvider.notifier);

        await notifier.retrySegment(_anchorIdA, _anchorIdB);

        expect(capturedBody?['costing_options'], {
          'bicycle': {'bicycle_type': 'Road'},
        });
      },
    );

    test(
      '_resolveSegment omits costing_options from the POST body when '
      'state.costingOptions is null (back-compat)',
      () async {
        Map<String, dynamic>? capturedBody;
        final shape = PolylineUtil.encode([_anchorA, _anchorB], precision: 6);
        final container = _buildSeededContainer((options, index) async {
          capturedBody = options.data as Map<String, dynamic>;
          return _CannedResponse.success(_tripData(shape));
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        await notifier.retrySegment(_anchorIdA, _anchorIdB);

        expect(capturedBody?.containsKey('costing_options'), isFalse);
      },
    );

    test(
      'a stale, superseded dispatch for the same segment never corrupts '
      'state and never throws uncaught (CancelToken + generation-counter '
      'guard combo)',
      () async {
        final shape = PolylineUtil.encode([_anchorA, _anchorC], precision: 6);
        final container = _buildSeededContainer(
          (options, index) async => _CannedResponse.success(_tripData(shape)),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);

        // Two dispatches in quick succession for the SAME segment key.
        // `_resolveSegment` cancels the first's in-flight `CancelToken`
        // before starting the second — the first must settle cleanly
        // (never throw uncaught, never apply its result over the second's)
        // regardless of whether it's caught by the cancellation branch or
        // the generation-counter check.
        final f1 = notifier.retrySegment(_anchorIdA, _anchorIdB);
        final f2 = notifier.retrySegment(_anchorIdA, _anchorIdB);

        await expectLater(f1, completes);
        await expectLater(f2, completes);

        final segment = container.read(routeAnchorsProvider).segments.single;
        expect(segment.state, SegmentState.routed);
        expect(segment.polyline, PolylineUtil.decode(shape, precision: 6));
      },
    );

    test(
      'retrySegment on a blocked segment transitions it to routed given a '
      'subsequent successful response (D-09)',
      () async {
        var shouldFail = true;
        final shape = PolylineUtil.encode([_anchorA, _anchorB], precision: 6);
        final container = _buildSeededContainer((options, index) async {
          if (shouldFail) return const _CannedResponse.failure();
          return _CannedResponse.success(_tripData(shape));
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        await notifier.retrySegment(_anchorIdA, _anchorIdB);
        expect(
          container.read(routeAnchorsProvider).segments.single.state,
          SegmentState.blocked,
        );

        shouldFail = false;
        await notifier.retrySegment(_anchorIdA, _anchorIdB);

        final segment = container.read(routeAnchorsProvider).segments.single;
        expect(segment.state, SegmentState.routed);
        expect(segment.polyline, PolylineUtil.decode(shape, precision: 6));
      },
    );

    test(
      'toggleAutoRouting ON to OFF leaves the existing segment untouched '
      'and issues zero Dio calls (corrected must_haves truth — existing '
      'segments are never rewritten to straight lines on toggle-off)',
      () async {
        var callCount = 0;
        final container = _buildSeededContainer((options, index) async {
          callCount++;
          return const _CannedResponse.failure();
        });
        final notifier = container.read(routeAnchorsProvider.notifier);
        final before = container.read(routeAnchorsProvider).segments.single;

        await notifier.toggleAutoRouting(); // ON -> OFF

        final result = container.read(routeAnchorsProvider);
        expect(result.autoRoutingEnabled, isFalse);
        final segment = result.segments.single;
        expect(segment.state, before.state);
        expect(segment.polyline, before.polyline);
        expect(callCount, 0);
      },
    );

    test(
      'toggleAutoRouting OFF to ON leaves the existing segment untouched '
      'and issues zero Dio calls (symmetric with the ON-to-OFF direction — '
      'toggling never retroactively rewrites existing segments; only new '
      'mutations made while the flag is on trigger a Valhalla resolve)',
      () async {
        var callCount = 0;
        final container = _buildSeededContainer((options, index) async {
          callCount++;
          return const _CannedResponse.failure();
        });
        final notifier = container.read(routeAnchorsProvider.notifier);
        final before = container.read(routeAnchorsProvider).segments.single;

        await notifier.toggleAutoRouting(); // ON -> OFF (default true)
        await notifier.toggleAutoRouting(); // OFF -> ON

        final result = container.read(routeAnchorsProvider);
        expect(result.autoRoutingEnabled, isTrue);
        final segment = result.segments.single;
        expect(segment.state, before.state);
        expect(segment.polyline, before.polyline);
        expect(callCount, 0);
      },
    );
  });

  group('RouteAnchors - Rec B: switchProfile / resolveAllSegments / '
      'resetForSession', () {
    test(
      'switchProfile with auto-routing off: sets travelProfile + '
      'costingOptions, leaves every segment straight, and issues zero Dio '
      'calls',
      () async {
        var callCount = 0;
        final container = _buildThreeAnchorSeededContainer((
          options,
          index,
        ) async {
          callCount++;
          return const _CannedResponse.failure();
        }, autoRoutingEnabled: false);
        final notifier = container.read(routeAnchorsProvider.notifier);
        const roadOpts = {'bicycle_type': 'Road', 'cycling_speed': 25};

        notifier.switchProfile('bicycle', roadOpts);
        await _flushAsyncWork();

        final result = container.read(routeAnchorsProvider);
        expect(result.travelProfile, 'bicycle');
        expect(result.costingOptions, roadOpts);
        expect(
          result.segments.every((s) => s.state == SegmentState.straight),
          isTrue,
        );
        expect(callCount, 0);
      },
    );

    test(
      'switchProfile clears both undoStack and redoStack (fresh baseline) '
      'and does not push a new undo snapshot',
      () async {
        final container = _buildThreeAnchorSeededContainer(
          (options, index) async => const _CannedResponse.failure(),
          autoRoutingEnabled: false,
        );
        final notifier = container.read(routeAnchorsProvider.notifier);

        // Seed a non-empty undo/redo stack via a real mutation first.
        notifier.dragAnchor(_anchorIdA, _anchorB);
        expect(
          container.read(routeAnchorsProvider).undoStack,
          isNotEmpty,
        );

        notifier.switchProfile('bicycle', const {'bicycle_type': 'Road'});
        await _flushAsyncWork();

        final result = container.read(routeAnchorsProvider);
        expect(result.undoStack, isEmpty);
        expect(result.redoStack, isEmpty);
      },
    );

    test(
      'switchProfile with auto-routing on re-dispatches a Valhalla resolve '
      'for every consecutive-anchor segment',
      () async {
        var callCount = 0;
        final shape = PolylineUtil.encode([_anchorA, _anchorB], precision: 6);
        final container = _buildThreeAnchorSeededContainer((
          options,
          index,
        ) async {
          callCount++;
          return _CannedResponse.success(_tripData(shape));
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.switchProfile('bicycle', const {'bicycle_type': 'Road'});
        await _flushAsyncWork();

        // 2 segments in the 3-anchor fixture (A-B, B-C).
        expect(callCount, 2);
        final result = container.read(routeAnchorsProvider);
        expect(
          result.segments.every((s) => s.state == SegmentState.routed),
          isTrue,
        );
      },
    );

    test(
      'resetForSession empties anchors, segments, undo/redo and sets the '
      'given travelProfile + costingOptions',
      () async {
        final container = _buildThreeAnchorSeededContainer(
          (options, index) async => const _CannedResponse.failure(),
          autoRoutingEnabled: false,
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        notifier.dragAnchor(_anchorIdA, _anchorB); // seed undo stack

        notifier.resetForSession('pedestrian', null);

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, isEmpty);
        expect(result.segments, isEmpty);
        expect(result.undoStack, isEmpty);
        expect(result.redoStack, isEmpty);
        expect(result.travelProfile, 'pedestrian');
        expect(result.costingOptions, isNull);
      },
    );
  });

  group('RouteAnchors - seedFromTrack (quick-260718-e9j)', () {
    test(
      'seeds one anchor per point, one straight-state segment per '
      'consecutive pair (straight-line polyline, no segmentPolylines '
      'given), and an empty undo/redo stack — and never dispatches a '
      'Valhalla resolve on seed (destroys off-road recordings otherwise)',
      () async {
        var callCount = 0;
        final container = _buildContainer((options, index) async {
          callCount++;
          return const _CannedResponse.failure();
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.seedFromTrack([
          _anchorA,
          _anchorB,
          _anchorC,
        ], 'bicycle', const {'bicycle_type': 'Road'});

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, hasLength(3));
        expect(result.segments, hasLength(2));
        for (final segment in result.segments) {
          expect(segment.state, SegmentState.straight);
        }
        expect(result.segments[0].polyline, [_anchorA, _anchorB]);
        expect(result.segments[1].polyline, [_anchorB, _anchorC]);
        expect(result.undoStack, isEmpty);
        expect(result.redoStack, isEmpty);
        expect(result.travelProfile, 'bicycle');
        expect(result.costingOptions, {'bicycle_type': 'Road'});

        // seedFromTrack must NOT auto-resolve segments on open — a segment
        // is only ever Valhalla-routed once the user actually edits it.
        await _flushAsyncWork();
        expect(callCount, 0);
      },
    );

    test(
      'given segmentPolylines, seeds each segment with its full original '
      'polyline (preserving off-road/recorded shape) rather than a '
      'straight line — falling back to straight for any missing entry',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        final recorded = [
          _anchorA,
          const Geographic(lat: 47.0005, lon: 9.0005),
          _anchorB,
        ];

        notifier.seedFromTrack(
          [_anchorA, _anchorB, _anchorC],
          'pedestrian',
          null,
          segmentPolylines: [recorded],
        );

        final result = container.read(routeAnchorsProvider);
        expect(result.segments, hasLength(2));
        expect(result.segments[0].polyline, recorded);
        expect(result.segments[0].state, SegmentState.straight);
        // No entry supplied for the second segment — falls back to a
        // straight line between its bounding anchors.
        expect(result.segments[1].polyline, [_anchorB, _anchorC]);
      },
    );

    test('a single point yields 1 anchor and 0 segments', () async {
      final container = _buildContainer(
        (options, index) async => const _CannedResponse.failure(),
      );
      final notifier = container.read(routeAnchorsProvider.notifier);

      notifier.seedFromTrack([_anchorA], 'pedestrian', null);

      final result = container.read(routeAnchorsProvider);
      expect(result.anchors, hasLength(1));
      expect(result.segments, isEmpty);
    });

    test('an empty point list yields an empty session', () async {
      final container = _buildContainer(
        (options, index) async => const _CannedResponse.failure(),
      );
      final notifier = container.read(routeAnchorsProvider.notifier);

      notifier.seedFromTrack(const [], 'pedestrian', null);

      final result = container.read(routeAnchorsProvider);
      expect(result.anchors, isEmpty);
      expect(result.segments, isEmpty);
    });

    test(
      'clears any prior in-flight/undo/redo bookkeeping from an earlier '
      'session (mirrors resetForSession)',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB);
        expect(
          container.read(routeAnchorsProvider).undoStack,
          isNotEmpty,
        );

        notifier.seedFromTrack([_anchorA, _anchorC], 'pedestrian', null);

        final result = container.read(routeAnchorsProvider);
        expect(result.undoStack, isEmpty);
        expect(result.redoStack, isEmpty);
        expect(result.anchors, hasLength(2));
      },
    );
  });

  group('RouteAnchors - anchor mutations, geometric split, undo/redo', () {
    test(
      'appendAnchor on an empty route creates 1 anchor and 0 segments; a '
      'second call creates a 2nd anchor and exactly 1 new segment',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        var state = container.read(routeAnchorsProvider);
        expect(state.anchors, hasLength(1));
        expect(state.segments, isEmpty);

        notifier.appendAnchor(_anchorB);
        state = container.read(routeAnchorsProvider);
        expect(state.anchors, hasLength(2));
        expect(state.segments, hasLength(1));
        expect(state.segments.single.beforeAnchorId, state.anchors[0].id);
        expect(state.segments.single.afterAnchorId, state.anchors[1].id);
      },
    );

    test(
      'appendAnchor with autoRoutingEnabled true triggers a Valhalla '
      'resolve for the new segment',
      () async {
        var callCount = 0;
        final shape = PolylineUtil.encode([_anchorA, _anchorB], precision: 6);
        final container = _buildContainer((options, index) async {
          callCount++;
          return _CannedResponse.success(_tripData(shape));
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB); // autoRoutingEnabled true by default
        await _flushAsyncWork();

        expect(callCount, 1);
        expect(
          container.read(routeAnchorsProvider).segments.single.state,
          SegmentState.routed,
        );
      },
    );

    test(
      'appendAnchor with autoRoutingEnabled false creates a straight '
      'segment and issues zero Dio calls',
      () async {
        var callCount = 0;
        final container = _buildContainer((options, index) async {
          callCount++;
          return const _CannedResponse.failure();
        });
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB);
        await _flushAsyncWork();

        final state = container.read(routeAnchorsProvider);
        expect(state.segments.single.state, SegmentState.straight);
        expect(state.segments.single.polyline, [_anchorA, _anchorB]);
        expect(callCount, 0);
      },
    );

    test(
      'dragAnchor moves the target anchor and only its adjacent segments '
      'change; a distant, unrelated segment is untouched',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB);
        notifier.appendAnchor(_anchorC);

        final seed = container.read(routeAnchorsProvider);
        expect(seed.segments, hasLength(2));
        final untouchedSegment = seed.segments[1]; // B-C, not adjacent to A

        const moved = Geographic(lat: 47.010, lon: 9.010);
        notifier.dragAnchor(seed.anchors[0].id, moved);

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors[0].lat, moved.lat);
        expect(result.anchors[0].lon, moved.lon);
        expect(result.segments[0].polyline, [moved, result.anchors[1].point]);
        expect(result.segments[1], untouchedSegment);
      },
    );

    test(
      'insertAnchorOnSegment splits the segment geometrically, inserts the '
      'new anchor at the correct list index, and issues zero Dio calls '
      '(WAYP-03)',
      () async {
        var callCount = 0;
        final container = _buildContainer((options, index) async {
          callCount++;
          return const _CannedResponse.failure();
        });
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorC);

        final seed = container.read(routeAnchorsProvider);
        final segment = seed.segments.single;
        const tapPoint = Geographic(lat: 47.001, lon: 9.000); // A-C midpoint

        notifier.insertAnchorOnSegment(
          segment.beforeAnchorId,
          segment.afterAnchorId,
          tapPoint,
        );

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, hasLength(3));
        expect(result.anchors[0].id, seed.anchors[0].id);
        expect(result.anchors[2].id, seed.anchors[1].id);
        final newAnchor = result.anchors[1];

        expect(result.segments, hasLength(2));
        expect(result.segments[0].afterAnchorId, newAnchor.id);
        expect(result.segments[1].beforeAnchorId, newAnchor.id);
        expect(result.segments[0].polyline.first, seed.anchors[0].point);
        expect(
          result.segments[0].polyline.last,
          result.segments[1].polyline.first,
        );
        expect(result.segments[1].polyline.last, seed.anchors[1].point);
        expect(callCount, 0);
      },
    );

    test(
      'undo restores the prior snapshot, redo re-applies it, and a fresh '
      'mutation clears the redo stack (D-11)',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        expect(
          container.read(routeAnchorsProvider).anchors,
          hasLength(1),
        );

        notifier.undo();
        expect(container.read(routeAnchorsProvider).anchors, isEmpty);

        notifier.redo();
        expect(
          container.read(routeAnchorsProvider).anchors,
          hasLength(1),
        );

        notifier.appendAnchor(_anchorB); // new mutation clears redo
        expect(
          container.read(routeAnchorsProvider).redoStack,
          isEmpty,
        );

        notifier.undo();
        notifier.undo();
        expect(container.read(routeAnchorsProvider).anchors, isEmpty);

        // undo() on an empty stack is a defensive no-op.
        expect(() => notifier.undo(), returnsNormally);
        expect(container.read(routeAnchorsProvider).anchors, isEmpty);
      },
    );

    test(
      'deleteAnchor(first) removes anchor 0 and its single leaving segment, '
      'creates no new segment, and pushes an undo snapshot',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB);
        notifier.appendAnchor(_anchorC);

        final seed = container.read(routeAnchorsProvider);
        final undoStackBefore = seed.undoStack.length;
        final firstId = seed.anchors[0].id;

        notifier.deleteAnchor(firstId);

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, hasLength(2));
        expect(result.anchors[0].id, seed.anchors[1].id);
        expect(result.anchors[1].id, seed.anchors[2].id);
        expect(result.segments, hasLength(1));
        expect(result.segments.single, seed.segments[1]); // B-C untouched
        expect(result.undoStack.length, undoStackBefore + 1);

        notifier.undo();
        final undone = container.read(routeAnchorsProvider);
        expect(undone.anchors, hasLength(3));
        expect(undone.anchors[0].id, firstId);
      },
    );

    test(
      'deleteAnchor(middle, auto-routing off) collapses its two touching '
      'segments into one new straight segment; unrelated distant segments '
      'are untouched',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting(); // OFF

        const anchorD = Geographic(lat: 47.003, lon: 9.000);
        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB);
        notifier.appendAnchor(_anchorC);
        notifier.appendAnchor(anchorD);

        final seed = container.read(routeAnchorsProvider);
        expect(seed.anchors, hasLength(4));
        expect(seed.segments, hasLength(3));
        final middleId = seed.anchors[1].id; // B
        final untouchedSegment = seed.segments[2]; // C-D, distant

        notifier.deleteAnchor(middleId);

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, hasLength(3));
        expect(result.anchors.map((a) => a.id), isNot(contains(middleId)));
        expect(result.segments, hasLength(2));

        final newSegment = result.segments.firstWhere(
          (s) => s.beforeAnchorId == seed.anchors[0].id,
        );
        expect(newSegment.afterAnchorId, seed.anchors[2].id);
        expect(newSegment.state, SegmentState.straight);
        expect(newSegment.polyline, [
          seed.anchors[0].point,
          seed.anchors[2].point,
        ]);
        expect(result.segments, contains(untouchedSegment));
      },
    );

    test(
      'deleteAnchor(last) removes the anchor and its single entering '
      'segment; no new segment is created',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB);
        notifier.appendAnchor(_anchorC);

        final seed = container.read(routeAnchorsProvider);
        final lastId = seed.anchors[2].id;

        notifier.deleteAnchor(lastId);

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, hasLength(2));
        expect(result.segments, hasLength(1));
        expect(result.segments.single, seed.segments[0]); // A-B untouched
      },
    );

    test(
      'deleteAnchor(only anchor) leaves anchors and segments empty',
      () async {
        final container = _buildContainer(
          (options, index) async => const _CannedResponse.failure(),
        );
        final notifier = container.read(routeAnchorsProvider.notifier);
        await notifier.toggleAutoRouting();

        notifier.appendAnchor(_anchorA);
        final onlyId = container
            .read(routeAnchorsProvider)
            .anchors
            .single
            .id;

        notifier.deleteAnchor(onlyId);

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, isEmpty);
        expect(result.segments, isEmpty);
      },
    );

    test(
      'deleteAnchor(middle, auto-routing on) dispatches a Valhalla resolve '
      'for the newly-collapsed segment',
      () async {
        var callCount = 0;
        final shape = PolylineUtil.encode([_anchorA, _anchorC], precision: 6);
        final container = _buildContainer((options, index) async {
          callCount++;
          return _CannedResponse.success(_tripData(shape));
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.appendAnchor(_anchorA);
        notifier.appendAnchor(_anchorB);
        notifier.appendAnchor(_anchorC);
        await _flushAsyncWork();
        callCount = 0; // reset — only interested in calls from deleteAnchor

        final seed = container.read(routeAnchorsProvider);
        notifier.deleteAnchor(seed.anchors[1].id);
        await _flushAsyncWork();

        expect(callCount, 1);
        final segment = container.read(routeAnchorsProvider).segments.single;
        expect(segment.state, SegmentState.routed);
      },
    );

    test(
      'reorderAnchors reuses the existing segment (with its routed '
      'polyline) for a pair that remains adjacent, issuing zero Dio calls '
      'for it, and creates a fresh straight segment for the newly-adjacent '
      'pair',
      () async {
        var callCount = 0;
        final container = _buildThreeAnchorSeededContainer(
          (options, index) async {
            callCount++;
            return const _CannedResponse.failure();
          },
          autoRoutingEnabled: false,
        );
        final notifier = container.read(routeAnchorsProvider.notifier);

        final seed = container.read(routeAnchorsProvider);
        final routedSegmentBefore = seed.segments.firstWhere(
          (s) => s.beforeAnchorId == _anchorIdB && s.afterAnchorId == _anchorIdC,
        );

        // New order [B, C, A]: B-C stays adjacent in the same direction
        // (reused verbatim); C-A is newly adjacent (fresh straight segment).
        notifier.reorderAnchors([_anchorIdB, _anchorIdC, _anchorIdA]);

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors.map((a) => a.id), [
          _anchorIdB,
          _anchorIdC,
          _anchorIdA,
        ]);
        expect(result.segments, hasLength(2));

        final reusedSegment = result.segments.firstWhere(
          (s) => s.beforeAnchorId == _anchorIdB && s.afterAnchorId == _anchorIdC,
        );
        expect(reusedSegment, same(routedSegmentBefore));
        expect(reusedSegment.state, SegmentState.routed);
        expect(reusedSegment.polyline, routedSegmentBefore.polyline);

        final newSegment = result.segments.firstWhere(
          (s) => s.beforeAnchorId == _anchorIdC && s.afterAnchorId == _anchorIdA,
        );
        expect(newSegment.state, SegmentState.straight);

        expect(callCount, 0); // auto-routing is off in this fixture
      },
    );

    test(
      'reorderAnchors with auto-routing on dispatches Valhalla only for the '
      'newly-adjacent pair, issuing zero calls for the still-adjacent pair',
      () async {
        var callCount = 0;
        final container = _buildThreeAnchorSeededContainer((
          options,
          index,
        ) async {
          callCount++;
          return const _CannedResponse.failure();
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.reorderAnchors([_anchorIdB, _anchorIdC, _anchorIdA]);
        await _flushAsyncWork();

        // Only the newly-adjacent C-A pair should have triggered a resolve.
        expect(callCount, 1);
      },
    );
  });

  group('RouteAnchors - deleteAllAnchors / reverseRoute', () {
    test(
      'deleteAllAnchors empties anchors and segments and pushes an undo '
      'snapshot',
      () async {
        var callCount = 0;
        final container = _buildThreeAnchorSeededContainer((
          options,
          index,
        ) async {
          callCount++;
          return const _CannedResponse.failure();
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.deleteAllAnchors();

        final result = container.read(routeAnchorsProvider);
        expect(result.anchors, isEmpty);
        expect(result.segments, isEmpty);
        expect(result.undoStack, hasLength(1));
        expect(callCount, 0);
      },
    );

    test('deleteAllAnchors is a no-op on an already-empty route', () async {
      var callCount = 0;
      final container = _buildContainer((options, index) async {
        callCount++;
        return const _CannedResponse.failure();
      });
      final notifier = container.read(routeAnchorsProvider.notifier);

      notifier.deleteAllAnchors();

      final result = container.read(routeAnchorsProvider);
      expect(result.anchors, isEmpty);
      expect(result.undoStack, isEmpty); // no snapshot pushed for a no-op
      expect(callCount, 0);
    });

    test(
      'reverseRoute flips the anchor order (former start becomes the goal) '
      'and rebuilds every segment with before/after swapped to match',
      () async {
        final container = _buildThreeAnchorSeededContainer((
          options,
          index,
        ) async {
          return const _CannedResponse.failure();
        }, autoRoutingEnabled: false);
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.reverseRoute();

        final result = container.read(routeAnchorsProvider);
        expect(
          result.anchors.map((a) => a.id).toList(),
          [_anchorIdC, _anchorIdB, _anchorIdA],
        );
        expect(
          result.segments.map(
            (s) => (s.beforeAnchorId, s.afterAnchorId),
          ),
          [(_anchorIdC, _anchorIdB), (_anchorIdB, _anchorIdA)],
        );
      },
    );

    test(
      'reverseRoute re-resolves every segment via Valhalla when '
      'auto-routing is on, rather than blindly mirroring the old polyline',
      () async {
        var callCount = 0;
        final container = _buildThreeAnchorSeededContainer((
          options,
          index,
        ) async {
          callCount++;
          return _CannedResponse.success(
            _tripData(PolylineUtil.encode([_anchorA, _anchorB], precision: 6)),
          );
        });
        final notifier = container.read(routeAnchorsProvider.notifier);

        notifier.reverseRoute();
        await _flushAsyncWork();

        // 2 segments (A-B, B-C originally) both re-resolved in the new
        // direction — never left as a stale mirror of the old polyline.
        expect(callCount, 2);
        final result = container.read(routeAnchorsProvider);
        expect(
          result.segments.every((s) => s.state == SegmentState.routed),
          isTrue,
        );
      },
    );

    test('reverseRoute is a no-op below 2 anchors', () async {
      var callCount = 0;
      final container = _buildContainer((options, index) async {
        callCount++;
        return const _CannedResponse.failure();
      });
      final notifier = container.read(routeAnchorsProvider.notifier);

      notifier.reverseRoute();

      final result = container.read(routeAnchorsProvider);
      expect(result.anchors, isEmpty);
      expect(result.undoStack, isEmpty);
      expect(callCount, 0);
    });
  });
}
