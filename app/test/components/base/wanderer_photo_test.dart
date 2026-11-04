import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/base/wanderer_photo.dart';
import 'package:wanderer/util/route_placeholder.dart';

const _svg =
    '<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" '
    'viewBox="0 0 1200 800"><rect width="1200" height="800"/></svg>';

Widget _app(Widget child, {Brightness brightness = Brightness.light}) =>
    MaterialApp(
      theme: ThemeData(brightness: brightness),
      home: Center(child: SizedBox(width: 288, height: 162, child: child)),
    );

RoutePlaceholderLoader _loader(WidgetTester tester) =>
    tester.widget<SvgPicture>(find.byType(SvgPicture)).bytesLoader
        as RoutePlaceholderLoader;

/// Lets real file reads finish. The test clock doesn't drive them, and each
/// step of a read resumes only on a pump.
Future<void> _settleIo(WidgetTester tester) async {
  for (var i = 0; i < 10; i++) {
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 20)),
    );
    await tester.pump();
  }
}

void main() {
  late Directory dir;
  late String path;

  setUp(() {
    dir = Directory.systemTemp.createTempSync('wanderer_photo');
    path = '${dir.path}/route.svg';
  });
  tearDown(() => dir.deleteSync(recursive: true));

  testWidgets('draws a downloaded SVG', (tester) async {
    File(path).writeAsStringSync(_svg);

    await tester.pumpWidget(_app(WandererPhoto.file(path)));
    await _settleIo(tester);

    expect(find.byType(Image), findsNothing);
    expect(_loader(tester).svg, _svg);
  });

  testWidgets('leaves other photos to Image', (tester) async {
    await tester.pumpWidget(_app(const WandererPhoto.file('/photos/a.jpg')));

    expect(find.byType(Image), findsOneWidget);
    expect(find.byType(SvgPicture), findsNothing);
  });

  testWidgets('never hands a server SVG to Image', (tester) async {
    await tester.pumpWidget(
      _app(const WandererPhoto.network('https://wanderer.example/route.svg')),
    );

    expect(find.byType(Image), findsNothing);
  });

  testWidgets('follows the theme of the app', (tester) async {
    File(path).writeAsStringSync(_svg);
    final photo = WandererPhoto.file(path);

    await tester.pumpWidget(_app(photo));
    await _settleIo(tester);
    expect(_loader(tester).dark, isFalse);

    await tester.pumpWidget(_app(photo, brightness: Brightness.dark));
    await tester.pumpAndSettle();
    expect(_loader(tester).dark, isTrue);
  });

  testWidgets('styles the image for the box it fills', (tester) async {
    File(path).writeAsStringSync(_svg);

    await tester.pumpWidget(
      _app(WandererPhoto.file(path, fit: BoxFit.contain)),
    );
    await _settleIo(tester);

    expect(_loader(tester).box, const Size(288, 162));
    expect(_loader(tester).fit, BoxFit.contain);
  });

  testWidgets('falls back when the image is missing', (tester) async {
    await tester.pumpWidget(
      _app(
        WandererPhoto.file(path, errorBuilder: (_) => const Text('fallback')),
      ),
    );
    await _settleIo(tester);

    expect(find.text('fallback'), findsOneWidget);
  });

  testWidgets('a failed load recovers on the next mount', (tester) async {
    final photo = WandererPhoto.file(
      path,
      errorBuilder: (_) => const Text('fallback'),
    );

    await tester.pumpWidget(_app(photo));
    await _settleIo(tester);
    expect(find.text('fallback'), findsOneWidget);

    File(path).writeAsStringSync(_svg);
    await tester.pumpWidget(_app(const SizedBox()));
    await tester.pumpWidget(_app(photo));
    await _settleIo(tester);

    expect(find.text('fallback'), findsNothing);
    expect(find.byType(SvgPicture), findsOneWidget);
  });

  testWidgets('reloads when a reused widget gets another photo', (
    tester,
  ) async {
    final other = '${dir.path}/other.svg';
    File(path).writeAsStringSync(_svg);
    File(other).writeAsStringSync(_svg.replaceAll('<rect', '<rect id="b"'));

    await tester.pumpWidget(_app(WandererPhoto.file(path)));
    await _settleIo(tester);
    await tester.pumpWidget(_app(WandererPhoto.file(other)));
    await _settleIo(tester);

    expect(_loader(tester).svg, contains('id="b"'));
  });
}
