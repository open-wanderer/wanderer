// RENDER-03 on-device spike harness for Phase 25 (map-rendering
// region-based-viewport-pipeline).
//
// This is NOT a `flutter test` unit test and NOT part of the production app
// -- it is a standalone Flutter entry point meant to be launched directly
// against a physical device with:
//
//   cd app && flutter run -t test/services/region_render_spike_harness.dart
//
// Purpose: materialize 10-20 duplicated region vector (and DEM) style
// sources+layers against a real maplibre 0.3.5 map -- the realistic upper
// bound for a trail spanning several downloaded regions x the basemap's
// per-cell layer duplication -- and let a human measure both the
// full-style-reload (`MapController.setStyle`) and incremental
// (`StyleController.addSource`/`removeSource`) region-swap paths. Settles
// RENDER-03 (composition strategy) via this plan's `checkpoint:decision`.
//
// It reuses the REAL production composition helper, `rewriteStyleForOffline`
// (`lib/util/offline_style_rewriter.dart`): feeding it N identical real
// on-disk cell paths materializes exactly N duplicated `__cellK`-suffixed
// native source+layer sets, which is the actual multi-cell composition the
// rest of Phase 25 will ship.
//
// This file is throwaway -- deleted or ignored once RENDER-03 is settled by
// the checkpoint decision. It is intentionally kept out of
// `router_provider.dart` / production navigation (its location under
// `app/test/` keeps it off the shipped app), mirroring the 23-06
// `tile_repository_manager_harness.dart` precedent.

import 'dart:convert';
import 'dart:io';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/cookie_jar_provider.dart';
import 'package:wanderer/provider/glyph_sprite_cache_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/util/region/offline_style_rewriter.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final appDocDir = await getApplicationDocumentsDirectory();

  final dbPath = p.join(appDocDir.path, 'objectbox');
  final store = await openStore(directory: dbPath);

  final cookiePath = p.join(appDocDir.path, '.cookies');
  final cookieDir = Directory(cookiePath);
  if (!await cookieDir.exists()) {
    await cookieDir.create(recursive: true);
  }
  final jar = PersistCookieJar(
    storage: FileStorage(cookiePath),
    ignoreExpires: false,
  );

  runApp(
    ProviderScope(
      overrides: [
        objectBoxProvider.overrideWithValue(store),
        cookieJarProvider.overrideWithValue(jar),
      ],
      child: const RegionRenderSpikeApp(),
    ),
  );
}

class RegionRenderSpikeApp extends StatelessWidget {
  const RegionRenderSpikeApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'RENDER-03 region-render spike',
      home: const RegionRenderSpikeScreen(),
    );
  }
}

/// Materializes N duplicated region vector/DEM style sources+layers (10-20,
/// human-adjustable) via the production [rewriteStyleForOffline] helper, and
/// exposes three measured actions: full-style-reload `setStyle`, incremental
/// `addSource`/`addLayer`, and incremental `removeLayer`/`removeSource`.
class RegionRenderSpikeScreen extends ConsumerStatefulWidget {
  const RegionRenderSpikeScreen({super.key});

  @override
  ConsumerState<RegionRenderSpikeScreen> createState() =>
      _RegionRenderSpikeScreenState();
}

