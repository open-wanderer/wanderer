import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/map_marker_gestures.dart';
import 'package:wanderer/components/map/screen_marker_layer.dart';

import 'map_test_harness.dart';

/// Marker content that records how often it is built and painted, so a test
/// can tell "moved" (cheap) from "rebuilt/repainted" (the per-frame cost the
/// layer exists to avoid).
class _Counters {
  final builds = <String, int>{};
  final paints = <String, int>{};
  int sheetPaints = 0;
  final taps = <String>[];
  final mapPointerDowns = <PointerEvent>[];
}

class _ProbeChild extends StatelessWidget {
  const _ProbeChild(this.id, this.counters);

  final String id;
  final _Counters counters;

  @override
  Widget build(BuildContext context) {
    counters.builds.update(id, (v) => v + 1, ifAbsent: () => 1);
    return CustomPaint(
      painter: PaintCounter(
        () => counters.paints.update(id, (v) => v + 1, ifAbsent: () => 1),
      ),
    );
  }
}

/// Remembers which marker id it was created for, to prove state follows the
/// key and not the list index.
class _StatefulProbe extends StatefulWidget {
  const _StatefulProbe(this.id);

  final String id;

  @override
  State<_StatefulProbe> createState() => _StatefulProbeState();
}

class _StatefulProbeState extends State<_StatefulProbe> {
  static int inits = 0;

  String? bornAs;

  @override
  void initState() {
    super.initState();
    inits++;
    bornAs = widget.id;
  }

  @override
  Widget build(BuildContext context) => const SizedBox.expand();
}

ml.Marker _marker(
  String id,
  ml.Geographic point,
  _Counters counters, {
  Widget? content,
  bool rotate = false,
}) => ml.Marker(
  point: point,
  size: const Size(32, 32),
  rotate: rotate,
  child: MapMarkerGestures(
    key: ValueKey(id),
    onTap: () => counters.taps.add(id),
    child: content ?? _ProbeChild(id, counters),
  ),
);

/// [count] markers on an on-screen grid (see [FakeMapController.project]).
List<ml.Marker> _grid(int count, _Counters counters) => [
  for (var i = 0; i < count; i++)
    _marker(
      'm$i',
      ml.Geographic(lon: -3.5 + (i % 10) * 0.7, lat: -2 + (i ~/ 10) * 0.8),
      counters,
    ),
];

class _Scene {
  _Scene(this.tester);

  final WidgetTester tester;
  final controller = FakeMapController();
  final counters = _Counters();
  final shift = ValueNotifier<Offset>(Offset.zero);
  final pitch = ValueNotifier<double>(0);
  late List<ml.Marker> markers;

