import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/route_anchor_layer.dart';
import 'package:wanderer/components/map/route_segment_layer.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/route_segment_util.dart';

/// Route Planner screen — hosts the native map and wires every Phase 19
/// gesture/state interaction to the UI: tap-to-add (WAYP-01), drag-to-
/// reposition (WAYP-02, handled inside [RouteAnchorLayer]), tap-to-insert /
/// tap-to-retry on a segment (WAYP-03/D-09), and the auto-routing toggle
/// (ROUTE-01/02, D-06). This is Phase 19's final integration point — 19-01/
/// 19-02/19-03 built correct, isolated components; this screen makes them
/// reachable by a user.
///
/// Router registration/entry-point wiring (the real [initialCenter]/
/// [travelProfile] values, the app-bar title) is HANDOFF-02/03 — explicitly
/// Phase 21's scope, not touched here.
class RoutePlannerScreen extends ConsumerStatefulWidget {
  /// `'pedestrian'` or `'bicycle'` — fixed for the entire planning session
  /// (D-07), set once by the Phase 21 entry-point dialog.
  final String travelProfile;

  /// The map's initial camera center. Required with no fallback default —
  /// inventing a hardcoded center here would silently mask a caller bug;
  /// Phase 21's entry point supplies the real value (e.g. the user's current
  /// location or the trail-source-select flow's last map camera).
  final ml.Geographic initialCenter;

  final String title;

  const RoutePlannerScreen({
    super.key,
    required this.travelProfile,
    required this.initialCenter,
    this.title = 'Plan a Route',
  });

  @override
  ConsumerState<RoutePlannerScreen> createState() =>
      _RoutePlannerScreenState();
}

class _RoutePlannerScreenState extends ConsumerState<RoutePlannerScreen> {
  static final _segmentLayer = RouteSegmentLayer();

  ml.MapController? _mapController;

  /// Buffers a style-loaded event that arrives before [_mapController] is
  /// set — copied verbatim from `TrailMap`'s pattern (Pitfall 5).
  ml.StyleController? _pendingStyle;

  /// The last resolved style. Cached so [ref.listen] can push segment
  /// updates to the native layer without waiting for another
  /// `onStyleLoaded` call.
  ml.StyleController? _resolvedStyle;

  /// Flips true the first time an anchor is placed and never re-shown again
  /// for the rest of this session (even if the anchor list is later emptied
  /// via undo).
  bool _hintDismissed = false;

  /// Segment keys a retry was just dispatched for, populated by the
  /// blocked-segment tap branch below. Consumed by Task 2's blocked-segment
  /// listener to distinguish a first-time failure from a second consecutive
  /// failed retry (UI-SPEC's Copywriting Contract).
  final Set<String> _retryAttempted = {};

  @override
  Widget build(BuildContext context) {
    final state = ref.watch(routeAnchorsProvider(widget.travelProfile));

    ref.listen(routeAnchorsProvider(widget.travelProfile), (prev, next) {
      final style = _resolvedStyle;
      if (style != null && !identical(prev?.segments, next.segments)) {
        // Keeps the native segment layer's GeoJSON in sync with every
        // mutation via updateGeoJsonSource — no remove/re-add flicker.
        _segmentLayer.update(style, next.segments).ignore();
      }
      if (next.anchors.isNotEmpty) _hintDismissed = true;
    });

    final styleJson = ref.watch(mapStyleJsonProvider).value;

    return _buildMap(context, state, styleJson);
  }

