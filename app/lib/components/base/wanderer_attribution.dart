import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:html/dom.dart' as dom;
import 'package:html/parser.dart' as html_parser;
import 'package:maplibre/maplibre.dart' as ml;
import 'package:pointer_interceptor/pointer_interceptor.dart';
import 'package:url_launcher/url_launcher.dart';

/// A collapsed-by-default attribution control — a drop-in
/// replacement for `ml.SourceAttribution` for use in [ml.MapLibreMap.children].
///
/// Modeled closely on maplibre 0.3.5's `SourceAttribution` (the pill layout +
/// tappable HTML-link rendering), with two deliberate differences: (1) the
/// info-button pill starts collapsed instead of expanded, and (2) there is no
/// camera-change auto-collapse logic — unnecessary once it already starts
/// collapsed.
@immutable
class WandererAttribution extends StatefulWidget {
  final Alignment alignment;
  final EdgeInsets padding;
  const WandererAttribution({
    super.key,
    this.alignment = Alignment.bottomRight,
    this.padding = const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
  });

  @override
  State<WandererAttribution> createState() => _WandererAttributionState();
}

class _WandererAttributionState extends State<WandererAttribution> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final style = ml.MapController.maybeOf(context)?.style;
    if (style == null) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final size = MediaQuery.sizeOf(context);

    final attributions = [
      '<a href="https://pub.dev/packages/maplibre">MapLibre</a>',
      ...style.getAttributionsSync(),
    ];

    // Use a SafeArea to ensure the widget is completely visible on devices
    // with rounded edges like iOS.
    return SafeArea(
      child: Container(
        alignment: widget.alignment,
        padding: widget.padding,
        child: PointerInterceptor(
          child: Container(
            decoration: BoxDecoration(
              color: theme.scaffoldBackgroundColor,
              borderRadius: BorderRadius.circular(20),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (_expanded)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 5, top: 5, left: 10),
                    child: ConstrainedBox(
                      constraints: BoxConstraints(maxWidth: size.width / 2),
                      child: Wrap(
                        crossAxisAlignment: WrapCrossAlignment.center,
                        spacing: 2,
                        runSpacing: 2,
                        children: attributions
                            .map(_AttributionHtml.new)
                            .toList(growable: false),
                      ),
                    ),
                  ),
                SizedBox.square(
                  dimension: 30,
                  child: IconButton(
                    onPressed: () => setState(() => _expanded = !_expanded),
                    icon: const Icon(Icons.info, size: 18),
                    padding: const EdgeInsets.all(4),
                    constraints: const BoxConstraints(),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _AttributionHtml extends StatefulWidget {
  const _AttributionHtml(this.html);

  final String html;

  @override
  State<_AttributionHtml> createState() => _AttributionHtmlState();
}

class _AttributionHtmlState extends State<_AttributionHtml> {
  bool _hovering = false;

  @override
  Widget build(BuildContext context) {
    var textStyle = Theme.of(context).textTheme.bodySmall;
    if (_hovering) {
      textStyle = textStyle?.copyWith(decoration: TextDecoration.underline);
    }
    final textSpans = <TextSpan>[];
    final document = html_parser.parse(widget.html);

    for (final node in document.body!.nodes) {
      if (node is dom.Text) {
        textSpans.add(TextSpan(text: node.text));
      } else if (node is dom.Element && node.localName == 'a') {
        textSpans.add(
          TextSpan(
            onEnter: (event) => setState(() => _hovering = true),
            onExit: (event) => setState(() => _hovering = false),
            text: node.text,
            style: textStyle,
            recognizer: TapGestureRecognizer()
              ..onTap = () {
                if (node.attributes['href'] case final String href) {
                  launchUrl(Uri.parse(href));
                }
              },
          ),
        );
      }
    }
    return RichText(
      text: TextSpan(style: textStyle, children: textSpans),
    );
  }
}
