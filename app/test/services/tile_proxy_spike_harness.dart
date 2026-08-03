// PROXY-03 on-device spike harness for Phase 25.1
// (local-http-tile-proxy-for-region-based-offline-map-rendering).
//
// This is NOT a `flutter test` unit test and NOT part of the production app
// -- it is a standalone Flutter entry point meant to be launched directly
// against a physical device with:
//
//   cd app && flutter run -t test/services/tile_proxy_spike_harness.dart
//
// Purpose: settle PROXY-03 -- does MapLibre Native reliably load a
// loopback-HTTP tile source (served by the REAL production TileProxyServer)
// while the device is offline, up to and including full airplane mode? This
// cannot be resolved by source reading (the HTTP fetch lives in MapLibre
// Native's C++/JNI/Swift core, outside the Dart surface), so a human must
// run this harness on real hardware. Settles PROXY-03 via this plan's
// `checkpoint:decision`.
//
// It starts the REAL `TileProxyServer` (`lib/services/tile_proxy_server.dart`)
// and composes the map's offline style through the REAL production
// `rewriteStyleForProxy` transform (`lib/util/region/offline_style_rewriter.dart`)
// -- exactly the wiring Plan 04 will add to `main.dart` -- so a pass here
// means the real pipeline works, not a bespoke test path.
//
// Exercises the five RESEARCH.md "Open Questions" item 1 test cases:
//   (a) tiles load online via the proxy
//   (b) tiles keep loading with Wi-Fi/cellular data OFF (airplane NOT
//       engaged)
//   (c) tiles keep loading with FULL airplane mode engaged -- the
//       load-bearing result
//   (d) a region finishing its download mid-session becomes visible
//       (reload regions + pan into the newly-covered area) WITHOUT
//       remounting this screen
//   (e) the Android cleartext (Plan 01) / iOS ATS (Plan 01) exceptions are
//       confirmed necessary/sufficient on real hardware (watch `adb logcat`
//       for `CLEARTEXT ... not permitted`, or iOS ATS failures, alongside
//       this harness's own log panel/debugPrint output)
//
// This file is throwaway -- deleted or ignored once PROXY-03 is settled by
// the checkpoint decision (Phase 27 legacy cleanup). It is intentionally
// kept out of `router_provider.dart` / production navigation (its location
// under `app/test/` keeps it off the shipped app), mirroring the
// `region_render_spike_harness.dart` (RENDER-03) / `tile_repository_manager_harness.dart`
// (23-06) precedents.

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
import 'package:wanderer/provider/region/tile_proxy_provider.dart';
import 'package:wanderer/services/tile_proxy_server.dart';
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

  // Start the REAL production proxy BEFORE runApp, and override the REAL
  // tileProxyBaseUrlProvider with its resolved baseUrl -- exactly the
  // wiring Plan 04 will add to main.dart. This harness never fakes the
  // server; a pass here is a pass on the real pipeline.
  final proxy = await TileProxyServer.start(store);
  debugPrint('[spike] TileProxyServer.start -> ${proxy.baseUrl}');

  runApp(
    ProviderScope(
      overrides: [
        objectBoxProvider.overrideWithValue(store),
        cookieJarProvider.overrideWithValue(jar),
        tileProxyBaseUrlProvider.overrideWithValue(proxy.baseUrl),
      ],
      child: const TileProxySpikeApp(),
    ),
  );
}

class TileProxySpikeApp extends StatelessWidget {
  const TileProxySpikeApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'PROXY-03 tile-proxy spike',
      home: const TileProxySpikeScreen(),
    );
  }
}

/// Runs the real `TileProxyServer` + real `rewriteStyleForProxy` against a
/// human-selected downloaded region and lets a human observe loopback-tile
/// rendering across the five airplane-mode test cases (see file header).
class TileProxySpikeScreen extends ConsumerStatefulWidget {
  const TileProxySpikeScreen({super.key});

  @override
  ConsumerState<TileProxySpikeScreen> createState() =>
      _TileProxySpikeScreenState();
}

class _TileProxySpikeScreenState extends ConsumerState<TileProxySpikeScreen> {
  final _serverUrlController = TextEditingController(
    text: 'https://demo.wanderer.to',
  );

  List<RegionEntity> _regions = [];
  RegionEntity? _selectedRegion;

  ml.MapController? _controller;

  /// The offline style composed by [_loadStyle] via the REAL production
  /// `rewriteStyleForProxy` -- a single static loopback XYZ source, no
  /// per-region duplication, since the proxy resolves per-tile coverage
  /// server-side. Null until a human taps "Load map".
  String? _styleJson;
  bool _loadingStyle = false;

  /// On-screen log mirroring [debugPrint] (test case e: enough detail on
  /// real hardware to confirm the Android cleartext / iOS ATS exceptions
  /// are actually necessary/sufficient, without needing a host machine for
  /// `adb logcat`/Xcode console in hand at all times).
  final List<String> _log = [];

  @override
  void dispose() {
    _serverUrlController.dispose();
    super.dispose();
  }

  void _appendLog(String message) {
    debugPrint('[spike] $message');
    setState(() {
      _log.insert(0, message);
      if (_log.length > 40) _log.removeLast();
    });
  }

