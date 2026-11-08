import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/map_marker_gestures.dart';
import 'package:wanderer/components/map/route_anchor_layer.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/util/geo/reverse_geocode.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';

import 'map_test_harness.dart';

class _SeededRouteAnchors extends RouteAnchors {
  _SeededRouteAnchors(this.anchors);

  final List<RouteAnchor> anchors;

  /// Anchor ids passed to [dragAnchor]. The real one also reverse-geocodes
  /// and re-routes over the network, which has no place in a widget test.
  final dragged = <String>[];

  /// Emits a new anchor list, as every provider write does.
  void emit(List<RouteAnchor> next) => state = state.copyWith(anchors: next);

  List<RouteAnchor> get current => state.anchors;

  @override
  void dragAnchor(String anchorId, ml.Geographic newPoint) {
    dragged.add(anchorId);
    state = state.copyWith(
      anchors: [
        for (final a in state.anchors)
          if (a.id == anchorId)
            a.copyWith(lat: newPoint.lat, lon: newPoint.lon)
          else
            a,
      ],
    );
  }

  @override
  RouteAnchorsState build() => RouteAnchorsState(
    anchors: anchors,
    segments: const [],
    autoRoutingEnabled: true,
    travelProfile: 'pedestrian',
    costingOptions: null,
    undoStack: const [],
    redoStack: const [],
  );
}

List<RouteAnchor> _anchors(int count) => [
  for (var i = 0; i < count; i++)
    RouteAnchor(
      id: 'a$i',
      lon: -3.5 + (i % 10) * 0.7,
      lat: -2 + (i ~/ 10) * 0.8,
    ),
];

class _Scene {
  _Scene(this.tester, {int anchors = 19, this.onAnchorTap}) {
    this.anchors = _anchors(anchors);
  }

  final WidgetTester tester;
  final void Function(String id)? onAnchorTap;
  late final List<RouteAnchor> anchors;
  final controller = FakeMapController();
  final shift = ValueNotifier<Offset>(Offset.zero);
  late final seeded = _SeededRouteAnchors(anchors);
  int sheetPaints = 0;

