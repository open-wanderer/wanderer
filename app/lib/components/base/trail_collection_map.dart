import 'package:wanderer/components/map/map_ui_controls.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/base/platform_view_pop_guard.dart';
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';

/// A lightweight, trail-agnostic native MapLibre GL map host.
///
/// Renders the basemap from [mapStyleJsonProvider] and swaps light/dark
/// styles live via [ml.MapController.setStyle] (mirrors `TrailMap`'s
/// style-swap pattern), but carries none of `TrailMap`'s single-trail or
/// offline surface: no `Trail` param, no offline style rewrite, no
/// bounds-fit-on-init. Bounds/camera work stays imperative in each caller's
/// `onStyleLoaded` — this host only exposes the seams (`layers`, `children`,
/// `onStyleLoaded`, `onMapEvent`, `onMapCreated`) callers build on.
///
/// Used by `list_detail_map_screen.dart`, `list_detail_screen.dart`'s
/// `_ListMap`, and `map_screen.dart` — screens that render N trails
/// (or none) rather than a single trail's track.
class TrailCollectionMap extends ConsumerStatefulWidget {
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

  const TrailCollectionMap({
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
  ConsumerState<TrailCollectionMap> createState() => _TrailCollectionMapState();
}

class _TrailCollectionMapState extends ConsumerState<TrailCollectionMap>
    with PlatformViewPopGuard<TrailCollectionMap> {
  ml.MapController? _controller;

  /// Buffers a style-loaded event that arrives before [_controller] is set.
  /// The native platform channel does not always fire `onMapCreated` before
  /// `onStyleLoaded` despite the package docs implying that order — callers
  /// that call back into [ml.MapController] from `onStyleLoaded` (e.g.
  /// `fitBounds`) would silently no-op on a still-null controller otherwise.
  ml.StyleController? _pendingStyle;

  /// The last successfully-resolved style JSON. Cached so a provider refresh
  /// (e.g. a theme toggle) never drops us back to the loading state and
  /// remounts the map — the live swap goes through [ml.MapController.setStyle]
  /// instead (mirrors `TrailMap`'s approach).
  String? _lastStyleJson;

  /// Cached [ml.MapOptions] + the `disabled` value it was built for --
  /// see the build-site comment.
  ml.MapOptions? _mapOptions;
  bool? _mapOptionsDisabled;

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
    // See [PlatformViewPopGuard] — drop the native surface as the pop begins
    // so it cannot outlive the transition.
    if (platformViewPopping) {
      return ColoredBox(color: Theme.of(context).colorScheme.surface);
    }

    // Cached on the `disabled` flip — see TrailMap's identical cache for why
    // (MapOptions has no value equality; a fresh instance per build
    // re-issues the plugin's option JNI setters on every rebuild).
    if (_mapOptions == null || _mapOptionsDisabled != widget.disabled) {
      _mapOptionsDisabled = widget.disabled;
      _mapOptions = ml.MapOptions(
        initStyle: styleJson,
        initCenter: widget.initCenter ?? const ml.Geographic(lat: 0, lon: 0),
        initZoom: widget.initZoom ?? 3,
        gestures: widget.disabled
            ? const ml.MapGestures.none()
            : const ml.MapGestures.all(),
        androidForegroundLoadColor: Theme.of(context).colorScheme.surface,
        // See TrailMap for why texture mode is off and `hc` is pinned.
        androidTextureMode: false,
        androidMode: ml.AndroidPlatformViewMode.hc,
      );
    }

    return ml.MapLibreMap(
      options: _mapOptions!,
      onMapCreated: (controller) {
        _controller = controller;
        widget.onMapCreated?.call(controller);
        final pending = _pendingStyle;
        if (pending != null) {
          _pendingStyle = null;
          widget.onStyleLoaded?.call(pending);
        }
      },
      onStyleLoaded: (style) {
        if (_controller == null) {
          _pendingStyle = style;
          return;
        }
        widget.onStyleLoaded?.call(style);
      },
      onEvent: (event) => widget.onMapEvent?.call(event),
      layers: widget.layers ?? const [],
      children:
          widget.children ??
          const [WandererMapScalebar(), WandererAttribution()],
    );
  }
}
