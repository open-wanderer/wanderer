import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

// ---------------------------------------------------------------------------
// Fixtures — mirrors route_anchor_provider_test.dart's _SeededRouteAnchors
// harness so plannedGpxProvider can be exercised against a known
// anchors/segments shape without depending on the mutator methods.
// ---------------------------------------------------------------------------

const _profile = 'pedestrian';

const _anchorA = Geographic(lat: 47.000, lon: 9.000);
const _anchorB = Geographic(lat: 47.001, lon: 9.000);
const _anchorC = Geographic(lat: 47.002, lon: 9.000);

const _anchorIdA = 'anchor-a';
const _anchorIdB = 'anchor-b';
const _anchorIdC = 'anchor-c';

class _SeededRouteAnchors extends RouteAnchors {
  _SeededRouteAnchors(this._anchors, this._segments);

  final List<RouteAnchor> _anchors;
  final List<RouteSegment> _segments;

  @override
  RouteAnchorsState build(String travelProfile) {
    return RouteAnchorsState(
      anchors: _anchors,
      segments: _segments,
      autoRoutingEnabled: true,
      travelProfile: travelProfile,
      undoStack: const [],
      redoStack: const [],
    );
  }
}

ProviderContainer _buildContainer({
  required List<RouteAnchor> anchors,
  required List<RouteSegment> segments,
}) {
  final container = ProviderContainer(
    overrides: [
      routeAnchorsProvider(
        _profile,
      ).overrideWith(() => _SeededRouteAnchors(anchors, segments)),
    ],
  );
  addTearDown(container.dispose);
  return container;
}

void main() {
  group('plannedGpxProvider', () {
    test('an empty route (0 anchors) yields an empty Gpx', () {
      final container = _buildContainer(anchors: const [], segments: const []);

      final gpx = container.read(plannedGpxProvider(_profile));

      expect(gpx.allPoints, isEmpty);
    });

    test(
      'walks the anchor-id chain (A-B-C) in path order, not array order, '
      'and does not duplicate the shared boundary point between segments',
      () {
        // Middle segment carries a distinctive multi-point polyline; the
        // shared boundary point (_anchorB / _anchorC) must not be duplicated
        // across the A-B and B-C segment appends.
        final anchors = [
          const RouteAnchor(id: _anchorIdA, lat: 47.000, lon: 9.000),
          const RouteAnchor(id: _anchorIdB, lat: 47.001, lon: 9.000),
          const RouteAnchor(id: _anchorIdC, lat: 47.002, lon: 9.000),
        ];
        const midPoint = Geographic(lat: 47.0015, lon: 9.0005);
        // Deliberately reversed array order from traversal order, proving
        // the walk follows the beforeAnchorId->afterAnchorId chain, not
        // state.segments' array index.
        final segments = [
          const RouteSegment(
            beforeAnchorId: _anchorIdB,
            afterAnchorId: _anchorIdC,
            polyline: [_anchorB, midPoint, _anchorC],
            state: SegmentState.routed,
          ),
          const RouteSegment(
            beforeAnchorId: _anchorIdA,
            afterAnchorId: _anchorIdB,
            polyline: [_anchorA, _anchorB],
            state: SegmentState.straight,
          ),
        ];
        final container = _buildContainer(anchors: anchors, segments: segments);

        final gpx = container.read(plannedGpxProvider(_profile));

        expect(gpx.allPoints, [_anchorA, _anchorB, midPoint, _anchorC]);
      },
    );

    test('recomputes when routeAnchorsProvider anchors/segments change', () {
      final container = ProviderContainer();
      addTearDown(container.dispose);

      final notifier = container.read(routeAnchorsProvider(_profile).notifier);
      notifier.appendAnchor(_anchorA);

      final before = container.read(plannedGpxProvider(_profile));
      expect(before.allPoints, [_anchorA]);

      notifier.appendAnchor(_anchorB);

      final after = container.read(plannedGpxProvider(_profile));
      expect(after.allPoints, [_anchorA, _anchorB]);
    });
  });
}