  Future<void> pump() => tester.pumpWidget(
    ProviderScope(
      overrides: [routeAnchorsProvider.overrideWith(() => seeded)],
      child: MaterialApp(
        home: Scaffold(
          body: Stack(
            children: [
              Positioned.fill(
                child: ValueListenableBuilder<Offset>(
                  valueListenable: shift,
                  builder: (context, offset, _) {
                    controller.shift = offset;
                    return MapHost(
                      controller: controller,
                      camera: cameraFor(),
                      child: RouteAnchorLayer(onAnchorTap: onAnchorTap),
                    );
                  },
                ),
              ),
              // The docked sheet / app bar: shares the page's repaint boundary
              // unless the marker layer isolates its own repaints.
              Positioned.fill(
                child: IgnorePointer(
                  child: CustomPaint(
                    painter: PaintCounter(() => sheetPaints++),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );

  Future<void> pan(Offset by) async {
    shift.value += by;
    await tester.pump();
  }

  Finder marker(String id) => find.byKey(ValueKey(id));
}

void main() {
  testWidgets(
    'a camera frame costs a constant number of native projections and '
    'leaves the rest of the page alone',
    (tester) async {
      final scene = _Scene(tester); // 19 anchors, as in the field reading
      await scene.pump();
      scene.controller.projections = 0;
      scene.sheetPaints = 0;

      const frames = 10;
      for (var i = 0; i < frames; i++) {
        await scene.pan(const Offset(3, 1));
      }

      expect(
        scene.controller.projections / frames,
        5,
        reason:
            'was one JNI projection per anchor per frame (19) with WidgetLayer',
      );
      expect(
        scene.sheetPaints,
        0,
        reason:
            'was 1 per frame: marker repaints dirtied the page-level repaint '
            'boundary and dragged the docked sheet into every pan frame',
      );
    },
  );

  testWidgets('markers show their 1-based position in the anchor list', (
    tester,
  ) async {
    final scene = _Scene(tester, anchors: 12);
    await scene.pump();

    for (var n = 1; n <= 12; n++) {
      expect(find.text('$n'), findsOneWidget);
    }
  });

  testWidgets('tapping a marker after a pan reports that anchor', (
    tester,
  ) async {
    final taps = <String>[];
    final scene = _Scene(tester, onAnchorTap: taps.add);
    await scene.pump();
    await scene.pan(const Offset(40, 25));

    await tester.tapAt(tester.getCenter(scene.marker('a4')));
    await tester.pump();

    expect(taps, ['a4']);
  });

  testWidgets(
    'a dragged marker follows the finger, the rest stay put, and the drop '
    'commits the anchor',
    (tester) async {
      final scene = _Scene(tester);
      await scene.pump();
      final start = tester.getCenter(scene.marker('a4'));
      final bystander = tester.getCenter(scene.marker('a5'));

      final gesture = await tester.startGesture(start);
      await gesture.moveBy(const Offset(40, 0));
      await tester.pump();
      await gesture.moveBy(const Offset(40, 0));
      await tester.pump();

      final midDrag = tester.getCenter(scene.marker('a4'));
      expect(
        midDrag.dx,
        greaterThan(start.dx + 30),
        reason: 'tracks the finger',
      );
      expect(tester.getCenter(scene.marker('a5')), bystander);
      expect(scene.seeded.dragged, isEmpty, reason: 'nothing commits mid-drag');

      await gesture.up();
      await tester.pump();

      expect(scene.seeded.dragged, ['a4']);
      expect(
        tester.getCenter(scene.marker('a4')),
        offsetMoreOrLessEquals(midDrag, epsilon: 0.5),
        reason: 'the anchor now lives where it was dropped',
      );
      expect(tester.getCenter(scene.marker('a5')), bystander);
    },
  );
  group('a state change that does not alter what the markers show', () {
    ReverseLocationResult location(String label) => ReverseLocationResult(
      label: label,
      fullLabel: '$label, Land',
      country: 'Land',
    );

    /// How many marker subtrees were rebuilt while [action] ran.
    Future<int> markerRebuilds(
      WidgetTester tester,
      Future<void> Function() action,
    ) async {
      var rebuilds = 0;
      debugOnRebuildDirtyWidget = (element, builtOnce) {
        if (element.widget is MapMarkerGestures) rebuilds++;
      };
      addTearDown(() => debugOnRebuildDirtyWidget = null);
      await action();
      debugOnRebuildDirtyWidget = null;
      return rebuilds;
    }

    testWidgets('a reverse-geocode result (location only) rebuilds no marker', (
      tester,
    ) async {
      final scene = _Scene(tester, anchors: 20);
      await scene.pump();

      final rebuilt = await markerRebuilds(tester, () async {
        // Exactly what _resolveAnchorLocation writes: a new list in which
        // one anchor carries a location. No marker reads it.
        scene.seeded.emit([
          for (final a in scene.seeded.current)
            if (a.id == 'a7') a.copyWith(location: location('Road')) else a,
        ]);
        await tester.pump();
      });

      expect(rebuilt, 0, reason: 'was one rebuild per anchor (20)');
      expect(find.text('8'), findsOneWidget);
    });

    testWidgets('moving an anchor still moves its marker', (tester) async {
      final scene = _Scene(tester);
      await scene.pump();
      final before = tester.getCenter(scene.marker('a3'));
      final bystander = tester.getCenter(scene.marker('a4'));

      final moved = ml.Geographic(
        lon: scene.anchors[3].lon + 0.2,
        lat: scene.anchors[3].lat,
      );
      scene.seeded.emit([
        for (final a in scene.seeded.current)
          if (a.id == 'a3') a.copyWith(lat: moved.lat, lon: moved.lon) else a,
      ]);
      await tester.pump();

      final after = tester.getCenter(scene.marker('a3'));
      expect(after, isNot(before));
      expect(
        after,
        offsetMoreOrLessEquals(FakeMapController.project(moved), epsilon: 0.5),
      );
      expect(tester.getCenter(scene.marker('a4')), bystander);
    });

    testWidgets('an appended anchor gets its marker', (tester) async {
      final scene = _Scene(tester, anchors: 5);
      await scene.pump();

      scene.seeded.emit([
        ...scene.seeded.current,
        RouteAnchor(id: 'new', lon: -3.0, lat: 1.0),
      ]);
      await tester.pump();

      expect(
        find.descendant(of: scene.marker('new'), matching: find.text('6')),
        findsOneWidget,
      );
    });

    testWidgets(
      'a different anchor at the same coordinates (undo/redo) owns the marker '
      'there: a tap reports it, not the one it replaced',
      (tester) async {
        final taps = <String>[];
        final scene = _Scene(tester, anchors: 5, onAnchorTap: taps.add);
        await scene.pump();

        final old = scene.seeded.current[2];
        scene.seeded.emit([
          for (final a in scene.seeded.current)
            if (a.id == old.id)
              RouteAnchor(id: 'x', lat: old.lat, lon: old.lon)
            else
              a,
        ]);
        await tester.pump();

        await tester.tapAt(tester.getCenter(scene.marker('x')));
        await tester.pump();

        expect(taps, ['x']);
        expect(scene.marker(old.id), findsNothing);
      },
    );

    testWidgets(
      'an anchor inserted in the middle renumbers the ones after it',
      (tester) async {
        final scene = _Scene(tester, anchors: 6);
        await scene.pump();

        final list = [...scene.seeded.current];
        list.insert(2, RouteAnchor(id: 'new', lon: -3.0, lat: -1.0));
        scene.seeded.emit(list);
        await tester.pump();

        for (var n = 1; n <= 7; n++) {
          expect(find.text('$n'), findsOneWidget, reason: 'number $n');
        }
        expect(
          find.descendant(of: scene.marker('new'), matching: find.text('3')),
          findsOneWidget,
        );
        expect(
          find.descendant(of: scene.marker('a2'), matching: find.text('4')),
          findsOneWidget,
          reason: 'the old third anchor is now fourth',
        );
      },
    );

    testWidgets('reordering anchors renumbers them (same ids, same points)', (
      tester,
    ) async {
      final scene = _Scene(tester, anchors: 4);
      await scene.pump();

      final list = [...scene.seeded.current];
      list.insert(0, list.removeAt(3)); // a3 jumps to the front
      scene.seeded.emit(list);
      await tester.pump();

      expect(
        find.descendant(of: scene.marker('a3'), matching: find.text('1')),
        findsOneWidget,
      );
      expect(
        find.descendant(of: scene.marker('a0'), matching: find.text('2')),
        findsOneWidget,
      );
    });
  });
}
