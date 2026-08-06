import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/map/map_marker_gestures.dart';

/// The map's platform view is not available in a widget test, so a full-screen
/// [Listener] underneath the marker stands in for it: whatever pointer events
/// it records are the events that would have reached the native map.
///
/// Marker geometry mirrors the real layers: a 36x36 box centered in the
/// viewport, exactly like an `ml.Marker(size: Size(36, 36))`.
void main() {
  const markerCenter = Offset(400, 300);
  const awayFromMarker = Offset(100, 100);

  late List<PointerEvent> mapEvents;
  late int taps;
  late List<String> panLog;

  Widget buildHarness({bool draggable = false}) {
    mapEvents = <PointerEvent>[];
    taps = 0;
    panLog = <String>[];

    return MediaQuery(
      data: const MediaQueryData(size: Size(800, 600)),
      child: Directionality(
        textDirection: TextDirection.ltr,
        child: Stack(
          children: [
            Listener(
              behavior: HitTestBehavior.opaque,
              onPointerDown: mapEvents.add,
              onPointerMove: mapEvents.add,
              onPointerUp: mapEvents.add,
              child: const SizedBox.expand(),
            ),
            Positioned(
              left: markerCenter.dx - 18,
              top: markerCenter.dy - 18,
              width: 36,
              height: 36,
              child: MapMarkerGestures(
                onTap: () => taps++,
                onPanStart: draggable ? (_) => panLog.add('start') : null,
                onPanUpdate: draggable ? (_) => panLog.add('update') : null,
                onPanEnd: draggable ? (_) => panLog.add('end') : null,
                onPanCancel: draggable ? () => panLog.add('cancel') : null,
                child: const ColoredBox(color: Color(0xff000000)),
              ),
            ),
          ],
        ),
      ),
    );
  }

  testWidgets('a single-finger tap on the marker fires onTap', (tester) async {
    await tester.pumpWidget(buildHarness());

    await tester.tapAt(markerCenter);
    await tester.pump();

    expect(taps, 1);
  });

  testWidgets('the map still receives pointers that land on the marker', (
    tester,
  ) async {
    await tester.pumpWidget(buildHarness());

    await tester.tapAt(markerCenter);
    await tester.pump();

    expect(
      mapEvents.whereType<PointerDownEvent>(),
      isNotEmpty,
      reason: 'the marker must not swallow the pointer',
    );
  });

  testWidgets('a drag across a tap-only marker pans the map, not the marker', (
    tester,
  ) async {
    await tester.pumpWidget(buildHarness());

    final gesture = await tester.startGesture(markerCenter);
    await gesture.moveBy(const Offset(120, 0));
    await gesture.up();
    await tester.pump();

    expect(taps, 0);
    expect(mapEvents.whereType<PointerMoveEvent>(), isNotEmpty);
  });

  testWidgets('a tap is ignored while a second finger is down', (tester) async {
    await tester.pumpWidget(buildHarness());

    final other = await tester.startGesture(awayFromMarker);
    final onMarker = await tester.startGesture(markerCenter);
    await onMarker.up();
    await other.up();
    await tester.pump();

    expect(taps, 0);
  });

  testWidgets('a second finger during a tap cancels it', (tester) async {
    await tester.pumpWidget(buildHarness());

    final onMarker = await tester.startGesture(markerCenter);
    final other = await tester.startGesture(awayFromMarker);
    await onMarker.up();
    await other.up();
    await tester.pump();

    expect(taps, 0);
  });

  testWidgets('a draggable marker still drags with one finger', (tester) async {
    await tester.pumpWidget(buildHarness(draggable: true));

    final gesture = await tester.startGesture(markerCenter);
    await gesture.moveBy(const Offset(60, 0));
    await gesture.moveBy(const Offset(60, 0));
    await gesture.up();
    await tester.pump();

    expect(panLog.first, 'start');
    expect(panLog, contains('update'));
    expect(panLog.last, 'end');
    expect(taps, 0);
  });

  testWidgets('a second finger stops a marker drag from starting', (
    tester,
  ) async {
    await tester.pumpWidget(buildHarness(draggable: true));

    final other = await tester.startGesture(awayFromMarker);
    final onMarker = await tester.startGesture(markerCenter);
    await onMarker.moveBy(const Offset(60, 0));
    await onMarker.up();
    await other.up();
    await tester.pump();

    expect(panLog, isEmpty);
  });
}