class _RegionRenderSpikeScreenState
    extends ConsumerState<RegionRenderSpikeScreen> {
  final _serverUrlController = TextEditingController(
    text: 'http://10.0.2.2:8090',
  );

  List<RegionEntity> _regions = [];
  RegionEntity? _selectedRegion;

  /// Duplicated source/layer count, human-adjustable 10..20 per ROADMAP
  /// success criterion 1.
  int _n = 10;

  ml.MapController? _controller;

  /// Buffers a style-loaded event that arrives before [_controller] is set --
  /// the native channel doesn't reliably fire onMapCreated first (same race
  /// class as TrailMap/SearchMap).
  ml.StyleController? _pendingStyle;

  /// Native source ids added by [_incrementalSwap] and not yet removed by
  /// [_removeAllIncremental] -- skipped on re-add (Pitfall 2: addSource
  /// throws on a duplicate id).
  final Set<String> _addedSourceIds = {};

  /// Native layer ids added alongside [_addedSourceIds]; removed before their
  /// source in [_removeAllIncremental] (Pitfall 3: removeLayer/removeSource
  /// have no missing-id guard on Android).
  final Set<String> _addedLayerIds = {};

  String? _lastMeasurement;

  @override
  void dispose() {
    _serverUrlController.dispose();
    super.dispose();
  }

  void _connect() {
    ref
        .read(apiProvider.notifier)
        .updateBaseUrl(_serverUrlController.text.trim());
    debugPrint('[spike] api baseUrl -> ${_serverUrlController.text.trim()}');
  }

  void _reloadRegions() {
    final store = ref.read(objectBoxProvider);
    final regions = store.box<RegionEntity>().getAll();
    setState(() {
      _regions = regions;
      if (_selectedRegion != null &&
          !regions.any((r) => r.obxId == _selectedRegion!.obxId)) {
        _selectedRegion = null;
      }
    });
    debugPrint('[spike] loaded ${regions.length} region(s) from ObjectBox');
  }

  /// Composes the duplicated style: obtains the real base style + glyph/
  /// sprite cache root, picks the selected region's downloaded vector (and,
  /// if present, DEM) archive path, and calls the production
  /// [rewriteStyleForOffline] helper with [n] identical real cell paths --
  /// which duplicates each cell into its own `__cellK`-suffixed native
  /// source+layer set, materializing exactly n duplicated sets. Returns null
  /// (and logs why) when no region is selected or it has no downloaded
  /// vector archive yet.
  Future<Map<String, dynamic>?> _composeDuplicatedStyle(int n) async {
    final region = _selectedRegion;
    if (region == null) {
      debugPrint('[spike] no region selected');
      return null;
    }
    final vectorPath = region.vectorPackage.target?.localFilePath;
    if (vectorPath == null) {
      debugPrint(
        '[spike] ${region.path} has no downloaded vector archive -- '
        'download it first via Settings -> Offline Maps/Regions',
      );
      return null;
    }
    final demPath = region.demPackage.target?.localFilePath;

    final baseJson = await ref.read(mapStyleJsonProvider.future);
    final cache = await ref.read(glyphSpriteCacheProvider.future);
    final decoded = jsonDecode(baseJson) as Map<String, dynamic>;

    return rewriteStyleForOffline(
      decoded,
      cacheRoot: cache.root,
      cellPaths: List.filled(n, vectorPath),
      demCellPaths: demPath == null ? const [] : List.filled(n, demPath),
      dark: false,
    );
  }

  /// Strategy (a): full-style-reload. Times the `setStyle` call itself with a
  /// [Stopwatch] -- the human observes the actual visual settle time/flicker
  /// on-device, which this call-latency number does not capture on its own.
  Future<void> _fullReloadSwap() async {
    final controller = _controller;
    if (controller == null) return;
    final style = await _composeDuplicatedStyle(_n);
    if (style == null) return;
    final json = jsonEncode(style);

    final stopwatch = Stopwatch()..start();
    controller.setStyle(json);
    stopwatch.stop();

    final msg =
        'Full reload N=$_n: setStyle() call took '
        '${stopwatch.elapsedMilliseconds}ms (judge visual flicker/settle '
        'time on-device, this is call latency only)';
    debugPrint('[spike] $msg');
    setState(() => _lastMeasurement = msg);
  }

  /// Strategy (b): incremental add. Iterates the duplicated style's sources
  /// (cell 0 plus every `-cell-<i>` clone), skipping any id already tracked
  /// (Pitfall 2), and calls `addSource` + a representative `addLayer` for
  /// each -- `VectorSource`/`FillStyleLayer` for tiled vector cells,
  /// `RasterDemSource`/`HillshadeStyleLayer` for the DEM cell. The url comes
  /// straight from [rewriteStyleForOffline]'s output
  /// (`pmtiles://file://<path>`), so this explicitly exercises whether that
  /// scheme resolves for an incrementally-added source (Open Question 1).
  Future<void> _incrementalSwap() async {
    final style = _controller?.style;
    if (style == null) return;
    final composed = await _composeDuplicatedStyle(_n);
    if (composed == null) return;
    final sources = composed['sources'];
    if (sources is! Map<String, dynamic>) return;

    final stopwatch = Stopwatch()..start();
    var added = 0;
    for (final entry in sources.entries) {
      final id = entry.key;
      if (_addedSourceIds.contains(id)) continue;
      final source = entry.value;
      if (source is! Map) continue;
      final url = source['url'] as String?;
      if (url == null) continue;
      final maxZoom = (source['maxzoom'] as num?)?.toDouble() ?? 14;

      try {
        if (source['type'] == 'raster-dem') {
          await style.addSource(
            ml.RasterDemSource(
              id: id,
              url: url,
              maxZoom: maxZoom,
              tileSize: (source['tileSize'] as num?)?.toInt() ?? 512,
              encoding: const ml.RasterDemTerrariumEncoding(),
            ),
          );
          final layerId = '$id-hillshade';
          await style.addLayer(
            ml.HillshadeStyleLayer(id: layerId, sourceId: id),
          );
          _addedLayerIds.add(layerId);
        } else {
          await style.addSource(
            ml.VectorSource(id: id, url: url, maxZoom: maxZoom),
          );
          final layerId = '$id-fill';
          await style.addLayer(
            ml.FillStyleLayer(
              id: layerId,
              sourceId: id,
              sourceLayerId: 'earth',
              paint: const {'fill-color': '#4caf50'},
            ),
          );
          _addedLayerIds.add(layerId);
        }
        _addedSourceIds.add(id);
        added++;
      } catch (e) {
        debugPrint('[spike] incremental addSource/addLayer failed for $id: $e');
      }
    }
    stopwatch.stop();

    final msg =
        'Incremental add N=$_n: +$added source(s) in '
        '${stopwatch.elapsedMilliseconds}ms (tracked=${_addedSourceIds.length})';
    debugPrint('[spike] $msg');
    setState(() => _lastMeasurement = msg);
  }

  /// Strategy (b) teardown: removes every tracked layer before its source
  /// (Pitfall 3 -- Android's removeSource/removeLayer have no missing-id
  /// guard, so each call is wrapped in try/catch), then clears the tracking
  /// sets so a subsequent [_incrementalSwap] can re-add the same ids.
  Future<void> _removeAllIncremental() async {
    final style = _controller?.style;
    if (style == null) return;

    final stopwatch = Stopwatch()..start();
    for (final layerId in _addedLayerIds.toList()) {
      try {
        await style.removeLayer(layerId);
      } catch (e) {
        debugPrint('[spike] removeLayer($layerId) failed: $e');
      }
    }
    for (final sourceId in _addedSourceIds.toList()) {
      try {
        await style.removeSource(sourceId);
      } catch (e) {
        debugPrint('[spike] removeSource($sourceId) failed: $e');
      }
    }
    final removed = _addedSourceIds.length;
    _addedLayerIds.clear();
    _addedSourceIds.clear();
    stopwatch.stop();

    final msg =
        'Incremental remove: cleared $removed source(s) in '
        '${stopwatch.elapsedMilliseconds}ms';
    debugPrint('[spike] $msg');
    setState(() => _lastMeasurement = msg);
  }

  Widget _buildMap(String baseJson) {
    return ml.MapLibreMap(
      options: ml.MapOptions(
        initStyle: baseJson,
        initCenter: ml.Geographic(lat: 47.0, lon: 8.0),
        initZoom: 8,
        gestures: const ml.MapGestures.all(),
      ),
      onMapCreated: (controller) {
        _controller = controller;
        final pending = _pendingStyle;
        if (pending != null) {
          _pendingStyle = null;
        }
      },
      onStyleLoaded: (style) {
        if (_controller == null) {
          _pendingStyle = style;
        }
      },
      layers: const [],
      children: const [],
    );
  }

  @override
  Widget build(BuildContext context) {
    final baseAsync = ref.watch(mapStyleJsonProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('RENDER-03 region-render spike')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: _serverUrlController,
                        decoration: const InputDecoration(
                          labelText: 'Backend base URL',
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    ElevatedButton(
                      onPressed: _connect,
                      child: const Text('Connect'),
                    ),
                    const SizedBox(width: 8),
                    OutlinedButton(
                      onPressed: _reloadRegions,
                      child: const Text('Reload regions'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                if (_regions.isEmpty)
                  const Text(
                    'No regions loaded yet. Connect, download at least one '
                    'region (ideally with DEM) via Settings -> Offline Maps/'
                    'Regions, then tap Reload regions.',
                  )
                else
                  DropdownButton<RegionEntity>(
                    isExpanded: true,
                    value: _selectedRegion,
                    hint: const Text('Select a downloaded region'),
                    items: [
                      for (final region in _regions)
                        DropdownMenuItem(
                          value: region,
                          child: Text(
                            '${region.name} (${region.path}) -- vector='
                            '${region.vectorPackage.target?.localFilePath != null} '
                            'dem=${region.demPackage.target?.localFilePath != null}',
                          ),
                        ),
                    ],
                    onChanged: (region) =>
                        setState(() => _selectedRegion = region),
                  ),
                Row(
                  children: [
                    Text('N = $_n duplicated cell(s)'),
                    Expanded(
                      child: Slider(
                        value: _n.toDouble(),
                        min: 10,
                        max: 20,
                        divisions: 10,
                        label: '$_n',
                        onChanged: (v) => setState(() => _n = v.round()),
                      ),
                    ),
                  ],
                ),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: [
                    ElevatedButton(
                      onPressed: _fullReloadSwap,
                      child: const Text('Full reload (setStyle)'),
                    ),
                    ElevatedButton(
                      onPressed: _incrementalSwap,
                      child: const Text('Incremental add'),
                    ),
                    ElevatedButton(
                      onPressed: _removeAllIncremental,
                      child: const Text('Incremental remove'),
                    ),
                  ],
                ),
                if (_lastMeasurement != null) Text(_lastMeasurement!),
              ],
            ),
          ),
          Expanded(
            child: baseAsync.when(
              data: _buildMap,
              loading: () => const Center(child: CircularProgressIndicator()),
              error: (e, st) => Center(child: Text('style load failed: $e')),
            ),
          ),
        ],
      ),
    );
  }
}
