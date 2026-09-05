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

  /// Attributions cached per native style object. Reading the controller
  /// subscribes this widget to every camera frame (the MapLibre inherited
  /// model notifies unconditionally), and `getAttributionsSync()` used to run
  /// on each of those frames — attributions only ever change when the style
  /// itself is swapped, so key the fetch on the style's identity instead.
  ml.StyleController? _attributionsStyle;
  List<String> _attributions = const [];

  @override
  Widget build(BuildContext context) {
    final style = ml.MapController.maybeOf(context)?.style;
    if (style == null) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final size = MediaQuery.sizeOf(context);

    if (!identical(style, _attributionsStyle)) {
      _attributionsStyle = style;
      _attributions = [
        '<a href="https://pub.dev/packages/maplibre">MapLibre</a>',
        ...style.getAttributionsSync(),
      ];
    }
    final attributions = _attributions;

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

/// One parsed HTML node: plain text, or a link with a long-lived tap
/// recognizer (created once at parse time, disposed with the state — the old
/// per-build parse allocated a fresh undisposed [TapGestureRecognizer] per
/// link per rebuild, and this widget rebuilds on every camera frame while
/// expanded).
class _ParsedNode {
  const _ParsedNode(this.text, {this.recognizer});
  final String text;
  final TapGestureRecognizer? recognizer;
}

class _AttributionHtmlState extends State<_AttributionHtml> {
  bool _hovering = false;
  List<_ParsedNode> _nodes = const [];

  @override
  void initState() {
    super.initState();
    _parse();
  }

  @override
  void didUpdateWidget(_AttributionHtml oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.html != widget.html) _parse();
  }

  @override
  void dispose() {
    _disposeRecognizers();
    super.dispose();
  }

  void _disposeRecognizers() {
    for (final node in _nodes) {
      node.recognizer?.dispose();
    }
  }

  void _parse() {
    _disposeRecognizers();
    final nodes = <_ParsedNode>[];
    final document = html_parser.parse(widget.html);
    for (final node in document.body!.nodes) {
      if (node is dom.Text) {
        nodes.add(_ParsedNode(node.text));
      } else if (node is dom.Element && node.localName == 'a') {
        nodes.add(
          _ParsedNode(
            node.text,
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
    _nodes = nodes;
  }

  @override
  Widget build(BuildContext context) {
    var textStyle = Theme.of(context).textTheme.bodySmall;
    if (_hovering) {
      textStyle = textStyle?.copyWith(decoration: TextDecoration.underline);
    }
    return RichText(
      text: TextSpan(
        style: textStyle,
        children: [
          for (final node in _nodes)
            if (node.recognizer == null)
              TextSpan(text: node.text)
            else
              TextSpan(
                onEnter: (event) => setState(() => _hovering = true),
                onExit: (event) => setState(() => _hovering = false),
                text: node.text,
                style: textStyle,
                recognizer: node.recognizer,
              ),
        ],
      ),
    );
  }
}
