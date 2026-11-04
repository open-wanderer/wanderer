import 'package:flutter/widgets.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:xml/xml.dart';

/// Whether [source], a URL or a file path, names an SVG.
///
/// The server stores the route image it generates for a trail or summit log
/// without photos as an SVG, in the same `photos` field as uploaded images
/// (`db/services/routeimage`). [Image] can't decode those.
bool isSvgPhoto(String source) =>
    (Uri.tryParse(source)?.path ?? source).toLowerCase().endsWith('.svg');

final _darkMediaQuery = RegExp(
  r'@media\s*\(\s*prefers-color-scheme\s*:\s*dark\s*\)\s*\{((?:[^{}]*\{[^{}]*\})*)[^{}]*\}',
);
final _cssRule = RegExp(r'([^{}]+)\{([^{}]*)\}');

/// Prepares a server-rendered route image for flutter_svg, which ignores
/// `<style>` and `vector-effect`, the two things the image relies on in a
/// browser.
///
/// With [dark], the rules of the `prefers-color-scheme: dark` media query are
/// written onto the elements they select. The light style needs no work: it
/// is what the presentation attributes already say. Only class selectors are
/// understood, which is all the renderer emits.
///
/// Strokes marked `non-scaling-stroke` keep their width on screen however
/// small the image is shown. Without that, the route of a 1200 unit wide image
/// would thin out to half a pixel in a trail card. Their widths are divided by
/// the scale the image is drawn at when it fills [box] according to [fit].
String themeRoutePlaceholder(
  String svg, {
  required bool dark,
  required Size box,
  required BoxFit fit,
}) {
  final document = XmlDocument.parse(svg);
  final root = document.rootElement;
  final rules = dark ? _darkRules(root) : const <String, Map<String, String>>{};
  final scale = _displayScale(root, box, fit);
  root.findAllElements('style').toList().forEach((style) => style.remove());

  for (final element in root.descendantElements) {
    final classes = element.getAttribute('class')?.split(RegExp(r'\s+'));
    for (final name in classes ?? const <String>[]) {
      rules[name]?.forEach(element.setAttribute);
    }

    if (element.getAttribute('vector-effect') == 'non-scaling-stroke') {
      final width = double.tryParse(element.getAttribute('stroke-width') ?? '');
      if (width != null) {
        element.setAttribute('stroke-width', '${width / scale}');
      }
    }
  }
  return document.toXmlString();
}

/// The declarations of the dark media query, by the class they select.
Map<String, Map<String, String>> _darkRules(XmlElement root) {
  final rules = <String, Map<String, String>>{};
  for (final style in root.findAllElements('style')) {
    for (final media in _darkMediaQuery.allMatches(style.innerText)) {
      for (final rule in _cssRule.allMatches(media.group(1)!)) {
        final declarations = <String, String>{};
        for (final declaration in rule.group(2)!.split(';')) {
          final colon = declaration.indexOf(':');
          if (colon < 0) continue;
          declarations[declaration.substring(0, colon).trim()] = declaration
              .substring(colon + 1)
              .trim();
        }
        for (final selector in rule.group(1)!.split(',')) {
          final trimmed = selector.trim();
          if (!trimmed.startsWith('.')) continue;
          (rules[trimmed.substring(1)] ??= {}).addAll(declarations);
        }
      }
    }
  }
  return rules;
}

/// Logical pixels per viewBox unit when the image fills [box].
double _displayScale(XmlElement root, Size box, BoxFit fit) {
  final viewBox = root
      .getAttribute('viewBox')
      ?.trim()
      .split(RegExp(r'[\s,]+'))
      .map(double.tryParse)
      .toList();
  if (viewBox == null || viewBox.length != 4) return 1;
  final width = viewBox[2];
  final height = viewBox[3];
  if (width == null || height == null) return 1;

  final sizes = applyBoxFit(fit, Size(width, height), box);
  final scale = sizes.destination.width / sizes.source.width;
  return scale.isFinite && scale > 0 ? scale : 1;
}

/// Compiles a route image for the current theme and display size.
///
/// Takes the SVG as text rather than fetching it: a failed load must never
/// reach flutter_svg, whose cache keeps a failed future forever, or
/// vector_graphics, which rethrows it as an uncaught error.
@immutable
class RoutePlaceholderLoader extends SvgLoader<void> {
  final String svg;
  final bool dark;
  final Size box;
  final BoxFit fit;

  const RoutePlaceholderLoader(
    this.svg, {
    required this.dark,
    required this.box,
    required this.fit,
  });

  @override
  String provideSvg(void message) =>
      themeRoutePlaceholder(svg, dark: dark, box: box, fit: fit);

  @override
  int get hashCode => Object.hash(svg, dark, box, fit);

  @override
  bool operator ==(Object other) =>
      other is RoutePlaceholderLoader &&
      other.svg == svg &&
      other.dark == dark &&
      other.box == box &&
      other.fit == fit;
}