  void _connect() {
    final url = _serverUrlController.text.trim();
    ref.read(apiProvider.notifier).updateBaseUrl(url);
    _appendLog('api baseUrl -> $url');
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
    _appendLog(
      'reloaded ${regions.length} region(s) -- test case (d): after a '
      'second region finishes downloading, tap this again then pan into '
      'the newly-covered area WITHOUT remounting this screen and watch '
      'whether tiles appear',
    );
  }

  /// Composes the offline style through the REAL production
  /// `rewriteStyleForProxy` transform -- mirrors the body Plan 04 will give
  /// `TrailMap`/`navigation_screen`'s `_composeStyle`. The resulting style
  /// carries exactly one static `<proxyBaseUrl>/vector/{z}/{x}/{y}.pbf`
  /// source (and, if a DEM archive is downloaded anywhere, one
  /// `.../dem/{z}/{x}/{y}.png` source) -- the proxy itself resolves per-tile
  /// region coverage server-side ([resolveRegionForTile]), so this composed
  /// style is NOT region-specific; the region picker only drives which
  /// region the map flies to.
  Future<void> _loadStyle() async {
    setState(() => _loadingStyle = true);
    try {
      final baseJson = await ref.read(mapStyleJsonProvider.future);
      final cache = await ref.read(glyphSpriteCacheProvider.future);
      final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
      final proxyBaseUrl = ref.read(tileProxyBaseUrlProvider);

      final composed = rewriteStyleForProxy(
        decoded,
        cacheRoot: cache.root,
        proxyBaseUrl: proxyBaseUrl,
        dark: false,
      );

      _appendLog(
        'style composed via rewriteStyleForProxy -- proxy=$proxyBaseUrl '
        '(test case a: with normal connectivity, confirm basemap + '
        'hillshade render through the proxy now)',
      );
      if (!mounted) return;
      setState(() {
        _styleJson = jsonEncode(composed);
        _loadingStyle = false;
      });
    } catch (e) {
      _appendLog('style compose FAILED: $e');
      if (!mounted) return;
      setState(() => _loadingStyle = false);
    }
  }

  /// Flies the already-mounted map to the selected region's bounds -- used
  /// so a human can move between regions (including a newly-downloaded one,
  /// test case d) without ever remounting the `ml.MapLibreMap` widget.
  Future<void> _flyToSelectedRegion() async {
    final controller = _controller;
    final region = _selectedRegion;
    if (controller == null || region == null) return;
    await controller.fitBounds(
      bounds: ml.LngLatBounds(
        longitudeWest: region.minLon,
        longitudeEast: region.maxLon,
        latitudeSouth: region.minLat,
        latitudeNorth: region.maxLat,
      ),
      nativeDuration: const Duration(milliseconds: 1),
    );
    _appendLog(
      'camera flown to "${region.path}" bounds -- test case (b): now turn '
      'Wi-Fi + cellular data OFF (airplane mode NOT engaged) and pan/zoom '
      'here; test case (c): then engage FULL airplane mode and repeat -- '
      'this is the load-bearing result',
    );
  }

  Widget _buildMap() {
    final styleJson = _styleJson;
    if (styleJson == null) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(16),
          child: Text(
            'Connect, Reload regions, select a downloaded region, then tap '
            '"Load map (rewriteStyleForProxy)" to start the map against '
            'the real loopback proxy.',
            textAlign: TextAlign.center,
          ),
        ),
      );
    }
    return ml.MapLibreMap(
      options: ml.MapOptions(
        initStyle: styleJson,
        initCenter: ml.Geographic(lat: 47.0, lon: 8.0),
        initZoom: 8,
        gestures: const ml.MapGestures.all(),
      ),
      onMapCreated: (controller) => _controller = controller,
      onStyleLoaded: (_) => _appendLog('onStyleLoaded fired'),
      layers: const [],
      children: const [],
    );
  }

  @override
  Widget build(BuildContext context) {
    final proxyBaseUrl = ref.watch(tileProxyBaseUrlProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('PROXY-03 tile-proxy spike')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Proxy baseUrl: $proxyBaseUrl',
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
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
                const SizedBox(height: 8),
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: [
                    ElevatedButton(
                      onPressed: _loadingStyle ? null : _loadStyle,
                      child: Text(
                        _loadingStyle
                            ? 'Loading style...'
                            : 'Load map (rewriteStyleForProxy)',
                      ),
                    ),
                    ElevatedButton(
                      onPressed: _selectedRegion == null
                          ? null
                          : _flyToSelectedRegion,
                      child: const Text('Fly to selected region'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                const Text(
                  'Log (newest first) -- watch for proxy 404s / connection '
                  'errors, and cross-reference with adb logcat '
                  '(Android CLEARTEXT ... not permitted) or the iOS console '
                  '(ATS failures) for test case (e):',
                  style: TextStyle(fontWeight: FontWeight.bold),
                ),
                SizedBox(
                  height: 96,
                  child: ListView(
                    padding: const EdgeInsets.symmetric(vertical: 4),
                    children: [
                      for (final entry in _log)
                        Text(entry, style: const TextStyle(fontSize: 11)),
                    ],
                  ),
                ),
              ],
            ),
          ),
          Expanded(child: _buildMap()),
        ],
      ),
    );
  }
}
