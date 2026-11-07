import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// `splash_logo_*.svg` is a derived copy of `logo_text_twoline_*.svg` with the
/// trail ribbon deleted, so the trail is left as an open hole for
/// `home_screen.dart` to paint the reveal through.
///
/// A derived copy can go stale: redesign the logo and the splash silently keeps
/// animating the old mark. Nothing in the build would catch that, so this
/// asserts the relationship instead of trusting it — regenerate the pair when
/// this fails, rather than editing the derived file to match.
void main() {
  const String dir = 'assets/svgs';

  /// The ribbon is the third path in the source: text, disc, trail, then the
  /// mountains, clouds and trees.
  const int trailPathIndex = 2;

  final RegExp pathElement = RegExp(r'<path[^>]*?/>', dotAll: true);
  final RegExp comment = RegExp(r'<!--.*?-->\s*', dotAll: true);

  String normalise(String svg) => svg.replaceAll(comment, '').trim();

  for (final String theme in <String>['light', 'dark']) {
    group('splash_logo_$theme.svg', () {
      final String source = File('$dir/logo_text_twoline_$theme.svg')
          .readAsStringSync();
      final String derived = File('$dir/splash_logo_$theme.svg')
          .readAsStringSync();
      final List<RegExpMatch> paths = pathElement
          .allMatches(source)
          .toList(growable: false);

      test('is the source logo with exactly the trail ribbon removed', () {
        final String trail = paths[trailPathIndex].group(0)!;
        expect(
          normalise(derived),
          normalise(source.replaceFirst(trail, '')),
          reason:
              'splash_logo_$theme.svg is out of sync with its source. '
              'Regenerate it as logo_text_twoline_$theme.svg minus its trail '
              'ribbon path, rather than hand-editing it.',
        );
      });

      test('the removed path is the trail ribbon, not some other shape', () {
        // Guards the index above: if the logo is redrawn with a different path
        // order, the test above would happily delete the wrong shape.
        final String d = RegExp(
          r'd="([^"]*)"',
        ).firstMatch(paths[trailPathIndex].group(0)!)!.group(1)!;
        expect(
          d,
          startsWith('M40.9546 119.979'),
          reason: 'path[$trailPathIndex] no longer starts at the trail\'s foot '
              'on the rim — the source logo\'s path order has changed.',
        );
      });

      test('leaves the trail as a hole rather than a filled shape', () {
        expect(
          pathElement.allMatches(derived).length,
          paths.length - 1,
          reason: 'exactly one path should have been removed',
        );
        expect(
          derived,
          isNot(contains('M40.9546 119.979')),
          reason: 'the trail ribbon is still present in the derived asset',
        );
      });
    });
  }
}
