import 'package:maplibre/maplibre.dart' as ml;

/// GeoJSON source id the cluster style layers below are drawn from.
const String kClusterSourceId = 'cluster-trails';

/// Adds the `clusters` circle layer + `cluster-count` symbol layer over a new
/// `'cluster-trails'` `GeoJsonSource` (CLUS-02). Filter/paint/layout values
/// are ported verbatim from `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts`
/// (D-04) — do not restyle.
///
/// Deliberately does NOT port `cluster-layer.ts`'s `point_count == 1` circle
/// layer (D-05, explicit user correction overriding a literal verbatim port):
/// individual trail markers stay category-icon `WidgetLayer` markers rendered
/// by `map_screen` (16-03), not a native circle — porting that native
/// single-point circle would flatten today's category glyph to a plain dot.
///
/// `[geojson]` is a JSON-encoded `FeatureCollection` string (`GeoJsonSource.data`
/// is a `String`, not a typed object — RESEARCH Pitfall 5). Call from
/// `onStyleLoaded` so the source/layers are re-added after every style swap
/// (CORE-02 theme toggle rebuilds the style and drops added layers/sources).
Future<void> addClusterLayers(ml.StyleController style, String geojson) async {
  await style.addSource(ml.GeoJsonSource(id: kClusterSourceId, data: geojson));

  // `is_large` trails are filtered out of both layers below (D-03) — matching
  // web, no client-side extra filtering beyond what the layer filter already
  // does.
  await style.addLayer(
    const ml.CircleStyleLayer(
      id: 'clusters',
      sourceId: kClusterSourceId,
      filter: <Object>[
        'all',
        <Object>[
          '!=',
          <Object>['get', 'is_large'],
          true,
        ],
        <Object>[
          '>',
          <Object>['get', 'point_count'],
          1,
        ],
      ],
      paint: <String, Object>{
        'circle-color': '#242734',
        // Step ramp verbatim from cluster-layer.ts — radius by point_count.
        // dart format off
        'circle-radius': <Object>[
          'step', <Object>['get', 'point_count'],
          10, 5, 12, 10, 15, 50, 18, 100, 22, 500, 25,
        ],
        // dart format on
        'circle-stroke-width': 2,
        'circle-stroke-color': '#fff',
      },
    ),
  );

  await style.addLayer(
    const ml.SymbolStyleLayer(
      id: 'cluster-count',
      sourceId: kClusterSourceId,
      filter: <Object>[
        'all',
        <Object>[
          '!=',
          <Object>['get', 'is_large'],
          true,
        ],
        <Object>[
          '>',
          <Object>['get', 'point_count'],
          1,
        ],
      ],
      layout: <String, Object>{
        'text-field': <Object>['get', 'point_count_abbreviated'],
        'text-font': <Object>['Noto Sans Regular'],
        'text-size': 11,
        'text-allow-overlap': true,
        'text-ignore-placement': true,
      },
      paint: <String, Object>{'text-color': '#fff'},
    ),
  );
}

/// Swaps the `'cluster-trails'` source's data on re-query (CLUS-04) without
/// removing/re-adding the source or its layers — avoids the id-collision and
/// flicker risk of `removeSource`/`addSource` churn (RESEARCH Anti-Patterns).
Future<void> updateClusterSource(ml.StyleController style, String geojson) =>
    style.updateGeoJsonSource(id: kClusterSourceId, data: geojson);
