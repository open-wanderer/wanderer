import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/util/route_segment_util.dart';

/// Owns the native MapLibre GL style layers/source that draw the route
/// planner's segment lines: one `GeoJsonSource` feeding three `state`-
/// filtered `LineStyleLayer`s (routed/straight/blocked, per UI-SPEC's
/// Segment Rendering Contract) plus one invisible, wide hit-test layer
/// spanning all states (Pitfall 4).
///
/// Mirrors `TrailLayer`'s plain-class shape (`trail_layer.dart`) — this is
/// not a widget, but a small stateful helper a screen calls into from its
/// own `onStyleLoaded`/mutation-listener lifecycle.
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
  /// On first call, adds the `GeoJsonSource` and all five style layers (in
  /// draw order: routed casing under routed, so 2px of white shows as an
  /// outline — matching `TrailLayer.add`'s ordering). On every subsequent
  /// call, updates the existing source's data in place via
  /// [ml.StyleController.updateGeoJsonSource] — never removes/re-adds the
  /// source, avoiding a flicker on every route mutation.
  Future<void> update(ml.StyleController style, List<RouteSegment> segments) async {
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
          paint: const <String, Object>{'line-color': '#ffffff', 'line-width': 9},
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
          paint: const <String, Object>{'line-color': '#3549bb', 'line-width': 5},
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

      // Invisible wide tap-detection layer, deliberately unfiltered so it
      // matches segments in every state. A thin (3-5px) rendered line is a
      // tiny, hard-to-tap native-GL target; this 24px-wide, zero-opacity
      // layer sharing the same source is queried by a tap handler via
      // `featuresAtPoint(point, layerIds: [hitLayerId])` instead of the
      // visible layers (Pitfall 4), mirroring D-04's 36px invisible
      // marker tap-radius precedent.
      await style.addLayer(
        ml.LineStyleLayer(
          id: hitLayerId,
          sourceId: sourceId,
          paint: const <String, Object>{'line-color': '#000000', 'line-width': 24, 'line-opacity': 0},
        ),
      );

      _added = true;
      return;
    }

    // Already added — update the source's data in place. Verified no-flicker
    // per RESEARCH.md: this never removes/re-adds the source or layers.
    await style.updateGeoJsonSource(id: sourceId, data: data);
  }

  /// Removes the layers/source added by [update], in reverse order (layers
  /// before the source they depend on). Safe to call even if [update] was
  /// never called or already removed — each removal is individually
  /// guarded since [ml.StyleController.removeLayer]/`removeSource` throw on
  /// an unknown id, mirroring `TrailLayer.remove`.
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
