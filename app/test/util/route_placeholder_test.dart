import 'package:flutter/widgets.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/util/route_placeholder.dart';
import 'package:xml/xml.dart';

/// The markup of `routeimage.Render` in `db/services/routeimage/render.go`,
/// cut down to one element of each kind.
const _svg = '''
<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" viewBox="0 0 1200 800"><style>@media (prefers-color-scheme: dark) {
.bg0 { stop-color: #0b0d16 }
.contours { stroke: #fff; opacity: 0.625 }
.casing { stroke: #0b0d16; stroke-opacity: 0.6 }
.route, .end { stroke: #fff }
.ring { stroke: #0b0d16 }
}</style><defs><linearGradient id="bg" x1="0" y1="0" x2="0" y2="1"><stop offset="0" class="bg0" stop-color="#f8f9fc"/></linearGradient></defs><rect width="1200" height="800" fill="url(#bg)"/><g class="contours" fill="none" stroke="#000" stroke-linejoin="round"><path d="M0 42L10 43" stroke-width="0.3" stroke-opacity="0.029"/></g><path class="casing" d="M10 10L20 20" fill="none" stroke="#ffffff" stroke-width="6" vector-effect="non-scaling-stroke" stroke-opacity="0.7"/><path class="route" d="M10 10L20 20" fill="none" stroke="#12141c" stroke-width="2.5" vector-effect="non-scaling-stroke"/><path class="ring" d="M20 20h0" fill="none" stroke="#ffffff" stroke-width="13" vector-effect="non-scaling-stroke"/><path class="end" d="M20 20h0" fill="none" stroke="#12141c" stroke-width="8" vector-effect="non-scaling-stroke"/><path class="start" d="M10 10h0" fill="none" stroke="#4ade80" stroke-width="8" vector-effect="non-scaling-stroke"/></svg>
''';

XmlElement _element(String svg, String className) =>
    XmlDocument.parse(svg).descendantElements
        .firstWhere((e) => e.getAttribute('class') == className);

String _themed({
  bool dark = false,
  Size box = const Size(1200, 800),
  BoxFit fit = BoxFit.cover,
}) => themeRoutePlaceholder(_svg, dark: dark, box: box, fit: fit);

void main() {
  group('isSvgPhoto', () {
    test('recognises SVG file names, URLs and paths', () {
      expect(isSvgPhoto('wanderer_route_placeholder_a1b2c3.svg'), isTrue);
      expect(
        isSvgPhoto(
          'https://wanderer.example/api/v1/files/trails/abc/route.svg?thumb=600x0',
        ),
        isTrue,
      );
      expect(isSvgPhoto('/data/library/abc/photos/ROUTE.SVG'), isTrue);
    });

    test('leaves other images alone', () {
      expect(isSvgPhoto('https://wanderer.example/photo.jpg'), isFalse);
      expect(isSvgPhoto('https://wanderer.example/a.jpg?name=b.svg'), isFalse);
      expect(isSvgPhoto('/data/library/abc/photos/photo.webp'), isFalse);
    });
  });

  group('theme', () {
    test('the light theme keeps the presentation attributes', () {
      final svg = _themed();

      final stop = _element(svg, 'bg0');
      expect(stop.getAttribute('stop-color'), '#f8f9fc');
      expect(_element(svg, 'route').getAttribute('stroke'), '#12141c');
      expect(_element(svg, 'contours').getAttribute('stroke'), '#000');
    });

    test('the dark theme applies the media query to the elements', () {
      final svg = _themed(dark: true);

      expect(_element(svg, 'bg0').getAttribute('stop-color'), '#0b0d16');
      expect(_element(svg, 'ring').getAttribute('stroke'), '#0b0d16');

      final contours = _element(svg, 'contours');
      expect(contours.getAttribute('stroke'), '#fff');
      expect(contours.getAttribute('opacity'), '0.625');

      final casing = _element(svg, 'casing');
      expect(casing.getAttribute('stroke'), '#0b0d16');
      expect(casing.getAttribute('stroke-opacity'), '0.6');
    });

    test('a rule reaches every class in its selector list', () {
      final svg = _themed(dark: true);

      expect(_element(svg, 'route').getAttribute('stroke'), '#fff');
      expect(_element(svg, 'end').getAttribute('stroke'), '#fff');
    });

    test('elements no rule selects keep their style', () {
      final svg = _themed(dark: true);

      expect(_element(svg, 'start').getAttribute('stroke'), '#4ade80');
    });

    test('drops <style>, which flutter_svg warns about', () {
      expect(_themed(dark: true), isNot(contains('<style')));
      expect(_themed(), isNot(contains('<style')));
    });
  });

  group('non-scaling strokes', () {
    double strokeWidth(String svg, String className) =>
        double.parse(_element(svg, className).getAttribute('stroke-width')!);

    test('keep their width at the size of the viewBox', () {
      expect(strokeWidth(_themed(), 'route'), 2.5);
    });

    test('keep their on-screen width in a trail card', () {
      // Cover scales by the larger ratio: 288 / 1200 = 0.24.
      final svg = _themed(box: const Size(288, 162));

      expect(strokeWidth(svg, 'route') * 0.24, closeTo(2.5, 1e-9));
      expect(strokeWidth(svg, 'ring') * 0.24, closeTo(13, 1e-9));
    });

    test('follow the fit of the image', () {
      // Contain scales by the smaller ratio: 400 / 1200.
      final svg = _themed(box: const Size(400, 800), fit: BoxFit.contain);

      expect(strokeWidth(svg, 'route') / 3, closeTo(2.5, 1e-9));
    });

    test('work without a bounded height', () {
      final svg = _themed(
        box: const Size(600, double.infinity),
        fit: BoxFit.fitWidth,
      );

      expect(strokeWidth(svg, 'route') / 2, closeTo(2.5, 1e-9));
    });

    test('survive an empty box', () {
      expect(strokeWidth(_themed(box: Size.zero), 'route'), 2.5);
    });

    test('leave the contours scaling with the image', () {
      final svg = _themed(box: const Size(288, 162));
      final contour = XmlDocument.parse(svg).descendantElements
          .firstWhere((e) => e.getAttribute('d') == 'M0 42L10 43');

      expect(contour.getAttribute('stroke-width'), '0.3');
    });
  });

  group('RoutePlaceholderLoader', () {
    tearDown(svg.cache.clear);

    RoutePlaceholderLoader loader({bool dark = false}) =>
        RoutePlaceholderLoader(
          _svg,
          dark: dark,
          box: const Size(288, 162),
          fit: BoxFit.cover,
        );

    test('compiles the image', () async {
      final bytes = await loader().loadBytes(null);

      expect(bytes.lengthInBytes, greaterThan(0));
    });

    test('compiles the two themes separately', () async {
      final light = await loader().loadBytes(null);
      final dark = await loader(dark: true).loadBytes(null);

      expect(
        light.buffer.asUint8List(),
        isNot(orderedEquals(dark.buffer.asUint8List())),
      );
    });

    test('is the same loader for the same image, theme and size', () {
      expect(loader(), loader());
      expect(loader().hashCode, loader().hashCode);
      expect(loader(), isNot(loader(dark: true)));
    });
  });
}
