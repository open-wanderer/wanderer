import 'package:maplibre/maplibre.dart' as ml;

/// GeoJSON source id the cluster style layers below are drawn from.
const String kClusterSourceId = 'cluster-trails';

/// Adds the `clusters` circle layer + `cluster-count` symbol layer over a new
/// `'cluster-trails'` `GeoJsonSource`. Filter/paint/layout values
/// are ported verbatim from `web/src/lib/vendor/maplibre-layer-manager/cluster-layer.ts`
/// — do not restyle.
///
/// Deliberately does NOT port `cluster-layer.ts`'s `point_count == 1` circle
/// layer: individual trail markers stay category-icon `WidgetLayer` markers
/// rendered by `map_screen`, not a native circle — porting that native
/// single-point circle would flatten today's category glyph to a plain dot.
///
/// `[geojson]` is a JSON-encoded `FeatureCollection` string (`GeoJsonSource.data`
/// is a `String`, not a typed object). Call from
/// `onStyleLoaded` so the source/layers are re-added after every style swap
/// (a theme toggle rebuilds the style and drops added layers/sources).
Future<void> addClusterLayers(ml.StyleController style, String geojson) async {
  await style.addSource(ml.GeoJsonSource(id: kClusterSourceId, data: geojson));

  // The `is_large` filter below is inert for practical purposes: is_large
  // features always carry point_count:1 (server-set), so the `point_count >
  // 1` clause already excludes them from these two layers regardless. Kept
  // for parity with web's filter shape. Unclustered category-icon markers
  // (map_screen.dart) intentionally do NOT filter on is_large
  // — see that file's comment for why.
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

/// Swaps the `'cluster-trails'` source's data on re-query without
/// removing/re-adding the source or its layers — avoids the id-collision and
/// flicker risk of `removeSource`/`addSource` churn.
Future<void> updateClusterSource(ml.StyleController style, String geojson) =>
    style.updateGeoJsonSource(id: kClusterSourceId, data: geojson);
