import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/provider/map_style_json_provider.dart';

/// A lightweight, trail-agnostic native MapLibre GL map host (CORE-08).
///
/// Renders the basemap from [mapStyleJsonProvider] and swaps light/dark
/// styles live via [ml.MapController.setStyle] (mirrors `WandererMap`'s
/// CORE-02 pattern), but carries none of `WandererMap`'s single-trail or
/// offline surface: no `Trail` param, no offline style rewrite, no
/// bounds-fit-on-init. Bounds/camera work stays imperative in each caller's
/// `onStyleLoaded` — this host only exposes the seams (`layers`, `children`,
/// `onStyleLoaded`, `onMapEvent`, `onMapCreated`) callers build on.
///
/// Used by `list_detail_map_screen.dart`, `list_detail_screen.dart`'s
/// `_ListMap`, and `map_screen.dart` (16-03) — screens that render N trails
/// (or none) rather than a single trail's track.
class SearchMap extends ConsumerStatefulWidget {
  final ml.Geographic? initCenter;
  final double? initZoom;
  final bool disabled;
  final void Function(ml.MapController controller)? onMapCreated;
  final void Function(ml.StyleController style)? onStyleLoaded;
  final void Function(ml.MapEvent event)? onMapEvent;

  /// Declarative style layers (e.g. [ml.PolylineLayer]) forwarded to
  /// [ml.MapLibreMap.layers]. Note: despite the name, [ml.MapLibreMap.layers]
  /// is typed `List<Layer>` (native style-layer builders), not Flutter
  /// widgets — distinct from [children] below.
  final List<ml.Layer>? layers;
  final List<Widget>? children;

  const SearchMap({
    super.key,
    this.initCenter,
    this.initZoom,
    this.disabled = false,
    this.onMapCreated,
    this.onStyleLoaded,
    this.onMapEvent,
    this.layers,
    this.children,
  });

  @override
  ConsumerState<SearchMap> createState() => _SearchMapState();
}

class _SearchMapState extends ConsumerState<SearchMap> {
  ml.MapController? _controller;

  /// The last successfully-resolved style JSON. Cached so a provider refresh
  /// (e.g. a theme toggle) never drops us back to the loading state and
  /// remounts the map — the live swap goes through [ml.MapController.setStyle]
  /// instead (mirrors `WandererMap`'s CORE-02).
  String? _lastStyleJson;

  @override
  Widget build(BuildContext context) {
    // Live style swap: when the style JSON changes (theme toggle), swap the
    // style in place on the already-mounted map — no remount, no flash.
    ref.listen(mapStyleJsonProvider, (_, _) => _swapStyle());

    final baseAsync = ref.watch(mapStyleJsonProvider);
    final baseJson = baseAsync.value;
    final error = baseAsync.error;

    // No offline branch here — baseJson IS the style.
    if (baseJson == null) {
      if (error != null) {
        return Center(child: Text(error.toString()));
      }
      return ColoredBox(color: Theme.of(context).colorScheme.surface);
    }

    _lastStyleJson = baseJson;
    return _buildMap(context, baseJson);
  }

  void _swapStyle() {
    final controller = _controller;
    if (controller == null) return;
    final json = ref.read(mapStyleJsonProvider).value;
    if (json != null && json != _lastStyleJson) {
      _lastStyleJson = json;
      controller.setStyle(json);
    }
  }

  Widget _buildMap(BuildContext context, String styleJson) {
    return ml.MapLibreMap(
      options: ml.MapOptions(
        initStyle: styleJson,
        initCenter: widget.initCenter ?? const ml.Geographic(lat: 0, lon: 0),
        initZoom: widget.initZoom ?? 3,
        gestures: widget.disabled
            ? const ml.MapGestures.none()
            : const ml.MapGestures.all(),
        androidForegroundLoadColor: Theme.of(context).colorScheme.surface,
      ),
      onMapCreated: (controller) {
        _controller = controller;
        widget.onMapCreated?.call(controller);
      },
      onStyleLoaded: (style) => widget.onStyleLoaded?.call(style),
      onEvent: (event) => widget.onMapEvent?.call(event),
      layers: widget.layers ?? const [],
      children:
          widget.children ??
          const [ml.MapScalebar(), ml.SourceAttribution()],
    );
  }
}