  Widget _buildMap(
    BuildContext context,
    RouteAnchorsState state,
    String? styleJson,
  ) {
    if (styleJson == null) {
      return ColoredBox(color: Theme.of(context).colorScheme.surface);
    }

    return ml.MapLibreMap(
      options: ml.MapOptions(
        initStyle: styleJson,
        initCenter: widget.initialCenter,
        initZoom: 15,
        gestures: const ml.MapGestures.all(),
        androidForegroundLoadColor: Theme.of(context).colorScheme.surface,
      ),
      onMapCreated: (controller) {
        _mapController = controller;
        final pending = _pendingStyle;
        if (pending != null) {
          _pendingStyle = null;
          _onStyleLoaded(pending);
        }
      },
      onStyleLoaded: (style) {
        if (_mapController == null) {
          _pendingStyle = style;
          return;
        }
        _onStyleLoaded(style);
      },
      onEvent: (event) {
        if (event is! ml.MapEventClick) return;

        // A marker tap never reaches here — RouteAnchorLayer's own
        // GestureDetector.onTap consumes it first (D-04: marker wins on
        // overlap).
        final hits =
            _mapController?.featuresAtPoint(
              event.screenPoint,
              layerIds: const ['route-segments-hit'],
            ) ??
            const [];
        final notifier = ref.read(
          routeAnchorsProvider(widget.travelProfile).notifier,
        );

        if (hits.isNotEmpty) {
          final props = hits.first.properties;
          final before = props['beforeAnchorId'] as String;
          final after = props['afterAnchorId'] as String;
          final segState = props['state'] as String;

          if (segState == 'blocked') {
            // D-09: a blocked segment retries instead of inserting — mutually
            // exclusive by segment state, never a competing gesture.
            _retryAttempted.add(segmentKey(before, after));
            ref
                .read(toastProvider.notifier)
                .add(
                  ToastMessage(
                    type: ToastType.info,
                    icon: FontAwesomeIcons.route,
                    text: 'Retrying route…',
                  ),
                );
            notifier.retrySegment(before, after);
          } else {
            notifier.insertAnchorOnSegment(before, after, event.point);
          }
          return;
        }

        // Empty-map tap always appends (D-03) — no add-mode toggle exists.
        notifier.appendAnchor(event.point);
      },
      children: [
        RouteAnchorLayer(travelProfile: widget.travelProfile),
        // D-06: shared top-right controls slot — Phase 20's future
        // list/elevation toggle buttons will join this Column, so this does
        // not assume it is the only child.
        Align(
          alignment: Alignment.topRight,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [_buildAutoRoutingToggle(state.autoRoutingEnabled)],
          ),
        ),
        if (state.anchors.isEmpty && !_hintDismissed)
          const Center(child: Text('Tap the map to start your route.')),
      ],
    );
  }

  /// (Re)runs the style-loaded work: caches the resolved style and does the
  /// first-time add of all 5 native segment layers. Buffered/replayed via
  /// [_pendingStyle] when the native platform channel fires `onStyleLoaded`
  /// before `onMapCreated` (Pitfall 5).
  void _onStyleLoaded(ml.StyleController style) {
    _resolvedStyle = style;
    _segmentLayer
        .update(
          style,
          ref.read(routeAnchorsProvider(widget.travelProfile)).segments,
        )
        .ignore();
  }

  /// D-06: top-right pill control toggling ROUTE-01/02's auto-routing flag.
  /// Matches `_buildMapControls`' existing pill container styling exactly
  /// (UI-SPEC's Map Control & App Bar Contract).
  Widget _buildAutoRoutingToggle(bool enabled) {
    return Padding(
      padding: const EdgeInsets.all(8.0),
      child: Container(
        decoration: BoxDecoration(
          color: Theme.of(context).canvasColor,
          borderRadius: BorderRadius.circular(24),
        ),
        child: IconButton(
          visualDensity: VisualDensity.compact,
          tooltip: enabled ? 'Auto-routing on' : 'Auto-routing off',
          icon: FaIcon(
            FontAwesomeIcons.route,
            size: 18,
            color: enabled
                ? (Theme.of(context).brightness == Brightness.dark
                      ? const Color(0xff3E435B)
                      : const Color(0xff242734))
                : Theme.of(
                    context,
                  ).colorScheme.onSurface.withValues(alpha: .4),
          ),
          onPressed: () => ref
              .read(routeAnchorsProvider(widget.travelProfile).notifier)
              .toggleAutoRouting(),
        ),
      ),
    );
  }
}
