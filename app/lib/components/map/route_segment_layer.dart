import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/util/route/segment.dart';

/// Owns the native MapLibre GL style layers/source that draw the route
/// planner's segment lines: one `GeoJsonSource` feeding three `state`-
/// filtered `LineStyleLayer`s (routed/straight/blocked) plus one invisible,
/// wide hit-test layer spanning all states.
///
/// Not a widget — a small stateful helper a screen calls into from its own
/// `onStyleLoaded`/mutation-listener lifecycle.
class RouteSegmentLayer {
  RouteSegmentLayer({
    this.sourceId = 'route-segments',
    this.hitLayerId = 'route-segments-hit',
  });

  final String sourceId;
  final String hitLayerId;

  static const _routedCasingLayerId = 'route-segments-routed-casing';
  static const _routedLayerId = 'route-segments-routed';
  static const _straightLayerId = 'route-segments-straight';
  static const _blockedLayerId = 'route-segments-blocked';

  bool _added = false;

  /// Draws/updates the segment lines for [segments].
  ///
  /// On first call, adds the `GeoJsonSource` and all five style layers (draw
  /// order: routed casing under routed, so 2px of white shows as an
  /// outline). On subsequent calls, updates the existing source's data in
  /// place via [ml.StyleController.updateGeoJsonSource] to avoid flicker.
  Future<void> update(
    ml.StyleController style,
    List<RouteSegment> segments,
  ) async {
    final data = buildSegmentsGeoJson(segments);

    if (!_added) {
      await style.addSource(ml.GeoJsonSource(id: sourceId, data: data));

      await style.addLayer(
        ml.LineStyleLayer(
          id: _routedCasingLayerId,
          sourceId: sourceId,
          filter: const [
            '==',
            ['get', 'state'],
            'routed',
          ],
          layout: const <String, Object>{
            'line-cap': 'round',
            'line-join': 'round',
          },
          paint: const <String, Object>{
            'line-color': '#ffffff',
            'line-width': 9,
          },
        ),
      );
      await style.addLayer(
        ml.LineStyleLayer(
          id: _routedLayerId,
          sourceId: sourceId,
          filter: const [
            '==',
            ['get', 'state'],
            'routed',
          ],
          layout: const <String, Object>{
            'line-cap': 'round',
            'line-join': 'round',
          },
          paint: const <String, Object>{
            'line-color': '#3549bb',
            'line-width': 5,
          },
        ),
      );
      await style.addLayer(
        ml.LineStyleLayer(
          id: _straightLayerId,
          sourceId: sourceId,
          filter: const [
            '==',
            ['get', 'state'],
            'straight',
          ],
          layout: const <String, Object>{
            'line-cap': 'round',
            'line-join': 'round',
          },
          paint: const <String, Object>{
            'line-color': '#3549bb',
            'line-width': 3,
            'line-opacity': 0.55,
          },
        ),
      );
      await style.addLayer(
        ml.LineStyleLayer(
          id: _blockedLayerId,
          sourceId: sourceId,
          filter: const [
            '==',
            ['get', 'state'],
            'blocked',
          ],
          layout: const <String, Object>{
            'line-cap': 'round',
            'line-join': 'round',
          },
          paint: const <String, Object>{
            'line-color': '#EF5350',
            'line-width': 3,
            'line-dasharray': [2, 2],
          },
        ),
      );

      // Invisible wide tap-detection layer, unfiltered so it matches
      // segments in every state — the visible lines are too thin (3-5px) to
      // tap reliably, so a tap handler queries this 24px layer instead via
      // `featuresAtPoint(point, layerIds: [hitLayerId])`.
      //
      // line-opacity is 0.01, not 0: native featuresAtPoint implementations
      // skip fully transparent (opacity: 0) layers entirely, so taps would
      // silently miss. 0.01 is visually indistinguishable but stays queryable.
      await style.addLayer(
        ml.LineStyleLayer(
          id: hitLayerId,
          sourceId: sourceId,
          paint: const <String, Object>{
            'line-color': '#000000',
            'line-width': 24,
            'line-opacity': 0.01,
          },
        ),
      );

      _added = true;
      return;
    }

    // Already added — update the source's data in place rather than
    // removing/re-adding the source or layers.
    await style.updateGeoJsonSource(id: sourceId, data: data);
  }

  /// Removes the layers/source added by [update], in reverse order. Safe to
  /// call even if [update] was never called — each removal is individually
  /// guarded since removeLayer/removeSource throw on an unknown id.
  Future<void> remove(ml.StyleController style) async {
    for (final id in [
      hitLayerId,
      _blockedLayerId,
      _straightLayerId,
      _routedLayerId,
      _routedCasingLayerId,
    ]) {
      try {
        await style.removeLayer(id);
      } catch (_) {}
    }
    try {
      await style.removeSource(sourceId);
    } catch (_) {}
    _added = false;
  }
}