  Future<void> pump(List<ml.Marker> markers) async {
    this.markers = markers;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: Stack(
            children: [
              // The map's platform view: records whatever reaches it.
              Positioned.fill(
                child: Listener(
                  behavior: HitTestBehavior.opaque,
                  onPointerDown: counters.mapPointerDowns.add,
                  child: const SizedBox.expand(),
                ),
              ),
              Positioned.fill(
                child: ListenableBuilder(
                  listenable: Listenable.merge([shift, pitch]),
                  builder: (context, _) {
                    controller.shift = shift.value;
                    return MapHost(
                      controller: controller,
                      camera: cameraFor(pitch: pitch.value),
                      child: ScreenMarkerLayer(markers: this.markers),
                    );
                  },
                ),
              ),
              // A docked sheet / app bar stand-in with no boundary of its own.
              Positioned.fill(
                child: IgnorePointer(
                  child: CustomPaint(
                    painter: PaintCounter(() => counters.sheetPaints++),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// One camera frame: the map moves by [by], the way a pan event would.
  Future<void> pan(Offset by) async {
    shift.value += by;
    await tester.pump();
  }

  Offset centerOf(String id) => tester.getCenter(find.byKey(ValueKey(id)));
}

void main() {
  testWidgets('places every marker at its projected screen position', (
    tester,
  ) async {
    final scene = _Scene(tester);
    await scene.pump(_grid(12, scene.counters));

    for (var i = 0; i < 12; i++) {
      final expected =
          FakeMapController.project(scene.markers[i].point) +
          scene.controller.shift;
      expect(scene.centerOf('m$i'), offsetMoreOrLessEquals(expected));
    }

    await scene.pan(const Offset(30, -20));

    for (var i = 0; i < 12; i++) {
      final expected =
          FakeMapController.project(scene.markers[i].point) +
          const Offset(30, -20);
      expect(scene.centerOf('m$i'), offsetMoreOrLessEquals(expected));
    }
  });

  testWidgets('honours the marker alignment', (tester) async {
    final scene = _Scene(tester);
    final markers = [
      for (var i = 0; i < 9; i++)
        ml.Marker(
          point: ml.Geographic(lon: 1 + i * 0.2, lat: 1 + i * 0.2),
          size: const Size(20, 40),
          alignment: Alignment.bottomCenter, // a pin: tip on the point
          child: SizedBox.expand(key: ValueKey('pin$i')),
        ),
    ];
    await scene.pump(markers);

    for (var i = 0; i < 9; i++) {
      final tip = FakeMapController.project(markers[i].point);
      final box = tester.getRect(find.byKey(ValueKey('pin$i')));
      expect(box.bottomCenter, offsetMoreOrLessEquals(tip));
    }
  });

  group('per-frame cost does not grow with the number of markers', () {
    testWidgets('native projections stay constant', (tester) async {
      final scene = _Scene(tester);
      await scene.pump(_grid(40, scene.counters));

      scene.controller.projections = 0;
      for (var i = 0; i < 10; i++) {
        await scene.pan(const Offset(3, 1));
      }

      // Five reference projections per frame, not 40.
      expect(scene.controller.projections, 10 * 5);
    });

    testWidgets('a tilted camera projects every marker natively instead', (
      tester,
    ) async {
      final scene = _Scene(tester);
      await scene.pump(_grid(40, scene.counters));

      scene.controller.projections = 0;
      scene.pitch.value = 30;
      await tester.pump();

      expect(scene.controller.projections, 40);
      // ...and they are still where the platform says they are.
      expect(
        scene.centerOf('m7'),
        offsetMoreOrLessEquals(
          FakeMapController.project(scene.markers[7].point),
        ),
      );
    });

    testWidgets('markers are neither rebuilt nor repainted when only moved', (
      tester,
    ) async {
      final scene = _Scene(tester);
      await scene.pump(_grid(30, scene.counters));
      final before = scene.centerOf('m4');
      scene.counters.builds.clear();
      scene.counters.paints.clear();

      for (var i = 0; i < 10; i++) {
        await scene.pan(const Offset(2, 2));
      }

      expect(scene.centerOf('m4'), isNot(before), reason: 'they did move');
      expect(scene.counters.builds, isEmpty, reason: 'no marker rebuilds');
      expect(
        scene.counters.paints,
        isEmpty,
        reason: 'marker pictures are reused; only their offset changes',
      );
    });

    testWidgets('camera frames do not repaint the rest of the page', (
      tester,
    ) async {
      final scene = _Scene(tester);
      await scene.pump(_grid(30, scene.counters));
      scene.counters.sheetPaints = 0;

      for (var i = 0; i < 10; i++) {
        await scene.pan(const Offset(2, 2));
      }

      expect(
        scene.counters.sheetPaints,
        0,
        reason:
            'the marker layer must be its own repaint boundary, or every pan '
            'frame re-records and re-rasterises the docked sheet',
      );
    });
  });

  testWidgets('markers outside the viewport are not painted until they enter', (
    tester,
  ) async {
    final scene = _Scene(tester);
    final markers = [
      ..._grid(10, scene.counters),
      // ~100 px per degree: lon 60 is x = 6400, far right of an 800 px surface.
      _marker('far', ml.Geographic(lon: 60, lat: 0), scene.counters),
    ];
    await scene.pump(markers);

    expect(scene.counters.paints['m0'], isNonZero);
    expect(scene.counters.paints['far'], isNull, reason: 'culled');

    await scene.pan(const Offset(-6000, 0));

    expect(
      scene.counters.paints['far'],
      1,
      reason: 'painted once it is in view',
    );
  });

  testWidgets('a tap reaches the marker where it was moved to', (tester) async {
    final scene = _Scene(tester);
    await scene.pump(_grid(12, scene.counters));
    await scene.pan(const Offset(40, 25));

    await tester.tapAt(scene.centerOf('m3'));
    await tester.pump();

    expect(scene.counters.taps, ['m3']);
  });

  testWidgets('pointers off the markers still reach the map underneath', (
    tester,
  ) async {
    final scene = _Scene(tester);
    await scene.pump(_grid(12, scene.counters));
    scene.counters.mapPointerDowns.clear();

    await tester.tapAt(const Offset(780, 580)); // empty corner
    await tester.pump();

    expect(scene.counters.mapPointerDowns, hasLength(1));
    expect(scene.counters.taps, isEmpty);
  });

  testWidgets('per-marker state follows the key, not the list index', (
    tester,
  ) async {
    final scene = _Scene(tester);
    List<ml.Marker> build(List<String> ids) => [
      for (var i = 0; i < ids.length; i++)
        _marker(
          ids[i],
          ml.Geographic(lon: -3 + i * 0.4, lat: 0),
          scene.counters,
          content: _StatefulProbe(ids[i]),
        ),
    ];
    final ids = [for (var i = 0; i < 10; i++) 'a$i'];
    await scene.pump(build(ids));
    final initsBefore = _StatefulProbeState.inits;

    // Remove the first marker: every other marker shifts down one slot.
    scene.markers = build(ids.sublist(1));
    scene.shift.value += const Offset(1, 0); // any rebuild trigger
    await tester.pump();

    expect(
      _StatefulProbeState.inits,
      initsBefore,
      reason: 'surviving markers must keep their State, not be re-created',
    );

    for (final id in ids.sublist(1)) {
      final state = tester.state<_StatefulProbeState>(
        find.descendant(
          of: find.byKey(ValueKey(id)),
          matching: find.byType(_StatefulProbe),
        ),
      );
      expect(state.bornAs, id, reason: '$id kept its own State');
    }
  });

  testWidgets('rotate/flat markers are rejected in debug builds', (
    tester,
  ) async {
    final scene = _Scene(tester);
    await scene.pump([
      _marker(
        'r',
        const ml.Geographic(lon: 1, lat: 1),
        scene.counters,
        rotate: true,
      ),
    ]);

    expect(tester.takeException(), isA<AssertionError>());
  });
}
